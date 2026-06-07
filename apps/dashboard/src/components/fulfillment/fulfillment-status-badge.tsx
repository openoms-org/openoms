"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
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

/** Resolve the i18n key for a blocker code's human-readable reason. */
export function blockerReasonKey(code: string): string {
  return `fulfillment.blockerCode.${code}`;
}
