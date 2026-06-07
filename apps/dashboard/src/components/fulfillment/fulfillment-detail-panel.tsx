"use client";

import { useState } from "react";
import { toast } from "sonner";
import { formatDistanceToNow, type Locale } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { CheckCircle2, PackageX, RefreshCw, ShieldCheck } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { ApiClientError } from "@/lib/api-client";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  AggregateStatusBadge,
  HealthStatusBadge,
  StepStatusBadge,
  blockerReasonKey,
} from "@/components/fulfillment/fulfillment-status-badge";
import {
  useFulfillmentProcess,
  useOrderFulfillment,
  useResolveBlocker,
  useRetryStep,
} from "@/hooks/use-fulfillment";
import type {
  FulfillmentBlocker,
  FulfillmentProcessDetail,
  FulfillmentStep,
} from "@/types/fulfillment";

interface FulfillmentDetailPanelProps {
  /** Look up by process id (control tower drilldown). */
  processId?: string;
  /** Look up by order id (order detail drilldown). Ignored when processId set. */
  orderId?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const RETRYABLE_STEP_STATUSES = new Set(["failed", "blocked"]);

export function FulfillmentDetailPanel({
  processId,
  orderId,
  open,
  onOpenChange,
}: FulfillmentDetailPanelProps) {
  const t = useTranslations("dashboard");
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  const byProcess = useFulfillmentProcess(processId, {
    enabled: open && !!processId,
  });
  const byOrder = useOrderFulfillment(orderId, {
    enabled: open && !processId && !!orderId,
  });

  const query = processId ? byProcess : byOrder;
  const detail = query.data;

  const resolveBlocker = useResolveBlocker();
  const retryStep = useRetryStep();

  const [pendingBlocker, setPendingBlocker] = useState<FulfillmentBlocker | null>(
    null,
  );
  const [pendingStep, setPendingStep] = useState<FulfillmentStep | null>(null);

  const handleResolve = () => {
    if (!pendingBlocker) return;
    resolveBlocker.mutate(pendingBlocker.id, {
      onSuccess: () => {
        toast.success(t("fulfillment.actions.blockerResolved"));
        setPendingBlocker(null);
      },
      onError: (error) => {
        toast.error(actionErrorMessage(error, t, "fulfillment.actions.resolveError"));
      },
    });
  };

  const handleRetry = () => {
    if (!pendingStep) return;
    retryStep.mutate(pendingStep.id, {
      onSuccess: () => {
        toast.success(t("fulfillment.actions.stepRetried"));
        setPendingStep(null);
      },
      onError: (error) => {
        toast.error(actionErrorMessage(error, t, "fulfillment.actions.retryError"));
        setPendingStep(null);
      },
    });
  };

  const isNotFound =
    query.error instanceof ApiClientError && query.error.status === 404;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full gap-0 overflow-y-auto sm:max-w-xl"
        data-testid="fulfillment-detail-panel"
      >
        <SheetHeader className="border-b">
          <SheetTitle>{t("fulfillment.detail.title")}</SheetTitle>
          <SheetDescription>{t("fulfillment.detail.subtitle")}</SheetDescription>
        </SheetHeader>

        <div className="space-y-6 p-4">
          {query.isLoading ? (
            <DetailSkeleton />
          ) : isNotFound ? (
            <EmptyState
              icon={PackageX}
              title={t("fulfillment.detail.notFoundTitle")}
              description={t("fulfillment.detail.notFoundDescription")}
            />
          ) : query.isError ? (
            <div className="flex flex-col items-center gap-2 py-10 text-center">
              <p className="text-sm text-destructive">
                {t("fulfillment.detail.loadError")}
              </p>
              <Button variant="outline" size="sm" onClick={() => query.refetch()}>
                {t("fulfillment.retry")}
              </Button>
            </div>
          ) : detail ? (
            <DetailBody
              detail={detail}
              dateLocale={dateLocale}
              onResolveBlocker={setPendingBlocker}
              onRetryStep={setPendingStep}
              t={t}
            />
          ) : null}
        </div>
      </SheetContent>

      <ConfirmDialog
        open={!!pendingBlocker}
        onOpenChange={(o) => !o && setPendingBlocker(null)}
        title={t("fulfillment.actions.resolveConfirmTitle")}
        description={t("fulfillment.actions.resolveConfirmDescription")}
        confirmLabel={t("fulfillment.actions.resolve")}
        onConfirm={handleResolve}
        isPending={resolveBlocker.isPending}
      />

      <ConfirmDialog
        open={!!pendingStep}
        onOpenChange={(o) => !o && setPendingStep(null)}
        title={t("fulfillment.actions.retryConfirmTitle")}
        description={t("fulfillment.actions.retryConfirmDescription")}
        confirmLabel={t("fulfillment.actions.retry")}
        onConfirm={handleRetry}
        isPending={retryStep.isPending}
      />
    </Sheet>
  );
}

interface DetailBodyProps {
  detail: FulfillmentProcessDetail;
  dateLocale: Locale;
  onResolveBlocker: (blocker: FulfillmentBlocker) => void;
  onRetryStep: (step: FulfillmentStep) => void;
  t: ReturnType<typeof useTranslations>;
}

function DetailBody({
  detail,
  dateLocale,
  onResolveBlocker,
  onRetryStep,
  t,
}: DetailBodyProps) {
  const openBlockers = detail.blockers.filter((b) => b.status !== "resolved");

  return (
    <>
      {/* Process header */}
      <section className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <AggregateStatusBadge status={detail.process.aggregate_status} />
          <HealthStatusBadge status={detail.process.health_status} />
        </div>
        <p className="text-xs text-muted-foreground">
          {t("fulfillment.detail.orderRef", { orderId: detail.process.order_id })}
        </p>
        <p className="text-xs text-muted-foreground">
          {t("fulfillment.detail.updatedAt", {
            time: formatDistanceToNow(new Date(detail.process.updated_at), {
              addSuffix: true,
              locale: dateLocale,
            }),
          })}
        </p>
      </section>

      {/* Blockers */}
      <section className="space-y-2">
        <h3 className="text-sm font-semibold">
          {t("fulfillment.detail.blockersHeading")}
        </h3>
        {openBlockers.length === 0 ? (
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4 text-success" aria-hidden="true" />
            {t("fulfillment.detail.noBlockers")}
          </p>
        ) : (
          <ul className="space-y-2">
            {openBlockers.map((blocker) => (
              <li
                key={blocker.id}
                className="rounded-md border border-destructive/30 bg-destructive/5 p-3"
                data-testid="fulfillment-blocker"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">
                      {t(blockerReasonKey(blocker.code))}
                    </p>
                    {blocker.description && (
                      <p className="mt-1 text-sm text-muted-foreground">
                        {blocker.description}
                      </p>
                    )}
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    className="shrink-0"
                    onClick={() => onResolveBlocker(blocker)}
                    data-testid="resolve-blocker-button"
                  >
                    <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                    {t("fulfillment.actions.resolve")}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Units + steps timeline */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold">
          {t("fulfillment.detail.unitsHeading")}
        </h3>
        {detail.units.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("fulfillment.detail.noUnits")}
          </p>
        ) : (
          detail.units.map((unit) => (
            <div
              key={unit.unit.id}
              className="rounded-md border bg-background p-3"
              data-testid="fulfillment-unit"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-medium">
                  {t(`fulfillment.unitType.${unit.unit.unit_type}`)}
                </span>
                <StepStatusBadge status={unit.unit.status} />
              </div>
              <ol className="mt-3 space-y-2 border-l pl-4">
                {unit.steps.map((step) => {
                  const retryable = RETRYABLE_STEP_STATUSES.has(step.status);
                  return (
                    <li
                      key={step.id}
                      className="flex items-center justify-between gap-3"
                      data-testid="fulfillment-step"
                    >
                      <span className="min-w-0">
                        <span className="block text-sm">
                          {t(`fulfillment.stepKey.${step.step_key}`)}
                        </span>
                        {step.attempts > 0 && (
                          <span className="block text-xs text-muted-foreground">
                            {t("fulfillment.detail.attempts", {
                              count: step.attempts,
                            })}
                          </span>
                        )}
                      </span>
                      <span className="flex shrink-0 items-center gap-2">
                        <StepStatusBadge status={step.status} />
                        {retryable && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onRetryStep(step)}
                            data-testid="retry-step-button"
                          >
                            <RefreshCw className="h-4 w-4" aria-hidden="true" />
                            {t("fulfillment.actions.retry")}
                          </Button>
                        )}
                      </span>
                    </li>
                  );
                })}
                {unit.steps.length === 0 && (
                  <li className="text-xs text-muted-foreground">
                    {t("fulfillment.detail.noSteps")}
                  </li>
                )}
              </ol>
            </div>
          ))
        )}
      </section>

      {/* Provider attempts */}
      <section className="space-y-2">
        <h3 className="text-sm font-semibold">
          {t("fulfillment.detail.attemptsHeading")}
        </h3>
        {detail.provider_attempts.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("fulfillment.detail.noAttempts")}
          </p>
        ) : (
          <ul className="space-y-2">
            {detail.provider_attempts.map((attempt) => (
              <li
                key={attempt.id}
                className="flex items-center justify-between gap-3 rounded-md border bg-background p-3"
                data-testid="provider-attempt"
              >
                <span className="min-w-0">
                  <span className="block text-sm font-medium">
                    {t(`fulfillment.providerOp.${attempt.operation}`)}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {attempt.provider}
                    {attempt.error_code ? ` · ${attempt.error_code}` : ""}
                  </span>
                </span>
                <ProviderAttemptBadge status={attempt.status} label={t} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

function ProviderAttemptBadge({
  status,
  label,
}: {
  status: string;
  label: ReturnType<typeof useTranslations>;
}) {
  const variant =
    status === "succeeded"
      ? "success"
      : status === "failed"
        ? "destructive"
        : "secondary";
  return (
    <Badge variant={variant}>{label(`fulfillment.providerStatus.${status}`)}</Badge>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-6 w-40" />
      <Skeleton className="h-20 w-full" />
      <Skeleton className="h-32 w-full" />
      <Skeleton className="h-24 w-full" />
    </div>
  );
}

function EmptyState({
  icon: Icon,
  title,
  description,
}: {
  icon: typeof PackageX;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-12 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-lg bg-muted">
        <Icon className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
      </span>
      <p className="font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

function actionErrorMessage(
  error: unknown,
  t: ReturnType<typeof useTranslations>,
  fallbackKey: string,
): string {
  // Backend returns a 400 with a human message for non-retryable steps; surface
  // it. Everything else falls back to the localized generic action error.
  if (error instanceof ApiClientError && error.status === 400 && error.message) {
    return error.message;
  }
  return t(fallbackKey);
}
