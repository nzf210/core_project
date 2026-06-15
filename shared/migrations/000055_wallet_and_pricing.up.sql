CREATE TABLE addon_prices (
    addon_key VARCHAR(50) PRIMARY KEY,
    price_cents BIGINT NOT NULL DEFAULT 0,
    unit VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed default pricing
INSERT INTO addon_prices (addon_key, price_cents, unit) VALUES
    ('ai_vision', 500000, 'per_request'),        -- Rp 5.000
    ('ai_audio_stt', 700000, 'per_minute'),      -- Rp 7.000
    ('wa_blast_api', 200000, 'per_request'),     -- Rp 2.000
    ('wa_session_meta', 1500000, 'per_session'); -- Rp 15.000

CREATE TABLE wallet_credits (
    tenant_id VARCHAR(50) PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE wallet_transactions (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(50) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL, -- positive for topup, negative for consumption
    transaction_type VARCHAR(20) NOT NULL, -- 'topup', 'consume'
    reference VARCHAR(100) NOT NULL, -- e.g., 'ai_vision', 'invoice_123', 'wa_session_meta'
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wallet_transactions_tenant ON wallet_transactions(tenant_id);
