import type { PaginationParams } from "./common";

// === Integrations (NOT paginated, admin only) ===
export interface Integration {
  id: string;
  tenant_id: string;
  provider: string;
  label?: string;
  status: "active" | "inactive" | "error";
  has_credentials: boolean;
  settings?: Record<string, unknown>;
  sync_cursor?: string;
  error_message?: string;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateIntegrationRequest {
  provider: string;
  label?: string;
  credentials: Record<string, unknown>;
  settings?: Record<string, unknown>;
}

export interface UpdateIntegrationRequest {
  label?: string;
  status?: "active" | "inactive" | "error";
  credentials?: Record<string, unknown>;
  settings?: Record<string, unknown>;
  sync_cursor?: string;
  error_message?: string;
}

// === Outgoing Webhooks ===
export interface WebhookEndpoint {
  id: string;
  name: string;
  url: string;
  secret: string;
  events: string[];
  active: boolean;
}

export interface WebhookConfig {
  endpoints: WebhookEndpoint[];
}

export interface WebhookDelivery {
  id: string;
  tenant_id: string;
  url: string;
  event_type: string;
  payload: Record<string, unknown>;
  status: string;
  response_code?: number;
  error?: string;
  created_at: string;
}

export interface WebhookDeliveryParams extends PaginationParams {
  event_type?: string;
  status?: string;
}

// === Sync Jobs ===
export interface SyncJob {
  id: string;
  tenant_id: string;
  integration_id: string;
  job_type: string;
  status: string;
  started_at?: string;
  finished_at?: string;
  items_processed: number;
  items_failed: number;
  error_message?: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface SyncJobListParams extends PaginationParams {
  integration_id?: string;
  job_type?: string;
  status?: string;
}
