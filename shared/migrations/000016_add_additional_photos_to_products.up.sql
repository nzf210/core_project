ALTER TABLE products ADD COLUMN IF NOT EXISTS additional_photos JSONB DEFAULT '[]'::jsonb;
