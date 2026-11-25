CREATE TABLE inbox_events (
    id TEXT PRIMARY KEY,

    task_id UUID NOT NULL,

    status VARCHAR(20) NOT NULL,

    received_order BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,

    received_at TIMESTAMPTZ(3)
        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    processed_at TIMESTAMPTZ(3),

    CONSTRAINT inbox_task_fk FOREIGN KEY (task_id)
        REFERENCES tasks(id)
        ON DELETE RESTRICT,

    CONSTRAINT inbox_status_check CHECK (
        status IN (
            'in_progress',
            'done',
            'failed'
        )
    ),

    CONSTRAINT inbox_id_check CHECK (
        LENGTH(id) BETWEEN 1 AND 200
    )
);

CREATE INDEX idx_inbox_pending
    ON inbox_events(task_id, received_order)
    WHERE processed_at IS NULL;