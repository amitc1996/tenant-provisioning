package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
)

func (q *Queries) CreateTenant(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (
			id,
			slug,
			name,
			status,
			version,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := q.db.Exec(
		ctx,
		query,
		tenant.ID,
		tenant.Slug,
		tenant.Name,
		tenant.Status,
		tenant.Version,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	return mapTenantError(err)
}

func (q *Queries) GetTenantByID(
	ctx context.Context,
	id string,
) (*domain.Tenant, error) {

	query := `
		SELECT
			id::text,
			slug,
			name,
			status,
			version,
			created_at,
			updated_at
		FROM tenants
		WHERE id = $1
	`

	return scanTenant(q.db.QueryRow(ctx, query, id))
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTenant(row scanner) (*domain.Tenant, error) {
	var tenant domain.Tenant

	err := row.Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Status,
		&tenant.Version,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}

	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (q *Queries) ListTenants(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.Tenant, error) {

	query := `
		SELECT
			id::text,
			slug,
			name,
			status,
			version,
			created_at,
			updated_at
		FROM tenants
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.db.Query(
		ctx,
		query,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tenants := make([]domain.Tenant, 0)

	for rows.Next() {
		tenant, err := scanTenant(rows)

		if err != nil {
			return nil, err
		}

		tenants = append(tenants, *tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tenants, nil
}

func (q *Queries) GetTenantForUpdate(
	ctx context.Context,
	id string,
) (*domain.Tenant, error) {

	query := `
		SELECT
			id::text,
			slug,
			name,
			status,
			version,
			created_at,
			updated_at
		FROM tenants
		WHERE id = $1
		FOR UPDATE
	`

	return scanTenant(q.db.QueryRow(ctx, query, id))
}

func mapTenantError(err error) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" &&
			(pgErr.ConstraintName == "tenants_slug_unique" ||
				pgErr.ConstraintName == "tenants_slug_key") {

			return domain.ErrTenantAlreadyExists
		}
	}

	return err
}

func (q *Queries) UpdateTenant(
	ctx context.Context,
	tenant *domain.Tenant,
	expectedVersion int,
	expectedStatus domain.TenantStatus,
) error {

	query := `
		UPDATE tenants
		SET
			slug = $2,
			name = $3,
			status = $4,
			version = $5,
			updated_at = $6
		WHERE
			id = $1
			AND version = $7
			AND status = $8
	`

	result, err := q.db.Exec(
		ctx,
		query,
		tenant.ID,
		tenant.Slug,
		tenant.Name,
		tenant.Status,
		tenant.Version,
		tenant.UpdatedAt,
		expectedVersion,
		expectedStatus,
	)

	if err != nil {
		return mapTenantError(err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrTenantVersionConflict
	}

	return nil
}
