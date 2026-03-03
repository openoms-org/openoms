# BaseLinker CSV Import — Design

## Goal

Enable one-time data migration from BaseLinker: orders (with item grouping), products (with variants and images), customers (extracted from orders). Upload CSV files exported from BL panel.

## Architecture

Extend existing CSV import infrastructure with BL-specific parsers. No BaseLinker API dependency — purely CSV-based.

## Key Challenges

1. **Orders: row = line item, not order.** BL exports one row per order item. Need parser that groups by `order_id` → single order with `items[]`, addresses, customer data from first row.

2. **Product variants.** BL exports variants as rows with `variant_id` and `product_id` (parent). Parse parent/child, create `product` + `product_variants`.

3. **Images.** Import stores URLs immediately. Separate background job "Re-download images" fetches from URLs and uploads to S3 (Supabase Storage).

## Components

### Backend

1. **BL Order Parser** (`service/baselinker_import_service.go`) — new parser:
   - Groups rows by `order_id`
   - Aggregates items (name, sku, qty, price per row)
   - Extracts shipping/billing address from `delivery_*` / `invoice_*` columns
   - Extracts unique customers (dedup by email) → optional auto-import to `customers`
   - Source: `"baselinker"`

2. **BL Product Parser** (extension of `product_import_service.go`) — aliases + variants:
   - BL column aliases → OpenOMS fields
   - Variant grouping by `product_id` (parent)
   - Image URLs → `images` JSONB array
   - Category mapping → `product_categories` (create if not exists)

3. **Customer Import** (new) — `POST /v1/customers/import/preview` + `/import`:
   - Upsert by email (or NIP for B2B)
   - Follows existing order/product import pattern (preview → mapping → import)

4. **Image Re-download Job** (new) — `POST /v1/products/redownload-images`:
   - Background job: downloads images from external URLs
   - Uploads to S3 (Supabase Storage)
   - Updates `product.images` with new URLs
   - Progress tracking via polling

### Frontend

- Import button on `/customers` page (new)
- "Download images" button on `/products` page (new) — triggers re-download job
- No dedicated migration wizard — reuse existing preview → mapping → import flow

## Data Mapping

### BL Order CSV → OpenOMS Order

| BL Column | OpenOMS Field |
|-----------|---------------|
| `order_id` | `external_id` (grouping key) |
| `date_add` | `ordered_at` |
| `order_status_name` | `status` |
| `buyer_name` / `delivery_fullname` | `customer_name` |
| `buyer_email` | `customer_email` |
| `buyer_phone` | `customer_phone` |
| `payment_sum` / `order_amount` | `total_amount` |
| `currency` | `currency` |
| `payment_method` | `payment_method` |
| `payment_done` | `payment_status` (paid if > 0) |
| `delivery_*` columns | `shipping_address` JSONB |
| `invoice_*` columns | `billing_address` JSONB |
| `product_name` | `items[].name` (per row) |
| `product_sku` | `items[].sku` (per row) |
| `product_quantity` | `items[].quantity` (per row) |
| `product_price_brutto` | `items[].price` (per row) |

### BL Product CSV → OpenOMS Product

| BL Column | OpenOMS Field |
|-----------|---------------|
| `product_id` | `external_id` |
| `name` | `name` |
| `sku` | `sku` |
| `ean` | `ean` |
| `price_brutto` | `price` |
| `quantity` / `stock` | `stock_quantity` |
| `description` | `description_short` |
| `description_extra` | `description_long` |
| `category` | `category` → `product_categories` |
| `image_url` / `images` | `images` JSONB (URL array) |
| `weight` | `weight` |
| `variant_id` | → `product_variants` child |
| `variant_name` | → variant attributes |

### BL Customer (extracted from orders)

| BL Column | OpenOMS Field |
|-----------|---------------|
| `buyer_name` / `delivery_fullname` | `name` |
| `buyer_email` | `email` (dedup key) |
| `buyer_phone` | `phone` |
| `invoice_company` | `company_name` |
| `invoice_nip` | `nip` |
| `delivery_*` | `default_shipping_address` |
| `invoice_*` | `default_billing_address` |

## Out of Scope

- BaseLinker API/SDK (no live connector)
- Dedicated migration wizard (reuse existing import UI)
- Invoice import (client generates new invoices in OpenOMS)
- Continuous sync

## Testing

- Unit: BL order parser (grouping, items aggregation, address extraction)
- Unit: BL product parser (variants, image URLs, category creation)
- Unit: customer dedup by email
- Integration: import with sample BL CSV files
- E2E: customer import page
