CREATE TABLE affiliates (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(30) UNIQUE NOT NULL,
    bank_info JSONB,
    cash_balance_cents BIGINT NOT NULL DEFAULT 0,
    total_earnings_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_affiliates_user ON affiliates(user_id);
CREATE INDEX idx_affiliates_refcode ON affiliates(referral_code);

CREATE TABLE affiliate_earnings (
    id SERIAL PRIMARY KEY,
    affiliate_id INT NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(50) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id VARCHAR(100) NOT NULL,
    amount_cents BIGINT NOT NULL,
    commission_rate_percent INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE affiliate_withdrawals (
    id SERIAL PRIMARY KEY,
    affiliate_id INT NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    admin_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Lock for lifetime recurring
ALTER TABLE tenants ADD COLUMN referred_by_affiliate_id INT REFERENCES affiliates(id) ON DELETE SET NULL;
