-- services/billing-service/schema.sql
-- Subscription plans definition
CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price_idr DECIMAL(15, 2) NOT NULL,
    max_bots INT DEFAULT 1,
    max_ai_requests INT DEFAULT 100,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO plans (id, name, price_idr, max_bots, max_ai_requests) VALUES
('lite', 'Lite', 150000, 3, 500),
('pro', 'Pro', 450000, 10, 5000),
('ultimate', 'Ultimate', 1500000, 50, 50000)
ON CONFLICT DO NOTHING;

-- Tenants Subscriptions
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    tenant_id VARCHAR(100) PRIMARY KEY,
    plan_id VARCHAR(50) REFERENCES plans(id),
    status VARCHAR(20) DEFAULT 'active', -- active, past_due, canceled
    current_period_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Invoices for Payment Tracking
CREATE TABLE IF NOT EXISTS invoices (
    id VARCHAR(100) PRIMARY KEY,
    tenant_id VARCHAR(100) NOT NULL,
    plan_id VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, paid, failed
    payment_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    paid_at TIMESTAMP
);

-- Note: Usage records are better stored in Redis for fast increments,
-- but a daily rollup table can be added here if needed.
