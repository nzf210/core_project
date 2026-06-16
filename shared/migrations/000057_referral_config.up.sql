CREATE TABLE referral_config (
    id INT PRIMARY KEY CHECK (id = 1),
    discount_percent NUMERIC(5,2) NOT NULL DEFAULT 10.00 CHECK (discount_percent >= 0 AND discount_percent <= 100),
    commission_percent NUMERIC(5,2) NOT NULL DEFAULT 10.00 CHECK (commission_percent >= 0 AND commission_percent <= 100),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO referral_config (id, discount_percent, commission_percent)
VALUES (1, 10.00, 10.00)
ON CONFLICT (id) DO NOTHING;
