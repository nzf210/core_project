-- 000041_quota_counters.up.sql
-- Per-tenant per-feature counter with period key (monthly)

CREATE TABLE IF NOT EXISTS quota_counters (
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_yyyymm CHAR(6) NOT NULL,           -- '202606'
    feature_key  VARCHAR(50) NOT NULL,        -- 'ai_text', 'ai_vision', 'ai_audio_stt', 'ai_audio_tts', 'image_gen', 'ocr_scans', 'chatbot_messages'
    count        BIGINT NOT NULL DEFAULT 0,
    reset_at     TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, period_yyyymm, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_quota_counters_period ON quota_counters(period_yyyymm);
CREATE INDEX IF NOT EXISTS idx_quota_counters_reset ON quota_counters(reset_at);
