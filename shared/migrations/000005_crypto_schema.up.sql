-- =============================================
-- Crypto Trading Bot Schema
-- Migration: 000005_crypto_schema
-- Tables: exchange_api_keys, bots, bot_orders, bot_pnl_snapshots
-- =============================================

-- Exchange API Keys (encrypted with AES-256-GCM)
CREATE TABLE exchange_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,          -- 'binance', 'tokocrypto', 'indodax'
    label VARCHAR(100) NOT NULL,            -- user-defined label
    encrypted_api_key TEXT NOT NULL,         -- AES-256-GCM encrypted
    encrypted_api_secret TEXT NOT NULL,      -- AES-256-GCM encrypted
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_exchange_api_keys_user_id ON exchange_api_keys(user_id);
CREATE INDEX idx_exchange_api_keys_tenant_id ON exchange_api_keys(tenant_id);

-- Trading Bots
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id UUID REFERENCES exchange_api_keys(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    bot_type VARCHAR(20) NOT NULL,          -- 'dca', 'grid', 'signal'
    pair VARCHAR(20) NOT NULL,              -- e.g. 'BTCUSDT'
    status VARCHAR(20) NOT NULL DEFAULT 'stopped',  -- 'running', 'paused', 'stopped'
    is_paper_trading BOOLEAN NOT NULL DEFAULT true,  -- paper trading mode
    -- DCA Configuration
    dca_interval VARCHAR(20),               -- 'hourly', 'daily', 'weekly', 'monthly'
    dca_amount_per_order BIGINT DEFAULT 0,  -- amount in USDT rupiah (x100)
    -- Grid Configuration
    grid_lower_price BIGINT DEFAULT 0,      -- lower price bound in USDT rupiah
    grid_upper_price BIGINT DEFAULT 0,      -- upper price bound in USDT rupiah
    grid_count INT DEFAULT 0,               -- number of grid levels
    grid_investment BIGINT DEFAULT 0,       -- total grid investment in USDT rupiah
    -- Aggregated Stats
    total_invested BIGINT NOT NULL DEFAULT 0,   -- total invested in USDT rupiah
    total_profit BIGINT NOT NULL DEFAULT 0,     -- realized profit in USDT rupiah
    total_trades INT NOT NULL DEFAULT 0,
    last_executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bots_user_id ON bots(user_id);
CREATE INDEX idx_bots_tenant_id ON bots(tenant_id);
CREATE INDEX idx_bots_status ON bots(status);

-- Bot Orders (trade history)
CREATE TABLE bot_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    side VARCHAR(4) NOT NULL,               -- 'buy' or 'sell'
    price BIGINT NOT NULL,                  -- price in USDT rupiah (x100)
    quantity BIGINT NOT NULL,               -- quantity in satoshi-level (x10^8)
    total BIGINT NOT NULL,                  -- total cost in USDT rupiah
    fee BIGINT NOT NULL DEFAULT 0,          -- fee in USDT rupiah
    exchange_order_id VARCHAR(100),         -- ID from exchange (null for paper trades)
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'filled', 'failed', 'cancelled'
    is_paper BOOLEAN NOT NULL DEFAULT true, -- whether this is a simulated trade
    error_message TEXT,                     -- error details if failed
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bot_orders_bot_id ON bot_orders(bot_id);
CREATE INDEX idx_bot_orders_status ON bot_orders(status);
CREATE INDEX idx_bot_orders_executed_at ON bot_orders(executed_at);

-- PnL Snapshots (periodic performance snapshots for charts)
CREATE TABLE bot_pnl_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    total_invested BIGINT NOT NULL DEFAULT 0,   -- cumulative invested
    current_value BIGINT NOT NULL DEFAULT 0,    -- current portfolio value
    realized_pnl BIGINT NOT NULL DEFAULT 0,     -- realized profit/loss
    unrealized_pnl BIGINT NOT NULL DEFAULT 0,   -- unrealized profit/loss
    snapshot_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bot_pnl_snapshots_bot_id ON bot_pnl_snapshots(bot_id);
CREATE INDEX idx_bot_pnl_snapshots_snapshot_at ON bot_pnl_snapshots(snapshot_at);
