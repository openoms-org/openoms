import type { PaginationParams } from "./common";

// === Invoices ===
export interface Invoice {
  id: string;
  tenant_id: string;
  order_id: string;
  provider: string;
  external_id?: string;
  external_number?: string;
  status: string;
  invoice_type: string;
  total_net?: number;
  total_gross?: number;
  currency: string;
  issue_date?: string;
  due_date?: string;
  pdf_url?: string;
  metadata: Record<string, unknown>;
  error_message?: string;
  ksef_number?: string;
  ksef_status: string;
  ksef_sent_at?: string;
  ksef_response?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateInvoiceRequest {
  order_id: string;
  provider: string;
  invoice_type?: string;
  customer_name: string;
  customer_email?: string;
  nip?: string;
  payment_method?: string;
  notes?: string;
}

export interface InvoiceListParams extends PaginationParams {
  status?: string;
  provider?: string;
  order_id?: string;
}

// === Invoicing Settings ===
export interface InvoicingSettings {
  provider: string;
  auto_create_on_status: string[];
  default_tax_rate: number;
  payment_days: number;
  credentials: Record<string, string>;
}

// === KSeF Settings ===
export interface KSeFSettings {
  enabled: boolean;
  environment: string;
  nip: string;
  token: string;
  auto_send: boolean;
  company_name: string;
  company_street: string;
  company_city: string;
  company_postal: string;
  company_country: string;
}

export interface KSeFTestResult {
  success: boolean;
  message: string;
  timestamp?: string;
  challenge?: string;
}

export interface KSeFBulkSendResult {
  sent: number;
  errors: string[];
  total: number;
}

// === Exchange Rates (Multi-Currency) ===
export interface ExchangeRate {
  id: string;
  tenant_id: string;
  base_currency: string;
  target_currency: string;
  rate: number;
  source: string;
  fetched_at: string;
  created_at: string;
}

export interface CreateExchangeRateRequest {
  base_currency: string;
  target_currency: string;
  rate: number;
  source?: string;
}

export interface UpdateExchangeRateRequest {
  rate?: number;
  source?: string;
}

export interface ConvertAmountRequest {
  amount: number;
  from: string;
  to: string;
}

export interface ConvertAmountResponse {
  original_amount: number;
  converted_amount: number;
  from: string;
  to: string;
  rate: number;
}

export interface ExchangeRateListParams extends PaginationParams {
  base_currency?: string;
  target_currency?: string;
}

export interface FetchNBPResponse {
  fetched: number;
  source: string;
}

// === Payment Reconciliation ===
export interface PaymentSettlement {
  id: string;
  tenant_id: string;
  provider: string;
  settlement_id?: string;
  settlement_date: string;
  total_amount: number;
  fee_amount: number;
  net_amount: number;
  currency: string;
  status: string;
  notes?: string;
  imported_at: string;
  created_at: string;
  updated_at: string;
}

export interface PaymentSettlementWithTransactions extends PaymentSettlement {
  transactions: PaymentTransaction[];
}

export interface PaymentTransaction {
  id: string;
  tenant_id: string;
  settlement_id?: string;
  order_id?: string;
  provider: string;
  external_transaction_id?: string;
  amount: number;
  fee: number;
  net_amount: number;
  currency: string;
  transaction_type: string;
  transaction_date: string;
  match_status: string;
  match_notes?: string;
  created_at: string;
}

export interface CreateSettlementRequest {
  provider: string;
  settlement_id?: string;
  settlement_date: string;
  total_amount: number;
  fee_amount: number;
  net_amount: number;
  currency?: string;
  notes?: string;
  transactions?: CreateTransactionRequest[];
}

export interface CreateTransactionRequest {
  external_transaction_id?: string;
  amount: number;
  fee: number;
  net_amount: number;
  currency?: string;
  transaction_type?: string;
  transaction_date: string;
}

export interface ManualMatchRequest {
  order_id: string;
  notes?: string;
}

export interface ReconciliationSummary {
  total_transactions: number;
  matched_count: number;
  unmatched_count: number;
  discrepancy_count: number;
  manual_match_count: number;
  total_amount: number;
  total_fees: number;
  total_net: number;
  matched_amount: number;
  unmatched_amount: number;
}

interface MatchResult {
  transaction_id: string;
  order_id?: string;
  status: string;
  notes?: string;
}

export interface AutoMatchResponse {
  settlement_id: string;
  results: MatchResult[];
  matched: number;
  unmatched: number;
  discrepancy: number;
}

export interface SettlementListParams extends PaginationParams {
  provider?: string;
  status?: string;
  date_from?: string;
  date_to?: string;
}

export interface TransactionListParams extends PaginationParams {
  settlement_id?: string;
  match_status?: string;
  provider?: string;
  transaction_type?: string;
  date_from?: string;
  date_to?: string;
}

// === VAT OSS ===
export interface VATRateSet {
  standard: number;
  reduced_1?: number;
  reduced_2?: number;
  super_reduced?: number;
}

export interface VATCalculation {
  net_amount: number;
  vat_amount: number;
  gross_amount: number;
  vat_rate: number;
  country: string;
  rate_type: string;
}

export interface OSSConfig {
  enabled: boolean;
  home_country: string;
  default_vat_rate: string;
}

export interface OSSReport {
  quarter: number;
  year: number;
  home_country: string;
  total_sales: number;
  total_vat: number;
  by_country: OSSCountryReport[];
  generated_at: string;
}

export interface OSSCountryReport {
  country: string;
  country_name: string;
  order_count: number;
  total_sales: number;
  vat_rate: number;
  vat_amount: number;
}

export interface ThresholdStatus {
  year: number;
  total_cross_border_eur: number;
  threshold_eur: number;
  exceeded: boolean;
  remaining_eur: number;
}

export interface VATCalculateRequest {
  amount: number;
  country: string;
  rate_type: string;
}
