-- +goose Up
-- Create jobs table for job queue system
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Job identification
    queue_name VARCHAR(50) NOT NULL,
    job_type VARCHAR(100) NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Job data
    payload JSONB NOT NULL,

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    CHECK (status IN ('pending', 'processing', 'completed', 'failed')),

    -- Priority and scheduling
    priority INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Results and errors
    result JSONB,
    error_message TEXT,

    -- Worker tracking
    worker_id VARCHAR(100)
);

-- Index for efficient job dequeuing (SELECT FOR UPDATE SKIP LOCKED)
CREATE INDEX idx_jobs_dequeue ON jobs(queue_name, status, scheduled_at, priority DESC)
    WHERE status = 'pending';

-- Index for organization queries
CREATE INDEX idx_jobs_organization ON jobs(organization_id, created_at DESC);

-- Index for monitoring and stats
CREATE INDEX idx_jobs_status ON jobs(status, created_at DESC);

-- Index for processing jobs by organization
CREATE INDEX idx_jobs_processing ON jobs(organization_id, queue_name, status)
    WHERE status = 'processing';

-- Create organization_rate_limits table
CREATE TABLE organization_rate_limits (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    queue_name VARCHAR(50) NOT NULL,

    -- Tier configuration
    tier VARCHAR(50) NOT NULL DEFAULT 'free',

    -- Rate limits
    max_jobs_per_hour INTEGER NOT NULL DEFAULT 10,
    max_concurrent_jobs INTEGER NOT NULL DEFAULT 2,

    -- Sliding window tracking
    jobs_in_current_window INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (organization_id, queue_name)
);

-- Index for rate limit checks
CREATE INDEX idx_rate_limits_lookup ON organization_rate_limits(organization_id, queue_name);

-- +goose Down
DROP TABLE IF EXISTS organization_rate_limits;
DROP TABLE IF EXISTS jobs;
