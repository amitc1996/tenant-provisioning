package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/application"
	"github.com/amitc1996/tenant-provisioning/internal/config"
	"github.com/amitc1996/tenant-provisioning/internal/consumer"
	"github.com/amitc1996/tenant-provisioning/internal/database"
	"github.com/amitc1996/tenant-provisioning/internal/handler"
	"github.com/amitc1996/tenant-provisioning/internal/messaging"
	"github.com/amitc1996/tenant-provisioning/internal/outbox"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	pool, err := database.Connect(
		context.Background(),
		cfg.DatabaseURL,
	)

	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL connection pool: %v", err)
	}

	defer pool.Close()
	repo := repository.New(pool)

	// Validate messaging configuration.
	if cfg.RabbitMQURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}

	// Initialize publisher.
	publisher := messaging.NewPublisher(
		cfg.RabbitMQURL,
	)

	// Initialize background dispatcher.
	dispatcher := outbox.New(
		repo,
		publisher,
	)

	service := application.New(repo)

	progressConsumer := consumer.NewProgressConsumer(
		cfg.RabbitMQURL,
		service,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	defer cancel()

	go dispatcher.Run(ctx)

	go progressConsumer.Run(ctx)

	apiHandler := handler.New(service)

	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.CustomRecovery(
		func(c *gin.Context, recovered any) {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code":    "internal_server_error",
						"message": "An unexpected error occurred",
					},
				},
			)
		},
	))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "control-plane",
		})
	})

	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)

		defer cancel()

		status := "ready"
		postgresStatus := "up"
		statusCode := http.StatusOK

		if err := pool.Ping(ctx); err != nil {
			status = "not_ready"
			postgresStatus = "down"
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status":  status,
			"service": "control-plane",
			"dependencies": gin.H{
				"postgresql": gin.H{
					"status": postgresStatus,
				},
			},
		})
	})

	// Initialize RabbitMQ connection.
	rabbitConn, err := messaging.Connect(cfg.RabbitMQURL)

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	defer rabbitConn.Close()

	// Initialize message queues.
	if err := messaging.SetupQueues(rabbitConn); err != nil {
		log.Fatalf("Failed to initialize RabbitMQ queues: %v", err)
	}

	log.Println("RabbitMQ connected successfully")

	// Register tenant and task APIs.
	apiHandler.RegisterRoutes(router)

	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
