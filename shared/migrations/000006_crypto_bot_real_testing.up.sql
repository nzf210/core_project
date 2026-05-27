ALTER TABLE bots ADD COLUMN has_open_position BOOLEAN DEFAULT FALSE;

CREATE TABLE bot_grid_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    price BIGINT NOT NULL,
    side VARCHAR(10) NOT NULL, -- 'buy' or 'sell'
    status VARCHAR(20) NOT NULL, -- 'pending', 'active', 'filled'
    exchange_order_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_bot_grid_lines_bot_id ON bot_grid_lines(bot_id);
