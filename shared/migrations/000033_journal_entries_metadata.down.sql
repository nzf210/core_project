-- shared/migrations/000033_journal_entries_metadata.down.sql

DROP INDEX IF EXISTS idx_journal_entries_metadata;
ALTER TABLE journal_entries DROP COLUMN IF EXISTS metadata;
