CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'accepted',
    created_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT tasks_tenant_fk FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE RESTRICT,

    CONSTRAINT tasks_type_check CHECK (
        type IN (
            'deploy',
            'update',
            'destroy'
        )
    ),

    CONSTRAINT tasks_status_check CHECK (
        status IN (
            'accepted',
            'in_progress',
            'done',
            'failed'
        )
    )
);

CREATE INDEX idx_tasks_tenant_id
    ON tasks(tenant_id);