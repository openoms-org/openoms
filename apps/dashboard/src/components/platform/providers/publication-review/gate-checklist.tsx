"use client";

import { useTranslations } from "next-intl";
import { AlertTriangle, CheckCircle2, CircleDashed, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { GateState, GateStatus } from "./publication-helpers";

const STATUS_TONE: Record<GateStatus, "success" | "warning" | "destructive" | "secondary"> = {
  pass: "success",
  warn: "warning",
  blocked: "destructive",
  manual: "secondary",
};

function GateIcon({ status }: { status: GateStatus }) {
  switch (status) {
    case "pass":
      return <CheckCircle2 className="h-4 w-4 text-success" aria-hidden />;
    case "warn":
      return <AlertTriangle className="h-4 w-4 text-warning" aria-hidden />;
    case "blocked":
      return <XCircle className="h-4 w-4 text-destructive" aria-hidden />;
    default:
      return <CircleDashed className="h-4 w-4 text-muted-foreground" aria-hidden />;
  }
}

/**
 * G1-G9 production-readiness gate checklist. Every gate is derived client-side
 * (no dedicated backend gate endpoint exists) and is labelled as derived; gates
 * that require a human sign-off render as "manual" rather than auto-passing.
 */
export function GateChecklist({ gates }: { gates: GateState[] }) {
  const t = useTranslations("platform");

  return (
    <ul className="space-y-1.5">
      {gates.map((gate) => (
        <li
          key={gate.id}
          className="flex flex-wrap items-center gap-2 rounded-md border bg-background p-2.5 text-sm"
        >
          <GateIcon status={gate.status} />
          <span className="font-medium">{gate.id}</span>
          <span className="min-w-0 flex-1 text-muted-foreground">
            {t(`publicationReview.gate.${gate.id}`)}
          </span>
          {gate.derived && (
            <Badge variant="outline" className="text-[11px]">
              {t("publicationReview.derived")}
            </Badge>
          )}
          <Badge variant={STATUS_TONE[gate.status]}>
            {t(`publicationReview.gateStatus.${gate.status}`)}
          </Badge>
          <span className="basis-full pl-6 text-xs text-muted-foreground">
            {t(`publicationReview.gateReason.${gate.reasonKey}`)}
          </span>
        </li>
      ))}
    </ul>
  );
}
