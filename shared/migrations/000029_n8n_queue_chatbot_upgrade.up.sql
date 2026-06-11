-- 000029_n8n_queue_chatbot_upgrade.up.sql
-- N8N Queue Mode + Multi-Tenant Chatbot + WA Session Pool + pgvector RAG
-- Keputusan: pgvector, Self-hosted Chatwoot, Queue-Based Scaling, Multi-Tenant WA Session Pool, Hybrid

-- ─────────────────────────────────────────────
-- 1. Enable pgvector extension for RAG
-- ─────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS vector;

-- ─────────────────────────────────────────────
-- 2. Multi-Tenant WA Session Pool
-- ─────────────────────────────────────────────
CREATE TABLE wa_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_name    VARCHAR(100) NOT NULL,           -- e.g., "wa-001", "wa-002"
    status          VARCHAR(50) NOT NULL DEFAULT 'disconnected',  -- connected, disconnected, qr_pending, banned
    wa_number       VARCHAR(50),                     -- nomor WA yang terhubung (e.g., 6281234567890)
    device_name     VARCHAR(255),                    -- nama device dari WhatsApp
    webhook_secret  VARCHAR(255),                    -- secret untuk validasi webhook
    qr_code         TEXT,                            -- QR code data (sementara, saat pairing)
    last_seen       TIMESTAMPTZ,                     -- kapan terakhir online
    connected_at    TIMESTAMPTZ,                     -- kapan pertama connect
    metadata        JSONB DEFAULT '{}',              -- extra info (battery, platform, etc.)
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_wa_sessions_tenant ON wa_sessions(tenant_id, session_name);
CREATE INDEX idx_wa_sessions_status ON wa_sessions(status);
CREATE INDEX idx_wa_sessions_wa_number ON wa_sessions(wa_number);

-- ─────────────────────────────────────────────
-- 3. Per-Tenant Chatbot Configuration
-- ─────────────────────────────────────────────
CREATE TABLE tenant_chatbot_configs (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- LLM Settings
    llm_provider                VARCHAR(50) DEFAULT 'minimax',       -- minimax, openai, gemini, claude
    llm_model                   VARCHAR(100) DEFAULT 'MiniMax-M2.7',
    temperature                 FLOAT DEFAULT 0.7,
    max_tokens                  INT DEFAULT 1024,
    -- Prompt & Behavior
    system_prompt               TEXT,                                -- custom system prompt per tenant
    tone                        VARCHAR(50) DEFAULT 'friendly',      -- friendly, formal, casual, professional
    language                    VARCHAR(10) DEFAULT 'id',            -- id, en
    max_context_messages        INT DEFAULT 10,                      -- berapa pesan terakhir dipakai sebagai context
    -- Welcome & Fallback
    welcome_message             TEXT DEFAULT 'Halo! Ada yang bisa saya bantu?',
    fallback_message            TEXT DEFAULT 'Maaf, saya belum bisa menjawab pertanyaan tersebut. Apakah Anda ingin dihubungkan dengan CS kami?',
    outside_hours_message       TEXT DEFAULT 'Terima kasih telah menghubungi kami. Saat ini di luar jam operasional. Pesan Anda akan dibalas saat jam kerja.',
    -- Business Hours
    business_hours_start        TIME DEFAULT '08:00',
    business_hours_end          TIME DEFAULT '22:00',
    business_days               INT[] DEFAULT '{1,2,3,4,5,6}',      -- 0=Sunday, 1=Monday, ..., 6=Saturday
    -- Escalation Settings
    escalation_enabled          BOOLEAN DEFAULT true,
    escalation_keywords         TEXT[] DEFAULT '{"bicara cs","hubungi admin","operator","manusia","human"}',
    escalation_confidence_threshold FLOAT DEFAULT 0.3,               -- jika confidence < ini → escalate
    auto_escalate_after_minutes INT DEFAULT 5,                       -- auto escalate jika stuck > N menit
    -- RAG Settings
    rag_enabled                 BOOLEAN DEFAULT true,
    rag_top_k                   INT DEFAULT 5,                       -- berapa dokumen teratas dari vector search
    rag_similarity_threshold    FLOAT DEFAULT 0.7,                   -- minimum similarity score
    -- Channels
    channels_enabled            TEXT[] DEFAULT '{"whatsapp"}',       -- whatsapp, telegram, webchat, instagram
    -- Status
    is_active                   BOOLEAN DEFAULT true,
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE INDEX idx_tenant_chatbot_configs_active ON tenant_chatbot_configs(is_active) WHERE is_active = true;

-- ─────────────────────────────────────────────
-- 4. Conversation Sessions (multi-channel)
-- ─────────────────────────────────────────────
CREATE TABLE conversation_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     VARCHAR(255) NOT NULL,            -- phone number or channel-specific ID
    customer_name   VARCHAR(255),                     -- nama pelanggan (jika diketahui)
    channel         VARCHAR(50) NOT NULL DEFAULT 'whatsapp',  -- whatsapp, telegram, webchat, instagram
    wa_session_id   UUID REFERENCES wa_sessions(id) ON DELETE SET NULL,  -- link ke WA session
    status          VARCHAR(50) DEFAULT 'active',     -- active, escalated, resolved, expired
    escalated_to    VARCHAR(255),                     -- Chatwoot conversation ID / agent ID
    escalated_at    TIMESTAMPTZ,
    context         JSONB DEFAULT '{}',               -- conversation context/metadata
    message_count   INT DEFAULT 0,                    -- counter untuk rate limiting
    last_message_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conv_sessions_tenant ON conversation_sessions(tenant_id);
CREATE INDEX idx_conv_sessions_customer ON conversation_sessions(customer_id);
CREATE INDEX idx_conv_sessions_status ON conversation_sessions(status);
CREATE INDEX idx_conv_sessions_channel ON conversation_sessions(channel);
CREATE INDEX idx_conv_sessions_active ON conversation_sessions(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_conv_sessions_wa_session ON conversation_sessions(wa_session_id);

-- ─────────────────────────────────────────────
-- 5. Conversation Logs (structured, analytics-ready)
-- ─────────────────────────────────────────────
CREATE TABLE conversation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES conversation_sessions(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role            VARCHAR(50) NOT NULL,              -- user, assistant, system, escalation
    content         TEXT NOT NULL,
    channel         VARCHAR(50) NOT NULL DEFAULT 'whatsapp',
    -- Metadata
    llm_provider    VARCHAR(50),                       -- minimax, openai, gemini, claude
    llm_model       VARCHAR(100),                      -- model yang dipakai
    tokens_used     INT DEFAULT 0,                     -- token count
    latency_ms      INT DEFAULT 0,                     -- response latency in ms
    confidence      FLOAT,                             -- AI confidence score (0-1)
    rag_sources     JSONB DEFAULT '[]',                -- dokumen RAG yang dipakai
    metadata        JSONB DEFAULT '{}',                -- extra metadata
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conv_logs_session ON conversation_logs(session_id);
CREATE INDEX idx_conv_logs_tenant ON conversation_logs(tenant_id);
CREATE INDEX idx_conv_logs_created ON conversation_logs(created_at);
CREATE INDEX idx_conv_logs_tenant_created ON conversation_logs(tenant_id, created_at DESC);

-- ─────────────────────────────────────────────
-- 6. Vector Embeddings (pgvector) — FAQ & Products
-- ─────────────────────────────────────────────
CREATE TABLE vector_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_type     VARCHAR(50) NOT NULL,              -- 'faq', 'product', 'document', 'custom'
    source_id       UUID NOT NULL,                     -- ID dari tenant_faqs atau products
    content         TEXT NOT NULL,                     -- raw text yang di-embed
    embedding       vector(1536),                      -- OpenAI ada-002 dimension (1536)
    metadata        JSONB DEFAULT '{}',                -- extra metadata (category, tags, etc.)
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_vector_embeddings_tenant ON vector_embeddings(tenant_id);
CREATE INDEX idx_vector_embeddings_source ON vector_embeddings(source_type, source_id);
-- IVFFlat index untuk fast similarity search
-- NOTE: Baru buat index ini SETELAH ada data (minimal 1000 rows)
-- CREATE INDEX idx_vector_embeddings_ivfflat ON vector_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- HNSW index (lebih cepat untuk real-time, tapi lebih banyak RAM)
CREATE INDEX idx_vector_embeddings_hnsw ON vector_embeddings USING hnsw (embedding vector_cosine_ops);

-- ─────────────────────────────────────────────
-- 7. Escalation History
-- ─────────────────────────────────────────────
CREATE TABLE escalation_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES conversation_sessions(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reason              VARCHAR(100) NOT NULL,          -- 'keyword', 'low_confidence', 'timeout', 'manual'
    trigger_message     TEXT,                            -- pesan yang trigger escalation
    chatwoot_conversation_id VARCHAR(255),               -- ID conversation di Chatwoot
    assigned_agent      VARCHAR(255),                    -- agent yang handle
    resolved_at         TIMESTAMPTZ,
    resolution_notes    TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_escalation_session ON escalation_history(session_id);
CREATE INDEX idx_escalation_tenant ON escalation_history(tenant_id);
CREATE INDEX idx_escalation_created ON escalation_history(created_at);
