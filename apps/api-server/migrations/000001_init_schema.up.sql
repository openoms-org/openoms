-- ============================================================
-- OpenOMS — Init Schema Migration
-- Squashed from 69 migrations into a single clean baseline.
-- Generated: 2026-02-20
-- ============================================================

-- ============================================================
-- Extensions
-- ============================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

-- ============================================================
-- Functions
-- ============================================================

-- Trigger function: auto-update updated_at timestamp
CREATE FUNCTION public.update_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- Trigger function: alternative updated_at (used by stocktakes, warehouse_documents)
CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$BEGIN NEW.updated_at = NOW(); RETURN NEW; END;$$;

-- SECURITY DEFINER: find tenant by slug (login flow, bypasses RLS)
CREATE FUNCTION public.find_tenant_by_slug(p_slug text) RETURNS TABLE(id uuid, name character varying, slug character varying, plan text, settings jsonb, created_at timestamp with time zone, updated_at timestamp with time zone)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    SELECT t.id, t.name, t.slug, t.plan, t.settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$;
ALTER FUNCTION public.find_tenant_by_slug(text) OWNER TO postgres;

-- SECURITY DEFINER: find user for auth (login flow, bypasses RLS)
CREATE FUNCTION public.find_user_for_auth(p_email text, p_tenant_id uuid) RETURNS TABLE(id uuid, tenant_id uuid, email text, name text, password_hash text, role text, role_id uuid, created_at timestamp with time zone, updated_at timestamp with time zone, totp_secret text, totp_enabled boolean)
    LANGUAGE sql SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash,
           u.role, u.role_id, u.created_at, u.updated_at,
           u.totp_secret, u.totp_enabled
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$;
ALTER FUNCTION public.find_user_for_auth(text, uuid) OWNER TO postgres;

-- SECURITY DEFINER: find order tenant_id (public return form, bypasses RLS)
CREATE FUNCTION public.find_order_tenant_id(p_order_id uuid) RETURNS TABLE(tenant_id uuid, customer_email text)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    SELECT o.tenant_id, o.customer_email
    FROM orders o
    WHERE o.id = p_order_id;
$$;
ALTER FUNCTION public.find_order_tenant_id(uuid) OWNER TO postgres;

-- SECURITY DEFINER: find return by token (public return status, bypasses RLS)
CREATE FUNCTION public.find_return_by_token(p_token text) RETURNS TABLE(id uuid, tenant_id uuid, order_id uuid, status character varying, reason text, items jsonb, refund_amount numeric, notes text, return_token text, customer_email text, customer_notes text, created_at timestamp with time zone, updated_at timestamp with time zone)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    SELECT r.id, r.tenant_id, r.order_id, r.status,
           r.reason, r.items, r.refund_amount,
           r.notes, r.return_token, r.customer_email,
           r.customer_notes, r.created_at, r.updated_at
    FROM returns r
    WHERE r.return_token = p_token;
$$;
ALTER FUNCTION public.find_return_by_token(text) OWNER TO postgres;

-- SECURITY DEFINER: find invitation by token (invite flow, bypasses RLS)
CREATE FUNCTION public.find_invitation_by_token(p_token text) RETURNS TABLE(id uuid, tenant_id uuid, email text, role text, expires_at timestamp with time zone, used_at timestamp with time zone)
    LANGUAGE sql SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    SELECT i.id, i.tenant_id, i.email, i.role, i.expires_at, i.used_at
    FROM invitations i
    WHERE i.token = p_token
    LIMIT 1;
$$;
ALTER FUNCTION public.find_invitation_by_token(text) OWNER TO postgres;

-- SECURITY DEFINER: mark invitation as used (invite flow, bypasses RLS)
CREATE FUNCTION public.use_invitation(p_token text) RETURNS void
    LANGUAGE sql SECURITY DEFINER
    SET search_path TO 'public'
    AS $$
    UPDATE invitations SET used_at = now() WHERE token = p_token AND used_at IS NULL;
$$;
ALTER FUNCTION public.use_invitation(text) OWNER TO postgres;

-- ============================================================
-- Tables (in dependency order)
-- ============================================================

-- 1. Tenants (root entity)
CREATE TABLE public.tenants (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    plan text DEFAULT 'free'::text NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_slug_key UNIQUE (slug);

-- 2. Roles (RBAC)
CREATE TABLE public.roles (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    is_system boolean DEFAULT false NOT NULL,
    permissions text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);
ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 3. Users
CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    email character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id uuid,
    last_logout_at timestamp with time zone,
    totp_secret text,
    totp_enabled boolean DEFAULT false NOT NULL,
    totp_verified_at timestamp with time zone
);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_email_key UNIQUE (tenant_id, email);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);

-- 4. Integrations
CREATE TABLE public.integrations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    provider text NOT NULL,
    status text DEFAULT 'inactive'::text NOT NULL,
    credentials jsonb DEFAULT '{}'::jsonb NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_sync_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    label text,
    sync_cursor text,
    error_message text
);

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 5. Product Categories
CREATE TABLE public.product_categories (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    parent_id uuid,
    name character varying(200) NOT NULL,
    slug character varying(200) NOT NULL,
    color character varying(7) DEFAULT '#6b7280'::character varying,
    icon character varying(50),
    "position" integer DEFAULT 0 NOT NULL,
    depth integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.product_categories(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 6. Price Lists
CREATE TABLE public.price_lists (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    currency text DEFAULT 'PLN'::text NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    discount_type text DEFAULT 'percentage'::text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    valid_from timestamp with time zone,
    valid_to timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.price_lists
    ADD CONSTRAINT price_lists_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.price_lists
    ADD CONSTRAINT price_lists_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 7. Customers
CREATE TABLE public.customers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    email text,
    phone text,
    name text NOT NULL,
    company_name text,
    nip text,
    default_shipping_address jsonb,
    default_billing_address jsonb,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    notes text,
    total_orders integer DEFAULT 0 NOT NULL,
    total_spent numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    price_list_id uuid
);

ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_price_list_id_fkey FOREIGN KEY (price_list_id) REFERENCES public.price_lists(id);

-- 8. Suppliers
CREATE TABLE public.suppliers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    code character varying(100),
    feed_url text,
    feed_format text DEFAULT 'iof'::text NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_sync_at timestamp with time zone,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    portal_enabled boolean DEFAULT false NOT NULL,
    sync_interval_minutes integer DEFAULT 60 NOT NULL,
    integration_id uuid,
    default_category_id uuid
);

ALTER TABLE ONLY public.suppliers
    ADD CONSTRAINT suppliers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.suppliers
    ADD CONSTRAINT suppliers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.suppliers
    ADD CONSTRAINT suppliers_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.suppliers
    ADD CONSTRAINT suppliers_default_category_id_fkey FOREIGN KEY (default_category_id) REFERENCES public.product_categories(id) ON DELETE SET NULL;

-- 9. Products
CREATE TABLE public.products (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    external_id character varying(255),
    source text DEFAULT 'manual'::text NOT NULL,
    name character varying(500) NOT NULL,
    sku character varying(100),
    ean character varying(20),
    price numeric(12,2) DEFAULT 0 NOT NULL,
    stock_quantity integer DEFAULT 0 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    image_url character varying(1024),
    images jsonb DEFAULT '[]'::jsonb NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    description_short text DEFAULT ''::text NOT NULL,
    description_long text DEFAULT ''::text NOT NULL,
    weight numeric(10,3) DEFAULT NULL::numeric,
    width numeric(10,2) DEFAULT NULL::numeric,
    height numeric(10,2) DEFAULT NULL::numeric,
    depth numeric(10,2) DEFAULT NULL::numeric,
    category text,
    has_variants boolean DEFAULT false NOT NULL,
    is_bundle boolean DEFAULT false NOT NULL,
    is_dropship boolean DEFAULT false NOT NULL,
    dropship_supplier_id uuid,
    category_id uuid
);

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_dropship_supplier_id_fkey FOREIGN KEY (dropship_supplier_id) REFERENCES public.suppliers(id);
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.product_categories(id) ON DELETE SET NULL;

-- 10. Product Variants
CREATE TABLE public.product_variants (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    product_id uuid NOT NULL,
    sku text,
    ean text,
    name text NOT NULL,
    attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    price_override numeric(12,2),
    stock_quantity integer DEFAULT 0 NOT NULL,
    weight numeric(8,3),
    image_url text,
    "position" integer DEFAULT 0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.product_variants
    ADD CONSTRAINT product_variants_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.product_variants
    ADD CONSTRAINT product_variants_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.product_variants
    ADD CONSTRAINT product_variants_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 11. Product Bundles
CREATE TABLE public.product_bundles (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    bundle_product_id uuid NOT NULL,
    component_product_id uuid NOT NULL,
    component_variant_id uuid,
    quantity integer DEFAULT 1 NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_bundle_product_id_component_product_id_comp_key UNIQUE (bundle_product_id, component_product_id, component_variant_id);
ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_bundle_product_id_fkey FOREIGN KEY (bundle_product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_component_product_id_fkey FOREIGN KEY (component_product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_component_variant_id_fkey FOREIGN KEY (component_variant_id) REFERENCES public.product_variants(id);
ALTER TABLE ONLY public.product_bundles
    ADD CONSTRAINT product_bundles_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 12. Product Listings (marketplace offers)
CREATE TABLE public.product_listings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    product_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    external_id text,
    status text DEFAULT 'pending'::text NOT NULL,
    url text,
    price_override numeric(12,2),
    stock_override integer,
    sync_status text DEFAULT 'pending'::text NOT NULL,
    last_synced_at timestamp with time zone,
    error_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    stock_sync_mode text DEFAULT 'auto'::text NOT NULL
);

ALTER TABLE ONLY public.product_listings
    ADD CONSTRAINT product_listings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.product_listings
    ADD CONSTRAINT product_listings_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.product_listings
    ADD CONSTRAINT product_listings_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.product_listings
    ADD CONSTRAINT product_listings_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 13. Price List Items
CREATE TABLE public.price_list_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    price_list_id uuid NOT NULL,
    product_id uuid NOT NULL,
    variant_id uuid,
    price numeric(12,2),
    discount numeric(5,2),
    min_quantity integer DEFAULT 1,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_price_list_id_product_id_variant_id_min_qu_key UNIQUE (price_list_id, product_id, variant_id, min_quantity);
ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_price_list_id_fkey FOREIGN KEY (price_list_id) REFERENCES public.price_lists(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_variant_id_fkey FOREIGN KEY (variant_id) REFERENCES public.product_variants(id);
ALTER TABLE ONLY public.price_list_items
    ADD CONSTRAINT price_list_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 14. Orders
CREATE TABLE public.orders (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    external_id character varying(255),
    source text DEFAULT 'manual'::text NOT NULL,
    integration_id uuid,
    status text DEFAULT 'new'::text NOT NULL,
    customer_name character varying(255) NOT NULL,
    customer_email character varying(255),
    customer_phone character varying(50),
    shipping_address jsonb DEFAULT '{}'::jsonb NOT NULL,
    billing_address jsonb,
    items jsonb DEFAULT '[]'::jsonb NOT NULL,
    total_amount numeric(12,2) DEFAULT 0 NOT NULL,
    currency character varying(3) DEFAULT 'PLN'::character varying NOT NULL,
    notes text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    ordered_at timestamp with time zone DEFAULT now() NOT NULL,
    shipped_at timestamp with time zone,
    delivered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    payment_status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    payment_method character varying(50),
    paid_at timestamp with time zone,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    delivery_method text,
    pickup_point_id text,
    customer_id uuid,
    merged_into uuid,
    split_from uuid,
    internal_notes text DEFAULT ''::text NOT NULL,
    priority text DEFAULT 'normal'::text NOT NULL
);

ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id);
ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_merged_into_fkey FOREIGN KEY (merged_into) REFERENCES public.orders(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_split_from_fkey FOREIGN KEY (split_from) REFERENCES public.orders(id) ON DELETE SET NULL;

-- 15. Order Groups (merge/split tracking)
CREATE TABLE public.order_groups (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    group_type text NOT NULL,
    source_order_ids uuid[] NOT NULL,
    target_order_ids uuid[] NOT NULL,
    notes text,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.order_groups
    ADD CONSTRAINT order_groups_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.order_groups
    ADD CONSTRAINT order_groups_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.order_groups
    ADD CONSTRAINT order_groups_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- 16. Shipments
CREATE TABLE public.shipments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    provider text NOT NULL,
    integration_id uuid,
    tracking_number character varying(255),
    status text DEFAULT 'created'::text NOT NULL,
    label_url character varying(1024),
    carrier_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    warehouse_id uuid,
    external_id character varying(255),
    package_number integer DEFAULT 1 NOT NULL,
    weight numeric(10,3),
    dimensions_length numeric(10,2),
    dimensions_width numeric(10,2),
    dimensions_height numeric(10,2),
    notes text,
    carbon_kg numeric(8,3),
    distance_km numeric(10,1),
    carbon_method text DEFAULT 'estimate'::text
);

ALTER TABLE ONLY public.shipments
    ADD CONSTRAINT shipments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.shipments
    ADD CONSTRAINT shipments_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.shipments
    ADD CONSTRAINT shipments_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.shipments
    ADD CONSTRAINT shipments_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE SET NULL;

-- 17. Returns
CREATE TABLE public.returns (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    status character varying(20) DEFAULT 'requested'::character varying NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    items jsonb DEFAULT '[]'::jsonb NOT NULL,
    refund_amount numeric(12,2) DEFAULT 0 NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    return_token text,
    customer_email text,
    customer_notes text
);

ALTER TABLE ONLY public.returns
    ADD CONSTRAINT returns_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.returns
    ADD CONSTRAINT returns_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.returns
    ADD CONSTRAINT returns_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id) ON DELETE CASCADE;

-- 18. Invoices
CREATE TABLE public.invoices (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    provider text NOT NULL,
    external_id text,
    external_number text,
    status text DEFAULT 'draft'::text NOT NULL,
    invoice_type text DEFAULT 'vat'::text NOT NULL,
    total_net numeric(12,2),
    total_gross numeric(12,2),
    currency text DEFAULT 'PLN'::text NOT NULL,
    issue_date date,
    due_date date,
    pdf_url text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ksef_number text,
    ksef_status text DEFAULT 'not_sent'::text NOT NULL,
    ksef_sent_at timestamp with time zone,
    ksef_response jsonb
);

ALTER TABLE ONLY public.invoices
    ADD CONSTRAINT invoices_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.invoices
    ADD CONSTRAINT invoices_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.invoices
    ADD CONSTRAINT invoices_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id) ON DELETE CASCADE;

-- 19. Warehouses
CREATE TABLE public.warehouses (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    code text,
    address jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.warehouses
    ADD CONSTRAINT warehouses_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.warehouses
    ADD CONSTRAINT warehouses_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- Add warehouse FK to shipments (deferred due to creation order)
ALTER TABLE ONLY public.shipments
    ADD CONSTRAINT shipments_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);

-- 20. Warehouse Stock
CREATE TABLE public.warehouse_stock (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    warehouse_id uuid NOT NULL,
    product_id uuid NOT NULL,
    variant_id uuid,
    quantity integer DEFAULT 0 NOT NULL,
    reserved integer DEFAULT 0 NOT NULL,
    min_stock integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_quantity_non_negative CHECK ((quantity >= 0)),
    CONSTRAINT chk_reserved_non_negative CHECK ((reserved >= 0))
);

ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_warehouse_id_product_id_variant_id_key UNIQUE (warehouse_id, product_id, variant_id);
ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_stock
    ADD CONSTRAINT warehouse_stock_variant_id_fkey FOREIGN KEY (variant_id) REFERENCES public.product_variants(id) ON DELETE CASCADE;

-- 21. Warehouse Documents (PZ/WZ/MM)
CREATE TABLE public.warehouse_documents (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    document_number text NOT NULL,
    document_type text NOT NULL,
    status text DEFAULT 'draft'::text NOT NULL,
    warehouse_id uuid NOT NULL,
    target_warehouse_id uuid,
    supplier_id uuid,
    order_id uuid,
    notes text,
    confirmed_at timestamp with time zone,
    confirmed_by uuid,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_target_warehouse_id_fkey FOREIGN KEY (target_warehouse_id) REFERENCES public.warehouses(id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_confirmed_by_fkey FOREIGN KEY (confirmed_by) REFERENCES public.users(id);
ALTER TABLE ONLY public.warehouse_documents
    ADD CONSTRAINT warehouse_documents_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- 22. Warehouse Document Items
CREATE TABLE public.warehouse_document_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    document_id uuid NOT NULL,
    product_id uuid NOT NULL,
    variant_id uuid,
    quantity integer NOT NULL,
    unit_price numeric(12,2),
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.warehouse_document_items
    ADD CONSTRAINT warehouse_document_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.warehouse_document_items
    ADD CONSTRAINT warehouse_document_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_document_items
    ADD CONSTRAINT warehouse_document_items_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.warehouse_documents(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.warehouse_document_items
    ADD CONSTRAINT warehouse_document_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
ALTER TABLE ONLY public.warehouse_document_items
    ADD CONSTRAINT warehouse_document_items_variant_id_fkey FOREIGN KEY (variant_id) REFERENCES public.product_variants(id);

-- 23. Stocktakes
CREATE TABLE public.stocktakes (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    warehouse_id uuid NOT NULL,
    name text NOT NULL,
    status text DEFAULT 'draft'::text NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    notes text,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.stocktakes
    ADD CONSTRAINT stocktakes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.stocktakes
    ADD CONSTRAINT stocktakes_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stocktakes
    ADD CONSTRAINT stocktakes_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);
ALTER TABLE ONLY public.stocktakes
    ADD CONSTRAINT stocktakes_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- 24. Stocktake Items
CREATE TABLE public.stocktake_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    stocktake_id uuid NOT NULL,
    product_id uuid NOT NULL,
    expected_quantity integer DEFAULT 0 NOT NULL,
    counted_quantity integer,
    difference integer GENERATED ALWAYS AS ((COALESCE(counted_quantity, 0) - expected_quantity)) STORED,
    notes text,
    counted_at timestamp with time zone,
    counted_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.stocktake_items
    ADD CONSTRAINT stocktake_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.stocktake_items
    ADD CONSTRAINT stocktake_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stocktake_items
    ADD CONSTRAINT stocktake_items_stocktake_id_fkey FOREIGN KEY (stocktake_id) REFERENCES public.stocktakes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stocktake_items
    ADD CONSTRAINT stocktake_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
ALTER TABLE ONLY public.stocktake_items
    ADD CONSTRAINT stocktake_items_counted_by_fkey FOREIGN KEY (counted_by) REFERENCES public.users(id);

-- 25. Automation Rules
CREATE TABLE public.automation_rules (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    enabled boolean DEFAULT true NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    trigger_event text NOT NULL,
    conditions jsonb DEFAULT '[]'::jsonb NOT NULL,
    actions jsonb DEFAULT '[]'::jsonb NOT NULL,
    last_fired_at timestamp with time zone,
    fire_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.automation_rules
    ADD CONSTRAINT automation_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.automation_rules
    ADD CONSTRAINT automation_rules_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 26. Automation Rule Logs
CREATE TABLE public.automation_rule_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    rule_id uuid NOT NULL,
    trigger_event text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    conditions_met boolean NOT NULL,
    actions_executed jsonb DEFAULT '[]'::jsonb NOT NULL,
    error_message text,
    executed_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.automation_rule_logs
    ADD CONSTRAINT automation_rule_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.automation_rule_logs
    ADD CONSTRAINT automation_rule_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.automation_rule_logs
    ADD CONSTRAINT automation_rule_logs_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES public.automation_rules(id) ON DELETE CASCADE;

-- 27. Automation Delayed Actions
CREATE TABLE public.automation_delayed_actions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    rule_id uuid NOT NULL,
    action_index integer NOT NULL,
    order_id uuid,
    execute_at timestamp with time zone NOT NULL,
    executed boolean DEFAULT false NOT NULL,
    executed_at timestamp with time zone,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    action_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    event_data jsonb DEFAULT '{}'::jsonb NOT NULL
);

ALTER TABLE ONLY public.automation_delayed_actions
    ADD CONSTRAINT automation_delayed_actions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.automation_delayed_actions
    ADD CONSTRAINT automation_delayed_actions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.automation_delayed_actions
    ADD CONSTRAINT automation_delayed_actions_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES public.automation_rules(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.automation_delayed_actions
    ADD CONSTRAINT automation_delayed_actions_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);

-- 28. Exchange Rates
CREATE TABLE public.exchange_rates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    base_currency text DEFAULT 'PLN'::text NOT NULL,
    target_currency text NOT NULL,
    rate numeric(12,6) NOT NULL,
    source text DEFAULT 'manual'::text NOT NULL,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.exchange_rates
    ADD CONSTRAINT exchange_rates_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.exchange_rates
    ADD CONSTRAINT exchange_rates_tenant_id_base_currency_target_currency_key UNIQUE (tenant_id, base_currency, target_currency);
ALTER TABLE ONLY public.exchange_rates
    ADD CONSTRAINT exchange_rates_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 29. Sync Jobs
CREATE TABLE public.sync_jobs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    job_type text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    items_processed integer DEFAULT 0 NOT NULL,
    items_failed integer DEFAULT 0 NOT NULL,
    error_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.sync_jobs
    ADD CONSTRAINT sync_jobs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.sync_jobs
    ADD CONSTRAINT sync_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.sync_jobs
    ADD CONSTRAINT sync_jobs_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE;

-- 30. Supplier Products
CREATE TABLE public.supplier_products (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    supplier_id uuid NOT NULL,
    product_id uuid,
    external_id text NOT NULL,
    name text NOT NULL,
    ean text,
    sku text,
    price numeric(12,2),
    stock_quantity integer DEFAULT 0 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_synced_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source_category text
);

ALTER TABLE ONLY public.supplier_products
    ADD CONSTRAINT supplier_products_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.supplier_products
    ADD CONSTRAINT supplier_products_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.supplier_products
    ADD CONSTRAINT supplier_products_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.supplier_products
    ADD CONSTRAINT supplier_products_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE SET NULL;

-- 31. Supplier Category Mappings
CREATE TABLE public.supplier_category_mappings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    supplier_id uuid NOT NULL,
    source_category text NOT NULL,
    category_id uuid,
    auto_matched boolean DEFAULT false NOT NULL,
    confirmed boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.supplier_category_mappings
    ADD CONSTRAINT supplier_category_mappings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.supplier_category_mappings
    ADD CONSTRAINT supplier_category_mappings_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.supplier_category_mappings
    ADD CONSTRAINT supplier_category_mappings_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.supplier_category_mappings
    ADD CONSTRAINT supplier_category_mappings_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.product_categories(id) ON DELETE SET NULL;

-- 32. Audit Log
CREATE TABLE public.audit_log (
    id bigint NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid,
    action character varying(100) NOT NULL,
    entity_type character varying(50) NOT NULL,
    entity_id uuid,
    changes jsonb DEFAULT '{}'::jsonb NOT NULL,
    ip_address inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE public.audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.audit_log_id_seq OWNED BY public.audit_log.id;
ALTER TABLE ONLY public.audit_log ALTER COLUMN id SET DEFAULT nextval('public.audit_log_id_seq'::regclass);

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;

-- 33. Webhook Events (incoming)
CREATE TABLE public.webhook_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    provider text NOT NULL,
    event_type character varying(100) NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'received'::text NOT NULL,
    processed_at timestamp with time zone,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.webhook_events
    ADD CONSTRAINT webhook_events_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.webhook_events
    ADD CONSTRAINT webhook_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 34. Webhook Deliveries (outgoing)
CREATE TABLE public.webhook_deliveries (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    url text NOT NULL,
    event_type character varying(100) NOT NULL,
    payload jsonb NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    response_code integer,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- 35. Invitations
CREATE TABLE public.invitations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    email text NOT NULL,
    token text NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_token_key UNIQUE (token);
ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- 36. Purchase Orders
CREATE TABLE public.purchase_orders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    po_number text NOT NULL,
    supplier_id uuid,
    supplier_name text NOT NULL,
    warehouse_id uuid,
    status text DEFAULT 'draft'::text NOT NULL,
    notes text,
    expected_delivery_date date,
    total_amount numeric(12,2) DEFAULT 0,
    currency text DEFAULT 'PLN'::text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_tenant_id_po_number_key UNIQUE (tenant_id, po_number);
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- 37. Purchase Order Items
CREATE TABLE public.purchase_order_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    purchase_order_id uuid NOT NULL,
    product_id uuid,
    sku text NOT NULL,
    product_name text NOT NULL,
    quantity_ordered integer NOT NULL,
    quantity_received integer DEFAULT 0 NOT NULL,
    unit_cost numeric(12,2) NOT NULL,
    total_cost numeric(12,2) GENERATED ALWAYS AS (((quantity_ordered)::numeric * unit_cost)) STORED,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.purchase_order_items
    ADD CONSTRAINT purchase_order_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.purchase_order_items
    ADD CONSTRAINT purchase_order_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.purchase_order_items
    ADD CONSTRAINT purchase_order_items_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.purchase_order_items
    ADD CONSTRAINT purchase_order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

-- 38-57. Remaining tables (payment, pick-pack, dropship, supplier portal, recurring, repricing, segments, loyalty, listing sync, stock sync)

CREATE TABLE public.payment_settlements (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    provider text NOT NULL,
    settlement_id text,
    settlement_date date NOT NULL,
    total_amount numeric(12,2) NOT NULL,
    fee_amount numeric(12,2) DEFAULT 0 NOT NULL,
    net_amount numeric(12,2) NOT NULL,
    currency text DEFAULT 'PLN'::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    notes text,
    imported_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.payment_settlements ADD CONSTRAINT payment_settlements_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.payment_settlements ADD CONSTRAINT payment_settlements_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

CREATE TABLE public.payment_transactions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    settlement_id uuid,
    order_id uuid,
    provider text NOT NULL,
    external_transaction_id text,
    amount numeric(12,2) NOT NULL,
    fee numeric(12,2) DEFAULT 0 NOT NULL,
    net_amount numeric(12,2) NOT NULL,
    currency text DEFAULT 'PLN'::text NOT NULL,
    transaction_type text DEFAULT 'payment'::text NOT NULL,
    transaction_date timestamp with time zone NOT NULL,
    match_status text DEFAULT 'unmatched'::text NOT NULL,
    match_notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.payment_transactions ADD CONSTRAINT payment_transactions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.payment_transactions ADD CONSTRAINT payment_transactions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.payment_transactions ADD CONSTRAINT payment_transactions_settlement_id_fkey FOREIGN KEY (settlement_id) REFERENCES public.payment_settlements(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.payment_transactions ADD CONSTRAINT payment_transactions_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);

CREATE TABLE public.pick_pack_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    session_type text DEFAULT 'single'::text NOT NULL,
    status text DEFAULT 'picking'::text NOT NULL,
    assigned_to uuid,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.pick_pack_sessions ADD CONSTRAINT pick_pack_sessions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.pick_pack_sessions ADD CONSTRAINT pick_pack_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.pick_pack_sessions ADD CONSTRAINT pick_pack_sessions_assigned_to_fkey FOREIGN KEY (assigned_to) REFERENCES public.users(id);

CREATE TABLE public.pick_pack_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    session_id uuid NOT NULL,
    order_id uuid NOT NULL,
    product_id uuid,
    sku text NOT NULL,
    product_name text NOT NULL,
    quantity_required integer NOT NULL,
    quantity_picked integer DEFAULT 0 NOT NULL,
    quantity_packed integer DEFAULT 0 NOT NULL,
    pick_location text,
    picked_at timestamp with time zone,
    packed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.pick_pack_items ADD CONSTRAINT pick_pack_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.pick_pack_items ADD CONSTRAINT pick_pack_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.pick_pack_items ADD CONSTRAINT pick_pack_items_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.pick_pack_sessions(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.pick_pack_items ADD CONSTRAINT pick_pack_items_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);
ALTER TABLE ONLY public.pick_pack_items ADD CONSTRAINT pick_pack_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

CREATE TABLE public.dropship_orders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    supplier_id uuid NOT NULL,
    supplier_name text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    supplier_reference text,
    tracking_number text,
    carrier text,
    notes text,
    total_cost numeric(12,2) DEFAULT 0 NOT NULL,
    currency text DEFAULT 'PLN'::text NOT NULL,
    sent_at timestamp with time zone,
    confirmed_at timestamp with time zone,
    shipped_at timestamp with time zone,
    delivered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.dropship_orders ADD CONSTRAINT dropship_orders_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.dropship_orders ADD CONSTRAINT dropship_orders_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.dropship_orders ADD CONSTRAINT dropship_orders_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders(id);
ALTER TABLE ONLY public.dropship_orders ADD CONSTRAINT dropship_orders_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);

CREATE TABLE public.dropship_order_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    dropship_order_id uuid NOT NULL,
    product_id uuid,
    sku text NOT NULL,
    product_name text NOT NULL,
    quantity integer NOT NULL,
    unit_cost numeric(12,2) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.dropship_order_items ADD CONSTRAINT dropship_order_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.dropship_order_items ADD CONSTRAINT dropship_order_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.dropship_order_items ADD CONSTRAINT dropship_order_items_dropship_order_id_fkey FOREIGN KEY (dropship_order_id) REFERENCES public.dropship_orders(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.dropship_order_items ADD CONSTRAINT dropship_order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

CREATE TABLE public.supplier_portal_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    supplier_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.supplier_portal_tokens ADD CONSTRAINT supplier_portal_tokens_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.supplier_portal_tokens ADD CONSTRAINT supplier_portal_tokens_token_hash_key UNIQUE (token_hash);
ALTER TABLE ONLY public.supplier_portal_tokens ADD CONSTRAINT supplier_portal_tokens_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.supplier_portal_tokens ADD CONSTRAINT supplier_portal_tokens_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);

CREATE TABLE public.supplier_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    purchase_order_id uuid NOT NULL,
    supplier_id uuid,
    user_id uuid,
    message text NOT NULL,
    is_from_supplier boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.supplier_messages ADD CONSTRAINT supplier_messages_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.supplier_messages ADD CONSTRAINT supplier_messages_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.supplier_messages ADD CONSTRAINT supplier_messages_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.supplier_messages ADD CONSTRAINT supplier_messages_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
ALTER TABLE ONLY public.supplier_messages ADD CONSTRAINT supplier_messages_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);

CREATE TABLE public.recurring_orders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    customer_id uuid,
    customer_name text NOT NULL,
    customer_email text,
    status text DEFAULT 'active'::text NOT NULL,
    frequency text NOT NULL,
    interval_days integer NOT NULL,
    next_order_date date NOT NULL,
    last_order_date date,
    end_date date,
    total_orders_created integer DEFAULT 0 NOT NULL,
    max_orders integer,
    shipping_address jsonb,
    notes text,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.recurring_orders ADD CONSTRAINT recurring_orders_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recurring_orders ADD CONSTRAINT recurring_orders_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.recurring_orders ADD CONSTRAINT recurring_orders_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id);
ALTER TABLE ONLY public.recurring_orders ADD CONSTRAINT recurring_orders_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

CREATE TABLE public.recurring_order_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    recurring_order_id uuid NOT NULL,
    product_id uuid,
    sku text NOT NULL,
    product_name text NOT NULL,
    quantity integer NOT NULL,
    unit_price numeric(12,2) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.recurring_order_items ADD CONSTRAINT recurring_order_items_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recurring_order_items ADD CONSTRAINT recurring_order_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.recurring_order_items ADD CONSTRAINT recurring_order_items_recurring_order_id_fkey FOREIGN KEY (recurring_order_id) REFERENCES public.recurring_orders(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recurring_order_items ADD CONSTRAINT recurring_order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

CREATE TABLE public.repricing_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    strategy text NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    scope_type text DEFAULT 'all'::text NOT NULL,
    scope_value jsonb,
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    min_price numeric(12,2),
    max_price numeric(12,2),
    channels jsonb DEFAULT '["internal"]'::jsonb,
    last_applied_at timestamp with time zone,
    products_affected integer DEFAULT 0 NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.repricing_rules ADD CONSTRAINT repricing_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.repricing_rules ADD CONSTRAINT repricing_rules_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.repricing_rules ADD CONSTRAINT repricing_rules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

CREATE TABLE public.repricing_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    rule_id uuid NOT NULL,
    product_id uuid NOT NULL,
    old_price numeric(12,2) NOT NULL,
    new_price numeric(12,2) NOT NULL,
    reason text,
    channel text DEFAULT 'internal'::text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.repricing_log ADD CONSTRAINT repricing_log_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.repricing_log ADD CONSTRAINT repricing_log_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.repricing_log ADD CONSTRAINT repricing_log_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES public.repricing_rules(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.repricing_log ADD CONSTRAINT repricing_log_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

CREATE TABLE public.customer_segments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    color text DEFAULT '#6366f1'::text,
    segment_type text DEFAULT 'manual'::text NOT NULL,
    rules jsonb,
    customer_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.customer_segments ADD CONSTRAINT customer_segments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.customer_segments ADD CONSTRAINT customer_segments_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

CREATE TABLE public.customer_segment_members (
    tenant_id uuid NOT NULL,
    segment_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    added_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.customer_segment_members ADD CONSTRAINT customer_segment_members_pkey PRIMARY KEY (segment_id, customer_id);
ALTER TABLE ONLY public.customer_segment_members ADD CONSTRAINT customer_segment_members_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.customer_segment_members ADD CONSTRAINT customer_segment_members_segment_id_fkey FOREIGN KEY (segment_id) REFERENCES public.customer_segments(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.customer_segment_members ADD CONSTRAINT customer_segment_members_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;

CREATE TABLE public.loyalty_programs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    program_type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.loyalty_programs ADD CONSTRAINT loyalty_programs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.loyalty_programs ADD CONSTRAINT loyalty_programs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

CREATE TABLE public.customer_loyalty (
    tenant_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    program_id uuid NOT NULL,
    points_balance integer DEFAULT 0 NOT NULL,
    total_points_earned integer DEFAULT 0 NOT NULL,
    total_points_redeemed integer DEFAULT 0 NOT NULL,
    current_tier text,
    total_spent numeric(12,2) DEFAULT 0 NOT NULL,
    order_count integer DEFAULT 0 NOT NULL,
    last_activity_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.customer_loyalty ADD CONSTRAINT customer_loyalty_pkey PRIMARY KEY (customer_id, program_id);
ALTER TABLE ONLY public.customer_loyalty ADD CONSTRAINT customer_loyalty_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.customer_loyalty ADD CONSTRAINT customer_loyalty_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id);
ALTER TABLE ONLY public.customer_loyalty ADD CONSTRAINT customer_loyalty_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE CASCADE;

CREATE TABLE public.listing_sync_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    sync_direction text DEFAULT 'bidirectional'::text NOT NULL,
    auto_sync boolean DEFAULT false NOT NULL,
    sync_interval_minutes integer DEFAULT 30 NOT NULL,
    field_mapping jsonb DEFAULT '{}'::jsonb NOT NULL,
    price_rule text DEFAULT 'same'::text NOT NULL,
    price_modifier numeric(10,2) DEFAULT 0 NOT NULL,
    stock_buffer integer DEFAULT 0 NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    last_sync_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.listing_sync_configs ADD CONSTRAINT listing_sync_configs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.listing_sync_configs ADD CONSTRAINT listing_sync_configs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.listing_sync_configs ADD CONSTRAINT listing_sync_configs_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE;

CREATE TABLE public.listing_sync_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    config_id uuid NOT NULL,
    direction text NOT NULL,
    entity_type text NOT NULL,
    product_id uuid,
    external_id text,
    status text DEFAULT 'success'::text NOT NULL,
    changes jsonb,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.listing_sync_log ADD CONSTRAINT listing_sync_log_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.listing_sync_log ADD CONSTRAINT listing_sync_log_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.listing_sync_log ADD CONSTRAINT listing_sync_log_config_id_fkey FOREIGN KEY (config_id) REFERENCES public.listing_sync_configs(id) ON DELETE CASCADE;

CREATE TABLE public.stock_sync_channels (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    integration_id uuid,
    channel_type text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    stock_buffer integer DEFAULT 0 NOT NULL,
    sync_mode text DEFAULT 'realtime'::text NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    last_sync_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.stock_sync_channels ADD CONSTRAINT stock_sync_channels_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.stock_sync_channels ADD CONSTRAINT stock_sync_channels_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stock_sync_channels ADD CONSTRAINT stock_sync_channels_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE SET NULL;

CREATE TABLE public.stock_sync_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    product_id uuid,
    sku text,
    trigger_type text NOT NULL,
    old_quantity integer DEFAULT 0 NOT NULL,
    new_quantity integer DEFAULT 0 NOT NULL,
    available_quantity integer DEFAULT 0 NOT NULL,
    channels_notified integer DEFAULT 0 NOT NULL,
    channels_failed integer DEFAULT 0 NOT NULL,
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
ALTER TABLE ONLY public.stock_sync_events ADD CONSTRAINT stock_sync_events_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.stock_sync_events ADD CONSTRAINT stock_sync_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stock_sync_events ADD CONSTRAINT stock_sync_events_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE SET NULL;

-- ============================================================
-- Indexes
-- ============================================================

CREATE INDEX idx_users_tenant ON public.users USING btree (tenant_id);
CREATE INDEX idx_roles_tenant ON public.roles USING btree (tenant_id);
CREATE INDEX idx_integrations_tenant ON public.integrations USING btree (tenant_id, provider);
CREATE UNIQUE INDEX idx_integrations_tenant_provider_label ON public.integrations USING btree (tenant_id, provider, COALESCE(label, ''::text));
CREATE INDEX idx_products_tenant ON public.products USING btree (tenant_id);
CREATE UNIQUE INDEX idx_products_tenant_sku ON public.products USING btree (tenant_id, sku) WHERE ((sku IS NOT NULL) AND ((sku)::text <> ''::text));
CREATE INDEX idx_products_ean ON public.products USING btree (tenant_id, ean) WHERE (ean IS NOT NULL);
CREATE INDEX idx_products_name_trgm ON public.products USING gin (name public.gin_trgm_ops);
CREATE INDEX idx_products_category_id ON public.products USING btree (tenant_id, category_id);
CREATE INDEX idx_product_variants_tenant ON public.product_variants USING btree (tenant_id);
CREATE INDEX idx_product_variants_product ON public.product_variants USING btree (product_id);
CREATE INDEX idx_product_variants_sku ON public.product_variants USING btree (sku) WHERE (sku IS NOT NULL);
CREATE INDEX idx_product_variants_ean ON public.product_variants USING btree (ean) WHERE (ean IS NOT NULL);
CREATE INDEX idx_product_variants_tenant_sku ON public.product_variants USING btree (tenant_id, sku) WHERE (sku IS NOT NULL);
CREATE INDEX idx_product_variants_tenant_ean ON public.product_variants USING btree (tenant_id, ean) WHERE (ean IS NOT NULL);
CREATE INDEX idx_product_bundles_bundle ON public.product_bundles USING btree (bundle_product_id);
CREATE INDEX idx_product_bundles_component ON public.product_bundles USING btree (component_product_id);
CREATE INDEX idx_product_categories_tenant ON public.product_categories USING btree (tenant_id);
CREATE INDEX idx_product_categories_parent ON public.product_categories USING btree (tenant_id, parent_id);
CREATE UNIQUE INDEX idx_product_categories_slug ON public.product_categories USING btree (tenant_id, slug);
CREATE INDEX idx_product_listings_tenant ON public.product_listings USING btree (tenant_id);
CREATE INDEX idx_product_listings_product ON public.product_listings USING btree (tenant_id, product_id);
CREATE INDEX idx_product_listings_integration ON public.product_listings USING btree (tenant_id, integration_id);
CREATE UNIQUE INDEX idx_product_listings_ext ON public.product_listings USING btree (tenant_id, integration_id, external_id) WHERE (external_id IS NOT NULL);
CREATE INDEX idx_product_listings_auto_sync ON public.product_listings USING btree (tenant_id, product_id) WHERE ((status = 'active'::text) AND (external_id IS NOT NULL) AND (stock_sync_mode = 'auto'::text));
CREATE INDEX idx_price_lists_tenant ON public.price_lists USING btree (tenant_id);
CREATE INDEX idx_price_list_items_list ON public.price_list_items USING btree (price_list_id);
CREATE INDEX idx_price_list_items_product ON public.price_list_items USING btree (product_id);
CREATE INDEX idx_customers_tenant ON public.customers USING btree (tenant_id);
CREATE INDEX idx_customers_email ON public.customers USING btree (email);
CREATE INDEX idx_customers_name ON public.customers USING btree (name);
CREATE INDEX idx_customers_phone ON public.customers USING btree (phone);
CREATE INDEX idx_customers_email_trgm ON public.customers USING gin (email public.gin_trgm_ops);
CREATE INDEX idx_customers_name_trgm ON public.customers USING gin (name public.gin_trgm_ops);
CREATE INDEX idx_orders_tenant_status ON public.orders USING btree (tenant_id, status, created_at DESC);
CREATE INDEX idx_orders_tenant_source ON public.orders USING btree (tenant_id, source);
CREATE INDEX idx_orders_external ON public.orders USING btree (tenant_id, source, external_id);
CREATE INDEX idx_orders_customer ON public.orders USING btree (customer_id) WHERE (customer_id IS NOT NULL);
CREATE INDEX idx_orders_delivery_method ON public.orders USING btree (tenant_id, delivery_method) WHERE (delivery_method IS NOT NULL);
CREATE INDEX idx_orders_customer_name_trgm ON public.orders USING gin (customer_name public.gin_trgm_ops);
CREATE INDEX idx_orders_customer_email_trgm ON public.orders USING gin (customer_email public.gin_trgm_ops);
CREATE INDEX idx_order_groups_tenant ON public.order_groups USING btree (tenant_id);
CREATE INDEX idx_shipments_tenant ON public.shipments USING btree (tenant_id, status);
CREATE INDEX idx_shipments_order ON public.shipments USING btree (order_id);
CREATE INDEX idx_shipments_tracking ON public.shipments USING btree (tracking_number) WHERE (tracking_number IS NOT NULL);
CREATE INDEX idx_shipments_external ON public.shipments USING btree (tenant_id, external_id) WHERE (external_id IS NOT NULL);
CREATE INDEX idx_shipments_order_package ON public.shipments USING btree (order_id, package_number);
CREATE INDEX idx_returns_tenant ON public.returns USING btree (tenant_id, created_at DESC);
CREATE INDEX idx_returns_order ON public.returns USING btree (tenant_id, order_id);
CREATE UNIQUE INDEX idx_returns_token ON public.returns USING btree (return_token) WHERE (return_token IS NOT NULL);
CREATE INDEX idx_invoices_tenant ON public.invoices USING btree (tenant_id, created_at DESC);
CREATE INDEX idx_invoices_order ON public.invoices USING btree (tenant_id, order_id);
CREATE INDEX idx_invoices_ksef_status ON public.invoices USING btree (ksef_status) WHERE (ksef_status <> 'not_sent'::text);
CREATE INDEX idx_warehouses_tenant ON public.warehouses USING btree (tenant_id);
CREATE INDEX idx_warehouse_stock_warehouse ON public.warehouse_stock USING btree (warehouse_id);
CREATE INDEX idx_warehouse_stock_product ON public.warehouse_stock USING btree (product_id);
CREATE INDEX idx_warehouse_docs_tenant ON public.warehouse_documents USING btree (tenant_id);
CREATE UNIQUE INDEX idx_warehouse_docs_number ON public.warehouse_documents USING btree (tenant_id, document_number);
CREATE INDEX idx_warehouse_docs_status ON public.warehouse_documents USING btree (status);
CREATE INDEX idx_warehouse_docs_type ON public.warehouse_documents USING btree (document_type);
CREATE INDEX idx_warehouse_docs_warehouse ON public.warehouse_documents USING btree (warehouse_id);
CREATE INDEX idx_warehouse_doc_items_document ON public.warehouse_document_items USING btree (document_id);
CREATE INDEX idx_warehouse_doc_items_product ON public.warehouse_document_items USING btree (product_id);
CREATE INDEX idx_stocktakes_tenant ON public.stocktakes USING btree (tenant_id);
CREATE INDEX idx_stocktakes_warehouse ON public.stocktakes USING btree (warehouse_id);
CREATE INDEX idx_stocktakes_status ON public.stocktakes USING btree (status);
CREATE INDEX idx_stocktake_items_stocktake ON public.stocktake_items USING btree (stocktake_id);
CREATE INDEX idx_stocktake_items_product ON public.stocktake_items USING btree (product_id);
CREATE INDEX idx_automation_rules_tenant_enabled ON public.automation_rules USING btree (tenant_id, enabled);
CREATE INDEX idx_automation_rule_logs_rule_id ON public.automation_rule_logs USING btree (rule_id);
CREATE INDEX idx_automation_rule_logs_executed_at ON public.automation_rule_logs USING btree (executed_at DESC);
CREATE INDEX idx_delayed_actions_tenant ON public.automation_delayed_actions USING btree (tenant_id);
CREATE INDEX idx_delayed_actions_execute ON public.automation_delayed_actions USING btree (execute_at) WHERE (NOT executed);
CREATE INDEX idx_sync_jobs_tenant ON public.sync_jobs USING btree (tenant_id, created_at DESC);
CREATE INDEX idx_sync_jobs_integration ON public.sync_jobs USING btree (tenant_id, integration_id, created_at DESC);
CREATE INDEX idx_sync_jobs_status ON public.sync_jobs USING btree (tenant_id, status) WHERE (status = ANY (ARRAY['pending'::text, 'running'::text]));
CREATE INDEX idx_suppliers_tenant ON public.suppliers USING btree (tenant_id);
CREATE INDEX idx_suppliers_integration_id ON public.suppliers USING btree (integration_id) WHERE (integration_id IS NOT NULL);
CREATE INDEX idx_supplier_products_tenant ON public.supplier_products USING btree (tenant_id);
CREATE INDEX idx_supplier_products_supplier ON public.supplier_products USING btree (supplier_id);
CREATE UNIQUE INDEX idx_supplier_products_unique ON public.supplier_products USING btree (tenant_id, supplier_id, external_id);
CREATE INDEX idx_supplier_products_ean ON public.supplier_products USING btree (tenant_id, ean) WHERE (ean IS NOT NULL);
CREATE INDEX idx_supplier_category_mappings_supplier ON public.supplier_category_mappings USING btree (supplier_id);
CREATE UNIQUE INDEX idx_supplier_category_mapping_unique ON public.supplier_category_mappings USING btree (tenant_id, supplier_id, source_category);
CREATE INDEX idx_audit_entity ON public.audit_log USING btree (tenant_id, entity_type, entity_id);
CREATE INDEX idx_webhook_deliveries_tenant ON public.webhook_deliveries USING btree (tenant_id, created_at DESC);
CREATE INDEX idx_invitations_tenant_id ON public.invitations USING btree (tenant_id);
CREATE INDEX idx_invitations_email ON public.invitations USING btree (email);
CREATE INDEX idx_invitations_token ON public.invitations USING btree (token);
CREATE INDEX idx_purchase_orders_tenant ON public.purchase_orders USING btree (tenant_id);
CREATE INDEX idx_purchase_orders_supplier ON public.purchase_orders USING btree (supplier_id);
CREATE INDEX idx_purchase_orders_status ON public.purchase_orders USING btree (status);
CREATE INDEX idx_purchase_order_items_po ON public.purchase_order_items USING btree (purchase_order_id);
CREATE INDEX idx_purchase_order_items_product ON public.purchase_order_items USING btree (product_id);
CREATE INDEX idx_payment_settlements_tenant ON public.payment_settlements USING btree (tenant_id);
CREATE INDEX idx_payment_settlements_provider ON public.payment_settlements USING btree (provider);
CREATE INDEX idx_payment_settlements_date ON public.payment_settlements USING btree (settlement_date);
CREATE INDEX idx_payment_transactions_settlement ON public.payment_transactions USING btree (settlement_id);
CREATE INDEX idx_payment_transactions_order ON public.payment_transactions USING btree (order_id);
CREATE INDEX idx_payment_transactions_external ON public.payment_transactions USING btree (external_transaction_id);
CREATE INDEX idx_payment_transactions_match ON public.payment_transactions USING btree (match_status);
CREATE INDEX idx_pick_pack_sessions_tenant ON public.pick_pack_sessions USING btree (tenant_id);
CREATE INDEX idx_pick_pack_sessions_status ON public.pick_pack_sessions USING btree (status);
CREATE INDEX idx_pick_pack_sessions_assigned ON public.pick_pack_sessions USING btree (assigned_to);
CREATE INDEX idx_pick_pack_items_session ON public.pick_pack_items USING btree (session_id);
CREATE INDEX idx_pick_pack_items_order ON public.pick_pack_items USING btree (order_id);
CREATE INDEX idx_dropship_orders_tenant ON public.dropship_orders USING btree (tenant_id);
CREATE INDEX idx_dropship_orders_order ON public.dropship_orders USING btree (order_id);
CREATE INDEX idx_dropship_orders_supplier ON public.dropship_orders USING btree (supplier_id);
CREATE INDEX idx_dropship_orders_status ON public.dropship_orders USING btree (status);
CREATE INDEX idx_dropship_items_order ON public.dropship_order_items USING btree (dropship_order_id);
CREATE INDEX idx_supplier_portal_tokens_hash ON public.supplier_portal_tokens USING btree (token_hash);
CREATE INDEX idx_supplier_portal_tokens_supplier ON public.supplier_portal_tokens USING btree (supplier_id);
CREATE INDEX idx_supplier_messages_po ON public.supplier_messages USING btree (purchase_order_id);
CREATE INDEX idx_recurring_orders_tenant ON public.recurring_orders USING btree (tenant_id);
CREATE INDEX idx_recurring_orders_customer ON public.recurring_orders USING btree (customer_id);
CREATE INDEX idx_recurring_orders_status ON public.recurring_orders USING btree (status);
CREATE INDEX idx_recurring_orders_next_date ON public.recurring_orders USING btree (next_order_date);
CREATE INDEX idx_recurring_order_items_recurring ON public.recurring_order_items USING btree (recurring_order_id);
CREATE INDEX idx_repricing_rules_tenant ON public.repricing_rules USING btree (tenant_id);
CREATE INDEX idx_repricing_rules_status ON public.repricing_rules USING btree (status);
CREATE INDEX idx_repricing_log_rule ON public.repricing_log USING btree (rule_id);
CREATE INDEX idx_repricing_log_product ON public.repricing_log USING btree (product_id);
CREATE INDEX idx_repricing_log_applied ON public.repricing_log USING btree (applied_at);
CREATE INDEX idx_customer_segments_tenant ON public.customer_segments USING btree (tenant_id);
CREATE INDEX idx_segment_members_customer ON public.customer_segment_members USING btree (customer_id);
CREATE INDEX idx_loyalty_programs_tenant ON public.loyalty_programs USING btree (tenant_id);
CREATE INDEX idx_customer_loyalty_customer ON public.customer_loyalty USING btree (customer_id);
CREATE INDEX idx_listing_sync_configs_tenant ON public.listing_sync_configs USING btree (tenant_id);
CREATE INDEX idx_listing_sync_configs_integration ON public.listing_sync_configs USING btree (integration_id);
CREATE INDEX idx_listing_sync_log_config_created ON public.listing_sync_log USING btree (tenant_id, config_id, created_at DESC);
CREATE INDEX idx_stock_sync_channels_tenant ON public.stock_sync_channels USING btree (tenant_id);
CREATE INDEX idx_stock_sync_channels_integration ON public.stock_sync_channels USING btree (integration_id);
CREATE INDEX idx_stock_sync_events_tenant_created ON public.stock_sync_events USING btree (tenant_id, created_at DESC);
CREATE INDEX idx_stock_sync_events_tenant_product ON public.stock_sync_events USING btree (tenant_id, product_id, created_at DESC);

-- ============================================================
-- Triggers (updated_at auto-update)
-- ============================================================

CREATE TRIGGER trigger_tenants_updated_at BEFORE UPDATE ON public.tenants FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_integrations_updated_at BEFORE UPDATE ON public.integrations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_orders_updated_at BEFORE UPDATE ON public.orders FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_products_updated_at BEFORE UPDATE ON public.products FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_product_listings_updated_at BEFORE UPDATE ON public.product_listings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_product_categories_updated_at BEFORE UPDATE ON public.product_categories FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_shipments_updated_at BEFORE UPDATE ON public.shipments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_returns_updated_at BEFORE UPDATE ON public.returns FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_invoices_updated_at BEFORE UPDATE ON public.invoices FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_suppliers_updated_at BEFORE UPDATE ON public.suppliers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER trigger_supplier_products_updated_at BEFORE UPDATE ON public.supplier_products FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON public.automation_rules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_customers_updated_at BEFORE UPDATE ON public.customers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_price_lists_updated_at BEFORE UPDATE ON public.price_lists FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_price_list_items_updated_at BEFORE UPDATE ON public.price_list_items FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_product_bundles_updated_at BEFORE UPDATE ON public.product_bundles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_product_variants_updated_at BEFORE UPDATE ON public.product_variants FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON public.roles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_warehouses_updated_at BEFORE UPDATE ON public.warehouses FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_warehouse_stock_updated_at BEFORE UPDATE ON public.warehouse_stock FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
CREATE TRIGGER update_warehouse_documents_updated_at BEFORE UPDATE ON public.warehouse_documents FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER update_stocktakes_updated_at BEFORE UPDATE ON public.stocktakes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER update_stock_sync_channels_updated_at BEFORE UPDATE ON public.stock_sync_channels FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- ============================================================
-- Row-Level Security (RLS)
-- ============================================================

-- Tenants
ALTER TABLE public.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.tenants USING ((id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Users
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.users FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.users USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Roles
ALTER TABLE public.roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.roles FORCE ROW LEVEL SECURITY;
CREATE POLICY roles_tenant_isolation ON public.roles USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Integrations
ALTER TABLE public.integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.integrations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.integrations USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Products
ALTER TABLE public.products ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.products FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.products USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Product Variants
ALTER TABLE public.product_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_variants FORCE ROW LEVEL SECURITY;
CREATE POLICY product_variants_tenant_isolation ON public.product_variants USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Product Bundles
ALTER TABLE public.product_bundles ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_bundles FORCE ROW LEVEL SECURITY;
CREATE POLICY product_bundles_tenant_isolation ON public.product_bundles USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Product Categories
ALTER TABLE public.product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_categories FORCE ROW LEVEL SECURITY;
CREATE POLICY product_categories_tenant_isolation ON public.product_categories USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Product Listings
ALTER TABLE public.product_listings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_listings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.product_listings USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Price Lists
ALTER TABLE public.price_lists ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.price_lists FORCE ROW LEVEL SECURITY;
CREATE POLICY price_lists_tenant_isolation ON public.price_lists USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Price List Items
ALTER TABLE public.price_list_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.price_list_items FORCE ROW LEVEL SECURITY;
CREATE POLICY price_list_items_tenant_isolation ON public.price_list_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Customers
ALTER TABLE public.customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.customers FORCE ROW LEVEL SECURITY;
CREATE POLICY customers_tenant_isolation ON public.customers USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Suppliers
ALTER TABLE public.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.suppliers FORCE ROW LEVEL SECURITY;
CREATE POLICY suppliers_tenant_isolation ON public.suppliers USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Supplier Products
ALTER TABLE public.supplier_products ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_products FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_products_tenant_isolation ON public.supplier_products USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Supplier Category Mappings
ALTER TABLE public.supplier_category_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_category_mappings FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_category_mappings_tenant_isolation ON public.supplier_category_mappings USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Orders
ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.orders USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Order Groups
ALTER TABLE public.order_groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.order_groups FORCE ROW LEVEL SECURITY;
CREATE POLICY order_groups_tenant_isolation ON public.order_groups USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Shipments
ALTER TABLE public.shipments ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.shipments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.shipments USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Returns
ALTER TABLE public.returns ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.returns FORCE ROW LEVEL SECURITY;
CREATE POLICY returns_tenant_isolation ON public.returns USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Invoices
ALTER TABLE public.invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.invoices FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.invoices USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Warehouses
ALTER TABLE public.warehouses ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.warehouses FORCE ROW LEVEL SECURITY;
CREATE POLICY warehouses_tenant_isolation ON public.warehouses USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Warehouse Stock
ALTER TABLE public.warehouse_stock ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.warehouse_stock FORCE ROW LEVEL SECURITY;
CREATE POLICY warehouse_stock_tenant_isolation ON public.warehouse_stock USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Warehouse Documents
ALTER TABLE public.warehouse_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.warehouse_documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.warehouse_documents USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Warehouse Document Items
ALTER TABLE public.warehouse_document_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.warehouse_document_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.warehouse_document_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Stocktakes
ALTER TABLE public.stocktakes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stocktakes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.stocktakes USING ((tenant_id = COALESCE((NULLIF(current_setting('app.current_tenant_id'::text, true), ''::text))::uuid, tenant_id)));

-- Stocktake Items
ALTER TABLE public.stocktake_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stocktake_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.stocktake_items USING ((tenant_id = COALESCE((NULLIF(current_setting('app.current_tenant_id'::text, true), ''::text))::uuid, tenant_id)));

-- Automation Rules
ALTER TABLE public.automation_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.automation_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY automation_rules_tenant_isolation ON public.automation_rules USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Automation Rule Logs
ALTER TABLE public.automation_rule_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.automation_rule_logs FORCE ROW LEVEL SECURITY;
CREATE POLICY automation_rule_logs_tenant_isolation ON public.automation_rule_logs USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Automation Delayed Actions
ALTER TABLE public.automation_delayed_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.automation_delayed_actions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.automation_delayed_actions USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Exchange Rates
ALTER TABLE public.exchange_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.exchange_rates FORCE ROW LEVEL SECURITY;
CREATE POLICY exchange_rates_tenant_isolation ON public.exchange_rates USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Sync Jobs
ALTER TABLE public.sync_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sync_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.sync_jobs USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Audit Log
ALTER TABLE public.audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.audit_log USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Webhook Events
ALTER TABLE public.webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.webhook_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.webhook_events USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Webhook Deliveries
ALTER TABLE public.webhook_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.webhook_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY webhook_deliveries_tenant_isolation ON public.webhook_deliveries USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Invitations
ALTER TABLE public.invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY invitations_tenant_isolation ON public.invitations USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));
CREATE POLICY invitations_tenant_write ON public.invitations FOR INSERT WITH CHECK ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Purchase Orders
ALTER TABLE public.purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.purchase_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.purchase_orders USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Purchase Order Items
ALTER TABLE public.purchase_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.purchase_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.purchase_order_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Payment Settlements
ALTER TABLE public.payment_settlements ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payment_settlements FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.payment_settlements USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Payment Transactions
ALTER TABLE public.payment_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.payment_transactions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.payment_transactions USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Pick & Pack Sessions
ALTER TABLE public.pick_pack_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.pick_pack_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.pick_pack_sessions USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Pick & Pack Items
ALTER TABLE public.pick_pack_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.pick_pack_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.pick_pack_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Dropship Orders
ALTER TABLE public.dropship_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.dropship_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.dropship_orders USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Dropship Order Items
ALTER TABLE public.dropship_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.dropship_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.dropship_order_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Supplier Portal Tokens
ALTER TABLE public.supplier_portal_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_portal_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_portal_tokens USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Supplier Messages
ALTER TABLE public.supplier_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_messages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_messages USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Recurring Orders
ALTER TABLE public.recurring_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.recurring_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.recurring_orders USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Recurring Order Items
ALTER TABLE public.recurring_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.recurring_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.recurring_order_items USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Repricing Rules
ALTER TABLE public.repricing_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.repricing_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.repricing_rules USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Repricing Log
ALTER TABLE public.repricing_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.repricing_log FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.repricing_log USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Customer Segments
ALTER TABLE public.customer_segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.customer_segments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.customer_segments USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Customer Segment Members
ALTER TABLE public.customer_segment_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.customer_segment_members FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.customer_segment_members USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Loyalty Programs
ALTER TABLE public.loyalty_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.loyalty_programs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.loyalty_programs USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Customer Loyalty
ALTER TABLE public.customer_loyalty ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.customer_loyalty FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.customer_loyalty USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Listing Sync Configs
ALTER TABLE public.listing_sync_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.listing_sync_configs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.listing_sync_configs USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Listing Sync Log
ALTER TABLE public.listing_sync_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.listing_sync_log FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.listing_sync_log USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Stock Sync Channels
ALTER TABLE public.stock_sync_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stock_sync_channels FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.stock_sync_channels USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- Stock Sync Events
ALTER TABLE public.stock_sync_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stock_sync_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.stock_sync_events USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid));

-- ============================================================
-- Grants (for openoms application role)
-- ============================================================

GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO openoms;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO openoms;
