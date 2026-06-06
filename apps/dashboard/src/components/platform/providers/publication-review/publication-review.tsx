"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Ban, Info } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ActionDialog } from "@/components/shared/action-dialog";
import { PublicationStateBadge } from "@/components/platform/publication-state-badge";
import { ApiClientError } from "@/lib/api-client";
import { formatDateTime } from "@/lib/utils";
import {
  useProviderCapabilities,
  useProviderGaps,
  useProviderSchema,
} from "@/hooks/use-platform-provider-config";
import {
  useEmergencyDisableProviderVersion,
  useProviderPublicationEvents,
  useProviderValidationRuns,
  usePublishProviderVersion,
} from "@/hooks/use-platform-provider-validation";
import { GateChecklist } from "./gate-checklist";
import {
  availableTransitions,
  blockingGaps,
  canEmergencyDisable,
  deriveGates,
  evaluatePublishAction,
  lastCompletedRun,
  warningGaps,
} from "./publication-helpers";
import { verdictTone } from "@/components/platform/providers/validation-lab/validation-helpers";
import type {
  ProviderPublicationState,
  ProviderVersion,
} from "@/types/platform";

interface PublicationReviewProps {
  providerId: string;
  version: ProviderVersion;
  versions: ProviderVersion[];
}

/**
 * Screen 9 — Publication Review. Renders the derived G1-G9 gate checklist,
 * blocking/warning gaps, last validation and rollback target, then offers the
 * legal lifecycle transitions. Exposure-increasing actions are disabled with an
 * explicit reason when gates/gaps block them; emergency disable requires a
 * reason. All transitions go through the shared ActionDialog.
 */
export function PublicationReview({
  providerId,
  version,
  versions,
}: PublicationReviewProps) {
  const t = useTranslations("platform");
  const versionId = version.id;

  const { data: schema } = useProviderSchema(providerId, versionId);
  const { data: capabilities = [] } = useProviderCapabilities(providerId, versionId);
  const { data: gaps = [] } = useProviderGaps(providerId, versionId);
  const { data: runs = [] } = useProviderValidationRuns(providerId, versionId);
  const { data: events = [] } = useProviderPublicationEvents(providerId, versionId);

  const publish = usePublishProviderVersion(providerId, versionId);
  const emergencyDisable = useEmergencyDisableProviderVersion(providerId, versionId);

  const state = version.publication_state;
  const gates = useMemo(
    () => deriveGates({ version, schema, capabilities, gaps, runs }),
    [version, schema, capabilities, gaps, runs],
  );
  const blocking = blockingGaps(gaps);
  const warnings = warningGaps(gaps);
  const lastRun = lastCompletedRun(runs);
  const transitions = availableTransitions(state);

  // Rollback target = the prior state recorded in publication history, if any.
  const rollbackTarget = useMemo<ProviderPublicationState | null>(() => {
    const transitionsHistory = [...events].sort((a, b) => b.id - a.id);
    const prior = transitionsHistory.find((e) => e.from_state);
    return (prior?.from_state as ProviderPublicationState | undefined) ?? null;
  }, [events]);

  const [target, setTarget] = useState<ProviderPublicationState | null>(null);
  const [reason, setReason] = useState("");
  const [disableOpen, setDisableOpen] = useState(false);
  const [disableReason, setDisableReason] = useState("");

  const runPublish = () => {
    if (!target) return;
    publish.mutate(
      { to_state: target, reason: reason.trim() },
      {
        onSuccess: () => {
          toast.success(t("publicationReview.published"));
          setTarget(null);
          setReason("");
        },
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  const runEmergencyDisable = () => {
    emergencyDisable.mutate(
      { reason: disableReason.trim() },
      {
        onSuccess: () => {
          toast.success(t("publicationReview.disabled"));
          setDisableOpen(false);
          setDisableReason("");
        },
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("publicationReview.title")}</CardTitle>
          <CardDescription>{t("publicationReview.description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <dl className="grid gap-2 sm:grid-cols-2">
            <div className="flex items-center gap-2">
              <dt className="text-sm text-muted-foreground">
                {t("publicationReview.currentState")}
              </dt>
              <dd>
                <PublicationStateBadge state={state} />
              </dd>
            </div>
            <div className="flex items-center gap-2">
              <dt className="text-sm text-muted-foreground">
                {t("publicationReview.lastValidation")}
              </dt>
              <dd>
                {lastRun ? (
                  <span className="flex items-center gap-2 text-sm">
                    <Badge variant={verdictTone(lastRun.verdict)}>
                      {t(`validationLab.verdict.${lastRun.verdict}`)}
                    </Badge>
                    <span className="text-muted-foreground">
                      {formatDateTime(lastRun.started_at)}
                    </span>
                  </span>
                ) : (
                  <span className="text-sm text-muted-foreground">
                    {t("publicationReview.noValidation")}
                  </span>
                )}
              </dd>
            </div>
            <div className="flex items-center gap-2">
              <dt className="text-sm text-muted-foreground">
                {t("publicationReview.rollbackTarget")}
              </dt>
              <dd>
                {rollbackTarget ? (
                  <PublicationStateBadge state={rollbackTarget} />
                ) : (
                  <span className="text-sm text-muted-foreground">
                    {t("publicationReview.noRollback")}
                  </span>
                )}
              </dd>
            </div>
            <div className="flex items-center gap-2">
              <dt className="text-sm text-muted-foreground">
                {t("publicationReview.versionCount")}
              </dt>
              <dd className="text-sm">{versions.length}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("publicationReview.gatesTitle")}</CardTitle>
          <CardDescription>{t("publicationReview.gatesDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <GateChecklist gates={gates} />
        </CardContent>
      </Card>

      {(blocking.length > 0 || warnings.length > 0) && (
        <Card>
          <CardHeader>
            <CardTitle>{t("publicationReview.gapsTitle")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {blocking.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-sm font-medium text-destructive">
                  {t("publicationReview.blockingGaps", { count: blocking.length })}
                </p>
                <ul className="space-y-1">
                  {blocking.map((gap) => (
                    <li
                      key={gap.id}
                      className="flex flex-wrap items-center gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-sm"
                    >
                      <Badge variant="destructive">
                        {t(`statusMapping.gaps.severity.${gap.severity}`)}
                      </Badge>
                      <Badge variant="outline">
                        {t(`statusMapping.gaps.type.${gap.gap_type}`)}
                      </Badge>
                      <span className="min-w-0 flex-1">
                        {gap.description || t("statusMapping.gaps.noDescription")}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {warnings.length > 0 && (
              <div className="space-y-1.5">
                <p className="text-sm font-medium text-warning">
                  {t("publicationReview.warningGaps", { count: warnings.length })}
                </p>
                <ul className="space-y-1">
                  {warnings.map((gap) => (
                    <li
                      key={gap.id}
                      className="flex flex-wrap items-center gap-2 rounded-md border border-warning/30 bg-warning/5 p-2 text-sm"
                    >
                      <Badge variant="warning">
                        {t(`statusMapping.gaps.severity.${gap.severity}`)}
                      </Badge>
                      <span className="min-w-0 flex-1">
                        {gap.description || t("statusMapping.gaps.noDescription")}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>{t("publicationReview.actionsTitle")}</CardTitle>
          <CardDescription>{t("publicationReview.actionsDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {transitions.length === 0 ? (
            <p className="flex items-center gap-2 text-sm text-muted-foreground">
              <Info className="h-4 w-4" />
              {t("publicationReview.noTransitions")}
            </p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {transitions.map((to) => {
                const decision = evaluatePublishAction(state, to, gates, gaps);
                return (
                  <TransitionButton
                    key={to}
                    to={to}
                    allowed={decision.allowed}
                    reasonKeys={decision.reasonKeys}
                    onClick={() => setTarget(to)}
                  />
                );
              })}
            </div>
          )}

          {canEmergencyDisable(state) && (
            <div className="border-t pt-3">
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDisableOpen(true)}
              >
                <Ban className="h-4 w-4" />
                {t("publicationReview.emergencyDisable")}
              </Button>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("publicationReview.emergencyDisableHint")}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Publish / transition confirmation */}
      <ActionDialog
        open={!!target}
        onOpenChange={(open) => {
          if (!open) {
            setTarget(null);
            setReason("");
          }
        }}
        title={
          target
            ? t("publicationReview.confirmTitle", {
                state: t(`publicationState.${target}`),
              })
            : ""
        }
        description={t("publicationReview.confirmDescription")}
        confirmLabel={t("publicationReview.confirmAction")}
        isPending={publish.isPending}
        onConfirm={runPublish}
      >
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="publish-reason">
            {t("publicationReview.reason")}
          </label>
          <Textarea
            id="publish-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={t("publicationReview.reasonPlaceholder")}
            rows={3}
          />
        </div>
      </ActionDialog>

      {/* Emergency disable — reason required */}
      <ActionDialog
        open={disableOpen}
        onOpenChange={(open) => {
          if (!open) {
            setDisableOpen(false);
            setDisableReason("");
          }
        }}
        variant="destructive"
        title={t("publicationReview.emergencyDisableTitle")}
        description={t("publicationReview.emergencyDisableDescription")}
        confirmLabel={t("publicationReview.emergencyDisableAction")}
        isPending={emergencyDisable.isPending}
        confirmDisabled={disableReason.trim().length === 0}
        onConfirm={runEmergencyDisable}
      >
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="disable-reason">
            {t("publicationReview.reasonRequired")}
          </label>
          <Textarea
            id="disable-reason"
            value={disableReason}
            onChange={(e) => setDisableReason(e.target.value)}
            placeholder={t("publicationReview.reasonPlaceholder")}
            rows={3}
            aria-required
          />
          {disableReason.trim().length === 0 && (
            <p className="text-xs text-destructive">
              {t("publicationReview.reasonRequiredHint")}
            </p>
          )}
        </div>
      </ActionDialog>
    </div>
  );
}

function TransitionButton({
  to,
  allowed,
  reasonKeys,
  onClick,
}: {
  to: ProviderPublicationState;
  allowed: boolean;
  reasonKeys: string[];
  onClick: () => void;
}) {
  const t = useTranslations("platform");

  const button = (
    <Button
      type="button"
      variant={to === "deprecated" || to === "retired" ? "outline" : "default"}
      size="sm"
      disabled={!allowed}
      onClick={onClick}
    >
      {t(`publicationReview.transitionTo.${to}`)}
    </Button>
  );

  if (allowed) {
    return button;
  }

  // Disabled actions must explain the concrete missing gates/gaps.
  return (
    <div className="flex flex-col gap-1">
      {button}
      <span className="max-w-[16rem] text-[11px] text-muted-foreground">
        {reasonKeys
          .map((key) => t(`publicationReview.disabledReason.${key}`))
          .join(" ")}
      </span>
    </div>
  );
}
