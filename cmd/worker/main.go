package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/messaging"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type workerConfig struct {
	failRate float64
	minDelay time.Duration
	maxDelay time.Duration
	stateDir string
}

type tenantState struct {
	TenantID  string    `json:"tenant_id"`
	TaskID    string    `json:"task_id"`
	LastType  string    `json:"last_type"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatalf("invalid worker flags: %v", err)
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	if cfg.stateDir != "" {
		if err := os.MkdirAll(cfg.stateDir, 0o755); err != nil {
			log.Fatalf("create state directory: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	publisher := messaging.NewPublisher(rabbitURL)
	defer publisher.Close()

	run(ctx, rabbitURL, publisher, cfg)
}

func parseFlags() (workerConfig, error) {
	failRate := flag.Float64("fail-rate", 0, "probability of simulated task failure (0.0 to 1.0)")
	minDelayMs := flag.Int("min-delay-ms", 1000, "minimum simulated work delay in milliseconds")
	maxDelayMs := flag.Int("max-delay-ms", 5000, "maximum simulated work delay in milliseconds")
	stateDir := flag.String("state-dir", "/var/lib/tenant-worker", "directory for simulated tenant state")
	flag.Parse()

	if *failRate < 0 || *failRate > 1 {
		return workerConfig{}, fmt.Errorf("fail-rate must be between 0 and 1, got %v", *failRate)
	}
	if *minDelayMs < 0 {
		return workerConfig{}, fmt.Errorf("min-delay-ms must be >= 0, got %d", *minDelayMs)
	}
	if *maxDelayMs < *minDelayMs {
		return workerConfig{}, fmt.Errorf("max-delay-ms (%d) must be >= min-delay-ms (%d)", *maxDelayMs, *minDelayMs)
	}

	return workerConfig{
		failRate: *failRate,
		minDelay: time.Duration(*minDelayMs) * time.Millisecond,
		maxDelay: time.Duration(*maxDelayMs) * time.Millisecond,
		stateDir: *stateDir,
	}, nil
}

func run(ctx context.Context, rabbitURL string, publisher *messaging.Publisher, cfg workerConfig) {
	delay := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		started := time.Now()
		err := consumeTasks(ctx, rabbitURL, publisher, cfg)
		if ctx.Err() != nil {
			return
		}

		if time.Since(started) >= 30*time.Second {
			delay = time.Second
		}

		log.Printf("worker interrupted: %v; retry in %s", err, delay)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		delay = min(delay*2, 30*time.Second)
	}
}

func consumeTasks(
	ctx context.Context,
	rabbitURL string,
	publisher *messaging.Publisher,
	cfg workerConfig,
) error {
	conn, err := messaging.Connect(rabbitURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := messaging.SetupQueues(conn); err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(
		messaging.TaskQueue,
		"",
		false, // Manual ACK
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("Worker listening on", messaging.TaskQueue)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("task delivery channel closed")
			}
			if err := handleDelivery(ctx, delivery, publisher, cfg); err != nil {
				return err
			}
		}
	}
}

func handleDelivery(
	ctx context.Context,
	delivery amqp.Delivery,
	publisher *messaging.Publisher,
	cfg workerConfig,
) error {
	var task messaging.TaskEvent
	if err := json.Unmarshal(delivery.Body, &task); err != nil {
		log.Println("Rejecting invalid task JSON")
		return delivery.Reject(false)
	}

	if !validTaskEvent(task) {
		log.Println("Rejecting invalid task event")
		return delivery.Reject(false)
	}

	if err := publishProgress(ctx, publisher, task.ID, "in_progress"); err != nil {
		return err
	}

	if err := sleepWorkDelay(ctx, cfg.minDelay, cfg.maxDelay); err != nil {
		return err
	}

	finalStatus := "done"
	if cfg.failRate > 0 && rand.Float64() < cfg.failRate {
		finalStatus = "failed"
	} else if err := applyTenantState(cfg.stateDir, task); err != nil {
		log.Printf("failed writing tenant state for task %s: %v", task.ID, err)
		finalStatus = "failed"
	}

	if err := publishProgress(ctx, publisher, task.ID, finalStatus); err != nil {
		return err
	}

	if err := delivery.Ack(false); err != nil {
		return err
	}

	log.Printf("task processed: id=%s tenant_id=%s type=%s status=%s", task.ID, task.TenantID, task.Type, finalStatus)
	return nil
}

func validTaskEvent(task messaging.TaskEvent) bool {
	if _, err := uuid.Parse(task.ID); err != nil {
		return false
	}
	if _, err := uuid.Parse(task.TenantID); err != nil {
		return false
	}
	switch task.Type {
	case "deploy", "update", "destroy":
		return true
	default:
		return false
	}
}

func publishProgress(
	ctx context.Context,
	publisher *messaging.Publisher,
	taskID string,
	status string,
) error {
	event := messaging.ProgressEvent{
		ID:     taskID + ":" + status,
		TaskID: taskID,
		Status: status,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return publisher.Publish(ctx, messaging.ProgressQueue, event.ID, payload)
}

func sleepWorkDelay(ctx context.Context, minDelay, maxDelay time.Duration) error {
	delay := minDelay
	if maxDelay > minDelay {
		delay += time.Duration(rand.Int64N(int64(maxDelay - minDelay + 1)))
	}
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func applyTenantState(stateDir string, task messaging.TaskEvent) error {
	if stateDir == "" {
		return nil
	}

	statePath := filepath.Join(stateDir, task.TenantID+".json")
	if task.Type == "destroy" {
		err := os.Remove(statePath)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	data, err := json.MarshalIndent(tenantState{
		TenantID:  task.TenantID,
		TaskID:    task.ID,
		LastType:  task.Type,
		UpdatedAt: time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0o644)
}

