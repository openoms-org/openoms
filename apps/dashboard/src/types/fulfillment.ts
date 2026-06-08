// === OPE-419 Fulfillment / Operations read API ===
//
// These types mirror the Go DTOs served by the read API merged in #540
// (internal/handler/fulfillment_handler.go, operations_handler.go,
// service/fulfillment_read_service.go, model/fulfillment.go,
// model/provider_attempt.go). Field names and casing are kept identical to the
// backend JSON tags — no invented fields.
//
// The endpoints are always registered but return EMPTY results while
// FULFILLMENT_PROCESS_ENABLED is off (nothing is being recorded), so every
// consumer must render an intentional empty/zero state by default.

import type { PaginationParams } from "./common";

// Process aggregate status (operator-visible progress).
export type AggregateStatus =
  | "new"
  | "validating"
  | "ready"
  | "in_progress"
  | "waiting_external"
  | "blocked"
  | "completed"
  | "cancelled";

// Process health status (independent from progress).
export type HealthStatus = "ok" | "warning" | "action_required" | "system_error";

// Fulfillment unit type.
export type UnitType =
  | "warehouse"
  | "dropship"
  | "backorder"
  | "mixed_child"
  | "manual";

// Shared unit/step lifecycle status.
export type FulfillmentStatus =
  | "pending"
  | "ready"
  | "running"
  | "waiting_external"
  | "blocked"
  | "succeeded"
  | "failed"
  | "cancelled"
  | "skipped";

// Blocker lifecycle status.
export type BlockerStatus = "open" | "acknowledged" | "resolved";

// Blocker category (drives the exception bucket refinement on the backend).
export type BlockerCategory =
  | "integration"
  | "supplier"
  | "operator"
  | "capability"
  | "mapping";

// Provider attempt operation + lifecycle status.
export type ProviderAttemptOperation =
  | "create_shipment"
  | "generate_label"
  | "download_label"
  | "sync_tracking";
export type ProviderAttemptStatus = "pending" | "succeeded" | "failed";

// Operator control-tower buckets. A process lands in exactly one bucket.
export type OperatorBucket =
  | "ready"
  | "processing"
  | "stuck"
  | "blocked"
  | "provider_issue"
  | "missing_data";

export const OPERATOR_BUCKETS: readonly OperatorBucket[] = [
  "ready",
  "processing",
  "stuck",
  "blocked",
  "provider_issue",
  "missing_data",
] as const;

// === Core models ===

export interface FulfillmentProcess {
  id: string;
  tenant_id: string;
  order_id: string;
  aggregate_status: AggregateStatus;
  health_status: HealthStatus;
  metadata?: unknown;
  created_at: string;
  updated_at: string;
}

export interface FulfillmentUnit {
  id: string;
  tenant_id: string;
  process_id: string;
  parent_unit_id?: string;
  unit_type: UnitType;
  status: FulfillmentStatus;
  metadata?: unknown;
  created_at: string;
  updated_at: string;
}

export interface FulfillmentStep {
  id: string;
  tenant_id: string;
  unit_id: string;
  step_key: string;
  status: FulfillmentStatus;
  attempts: number;
  metadata?: unknown;
  created_at: string;
  updated_at: string;
}

export interface FulfillmentBlocker {
  id: string;
  tenant_id: string;
  process_id: string;
  unit_id?: string;
  code: string;
  category: BlockerCategory | string;
  status: BlockerStatus;
  description: string;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
}

export interface ProviderAttempt {
  id: string;
  tenant_id: string;
  process_id?: string;
  provider: string;
  operation: ProviderAttemptOperation | string;
  status: ProviderAttemptStatus;
  request_id?: string;
  correlation_id?: string;
  raw_provider_status?: string;
  result_redacted?: unknown;
  payload_hash?: string;
  error_code?: string;
  created_at: string;
}

// === Assembled detail (GET process/{id} and orders/{orderID}) ===

export interface FulfillmentUnitDetail {
  unit: FulfillmentUnit;
  steps: FulfillmentStep[];
}

export interface FulfillmentProcessDetail {
  process: FulfillmentProcess;
  units: FulfillmentUnitDetail[];
  blockers: FulfillmentBlocker[];
  provider_attempts: ProviderAttempt[];
}

// === Operations control tower ===

export interface OperationsSummary {
  buckets: Record<OperatorBucket, number>;
  total: number;
}

export interface FulfillmentExceptionItem {
  process: FulfillmentProcess;
  bucket: OperatorBucket;
  top_blocker?: FulfillmentBlocker;
}

export interface FulfillmentExceptionsResponse {
  items: FulfillmentExceptionItem[];
  total: number;
}

export interface IntegrationCapabilitySummary {
  automation_coverage: number;
  manual_intervention_processes: number;
  unsupported_capability_processes: number;
  stale_data_processes: number;
  missing_mapping_processes: number;
  failed_provider_attempts: number;
  total_processes: number;
}

// === Parity / verification report (OPE-423b) ===
//
// Mirrors service.ParityReport (apps/api-server/internal/service/
// fulfillment_parity_service.go) served by GET /v1/operations/parity. It is a
// READ-ONLY decision aid for the rollout cutover: process_coverage_met is the
// only gate-relevant verdict (the two exception counts are advisory context,
// counted on different primitives so they are NOT expected to be equal). Field
// names/casing match the backend JSON tags exactly — no invented fields.
export interface FulfillmentParityReport {
  // Legacy active order population (the coverage denominator): orders whose
  // status is NOT terminal (completed/cancelled/refunded).
  non_terminal_orders: number;
  // Total process-backed population recorded for the tenant.
  fulfillment_processes: number;
  // Non-terminal orders with NO process yet — the coverage GAP, must trend to 0.
  orders_missing_process: number;
  // Share (0..1) of non-terminal orders backed by a process; 1.0 when there are
  // no non-terminal orders (vacuously covered).
  process_coverage: number;
  // Legacy "needs attention" order count (on_hold or problem-shipment), the
  // order-comparable subset of the dashboard heuristic. Advisory context.
  legacy_problem_orders: number;
  // Process-backed "needs attention" total (blocked + provider_issue + stuck
  // buckets). Advisory context — counted on processes, not orders.
  process_backed_exceptions: number;
  // The coverage threshold process_coverage_met was evaluated against.
  coverage_threshold: number;
  // The parity verdict: true when process_coverage >= coverage_threshold, i.e.
  // it is safe to cut the dashboard over to process-backed counts.
  process_coverage_met: boolean;
}

// === Query params ===

export interface FulfillmentProcessListParams extends PaginationParams {
  aggregate_status?: AggregateStatus | AggregateStatus[];
  health_status?: HealthStatus | HealthStatus[];
}
