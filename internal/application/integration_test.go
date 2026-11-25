package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/amitc1996/tenant-provisioning/internal/application"
	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/amitc1996/tenant-provisioning/internal/handler"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
)

func setupIntegration(
	t *testing.T,
) (*application.Service, *pgxpool.Pool) {

	t.Helper()

	ctx := context.Background()

	url := os.Getenv("TEST_DATABASE_URL")

	if url == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}

	admin, err := pgxpool.New(ctx, url)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(admin.Close)

	schema := "test_" + strings.ReplaceAll(
		uuid.NewString(),
		"-",
		"",
	)

	identifier := pgx.Identifier{schema}.Sanitize()

	_, err = admin.Exec(
		ctx,
		"CREATE SCHEMA "+identifier,
	)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, err := admin.Exec(
			context.Background(),
			"DROP SCHEMA "+identifier+" CASCADE",
		)

		if err != nil {
			t.Errorf("schema cleanup failed: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(url)

	if err != nil {
		t.Fatal(err)
	}

	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	files, err := filepath.Glob(
		"../../migrations/*.up.sql",
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("migration files not found")
	}

	for _, file := range files {

		content, err := os.ReadFile(file)

		if err != nil {
			t.Fatal(err)
		}

		if _, err := pool.Exec(
			ctx,
			string(content),
		); err != nil {
			t.Fatalf(
				"migration %s failed: %v",
				file,
				err,
			)
		}
	}

	repo := repository.New(pool)

	return application.New(repo), pool
}

func assertCounts(
	t *testing.T,
	pool *pgxpool.Pool,
	wantTenants int,
	wantTasks int,
	wantEvents int,
) {
	t.Helper()

	var tenants, tasks, events int

	err := pool.QueryRow(
		context.Background(),
		`
		SELECT
			(SELECT COUNT(*) FROM tenants),
			(SELECT COUNT(*) FROM tasks),
			(SELECT COUNT(*) FROM outbox_events)
		`,
	).Scan(&tenants, &tasks, &events)

	if err != nil {
		t.Fatal(err)
	}

	if tenants != wantTenants ||
		tasks != wantTasks ||
		events != wantEvents {

		t.Fatalf(
			"expected counts (%d,%d,%d), got (%d,%d,%d)",
			wantTenants,
			wantTasks,
			wantEvents,
			tenants,
			tasks,
			events,
		)
	}
}

func TestCreateTenantIntegration(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	result, err := service.CreateTenant(
		ctx,
		application.CreateTenantInput{
			Slug: "customer-one",
			Name: "Customer One",
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.Tenant.Status != domain.TenantStatusProvisioning {
		t.Fatal("unexpected tenant status")
	}

	if result.Task.Status != domain.TaskStatusAccepted {
		t.Fatal("unexpected task status")
	}

	if result.Task.Type != domain.TaskTypeDeploy {
		t.Fatal("unexpected task type")
	}

	if result.Task.TenantID != result.Tenant.ID {
		t.Fatal("incorrect tenant association")
	}

	assertCounts(t, pool, 1, 1, 1)

	var payload []byte

	err = pool.QueryRow(
		ctx,
		`SELECT payload FROM outbox_events WHERE task_id = $1`,
		result.Task.ID,
	).Scan(&payload)

	if err != nil {
		t.Fatal(err)
	}

	var event map[string]string

	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatal(err)
	}

	if event["id"] != result.Task.ID ||
		event["tenant_id"] != result.Tenant.ID ||
		event["type"] != "deploy" ||
		event["status"] != "accepted" {
		t.Fatal("incorrect outbox event payload")
	}
}

func TestDuplicateSlug(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	input := application.CreateTenantInput{
		Slug: "duplicate-tenant",
		Name: "Duplicate Tenant",
	}

	if _, err := service.CreateTenant(ctx, input); err != nil {
		t.Fatal(err)
	}

	_, err := service.CreateTenant(ctx, input)

	if !errors.Is(err, domain.ErrTenantAlreadyExists) {
		t.Fatalf("expected duplicate slug error, got %v", err)
	}

	assertCounts(t, pool, 1, 1, 1)
}

func TestConcurrentTenantCreation(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	const requests = 20

	start := make(chan struct{})
	results := make(chan error, requests)

	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {

		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			_, err := service.CreateTenant(
				ctx,
				application.CreateTenantInput{
					Slug: "concurrent-tenant",
					Name: "Concurrent Tenant",
				},
			)

			results <- err
		}()
	}

	close(start)

	wg.Wait()
	close(results)

	success := 0
	conflicts := 0

	for err := range results {

		switch {
		case err == nil:
			success++

		case errors.Is(
			err,
			domain.ErrTenantAlreadyExists,
		):
			conflicts++

		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if success != 1 || conflicts != requests-1 {
		t.Fatalf(
			"expected 1 success and %d conflicts; got %d and %d",
			requests-1,
			success,
			conflicts,
		)
	}

	assertCounts(t, pool, 1, 1, 1)
}

func completeTask(
	t *testing.T,
	service *application.Service,
	taskID string,
) {
	t.Helper()

	for _, status := range []domain.TaskStatus{
		domain.TaskStatusInProgress,
		domain.TaskStatusDone,
	} {

		err := service.ApplyTaskProgress(
			context.Background(),
			application.ProgressInput{
				EventID: taskID + ":" + string(status),
				TaskID:  taskID,
				Status:  status,
			},
		)

		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestConcurrentPatch(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	result, err := service.CreateTenant(
		ctx,
		application.CreateTenantInput{
			Slug: "patch-tenant",
			Name: "Patch Tenant",
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	completeTask(t, service, result.Task.ID)

	tenant, err := service.GetTenant(
		ctx,
		result.Tenant.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	if tenant.Version != 2 ||
		tenant.Status != domain.TenantStatusActive {
		t.Fatal("tenant must be active with version 2")
	}

	router := gin.New()

	handler.New(service).RegisterRoutes(router)

	const requests = 10

	type response struct {
		code int
		body string
	}

	results := make(chan response, requests)

	start := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {

		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			req := httptest.NewRequest(
				http.MethodPatch,
				"/api/v1/tenants/"+tenant.ID,
				strings.NewReader(
					`{"name":"Updated Tenant","version":2}`,
				),
			)

			req.Header.Set(
				"Content-Type",
				"application/json",
			)

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			results <- response{
				code: recorder.Code,
				body: recorder.Body.String(),
			}
		}()
	}

	close(start)

	wg.Wait()
	close(results)

	success := 0
	conflicts := 0

	for result := range results {

		switch result.code {

		case http.StatusAccepted:
			success++

		case http.StatusConflict:

			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}

			if err := json.Unmarshal(
				[]byte(result.body),
				&body,
			); err != nil {
				t.Fatal(err)
			}

			if body.Error.Code != "tenant_version_conflict" {
				t.Fatalf("unexpected conflict: %s", result.body)
			}

			conflicts++

		default:
			t.Fatalf(
				"unexpected response: %d %s",
				result.code,
				result.body,
			)
		}
	}

	if success != 1 || conflicts != requests-1 {
		t.Fatalf(
			"unexpected results: success=%d conflicts=%d",
			success,
			conflicts,
		)
	}

	updated, err := service.GetTenant(ctx, tenant.ID)

	if err != nil {
		t.Fatal(err)
	}

	if updated.Version != 3 ||
		updated.Status != domain.TenantStatusUpdating {
		t.Fatal("unexpected tenant state after PATCH")
	}

	assertCounts(t, pool, 1, 2, 2)

	var openTasks int

	err = pool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM tasks
		WHERE tenant_id = $1
		  AND status IN ('accepted', 'in_progress')
		`,
		tenant.ID,
	).Scan(&openTasks)

	if err != nil {
		t.Fatal(err)
	}

	if openTasks != 1 {
		t.Fatalf("expected one open task, got %d", openTasks)
	}
}

func TestDestroyedTenantSlug(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	input := application.CreateTenantInput{
		Slug: "destroyed-tenant",
		Name: "Destroyed Tenant",
	}

	result, err := service.CreateTenant(ctx, input)

	if err != nil {
		t.Fatal(err)
	}

	completeTask(t, service, result.Task.ID)

	deletion, err := service.DeleteTenant(
		ctx,
		result.Tenant.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	completeTask(t, service, deletion.Task.ID)

	tenant, err := service.GetTenant(
		ctx,
		result.Tenant.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	if tenant.Status != domain.TenantStatusDestroyed {
		t.Fatal("expected destroyed tenant")
	}

	if tenant.Version != 4 {
		t.Fatal("expected version 4")
	}

	_, err = service.CreateTenant(ctx, input)

	if !errors.Is(err, domain.ErrTenantAlreadyExists) {
		t.Fatalf("expected duplicate slug error, got %v", err)
	}

	assertCounts(t, pool, 1, 2, 2)
}

func TestCreateTenantRollback(t *testing.T) {
	service, pool := setupIntegration(t)

	ctx := context.Background()

	_, err := pool.Exec(
		ctx,
		`
		ALTER TABLE outbox_events
		ADD CONSTRAINT test_reject_outbox
		CHECK (false)
		`,
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateTenant(
		ctx,
		application.CreateTenantInput{
			Slug: "rollback-tenant",
			Name: "Rollback Tenant",
		},
	)

	if err == nil {
		t.Fatal("expected transaction failure")
	}

	assertCounts(t, pool, 0, 0, 0)
}
