-- shared/migrations/000033_journal_entries_metadata.up.sql
-- Adds metadata JSONB column to journal_entries for flexible annotation/storage

ALTER TABLE journal_entries ADD COLUMN IF NOT EXISTS metadata JSONB;

CREATE INDEX IF NOT EXISTS idx_journal_entries_metadata ON journal_entries USING GIN (metadata);
