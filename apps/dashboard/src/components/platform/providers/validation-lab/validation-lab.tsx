"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { FlaskConical, Play, ShieldAlert } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { EmptyState } from "@/components/shared/empty-state";
import { ActionDialog } from "@/components/shared/action-dialog";
import { ApiClientError } from "@/lib/api-client";
import {
  useProviderProbes,
  useProviderValidationRuns,
  useStartValidationRun,
} from "@/hooks/use-platform-provider-validation";
import { canViewSensitiveEvidence } from "@/components/platform/providers/shared/evidence-helpers";
import {
  groupProbesBySafety,
  hasDestructiveProbe,
  probeSafetyClass,
  safetyClassTone,
} from "./validation-helpers";
import { ValidationRunTimeline } from "./validation-run-timeline";
import type {
  PlatformMe,
  ProbeSafetyClass,
  ProviderValidationProbe,
  ValidationEnvironment,
} from "@/types/platform";

interface ValidationLabProps {
  providerId: string;
  versionId: string;
  me: PlatformMe | undefined;
}

/**
 * Screen 8 — Validation Lab. Lists declared probes grouped by safety class,
 * starts immutable validation runs and shows the run timeline + evidence. Probe
 * execution is performed externally (no synchronous "run probe" backend); this
 * surface opens a run and records the verdict path. Production writes require
 * the `providers:secrets` permission; destructive probes need explicit confirm.
 */
export function ValidationLab({ providerId, versionId, me }: ValidationLabProps) {
  const t = useTranslations("platform");
  const { data: probes = [], isLoading } = useProviderProbes(providerId, versionId);
  const runsQuery = useProviderValidationRuns(providerId, versionId);
  const startRun = useStartValidationRun(providerId, versionId);

  const [environment, setEnvironment] = useState<ValidationEnvironment>("sandbox");
  const [allowDestructive, setAllowDestructive] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [activeRunId, setActiveRunId] = useState<string | null>(null);

  const canProduction = canViewSensitiveEvidence(me?.permissions);
  const grouped = useMemo(
    () => groupProbesBySafety(probes, environment),
    [probes, environment],
  );
  const destructivePresent = hasDestructiveProbe(probes);

  // Production environment is opt-in and permission-gated (Screen 8).
  const environmentBlocked = environment === "production" && !canProduction;

  const beginRun = () => {
    // Destructive probes require an explicit confirmation before the run opens.
    if (destructivePresent && allowDestructive) {
      setConfirmOpen(true);
      return;
    }
    submitRun(false);
  };

  const submitRun = (destructive: boolean) => {
    startRun.mutate(
      { environment, allow_destructive: destructive },
      {
        onSuccess: (run) => {
          setActiveRunId(run.id);
          setConfirmOpen(false);
          toast.success(t("validationLab.runStarted"));
        },
        onError: (err) => {
          setConfirmOpen(false);
          toast.error(
            err instanceof ApiClientError ? err.message : t("saveError"),
          );
        },
      },
    );
  };

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <Card>
          <CardHeader>
            <CardTitle>{t("validationLab.probesTitle")}</CardTitle>
            <CardDescription>{t("validationLab.probesDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <p className="text-sm text-muted-foreground">{t("validationLab.loadingProbes")}</p>
            ) : probes.length === 0 ? (
              <EmptyState
                icon={FlaskConical}
                title={t("validationLab.noProbesTitle")}
                description={t("validationLab.noProbesDescription")}
              />
            ) : (
              <div className="space-y-5">
                {grouped.map((group) => (
                  <ProbeGroup
                    key={group.safety}
                    safety={group.safety}
                    probes={group.probes}
                    environment={environment}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("validationLab.runTitle")}</CardTitle>
            <CardDescription>{t("validationLab.runDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="validation-env">
                {t("validationLab.environment")}
              </label>
              <Select
                value={environment}
                onValueChange={(v) => setEnvironment(v as ValidationEnvironment)}
              >
                <SelectTrigger id="validation-env">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sandbox">
                    {t("validationLab.env.sandbox")}
                  </SelectItem>
                  <SelectItem value="production">
                    {t("validationLab.env.production")}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            {environmentBlocked && (
              <p
                role="alert"
                className="flex items-start gap-2 rounded-md border border-warning/40 bg-warning/10 p-2.5 text-xs text-warning"
              >
                <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
                {t("validationLab.productionPermissionRequired")}
              </p>
            )}

            {destructivePresent && (
              <label className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-2.5 text-xs">
                <Checkbox
                  checked={allowDestructive}
                  onCheckedChange={(c) => setAllowDestructive(c === true)}
                  aria-label={t("validationLab.allowDestructive")}
                  className="mt-0.5"
                />
                <span>
                  <span className="font-medium text-destructive">
                    {t("validationLab.allowDestructive")}
                  </span>
                  <span className="block text-muted-foreground">
                    {t("validationLab.allowDestructiveHint")}
                  </span>
                </span>
              </label>
            )}

            <Button
              type="button"
              onClick={beginRun}
              disabled={
                startRun.isPending || probes.length === 0 || environmentBlocked
              }
              className="w-full"
            >
              <Play className="h-4 w-4" />
              {t("validationLab.startRun")}
            </Button>

            <p className="text-xs text-muted-foreground">
              {t("validationLab.safetyNote")}
            </p>
          </CardContent>
        </Card>
      </div>

      <ValidationRunTimeline
        providerId={providerId}
        versionId={versionId}
        runs={runsQuery.data ?? []}
        isLoading={runsQuery.isLoading}
        isError={runsQuery.isError}
        refetch={() => runsQuery.refetch()}
        activeRunId={activeRunId}
        onSelectRun={setActiveRunId}
        me={me}
      />

      <ActionDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        variant="destructive"
        title={t("validationLab.destructiveConfirmTitle")}
        description={t("validationLab.destructiveConfirmDescription")}
        confirmLabel={t("validationLab.destructiveConfirmAction")}
        isPending={startRun.isPending}
        onConfirm={() => submitRun(true)}
      />
    </div>
  );
}

function ProbeGroup({
  safety,
  probes,
  environment,
}: {
  safety: ProbeSafetyClass;
  probes: ProviderValidationProbe[];
  environment: ValidationEnvironment;
}) {
  const t = useTranslations("platform");
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant={safetyClassTone(safety)}>
          {t(`validationLab.safety.${safety}`)}
        </Badge>
        <span className="text-xs text-muted-foreground">
          {t(`validationLab.safetyHint.${safety}`)}
        </span>
      </div>
      <ul className="space-y-1.5">
        {probes.map((probe) => {
          const cls = probeSafetyClass(probe, environment);
          return (
            <li
              key={probe.id}
              className="flex flex-wrap items-center gap-2 rounded-md border bg-background p-2.5"
            >
              <span className="min-w-0 flex-1 text-sm font-medium">
                {probe.label}
              </span>
              <Badge variant="outline" className="font-mono text-[11px]">
                {t(`validationLab.probeType.${probe.probe_type}`)}
              </Badge>
              {probe.required && (
                <Badge variant="secondary">{t("validationLab.probeRequired")}</Badge>
              )}
              {cls === "destructive" && (
                <Badge variant="destructive">{t("validationLab.safety.destructive")}</Badge>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
