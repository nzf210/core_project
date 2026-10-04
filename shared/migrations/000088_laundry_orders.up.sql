-- Migration: 000088_laundry_orders.up.sql
-- F071: Modular Business Workflow — Laundry Order & Wash Tracking

CREATE TABLE IF NOT EXISTS laundry_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(50) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50) NOT NULL,
    service_type VARCHAR(50) NOT NULL DEFAULT 'kiloan',
    weight_grams INT NOT NULL DEFAULT 0,
    item_count INT NOT NULL DEFAULT 0,
    rack_location VARCHAR(50) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'received',
    total_amount BIGINT NOT NULL DEFAULT 0,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'unpaid',
    estimated_completion_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_laundry_orders_tenant_order_no UNIQUE (tenant_id, order_no)
);

CREATE INDEX IF NOT EXISTS idx_laundry_orders_tenant_status ON laundry_orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_laundry_orders_tenant_phone ON laundry_orders(tenant_id, customer_phone);
CREATE INDEX IF NOT EXISTS idx_laundry_orders_created_at ON laundry_orders(tenant_id, created_at DESC);
