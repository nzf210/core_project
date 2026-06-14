-- Down migration for Campaign Data Integrity (Anti-Double)

DROP TABLE IF EXISTS endorsements CASCADE;
DROP TABLE IF EXISTS dpt_records CASCADE;
DROP TABLE IF EXISTS citizens CASCADE;