"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { FileSearch, History } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/shared/empty-state";
import { formatDateTime } from "@/lib/utils";
import { useProviderValidationRun } from "@/hooks/use-platform-provider-validation";
import { EvidenceDrawer } from "@/components/platform/providers/shared/evidence-drawer";
import { resultStatusTone, verdictTone } from "./validation-helpers";
import type {
  PlatformMe,
  ProviderValidationRun,
} from "@/types/platform";

interface ValidationRunTimelineProps {
  providerId: string;
  versionId: string;
  runs: ProviderValidationRun[];
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
  activeRunId: string | null;
  onSelectRun: (runId: string) => void;
  me: PlatformMe | undefined;
}

/**
 * Immutable validation-run history with per-probe results. Selecting a run loads
 * its full results; the evidence drawer (Screen 11) opens redacted evidence.
 */
export function ValidationRunTimeline({
  providerId,
  versionId,
  runs,
  isLoading,
  isError,
  refetch,
  activeRunId,
  onSelectRun,
  me,
}: ValidationRunTimelineProps) {
  const t = useTranslations("platform");
  const [evidenceRunId, setEvidenceRunId] = useState<string | null>(null);

  const selectedRunId = activeRunId ?? runs[0]?.id ?? null;
  const detail = useProviderValidationRun(providerId, versionId, selectedRunId);
  const evidenceDetail = useProviderValidationRun(
    providerId,
    versionId,
    evidenceRunId,
    { enabled: !!evidenceRunId },
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("validationLab.runsTitle")}</CardTitle>
        <CardDescription>{t("validationLab.runsDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isError ? (
          <div className="rounded-md border border-destructive bg-destructive/10 p-4">
            <p className="text-sm text-destructive">{t("loadError")}</p>
            <Button variant="outline" size="sm" className="mt-2" onClick={refetch}>
              {t("retry")}
            </Button>
          </div>
        ) : isLoading ? (
          <p className="text-sm text-muted-foreground">{t("validationLab.loadingRuns")}</p>
        ) : runs.length === 0 ? (
          <EmptyState
            icon={History}
            title={t("validationLab.noRunsTitle")}
            description={t("validationLab.noRunsDescription")}
          />
        ) : (
          <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
            <ul className="space-y-1.5" aria-label={t("validationLab.runsTitle")}>
              {runs.map((run) => {
                const selected = run.id === selectedRunId;
                return (
                  <li key={run.id}>
                    <button
                      type="button"
                      onClick={() => onSelectRun(run.id)}
                      aria-pressed={selected}
                      className={`flex w-full flex-wrap items-center gap-2 rounded-md border p-2.5 text-left text-sm transition-colors ${
                        selected
                          ? "border-primary bg-primary/5"
                          : "bg-background hover:bg-muted"
                      }`}
                    >
                      <Badge variant={verdictTone(run.verdict)}>
                        {t(`validationLab.verdict.${run.verdict}`)}
                      </Badge>
                      <Badge variant="outline">
                        {t(`validationLab.env.${run.environment}`)}
                      </Badge>
                      <span className="ml-auto text-xs text-muted-foreground">
                        {formatDateTime(run.started_at)}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>

            <div className="min-w-0">
              {detail.isLoading ? (
                <p className="text-sm text-muted-foreground">
                  {t("validationLab.loadingResults")}
                </p>
              ) : detail.data ? (
                <RunDetail
                  run={detail.data}
                  onOpenEvidence={() => setEvidenceRunId(detail.data!.id)}
                />
              ) : (
                <p className="text-sm text-muted-foreground">
                  {t("validationLab.selectRun")}
                </p>
              )}
            </div>
          </div>
        )}
      </CardContent>

      <EvidenceDrawer
        open={!!evidenceRunId}
        onOpenChange={(open) => {
          if (!open) setEvidenceRunId(null);
        }}
        run={evidenceDetail.data}
        me={me}
      />
    </Card>
  );
}

function RunDetail({
  run,
  onOpenEvidence,
}: {
  run: ProviderValidationRun;
  onOpenEvidence: () => void;
}) {
  const t = useTranslations("platform");
  const results = run.results ?? [];

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={verdictTone(run.verdict)}>
          {t(`validationLab.verdict.${run.verdict}`)}
        </Badge>
        <span className="text-xs text-muted-foreground">
          {run.finished_at
            ? t("validationLab.finishedAt", {
                time: formatDateTime(run.finished_at),
              })
            : t("validationLab.runInProgress")}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="ml-auto"
          onClick={onOpenEvidence}
          disabled={results.length === 0}
        >
          <FileSearch className="h-4 w-4" />
          {t("validationLab.openEvidence")}
        </Button>
      </div>

      {results.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          {t("validationLab.noResults")}
        </p>
      ) : (
        <ul className="space-y-1.5">
          {results.map((result) => (
            <li
              key={result.id}
              className="flex flex-wrap items-center gap-2 rounded-md border bg-background p-2.5 text-sm"
            >
              <Badge variant={resultStatusTone(result.status)}>
                {t(`validationLab.resultStatus.${result.status}`)}
              </Badge>
              <span className="min-w-0 flex-1">{result.label}</span>
              <span className="font-mono text-[11px] text-muted-foreground">
                {result.probe_type}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
