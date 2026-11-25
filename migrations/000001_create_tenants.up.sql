CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    slug VARCHAR(28) NOT NULL,
    name TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'provisioning',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT tenants_slug_unique UNIQUE (slug),

    CONSTRAINT tenants_slug_check CHECK (
        slug ~ '^[a-z][-a-z0-9]{1,26}[a-z0-9]$'
    ),

    CONSTRAINT tenants_name_check CHECK (
        LENGTH(TRIM(name)) > 0
    ),

    CONSTRAINT tenants_status_check CHECK (
        status IN (
            'provisioning',
            'active',
            'updating',
            'destroying',
            'destroyed',
            'failed'
        )
    ),

    CONSTRAINT tenants_version_check CHECK (
        version >= 1
    )
);