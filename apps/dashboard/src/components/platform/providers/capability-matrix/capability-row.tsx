"use client";

import { ChevronDown, ChevronRight, AlertTriangle, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SupportStateBadge } from "@/components/platform/providers/shared/support-state-badge";
import { EvidenceLink } from "@/components/platform/providers/shared/evidence-link";
import {
  hasProbeReference,
  isCustomerVisible,
  isEvidenceMissing,
} from "@/components/platform/providers/capability-matrix/capability-helpers";
import {
  SUPPORT_STATUSES,
  type ProviderCapability,
  type SupportStatus,
} from "@/types/platform";

interface CapabilityRowProps {
  capability: ProviderCapability;
  expanded: boolean;
  readOnly: boolean;
  onToggle: () => void;
  onChange: (next: ProviderCapability) => void;
  onRemove: () => void;
}

function commaList(values: string[]): string {
  return values.join(", ");
}

function parseList(value: string): string[] {
  return value
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
}

/**
 * One capability row with inline expansion (design spec §327: inline details in
 * an expanding row, not a per-capability modal). Collapsed view shows the key,
 * support state, channel/mode/freshness and an evidence marker; the expanded
 * view edits all fields including customer-visibility/evidence.
 */
export function CapabilityRow({
  capability,
  expanded,
  readOnly,
  onToggle,
  onChange,
  onRemove,
}: CapabilityRowProps) {
  const t = useTranslations("platform");
  const evidenceMissing = isEvidenceMissing(capability);
  const probeReferenced = hasProbeReference(capability);
  const customerVisible = isCustomerVisible(capability);

  const update = (patch: Partial<ProviderCapability>) => {
    onChange({ ...capability, ...patch });
  };

  return (
    <div className="rounded-md border">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-3 py-2.5 text-left"
        aria-expanded={expanded}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
        <span className="min-w-0 flex-1 truncate font-mono text-sm">
          {capability.capability_key || t("capabilityMatrix.unnamed")}
        </span>
        <SupportStateBadge status={capability.support_status} />
        {capability.channel && (
          <Badge variant="outline" className="hidden sm:inline-flex">
            {capability.channel}
          </Badge>
        )}
        {evidenceMissing && (
          <Badge variant="destructive" className="gap-1">
            <AlertTriangle className="h-3 w-3" />
            {t("capabilityMatrix.evidenceRequired")}
          </Badge>
        )}
      </button>

      {expanded && (
        <div className="space-y-4 border-t p-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.capabilityKey")}</Label>
              <Input
                value={capability.capability_key}
                disabled={readOnly}
                className="font-mono"
                onChange={(e) => update({ capability_key: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.supportState")}</Label>
              <Select
                value={capability.support_status}
                disabled={readOnly}
                onValueChange={(v) =>
                  update({ support_status: v as SupportStatus })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SUPPORT_STATUSES.map((status) => (
                    <SelectItem key={status} value={status}>
                      {t(`capabilityMatrix.supportStatus.${status}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.channel")}</Label>
              <Input
                value={capability.channel}
                disabled={readOnly}
                onChange={(e) => update({ channel: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.mode")}</Label>
              <Input
                value={capability.mode}
                disabled={readOnly}
                onChange={(e) => update({ mode: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.freshness")}</Label>
              <Input
                value={capability.freshness}
                disabled={readOnly}
                onChange={(e) => update({ freshness: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.latencySla")}</Label>
              <Input
                type="number"
                value={capability.latency_sla_seconds ?? ""}
                disabled={readOnly}
                onChange={(e) =>
                  update({
                    latency_sla_seconds:
                      e.target.value.trim() === ""
                        ? null
                        : Number(e.target.value),
                  })
                }
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.requiredInputs")}</Label>
              <Input
                value={commaList(capability.required_inputs)}
                disabled={readOnly}
                onChange={(e) =>
                  update({ required_inputs: parseList(e.target.value) })
                }
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t("capabilityMatrix.fields.providedOutputs")}</Label>
              <Input
                value={commaList(capability.provided_outputs)}
                disabled={readOnly}
                onChange={(e) =>
                  update({ provided_outputs: parseList(e.target.value) })
                }
              />
            </div>
          </div>

          <div className="grid gap-1.5">
            <Label>{t("capabilityMatrix.fields.evidence")}</Label>
            <Input
              value={capability.notes}
              disabled={readOnly}
              placeholder={t("capabilityMatrix.fields.evidencePlaceholder")}
              onChange={(e) => update({ notes: e.target.value })}
            />
            <div className="flex items-center gap-2">
              <EvidenceLink value={capability.notes} />
              {evidenceMissing && (
                <span className="text-xs text-destructive">
                  {t("capabilityMatrix.evidenceRequiredHint")}
                </span>
              )}
            </div>
          </div>

          {/*
            Validation probe and customer visibility are derived (read-only)
            views — the backend capability model persists only the fields above,
            so we surface these as computed indicators rather than inventing
            extra payload the server would silently drop.
          */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-md border p-3">
              <Label className="cursor-default">
                {t("capabilityMatrix.fields.validationProbe")}
              </Label>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("capabilityMatrix.fields.validationProbeHint")}
              </p>
              <Badge variant={probeReferenced ? "info" : "outline"} className="mt-2">
                {probeReferenced
                  ? t("capabilityMatrix.probeReferenced")
                  : t("capabilityMatrix.probeNone")}
              </Badge>
            </div>
            <div className="rounded-md border p-3">
              <Label className="cursor-default">
                {t("capabilityMatrix.fields.customerVisibility")}
              </Label>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("capabilityMatrix.fields.customerVisibilityHint")}
              </p>
              <Badge
                variant={customerVisible ? "success" : "secondary"}
                className="mt-2"
              >
                {customerVisible
                  ? t("capabilityMatrix.visibilityShown")
                  : t("capabilityMatrix.visibilityHidden")}
              </Badge>
            </div>
          </div>

          {!readOnly && (
            <div className="flex justify-end">
              <button
                type="button"
                onClick={onRemove}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" />
                {t("capabilityMatrix.removeRow")}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
