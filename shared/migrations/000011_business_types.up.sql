-- Business Type Catalog + Tenant Onboarding System
-- Supports: warung, laundry, industri_kreatif, toko_online, restoran, jasa, umum

CREATE TABLE IF NOT EXISTS business_types (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    default_modules JSONB NOT NULL DEFAULT '[]',
    default_dashboard_widgets JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO business_types (id, name, description, icon, default_modules, default_dashboard_widgets) VALUES
('umum', 'Umum / General', 'Untuk semua jenis usaha tanpa spesialisasi', 'store',
 '["transactions", "customers", "reports", "pos"]',
 '["income_summary", "expense_summary", "recent_transactions", "quick_actions"]'),

('warung', 'Warung / Toko Kelontong', 'Warung sembako, kelontong, dan toko retail kecil', 'shopping-bag',
 '["transactions", "customers", "reports", "pos", "inventory", "supplier", "best_seller"]',
 '["daily_sales", "best_selling_items", "stock_alert", "income_summary", "profit_margin"]'),

('laundry', 'Laundry', 'Jasa cuci baju, setrika, dry clean, dan laundry kiloan', 'shirt',
 '["transactions", "customers", "reports", "pos", "order_tracking", "package_pricing"]',
 '["active_orders", "daily_revenue", "package_breakdown", "customer_summary", "order_status_timeline"]'),

('industri_kreatif', 'Industri Kreatif', 'Desain grafis, fotografi, videografi, kerajinan, dan kreator konten', 'palette',
 '["transactions", "customers", "reports", "project_tracking", "material_costing", "invoice_generator"]',
 '["active_projects", "project_margin", "monthly_revenue", "material_spend", "invoice_status"]'),

('toko_online', 'Toko Online / E-Commerce', 'Penjual online di marketplace, dropship, atau toko sendiri', 'globe',
 '["transactions", "customers", "reports", "pos", "inventory", "shipment_tracking", "marketplace_sync"]',
 '["order_volume", "channel_breakdown", "revenue_trend", "top_products", "pending_shipments"]'),

('restoran', 'Restoran / F&B', 'Rumah makan, cafe, catering, dan bisnis kuliner', 'utensils',
 '["transactions", "customers", "reports", "pos", "menu_management", "ingredient_costing", "table_management"]',
 '["daily_revenue", "popular_items", "cost_ratio", "peak_hours", "table_turnover"]'),

('jasa', 'Jasa / Service', 'Konsultan, bengkel, salon, barber, dan jasa profesional', 'briefcase',
 '["transactions", "customers", "reports", "appointment_scheduling", "service_catalog", "staff_performance"]',
 '["appointments_today", "service_revenue", "top_services", "customer_retention", "staff_utilization"]')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_type VARCHAR(50) DEFAULT 'umum' REFERENCES business_types(id);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_name VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_address TEXT;

CREATE TABLE IF NOT EXISTS tenant_module_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_key VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, module_key)
);

CREATE INDEX idx_tenant_module_config_tenant ON tenant_module_config(tenant_id);

CREATE TABLE IF NOT EXISTS usage_quotas (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    plan_tier VARCHAR(20) NOT NULL DEFAULT 'free',
    transactions_used INT DEFAULT 0,
    transactions_limit INT DEFAULT 100,
    users_used INT DEFAULT 1,
    users_limit INT DEFAULT 1,
    ai_requests_used INT DEFAULT 0,
    ai_requests_limit INT DEFAULT 5,
    bots_used INT DEFAULT 0,
    bots_limit INT DEFAULT 0,
    period_start TIMESTAMPTZ DEFAULT NOW(),
    period_end TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '30 days'),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed usage quotas for existing tenants
INSERT INTO usage_quotas (tenant_id, plan_tier, transactions_limit, users_limit, ai_requests_limit, bots_limit)
SELECT id, COALESCE(plan, 'free'), 100, 1, 5, 0
FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;
