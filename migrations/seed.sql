-- Seed users
INSERT INTO users (username, name, email, password_hash, role) VALUES
(
    'admin',
    'Administrator',
    'admin@lattepos.com',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'admin'
),
(
    'kasir1',
    'Kasir Satu',
    'kasir1@lattepos.com',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'cashier'
) ON CONFLICT (username) DO NOTHING;

-- Seed organizations
INSERT INTO organizations (name, slug) VALUES
('LattePOS Central', 'lattepos-central'),
('LattePOS Branch One', 'lattepos-branch-one')
ON CONFLICT (slug) DO NOTHING;

-- Seed organization_members
-- Admin: anggota di kedua organisasi
INSERT INTO organization_members (org_id, user_id)
SELECT o.id, u.id
FROM organizations o, users u
WHERE u.username = 'admin'
ON CONFLICT (org_id, user_id) DO NOTHING;

-- Kasir1: anggota di lattepos-central saja
INSERT INTO organization_members (org_id, user_id)
SELECT o.id, u.id
FROM organizations o, users u
WHERE u.username = 'kasir1' AND o.slug = 'lattepos-central'
ON CONFLICT (org_id, user_id) DO NOTHING;

-- Seed permissions
INSERT INTO permissions (name, description) VALUES
('users:read', 'Melihat daftar dan detail user'),
('users:create', 'Membuat user baru'),
('users:update', 'Mengubah data atau role user'),
('permissions:read', 'Melihat daftar permission'),
('permissions:create', 'Membuat permission baru'),
('roles:read', 'Melihat daftar dan detail role'),
('roles:create', 'Membuat role baru'),
('roles:update', 'Mengubah role dan menetapkan permission')
ON CONFLICT (name) DO NOTHING;

-- Seed global roles (org_id NULL)
INSERT INTO roles (name, description, org_id) VALUES
('admin', 'Administrator global dengan akses penuh', NULL),
('cashier', 'Kasir global dengan akses terbatas', NULL)
ON CONFLICT (name) DO NOTHING;

-- Seed org-scoped roles (contoh untuk lattepos-central)
INSERT INTO roles (name, description, org_id)
SELECT 'org-admin', 'Administrator organisasi lattepos-central', o.id
FROM organizations o
WHERE o.slug = 'lattepos-central'
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, description, org_id)
SELECT 'org-cashier', 'Kasir organisasi lattepos-central', o.id
FROM organizations o
WHERE o.slug = 'lattepos-central'
ON CONFLICT (name) DO NOTHING;

-- Seed role_permissions
-- Admin role (global & org-scoped): semua permission
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'org-admin')
ON CONFLICT DO NOTHING;

-- Cashier role (global & org-scoped): permission users:read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('cashier', 'org-cashier') AND p.name = 'users:read'
ON CONFLICT DO NOTHING;

-- Seed user_roles (dengan org_id scoping)
-- Admin -> role 'admin' & 'org-admin' di lattepos-central
INSERT INTO user_roles (user_id, role_id, org_id)
SELECT u.id, r.id, o.id
FROM users u, roles r, organizations o
WHERE u.username = 'admin' AND r.name = 'admin' AND o.slug = 'lattepos-central'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id, org_id)
SELECT u.id, r.id, o.id
FROM users u, roles r, organizations o
WHERE u.username = 'admin' AND r.name = 'org-admin' AND o.slug = 'lattepos-central'
ON CONFLICT DO NOTHING;

-- Kasir1 -> role 'cashier' & 'org-cashier' di lattepos-central
INSERT INTO user_roles (user_id, role_id, org_id)
SELECT u.id, r.id, o.id
FROM users u, roles r, organizations o
WHERE u.username = 'kasir1' AND r.name = 'cashier' AND o.slug = 'lattepos-central'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id, org_id)
SELECT u.id, r.id, o.id
FROM users u, roles r, organizations o
WHERE u.username = 'kasir1' AND r.name = 'org-cashier' AND o.slug = 'lattepos-central'
ON CONFLICT DO NOTHING;
