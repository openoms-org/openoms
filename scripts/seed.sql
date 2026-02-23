-- OpenOMS — Default seed data (minimal)
-- Creates 1 tenant + 1 admin user for local development.
-- Login: slug "dev", email "admin@dev.local", password "password123"

INSERT INTO tenants (id, name, slug, plan, settings)
VALUES ('11111111-1111-1111-1111-111111111111', 'Dev Store', 'dev', 'pro',
        '{"default_currency": "PLN", "vat_rate": 23, "auto_confirm_orders": true}')
ON CONFLICT (id) DO NOTHING;

-- Password hash = bcrypt('password123')
INSERT INTO users (id, tenant_id, email, name, password_hash, role)
VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', '11111111-1111-1111-1111-111111111111',
        'admin@dev.local', 'Admin',
        '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'owner')
ON CONFLICT (tenant_id, email) DO NOTHING;
