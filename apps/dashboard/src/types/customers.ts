import type { PaginationParams } from "./common";
import type { ImportColumnMapping, ImportError, ImportPreviewRow } from "./products";

// === Customer Import ===
export interface CustomerImportPreview {
  headers: string[];
  total_rows: number;
  sample_rows: ImportPreviewRow[];
  new_count: number;
  update_count: number;
  mappings?: ImportColumnMapping[];
}

export interface CustomerImportResult {
  created: number;
  updated: number;
  skipped: number;
  errors: ImportError[];
}

// === Customers ===
export interface Customer {
  id: string;
  tenant_id: string;
  email?: string;
  phone?: string;
  name: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags: string[];
  notes?: string;
  total_orders: number;
  total_spent: number;
  price_list_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerRequest {
  name: string;
  email?: string;
  phone?: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags?: string[];
  notes?: string;
}

export interface UpdateCustomerRequest {
  name?: string;
  email?: string;
  phone?: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags?: string[];
  notes?: string;
  price_list_id?: string;
}

export interface CustomerListParams extends PaginationParams {
  search?: string;
  tags?: string;
}

// === Customer Segments ===
export interface CustomerSegment {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  color: string;
  segment_type: "manual" | "rfm_auto" | "rule_based";
  rules?: Record<string, unknown>;
  customer_count: number;
  created_at: string;
  updated_at: string;
}

export interface SegmentMember {
  tenant_id: string;
  segment_id: string;
  customer_id: string;
  added_at: string;
  customer_name: string;
  customer_email?: string;
  total_orders: number;
  total_spent: number;
}

export interface CustomerRFM {
  customer_id: string;
  customer_name: string;
  customer_email?: string;
  days_since_last_order: number;
  order_count: number;
  total_spent: number;
  rfm: { recency: number; frequency: number; monetary: number };
  segment_label: string;
}

export interface CreateSegmentRequest {
  name: string;
  description?: string;
  color: string;
  segment_type: "manual" | "rfm_auto" | "rule_based";
  rules?: Record<string, unknown>;
}

export interface UpdateSegmentRequest {
  name?: string;
  description?: string;
  color?: string;
  rules?: Record<string, unknown>;
}

export interface AddSegmentMemberRequest {
  customer_id: string;
}

export type SegmentListParams = PaginationParams;

// === Loyalty Programs ===
export interface LoyaltyProgram {
  id: string;
  tenant_id: string;
  name: string;
  status: "active" | "paused" | "ended";
  program_type: "points" | "tier" | "discount_after_n";
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CustomerLoyalty {
  tenant_id: string;
  customer_id: string;
  program_id: string;
  points_balance: number;
  total_points_earned: number;
  total_points_redeemed: number;
  current_tier?: string;
  total_spent: number;
  order_count: number;
  last_activity_at?: string;
  created_at: string;
  updated_at: string;
  program_name: string;
  program_type: string;
}

export interface LeaderboardEntry {
  customer_id: string;
  customer_name: string;
  points: number;
  total_spent: number;
  order_count: number;
  current_tier?: string;
}

export interface CreateLoyaltyProgramRequest {
  name: string;
  program_type: "points" | "tier" | "discount_after_n";
  config: Record<string, unknown>;
}

export interface UpdateLoyaltyProgramRequest {
  name?: string;
  status?: "active" | "paused" | "ended";
  config?: Record<string, unknown>;
}

export interface AwardPointsRequest {
  customer_id: string;
  points: number;
  reason: string;
}

export interface RedeemPointsRequest {
  customer_id: string;
  points: number;
}

export type LoyaltyProgramListParams = PaginationParams;
