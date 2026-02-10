-- OpenOMS — Seed data
-- Migration 000002: Test data for 3 tenants with Polish e-commerce context
--
-- Run as superuser (openoms), NOT openoms_app, because RLS would block
-- cross-tenant inserts in a single transaction.
--
-- Well-known UUIDs for testing:
--   Tenant A (MercPart):    11111111-1111-1111-1111-111111111111
--   Tenant B (ElektroMax):  22222222-2222-2222-2222-222222222222
--   Tenant C (ZielonyOgrod): 33333333-3333-3333-3333-333333333333

-- ============================================================
-- Tenants
-- ============================================================
INSERT INTO tenants (id, name, slug, plan, settings) VALUES
    ('11111111-1111-1111-1111-111111111111', 'MercPart — Części Samochodowe', 'mercpart', 'pro',
     '{"default_currency": "PLN", "vat_rate": 23, "auto_confirm_orders": true}'),
    ('22222222-2222-2222-2222-222222222222', 'ElektroMax', 'elektromax', 'lite',
     '{"default_currency": "PLN", "vat_rate": 23, "auto_confirm_orders": false}'),
    ('33333333-3333-3333-3333-333333333333', 'ZielonyOgród', 'zielonyogrod', 'free',
     '{"default_currency": "PLN", "vat_rate": 8, "auto_confirm_orders": false}')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Users (3 per tenant: owner, admin, member)
-- Password hash = bcrypt('password123')
-- ============================================================
INSERT INTO users (id, tenant_id, email, name, password_hash, role) VALUES
    -- MercPart
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa001', '11111111-1111-1111-1111-111111111111',
     'rafal@mercpart.pl', 'Rafał Strzelczyk',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'owner'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa002', '11111111-1111-1111-1111-111111111111',
     'adam@mercpart.pl', 'Adam Kowalski',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'admin'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa003', '11111111-1111-1111-1111-111111111111',
     'ewa@mercpart.pl', 'Ewa Nowak',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'member'),
    -- ElektroMax
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa004', '22222222-2222-2222-2222-222222222222',
     'jan@elektromax.pl', 'Jan Wiśniewski',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'owner'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa005', '22222222-2222-2222-2222-222222222222',
     'anna@elektromax.pl', 'Anna Kamińska',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'admin'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa006', '22222222-2222-2222-2222-222222222222',
     'piotr@elektromax.pl', 'Piotr Zieliński',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'member'),
    -- ZielonyOgród
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa007', '33333333-3333-3333-3333-333333333333',
     'maria@zielonyogrod.pl', 'Maria Lewandowska',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'owner'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa008', '33333333-3333-3333-3333-333333333333',
     'tomek@zielonyogrod.pl', 'Tomek Wójcik',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'admin'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa009', '33333333-3333-3333-3333-333333333333',
     'kasia@zielonyogrod.pl', 'Kasia Dąbrowska',
     '$2a$12$sOotPXgEIxhy/IRdwZSmcO918JDs5/pW6EPznANKHYjK8tXBb8TVa', 'member')
ON CONFLICT (tenant_id, email) DO NOTHING;

-- ============================================================
-- Integrations (Tenant A: Allegro + InPost)
-- ============================================================
INSERT INTO integrations (id, tenant_id, provider, status, credentials, settings) VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', '11111111-1111-1111-1111-111111111111',
     'allegro', 'active',
     '{"client_id": "placeholder-allegro-client-id", "client_secret": "placeholder-encrypted", "access_token": "placeholder-token", "refresh_token": "placeholder-refresh"}',
     '{"auto_import_orders": true, "sync_interval_minutes": 15}'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002', '11111111-1111-1111-1111-111111111111',
     'inpost', 'active',
     '{"api_token": "placeholder-inpost-token", "organization_id": "placeholder-org-id"}',
     '{"default_parcel_size": "A", "default_sender_address": {"city": "Warszawa", "postal_code": "02-495"}}')
ON CONFLICT (tenant_id, provider) DO NOTHING;

-- ============================================================
-- Orders — MercPart (5 orders — auto parts)
-- ============================================================
INSERT INTO orders (id, tenant_id, external_id, source, integration_id, status, customer_name, customer_email, customer_phone, shipping_address, items, total_amount, currency, ordered_at, shipped_at, delivered_at) VALUES
    ('cccccccc-cccc-cccc-cccc-ccccccccc001', '11111111-1111-1111-1111-111111111111',
     'ALG-90001', 'allegro', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'new',
     'Marek Jabłoński', 'marek.j@gmail.com', '+48601234567',
     '{"name": "Marek Jabłoński", "street": "ul. Słoneczna 15/3", "city": "Warszawa", "postal_code": "02-495", "country": "PL"}',
     '[{"name": "Klocki hamulcowe przód Audi A4 B8", "sku": "KH-AUDI-A4-P", "quantity": 1, "price": 189.99}, {"name": "Tarcze hamulcowe przód Audi A4 B8", "sku": "TH-AUDI-A4-P", "quantity": 1, "price": 270.00}]',
     459.99, 'PLN', '2025-01-15 10:30:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc002', '11111111-1111-1111-1111-111111111111',
     'ALG-90002', 'allegro', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'confirmed',
     'Zofia Kowalczyk', 'zofia.k@wp.pl', '+48502345678',
     '{"name": "Zofia Kowalczyk", "street": "os. Tysiąclecia 8/12", "city": "Kraków", "postal_code": "31-340", "country": "PL"}',
     '[{"name": "Filtr oleju Mann W712/95", "sku": "FO-MANN-712", "quantity": 2, "price": 32.50}, {"name": "Olej Castrol Edge 5W30 5L", "sku": "OL-CAST-5W30", "quantity": 1, "price": 259.50}]',
     324.50, 'PLN', '2025-01-16 14:15:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc003', '11111111-1111-1111-1111-111111111111',
     'ALG-90003', 'allegro', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001', 'processing',
     'Krzysztof Nowicki', 'k.nowicki@onet.pl', '+48603456789',
     '{"name": "Krzysztof Nowicki", "street": "ul. Mickiewicza 42", "city": "Poznań", "postal_code": "60-836", "country": "PL"}',
     '[{"name": "Amortyzator przód Kayaba Excel-G BMW E46", "sku": "AM-KYB-E46-P", "quantity": 2, "price": 425.00}, {"name": "Poduszka amortyzatora BMW E46", "sku": "PA-BMW-E46", "quantity": 2, "price": 200.00}]',
     1250.00, 'PLN', '2025-01-17 09:00:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc004', '11111111-1111-1111-1111-111111111111',
     'WOO-10044', 'woocommerce', NULL, 'shipped',
     'Agnieszka Mazur', 'agnieszka.m@interia.pl', '+48504567890',
     '{"name": "Agnieszka Mazur", "street": "ul. Kwiatowa 7", "city": "Wrocław", "postal_code": "50-001", "country": "PL"}',
     '[{"name": "Wycieraczki Bosch Aerotwin A638S", "sku": "WY-BOSCH-A638", "quantity": 1, "price": 89.90}]',
     89.90, 'PLN', '2025-01-10 16:45:00+01', '2025-01-12 08:00:00+01', NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc005', '11111111-1111-1111-1111-111111111111',
     NULL, 'manual', NULL, 'delivered',
     'Stanisław Grabowski', 'st.grabowski@gmail.com', '+48605678901',
     '{"name": "Stanisław Grabowski", "street": "ul. Długa 23", "city": "Gdańsk", "postal_code": "80-831", "country": "PL"}',
     '[{"name": "Sprzęgło komplet LuK VW Golf VII 1.6 TDI", "sku": "SP-LUK-GOLF7", "quantity": 1, "price": 1450.00}, {"name": "Koło zamachowe dwumasowe LuK", "sku": "KZ-LUK-DMF", "quantity": 1, "price": 700.00}]',
     2150.00, 'PLN', '2025-01-05 11:20:00+01', '2025-01-06 09:00:00+01', '2025-01-08 14:30:00+01')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Orders — ElektroMax (5 orders — electronics)
-- ============================================================
INSERT INTO orders (id, tenant_id, external_id, source, status, customer_name, customer_email, customer_phone, shipping_address, items, total_amount, currency, ordered_at, shipped_at, delivered_at) VALUES
    ('cccccccc-cccc-cccc-cccc-ccccccccc006', '22222222-2222-2222-2222-222222222222',
     'ALG-80001', 'allegro', 'new',
     'Tomasz Kaczmarek', 'tomek.k@gmail.com', '+48606789012',
     '{"name": "Tomasz Kaczmarek", "street": "ul. Piłsudskiego 100", "city": "Łódź", "postal_code": "90-001", "country": "PL"}',
     '[{"name": "Samsung Galaxy S24 128GB", "sku": "TEL-SAM-S24", "quantity": 1, "price": 2499.00}]',
     2499.00, 'PLN', '2025-01-18 12:00:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc007', '22222222-2222-2222-2222-222222222222',
     'ALG-80002', 'allegro', 'confirmed',
     'Patrycja Wróbel', 'patrycja.w@o2.pl', '+48507890123',
     '{"name": "Patrycja Wróbel", "street": "al. Niepodległości 22", "city": "Szczecin", "postal_code": "70-404", "country": "PL"}',
     '[{"name": "JBL Tune 770NC słuchawki", "sku": "AUD-JBL-770", "quantity": 1, "price": 449.00}]',
     449.00, 'PLN', '2025-01-18 15:30:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc008', '22222222-2222-2222-2222-222222222222',
     'ALG-80003', 'allegro', 'shipped',
     'Bartosz Szymański', 'b.szymanski@wp.pl', '+48608901234',
     '{"name": "Bartosz Szymański", "street": "ul. Reymonta 5/8", "city": "Katowice", "postal_code": "40-001", "country": "PL"}',
     '[{"name": "Dell Monitor 27\" 4K S2722QC", "sku": "MON-DELL-27-4K", "quantity": 1, "price": 1799.00}]',
     1799.00, 'PLN', '2025-01-14 10:00:00+01', '2025-01-15 14:00:00+01', NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc009', '22222222-2222-2222-2222-222222222222',
     'WOO-20015', 'woocommerce', 'delivered',
     'Monika Jankowska', 'monika.j@gmail.com', '+48509012345',
     '{"name": "Monika Jankowska", "street": "ul. Zielona 18", "city": "Lublin", "postal_code": "20-001", "country": "PL"}',
     '[{"name": "Kabel USB-C Anker 2m 100W", "sku": "ACC-ANK-USBC", "quantity": 3, "price": 53.00}]',
     159.00, 'PLN', '2025-01-08 09:45:00+01', '2025-01-09 11:00:00+01', '2025-01-11 16:00:00+01'),

    ('cccccccc-cccc-cccc-cccc-ccccccccc010', '22222222-2222-2222-2222-222222222222',
     NULL, 'manual', 'cancelled',
     'Robert Pawlak', 'r.pawlak@interia.pl', '+48610123456',
     '{"name": "Robert Pawlak", "street": "ul. Kościuszki 33", "city": "Białystok", "postal_code": "15-001", "country": "PL"}',
     '[{"name": "MacBook Air M2 256GB", "sku": "LAP-MAC-AIR-M2", "quantity": 1, "price": 3299.00}]',
     3299.00, 'PLN', '2025-01-12 17:00:00+01', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Orders — ZielonyOgród (5 orders — plants and garden)
-- ============================================================
INSERT INTO orders (id, tenant_id, external_id, source, status, customer_name, customer_email, customer_phone, shipping_address, items, total_amount, currency, ordered_at, shipped_at, delivered_at) VALUES
    ('cccccccc-cccc-cccc-cccc-ccccccccc011', '33333333-3333-3333-3333-333333333333',
     'ALG-70001', 'allegro', 'new',
     'Dorota Sikora', 'dorota.s@gmail.com', '+48611234567',
     '{"name": "Dorota Sikora", "street": "ul. Ogrodowa 12", "city": "Radom", "postal_code": "26-600", "country": "PL"}',
     '[{"name": "Hortensja ogrodowa niebieska", "sku": "ROL-HORT-BLUE", "quantity": 2, "price": 49.90}, {"name": "Ziemia do hortensji 50L", "sku": "ZIE-HORT-50L", "quantity": 1, "price": 34.90}, {"name": "Nawóz do hortensji 1kg", "sku": "NAW-HORT-1KG", "quantity": 1, "price": 55.00}]',
     189.70, 'PLN', '2025-01-19 08:30:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc012', '33333333-3333-3333-3333-333333333333',
     'ALG-70002', 'allegro', 'processing',
     'Henryk Woźniak', 'h.wozniak@wp.pl', '+48512345678',
     '{"name": "Henryk Woźniak", "street": "ul. Lipowa 45", "city": "Kielce", "postal_code": "25-001", "country": "PL"}',
     '[{"name": "Robot koszący Gardena Sileno 250", "sku": "NAR-GARD-S250", "quantity": 1, "price": 750.00}]',
     750.00, 'PLN', '2025-01-17 11:00:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc013', '33333333-3333-3333-3333-333333333333',
     'WOO-30008', 'woocommerce', 'confirmed',
     'Elżbieta Adamska', 'elzbieta.a@onet.pl', '+48613456789',
     '{"name": "Elżbieta Adamska", "street": "os. Piastów 3/16", "city": "Opole", "postal_code": "45-001", "country": "PL"}',
     '[{"name": "Donica ceramiczna antracyt 40cm", "sku": "DON-CER-ANT-40", "quantity": 2, "price": 89.00}, {"name": "Keramzyt drenaż 10L", "sku": "KER-DREN-10L", "quantity": 2, "price": 28.25}]',
     234.50, 'PLN', '2025-01-18 14:20:00+01', NULL, NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc014', '33333333-3333-3333-3333-333333333333',
     NULL, 'manual', 'shipped',
     'Andrzej Kamiński', 'a.kaminski@gmail.com', '+48514567890',
     '{"name": "Andrzej Kamiński", "street": "ul. Wiejska 88", "city": "Rzeszów", "postal_code": "35-001", "country": "PL"}',
     '[{"name": "Szklarnia ogrodowa 6m² aluminium", "sku": "SZK-ALU-6M2", "quantity": 1, "price": 1299.00}]',
     1299.00, 'PLN', '2025-01-11 13:00:00+01', '2025-01-13 10:00:00+01', NULL),

    ('cccccccc-cccc-cccc-cccc-ccccccccc015', '33333333-3333-3333-3333-333333333333',
     'ALG-70003', 'allegro', 'cancelled',
     'Natalia Kwiatkowska', 'n.kwiatkowska@o2.pl', '+48615678901',
     '{"name": "Natalia Kwiatkowska", "street": "ul. Parkowa 2", "city": "Toruń", "postal_code": "87-100", "country": "PL"}',
     '[{"name": "Nasiona traw sportowych 5kg", "sku": "NAS-TRAW-SPO-5", "quantity": 1, "price": 67.80}]',
     67.80, 'PLN', '2025-01-16 16:00:00+01', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Shipments (for shipped/delivered orders)
-- ============================================================
INSERT INTO shipments (id, tenant_id, order_id, provider, integration_id, tracking_number, status, label_url, carrier_data) VALUES
    -- MercPart: shipped order (Agnieszka Mazur — wycieraczki)
    ('dddddddd-dddd-dddd-dddd-ddddddddd001', '11111111-1111-1111-1111-111111111111',
     'cccccccc-cccc-cccc-cccc-ccccccccc004', 'inpost', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002',
     '620100000012345678901234', 'in_transit',
     'https://api.inpost.pl/labels/620100000012345678901234.pdf',
     '{"parcel_size": "A", "locker_id": "WRO04M", "sender": "MercPart"}'),
    -- MercPart: delivered order (Stanisław Grabowski — sprzęgło)
    ('dddddddd-dddd-dddd-dddd-ddddddddd002', '11111111-1111-1111-1111-111111111111',
     'cccccccc-cccc-cccc-cccc-ccccccccc005', 'inpost', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002',
     '620100000098765432109876', 'delivered',
     'https://api.inpost.pl/labels/620100000098765432109876.pdf',
     '{"parcel_size": "B", "locker_id": "GDA01A", "sender": "MercPart", "delivered_at": "2025-01-08T14:30:00+01:00"}'),
    -- ElektroMax: shipped order (Bartosz Szymański — monitor)
    ('dddddddd-dddd-dddd-dddd-ddddddddd003', '22222222-2222-2222-2222-222222222222',
     'cccccccc-cccc-cccc-cccc-ccccccccc008', 'dhl', NULL,
     'JD0145678901234', 'picked_up',
     'https://www.dhl.com/labels/JD0145678901234.pdf',
     '{"service_type": "DHL_PARCEL", "weight_kg": 8.5, "sender": "ElektroMax"}')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Products (extracted from order items)
-- ============================================================
INSERT INTO products (id, tenant_id, source, name, sku, ean, price, stock_quantity, metadata) VALUES
    -- MercPart products
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee001', '11111111-1111-1111-1111-111111111111',
     'allegro', 'Klocki hamulcowe przód Audi A4 B8', 'KH-AUDI-A4-P', '5901234567890',
     189.99, 45, '{"brand": "TRW", "category": "hamulce", "weight_kg": 2.1}'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee002', '11111111-1111-1111-1111-111111111111',
     'allegro', 'Olej Castrol Edge 5W30 5L', 'OL-CAST-5W30', '4008177072628',
     259.50, 22, '{"brand": "Castrol", "category": "oleje", "weight_kg": 4.5}'),
    -- ElektroMax products
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee003', '22222222-2222-2222-2222-222222222222',
     'allegro', 'Samsung Galaxy S24 128GB', 'TEL-SAM-S24', '8806095373102',
     2499.00, 8, '{"brand": "Samsung", "category": "telefony", "color": "black"}'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee004', '22222222-2222-2222-2222-222222222222',
     'allegro', 'Dell Monitor 27" 4K S2722QC', 'MON-DELL-27-4K', '5397184505137',
     1799.00, 5, '{"brand": "Dell", "category": "monitory", "resolution": "3840x2160"}'),
    -- ZielonyOgród product
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeee005', '33333333-3333-3333-3333-333333333333',
     'allegro', 'Hortensja ogrodowa niebieska', 'ROL-HORT-BLUE', NULL,
     49.90, 120, '{"category": "rośliny", "season": "wiosna-lato", "pot_size_cm": 15}')
ON CONFLICT (id) DO NOTHING;
