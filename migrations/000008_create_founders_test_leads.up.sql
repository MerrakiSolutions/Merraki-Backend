-- ============================================================================
-- MIGRATION: Create founders_test_leads table
-- ============================================================================

BEGIN;

CREATE TABLE founders_test_leads (
    id                      BIGSERIAL PRIMARY KEY,

    -- Contact (from TestContactScreen form)
    name                    TEXT        NOT NULL,
    email                   TEXT        NOT NULL,
    company                 TEXT,
    role                    TEXT,
    ip_address              INET,

    -- Test result (computed by frontend, stored verbatim)
    total_score             INTEGER     NOT NULL DEFAULT 0,
    total_max               INTEGER     NOT NULL DEFAULT 100,
    personality_type        TEXT        NOT NULL,
    personality_title       TEXT        NOT NULL,
    personality_badge       TEXT        NOT NULL,
    personality_color       TEXT        NOT NULL,
    personality_description TEXT        NOT NULL,
    section_scores          JSONB       NOT NULL DEFAULT '[]',

    -- Admin workflow
    status                  TEXT        NOT NULL DEFAULT 'new',
    notes                   TEXT,

    -- Timestamps
    submitted_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_ftl_email        ON founders_test_leads (email);
CREATE INDEX idx_ftl_status       ON founders_test_leads (status);
CREATE INDEX idx_ftl_submitted_at ON founders_test_leads (submitted_at DESC);
CREATE INDEX idx_ftl_personality  ON founders_test_leads (personality_type);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_ftl()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_ftl_updated_at
BEFORE UPDATE ON founders_test_leads
FOR EACH ROW EXECUTE FUNCTION set_updated_at_ftl();

COMMIT;