import type { Address, PaginationParams } from "./common";
import type { Product, ProductVariant } from "./products";

// === Orders ===
export interface OrderItem {
  product_id?: string;
  name: string;
  sku?: string;
  quantity: number;
  price: number;
  weight?: number;
}

interface ReturnItem {
  name: string;
  quantity: number;
}

export interface Order {
  id: string;
  tenant_id: string;
  external_id?: string;
  source: string;
  integration_id?: string;
  status: string;
  customer_name: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount: number;
  currency: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
  metadata?: Record<string, unknown>;
  tags: string[];
  ordered_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  delivery_method?: string;
  pickup_point_id?: string;
  payment_status: string;
  payment_method?: string;
  paid_at?: string;
  customer_id?: string;
  merged_into?: string;
  split_from?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateOrderRequest {
  external_id?: string;
  source: string;
  integration_id?: string;
  customer_name: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount: number;
  currency?: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
  metadata?: Record<string, unknown>;
  tags?: string[];
  delivery_method?: string;
  pickup_point_id?: string;
  ordered_at?: string;
  payment_status?: string;
  payment_method?: string;
  shipment_provider?: string;
  auto_create_shipment?: boolean;
}

export interface UpdateOrderRequest {
  external_id?: string;
  customer_name?: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount?: number;
  currency?: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
  metadata?: Record<string, unknown>;
  tags?: string[];
  delivery_method?: string;
  pickup_point_id?: string;
  payment_status?: string;
  payment_method?: string;
  paid_at?: string;
}

export interface StatusTransitionRequest {
  status: string;
  force?: boolean;
}

export interface OrderListParams extends PaginationParams {
  status?: string;
  source?: string;
  search?: string;
  payment_status?: string;
  tag?: string;
  priority?: string;
}

// === Returns/RMA ===
export interface Return {
  id: string;
  tenant_id: string;
  order_id: string;
  status: string;
  reason: string;
  items: ReturnItem[];
  refund_amount: number;
  notes?: string;
  return_token?: string;
  customer_email?: string;
  customer_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateReturnRequest {
  order_id: string;
  reason: string;
  items?: ReturnItem[];
  refund_amount: number;
  notes?: string;
}

export interface UpdateReturnRequest {
  reason?: string;
  items?: ReturnItem[];
  refund_amount?: number;
  notes?: string;
}

export interface ReturnStatusRequest {
  status: string;
}

export interface ReturnListParams extends PaginationParams {
  status?: string;
  order_id?: string;
}

// === Bulk Status ===
interface BulkStatusResult {
  order_id: string;
  success: boolean;
  error?: string;
}

export interface BulkStatusTransitionResponse {
  results: BulkStatusResult[];
  succeeded: number;
  failed: number;
}

// === Order Groups (Merge/Split) ===
export interface OrderGroup {
  id: string;
  tenant_id: string;
  group_type: "merged" | "split";
  source_order_ids: string[];
  target_order_ids: string[];
  notes?: string;
  created_by?: string;
  created_at: string;
}

export interface MergeOrdersRequest {
  order_ids: string[];
  notes?: string;
}

interface SplitSpec {
  items: OrderItem[];
  customer_name?: string;
  shipping_address?: Address;
}

export interface SplitOrderRequest {
  splits: SplitSpec[];
  notes?: string;
}

// === Barcode / Packing Station ===
export interface BarcodeLookupResponse {
  product?: Product;
  variants?: ProductVariant[];
}

export interface ScannedItem {
  sku: string;
  quantity: number;
}

export interface PackOrderRequest {
  scanned_items: ScannedItem[];
}

export interface PackOrderResponse {
  order_id: string;
  packed_at: string;
  packed_by: string;
  status: string;
}

// === Public Return Self-Service ===
export interface PublicReturnRequest {
  order_id: string;
  email: string;
  items?: ReturnItem[];
  reason: string;
  notes?: string;
}

export interface PublicReturnResponse {
  id: string;
  status: string;
  return_token: string;
  created_at: string;
}

export interface PublicReturnStatus {
  id: string;
  status: string;
  reason: string;
  items: ReturnItem[];
  created_at: string;
  updated_at: string;
}
