import type { PaginationParams } from "./common";

// === AI Text And Image Helpers ===
export interface AITextResult {
  description: string;
}

export interface AIDescribeRequest {
  product_id: string;
  style?: "professional" | "promotional" | "casual" | "seo";
  language?: "pl" | "en" | "de";
  length?: "short" | "medium" | "long";
  marketplace?: "allegro" | "amazon" | "ebay";
  format?: "text" | "html";
}

export interface BGRemovalResult {
  url: string;
  content_type: string;
  message?: string;
}

export interface BGRemovalStatus {
  configured: boolean;
}

// === Dashboard Stats ===
export interface DashboardStats {
  order_counts: OrderCounts;
  revenue: Revenue;
  recent_orders: OrderSummary[];
}

export interface OrderCounts {
  total: number;
  by_status: Record<string, number>;
  by_source: Record<string, number>;
}

export interface Revenue {
  total: number;
  currency: string;
  daily: DailyRevenue[];
}

export interface DailyRevenue {
  date: string;
  amount: number;
  count: number;
}

export interface OrderSummary {
  id: string;
  customer_name: string;
  status: string;
  source: string;
  total_amount: number;
  currency: string;
  created_at: string;
}

// === Advanced Reports ===
export interface TopProduct {
  name: string;
  sku?: string;
  total_quantity: number;
  total_revenue: number;
}

export interface SourceRevenue {
  source: string;
  revenue: number;
  count: number;
}

export interface DailyOrderTrend {
  date: string;
  count: number;
  avg_value: number;
}

// === Carbon Footprint ===
export interface CarbonStats {
  total_shipments: number;
  total_carbon_kg: number;
  avg_carbon_per_shipment: number;
  total_distance_km: number;
  by_carrier: CarrierCarbonStats[];
  monthly_trend: MonthlyCarbonStat[];
}

export interface CarrierCarbonStats {
  carrier: string;
  shipments: number;
  total_carbon_kg: number;
  avg_carbon_kg: number;
}

export interface MonthlyCarbonStat {
  month: string;
  shipments: number;
  total_carbon_kg: number;
}

// === Audit Log ===
export interface AuditLogEntry {
  id: number;
  user_name?: string;
  action: string;
  entity_type: string;
  entity_id: string;
  changes: Record<string, string>;
  ip_address?: string;
  created_at: string;
}

export interface AuditListParams extends PaginationParams {
  entity_type?: string;
  action?: string;
  user_id?: string;
}

// === Email Settings ===
export interface EmailSettings {
  enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_user: string;
  smtp_pass: string;
  from_email: string;
  from_name: string;
  notify_on: string[];
}

// === Company Settings ===
export interface CompanySettings {
  company_name: string;
  logo_url: string;
  address: string;
  city: string;
  post_code: string;
  nip: string;
  phone: string;
  email: string;
  website: string;
}

// === Order Status Config ===
export interface StatusDef {
  key: string;
  label: string;
  color: string; // preset name: "blue", "green", "red", etc.
  position: number;
}

export interface OrderStatusConfig {
  statuses: StatusDef[];
  transitions: Record<string, string[]>;
}

// === Custom Fields Config ===
export interface CustomFieldDef {
  key: string;
  label: string;
  type: "text" | "number" | "select" | "date" | "checkbox";
  required: boolean;
  position: number;
  options?: string[];
}

export interface CustomFieldsConfig {
  fields: CustomFieldDef[];
}

// === SMS Settings ===
export interface SMSSettings {
  enabled: boolean;
  api_token: string;
  from: string;
  notify_on: string[];
  templates: Record<string, string>;
}

// === Print Templates ===
export interface PrintTemplatesConfig {
  packing_slip_html: string;
  order_summary_html: string;
  return_slip_html: string;
}

// === AI Auto-Categorization (Phase 33) ===
export interface AISuggestion {
  product_id: string;
  categories: string[];
  tags: string[];
  description?: string;
  short_description?: string;
  long_description?: string;
}

interface AIBulkCategorizeResult {
  product_id: string;
  categories: string[];
  tags: string[];
  error?: string;
}

export interface AIBulkCategorizeResponse {
  results: AIBulkCategorizeResult[];
}

// === Marketing / Mailchimp (Phase 34) ===
export interface MailchimpSettings {
  api_key: string;
  list_id: string;
  enabled: boolean;
}

export interface MarketingSyncResponse {
  synced: number;
  failed: number;
}

export interface MarketingStatusResponse {
  enabled: boolean;
  configured: boolean;
}

export interface CreateCampaignRequest {
  name: string;
  subject: string;
  content: string;
}

export interface CreateCampaignResponse {
  campaign_id: string;
}

// === Helpdesk / Freshdesk (Phase 34) ===
export interface FreshdeskSettings {
  domain: string;
  api_key: string;
  enabled: boolean;
}

export interface FreshdeskTicket {
  id: number;
  subject: string;
  description?: string;
  status: number;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface CreateTicketRequest {
  subject: string;
  description: string;
  email: string;
}

export interface TicketListResponse {
  tickets: FreshdeskTicket[];
}

// === Onboarding ===
export interface OnboardingStatus {
  completed: boolean;
  current_step: number;
  completed_steps: number[];
  skipped_steps: number[];
  completed_at: string | null;
  dismissed: boolean;
}

export interface UpdateOnboardingStepRequest {
  action: "completed" | "skipped";
}

interface CompanySettingsRequest {
  company_name: string;
  nip: string;
  address: string;
  city: string;
  post_code: string;
  phone: string;
  email: string;
}
