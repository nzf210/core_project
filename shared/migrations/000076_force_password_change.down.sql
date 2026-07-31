-- Rollback: F055 - Force password change column
ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;
