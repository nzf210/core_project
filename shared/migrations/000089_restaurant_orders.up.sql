-- Migration: 000089_restaurant_orders.up.sql
-- F072: Modular Business Workflow — Restaurant KOT & Table Orders

CREATE TABLE IF NOT EXISTS restaurant_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(50) NOT NULL,
    table_number VARCHAR(50) NOT NULL DEFAULT 'Bawa Pulang',
    customer_name VARCHAR(255) NOT NULL DEFAULT 'Tamu',
    customer_phone VARCHAR(50) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    total_amount BIGINT NOT NULL DEFAULT 0,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'unpaid',
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_restaurant_orders_tenant_order_no UNIQUE (tenant_id, order_no)
);

CREATE INDEX IF NOT EXISTS idx_restaurant_orders_tenant_status ON restaurant_orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_restaurant_orders_tenant_table ON restaurant_orders(tenant_id, table_number);
CREATE INDEX IF NOT EXISTS idx_restaurant_orders_created_at ON restaurant_orders(tenant_id, created_at DESC);
