-- Rollback: remove additional_photos column from products
ALTER TABLE products DROP COLUMN IF EXISTS additional_photos;
