"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Eye, EyeOff, Lock } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatDateTime } from "@/lib/utils";
import {
  canViewSensitiveEvidence,
  evidenceTimeline,
  hasRedactableContent,
  redactObservation,
  shortHash,
} from "@/components/platform/providers/shared/evidence-helpers";
import { resultStatusTone } from "@/components/platform/providers/validation-lab/validation-helpers";
import type {
  PlatformMe,
  ProviderValidationResult,
  ProviderValidationRun,
} from "@/types/platform";

interface EvidenceDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run: ProviderValidationRun | undefined;
  me: PlatformMe | undefined;
}

/**
 * Screen 11 — Evidence drawer. Renders a run's immutable probe results as an
 * evidence timeline. Observations are redacted by default; only a viewer with
 * the `providers:secrets` permission can explicitly reveal the recorded
 * observation. Raw secrets are never rendered — payloads are referenced by hash.
 */
export function EvidenceDrawer({
  open,
  onOpenChange,
  run,
  me,
}: EvidenceDrawerProps) {
  const t = useTranslations("platform");
  const canViewSensitive = canViewSensitiveEvidence(me?.permissions);
  const results = evidenceTimeline(run?.results);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-xl">
        <SheetHeader>
          <SheetTitle>{t("evidence.title")}</SheetTitle>
          <SheetDescription>{t("evidence.description")}</SheetDescription>
        </SheetHeader>

        <div className="space-y-3 px-4 pb-6" aria-live="polite">
          {!run ? (
            <p className="text-sm text-muted-foreground">{t("evidence.noRun")}</p>
          ) : results.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("evidence.empty")}</p>
          ) : (
            <>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                {canViewSensitive ? (
                  <Badge variant="info">{t("evidence.sensitiveAllowed")}</Badge>
                ) : (
                  <Badge variant="secondary">
                    <Lock className="h-3 w-3" />
                    {t("evidence.redactedOnly")}
                  </Badge>
                )}
              </div>
              <ul className="space-y-3">
                {results.map((result) => (
                  <EvidenceRow
                    key={result.id}
                    result={result}
                    canViewSensitive={canViewSensitive}
                  />
                ))}
              </ul>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function EvidenceRow({
  result,
  canViewSensitive,
}: {
  result: ProviderValidationResult;
  canViewSensitive: boolean;
}) {
  const t = useTranslations("platform");
  const [revealed, setRevealed] = useState(false);
  const redactable = hasRedactableContent(result.observation);
  const observation = redactObservation(result.observation, {
    canViewSensitive,
    reveal: revealed,
  });

  return (
    <li className="space-y-2 rounded-md border bg-background p-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={resultStatusTone(result.status)}>
          {t(`validationLab.resultStatus.${result.status}`)}
        </Badge>
        <span className="min-w-0 flex-1 text-sm font-medium">{result.label}</span>
        <span className="font-mono text-[11px] text-muted-foreground">
          {result.probe_type}
        </span>
      </div>

      <div className="space-y-1 text-xs">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium text-muted-foreground">
            {t("evidence.observation")}
          </span>
          {redactable && canViewSensitive && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-[11px]"
              onClick={() => setRevealed((v) => !v)}
            >
              {revealed ? (
                <>
                  <EyeOff className="h-3 w-3" />
                  {t("evidence.hide")}
                </>
              ) : (
                <>
                  <Eye className="h-3 w-3" />
                  {t("evidence.reveal")}
                </>
              )}
            </Button>
          )}
        </div>
        <p className="whitespace-pre-wrap break-words rounded bg-muted p-2 font-mono text-[11px]">
          {observation || t("evidence.noObservation")}
        </p>
        {redactable && !canViewSensitive && (
          <p className="text-[11px] text-muted-foreground">
            {t("evidence.redactedNotice")}
          </p>
        )}
      </div>

      {result.findings && (
        <div className="text-xs">
          <span className="font-medium text-muted-foreground">
            {t("evidence.findings")}
          </span>
          <p className="break-words">{result.findings}</p>
        </div>
      )}

      <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-[11px]">
        <dt className="text-muted-foreground">{t("evidence.payloadHash")}</dt>
        <dd className="font-mono break-all" title={result.payload_hash}>
          {result.payload_hash ? shortHash(result.payload_hash) : "—"}
        </dd>
        <dt className="text-muted-foreground">{t("evidence.recordedAt")}</dt>
        <dd>{formatDateTime(result.created_at)}</dd>
      </dl>
    </li>
  );
}
