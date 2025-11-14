-- Seed data for SupaControl

-- Default admin user
-- Username: admin
-- Password: admin (CHANGE THIS IN PRODUCTION!)
-- Password hash generated using argon2
INSERT INTO users (username, password_hash, role)
VALUES ('admin', '$argon2id$v=19$m=65536,t=3,p=2$Bf6ExJJ5cMiNs0KvwcTt1g$yMF+Kkkk7JwmjLd+yZviCJo5FoTrKuLpKOSrk3cTLoM', 'admin')
ON CONFLICT (username) DO NOTHING;

-- Note: In production, you should:
-- 1. Change the default admin password immediately
-- 2. Use a secure random salt
-- 3. Consider using environment variables for the initial admin credentials
