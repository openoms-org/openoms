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

-- Second user for logout E2E tests (separate from main auth user to avoid token invalidation)
INSERT INTO users (id, tenant_id, email, name, password_hash, role)
VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', '11111111-1111-1111-1111-111111111111',
        'e2e-logout@dev.local', 'E2E Logout User',
        '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'admin')
ON CONFLICT (tenant_id, email) DO NOTHING;

-- Seed product for E2E tests (product-crud.spec.ts searches for this)
INSERT INTO products (id, tenant_id, name, sku, ean, price, stock_quantity)
VALUES ('cccccccc-cccc-cccc-cccc-cccccccc0001', '11111111-1111-1111-1111-111111111111',
        'Klocki hamulcowe przód Audi A4 B8', 'KH-AUDI-A4-P', '5901234567890', 149.99, 25)
ON CONFLICT (id) DO NOTHING;

-- Seed order for E2E tests (order-lifecycle.spec.ts and order-status-change.spec.ts need this)
INSERT INTO orders (id, tenant_id, customer_name, customer_email, customer_phone, status, items, total_amount)
VALUES ('dddddddd-dddd-dddd-dddd-dddddddd0001', '11111111-1111-1111-1111-111111111111',
        'Marek Jabłoński', 'marek@example.com', '+48 500 100 300', 'new',
        '[{"name": "Klocki hamulcowe przód Audi A4 B8", "sku": "KH-AUDI-A4-P", "quantity": 2, "price": 149.99}]',
        299.98)
ON CONFLICT (id) DO NOTHING;

-- Trial flow test tenant (onboarding incomplete — for E2E tests)
-- Uses DO UPDATE SET to reset onboarding state on re-seed (tests are destructive)
INSERT INTO tenants (id, name, slug, plan, settings)
VALUES ('22222222-2222-2222-2222-222222222222', 'Test Flow Sp. z o.o.', 'testflow', 'starter',
        '{"default_currency": "PLN", "vat_rate": 23, "onboarding": {"completed": false, "current_step": 1, "completed_steps": [], "skipped_steps": []}}')
ON CONFLICT (id) DO UPDATE SET
  settings = '{"default_currency": "PLN", "vat_rate": 23, "onboarding": {"completed": false, "current_step": 1, "completed_steps": [], "skipped_steps": []}}';

INSERT INTO users (id, tenant_id, email, name, password_hash, role)
VALUES ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', '22222222-2222-2222-2222-222222222222',
        'trial@testflow.pl', 'Trial User',
        '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'owner')
ON CONFLICT (tenant_id, email) DO NOTHING;
