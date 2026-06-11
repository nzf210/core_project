-- 000029_n8n_queue_chatbot_upgrade.down.sql
-- Rollback: N8N Queue Mode + Multi-Tenant Chatbot + WA Session Pool + pgvector RAG

DROP TABLE IF EXISTS escalation_history CASCADE;
DROP TABLE IF EXISTS vector_embeddings CASCADE;
DROP TABLE IF EXISTS conversation_logs CASCADE;
DROP TABLE IF EXISTS conversation_sessions CASCADE;
DROP TABLE IF EXISTS tenant_chatbot_configs CASCADE;
DROP TABLE IF EXISTS wa_sessions CASCADE;

-- NOTE: We intentionally do NOT drop the vector extension
-- because other tables might depend on it.
-- To manually remove: DROP EXTENSION IF EXISTS vector CASCADE;
