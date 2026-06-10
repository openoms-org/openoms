"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { enumLabel, type EnumTranslator } from "@/lib/i18n-fallback";
import type {
  AggregateStatus,
  FulfillmentStatus,
  HealthStatus,
  OperatorBucket,
} from "@/types/fulfillment";

type BadgeVariant =
  | "default"
  | "secondary"
  | "destructive"
  | "outline"
  | "success"
  | "warning"
  | "info";

const AGGREGATE_VARIANT: Record<AggregateStatus, BadgeVariant> = {
  new: "secondary",
  validating: "info",
  ready: "info",
  in_progress: "info",
  waiting_external: "warning",
  blocked: "destructive",
  completed: "success",
  cancelled: "outline",
};

const HEALTH_VARIANT: Record<HealthStatus, BadgeVariant> = {
  ok: "success",
  warning: "warning",
  action_required: "warning",
  system_error: "destructive",
};

const STEP_VARIANT: Record<FulfillmentStatus, BadgeVariant> = {
  pending: "secondary",
  ready: "info",
  running: "info",
  waiting_external: "warning",
  blocked: "destructive",
  succeeded: "success",
  failed: "destructive",
  cancelled: "outline",
  skipped: "outline",
};

const BUCKET_VARIANT: Record<OperatorBucket, BadgeVariant> = {
  ready: "info",
  processing: "info",
  stuck: "destructive",
  blocked: "destructive",
  provider_issue: "warning",
  missing_data: "warning",
};

export function AggregateStatusBadge({ status }: { status: AggregateStatus }) {
  const t = useTranslations("dashboard");
  return (
    <Badge variant={AGGREGATE_VARIANT[status] ?? "secondary"}>
      {t(`fulfillment.aggregateStatus.${status}`)}
    </Badge>
  );
}

export function HealthStatusBadge({ status }: { status: HealthStatus }) {
  const t = useTranslations("dashboard");
  // ok health is the quiet default — render nothing to reduce noise.
  if (status === "ok") {
    return null;
  }
  return (
    <Badge variant={HEALTH_VARIANT[status] ?? "secondary"}>
      {t(`fulfillment.healthStatus.${status}`)}
    </Badge>
  );
}

export function StepStatusBadge({ status }: { status: FulfillmentStatus }) {
  const t = useTranslations("dashboard");
  return (
    <Badge variant={STEP_VARIANT[status] ?? "secondary"}>
      {t(`fulfillment.stepStatus.${status}`)}
    </Badge>
  );
}

export function BucketBadge({ bucket }: { bucket: OperatorBucket }) {
  const t = useTranslations("dashboard");
  return (
    <Badge variant={BUCKET_VARIANT[bucket] ?? "secondary"}>
      {t(`fulfillment.buckets.${bucket}.label`)}
    </Badge>
  );
}

/**
 * Human-readable reason for a blocker code. Unknown codes (backend ahead of
 * the catalogs) degrade to a humanized form instead of the raw dotted key.
 */
export function blockerReasonLabel(t: EnumTranslator, code: string): string {
  return enumLabel(t, "fulfillment.blockerCode", code);
}

/** Human-readable label for a provider-attempt operation, with the same fallback. */
export function providerOpLabel(t: EnumTranslator, operation: string): string {
  return enumLabel(t, "fulfillment.providerOp", operation);
}
