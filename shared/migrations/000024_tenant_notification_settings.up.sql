CREATE TABLE IF NOT EXISTS tenant_notification_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    notify_email BOOLEAN DEFAULT TRUE,
    notify_wa BOOLEAN DEFAULT TRUE,
    notify_telegram BOOLEAN DEFAULT FALSE,
    telegram_chat_id VARCHAR(50),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
