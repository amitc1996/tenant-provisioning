CREATE TABLE outbox_events (
    task_id UUID PRIMARY KEY,

    payload JSONB NOT NULL,

    attempts INTEGER NOT NULL DEFAULT 0,

    next_attempt_at TIMESTAMPTZ(3)
        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    published_at TIMESTAMPTZ(3),

    created_at TIMESTAMPTZ(3)
        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT outbox_task_fk FOREIGN KEY (task_id)
        REFERENCES tasks(id)
        ON DELETE RESTRICT,

    CONSTRAINT outbox_attempts_check CHECK (
        attempts >= 0
    )
);

CREATE INDEX idx_outbox_pending
    ON outbox_events(next_attempt_at, created_at)
    WHERE published_at IS NULL;