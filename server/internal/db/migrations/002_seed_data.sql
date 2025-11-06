-- Seed data for SupaControl

-- Default admin user
-- Username: admin
-- Password: admin (CHANGE THIS IN PRODUCTION!)
-- Password hash generated using argon2
INSERT INTO users (username, password_hash, role)
VALUES ('admin', '$argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQxMjM0NTY3OA$YxLivdd9n0N6UjXPjpRwfC7UmvKGjjYEDyJmZ9w7hPs', 'admin')
ON CONFLICT (username) DO NOTHING;

-- Note: In production, you should:
-- 1. Change the default admin password immediately
-- 2. Use a secure random salt
-- 3. Consider using environment variables for the initial admin credentials
