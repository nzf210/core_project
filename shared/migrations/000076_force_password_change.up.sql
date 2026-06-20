-- F055: Reset password default — flag wajib ganti password
ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN DEFAULT false;