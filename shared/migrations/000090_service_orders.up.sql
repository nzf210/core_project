-- Migration: 000090_service_orders.up.sql
-- F073: Modular Business Workflow — Service Order & Repair Ticket (SPK)

CREATE TABLE IF NOT EXISTS service_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(50) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50) NOT NULL,
    unit_name VARCHAR(255) NOT NULL,
    unit_identifier VARCHAR(100) DEFAULT '',
    complaint TEXT NOT NULL,
    technician_name VARCHAR(100) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'received',
    estimated_cost BIGINT NOT NULL DEFAULT 0,
    final_cost BIGINT NOT NULL DEFAULT 0,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'unpaid',
    notes TEXT DEFAULT '',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_service_orders_tenant_order_no UNIQUE (tenant_id, order_no)
);

CREATE INDEX IF NOT EXISTS idx_service_orders_tenant_status ON service_orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_service_orders_tenant_phone ON service_orders(tenant_id, customer_phone);
CREATE INDEX IF NOT EXISTS idx_service_orders_created_at ON service_orders(tenant_id, created_at DESC);
