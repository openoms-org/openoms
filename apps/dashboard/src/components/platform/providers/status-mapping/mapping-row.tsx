"use client";

import { AlertTriangle, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { MappingWarning } from "@/components/platform/providers/status-mapping/status-mapping-helpers";
import {
  MAPPING_CONFIDENCES,
  type MappingConfidence,
  type ProviderStatusMapping,
} from "@/types/platform";

interface MappingRowProps {
  mapping: ProviderStatusMapping;
  warnings: MappingWarning[];
  readOnly: boolean;
  onChange: (next: ProviderStatusMapping) => void;
  onRemove: () => void;
}

/**
 * One status-mapping row: observed raw status on the left, canonical mapping
 * rule on the right. Per-row warnings (terminal low-confidence, shipment over
 * order, unknown-needs-gap) render inline so the operator cannot silently ship
 * an unsafe mapping.
 */
export function MappingRow({
  mapping,
  warnings,
  readOnly,
  onChange,
  onRemove,
}: MappingRowProps) {
  const t = useTranslations("platform");
  const hasBlocking = warnings.some((w) => w.blocking);

  const update = (patch: Partial<ProviderStatusMapping>) => {
    onChange({ ...mapping, ...patch });
  };

  return (
    <div
      className={cn(
        "grid gap-3 rounded-md border p-3 lg:grid-cols-[1fr_1.4fr]",
        hasBlocking ? "border-destructive/50 bg-destructive/5" : "border-border",
      )}
    >
      {/* Left: observed raw status */}
      <div className="space-y-2">
        <div className="grid gap-1.5">
          <Label>{t("statusMapping.fields.rawStatus")}</Label>
          <Input
            value={mapping.raw_status}
            disabled={readOnly}
            className="font-mono"
            onChange={(e) => update({ raw_status: e.target.value })}
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t("statusMapping.fields.notes")}</Label>
          <Input
            value={mapping.notes}
            disabled={readOnly}
            placeholder={t("statusMapping.fields.notesPlaceholder")}
            onChange={(e) => update({ notes: e.target.value })}
          />
        </div>
      </div>

      {/* Right: canonical mapping rule */}
      <div className="space-y-2">
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label>{t("statusMapping.fields.canonicalStatus")}</Label>
            <Input
              value={mapping.canonical_status}
              disabled={readOnly}
              className="font-mono"
              placeholder={t("statusMapping.fields.canonicalPlaceholder")}
              onChange={(e) => update({ canonical_status: e.target.value })}
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t("statusMapping.fields.confidence")}</Label>
            <Select
              value={mapping.confidence}
              disabled={readOnly}
              onValueChange={(v) => update({ confidence: v as MappingConfidence })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MAPPING_CONFIDENCES.map((confidence) => (
                  <SelectItem key={confidence} value={confidence}>
                    {t(`statusMapping.confidence.${confidence}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label>{t("statusMapping.fields.eventType")}</Label>
            <Input
              value={mapping.canonical_event_type}
              disabled={readOnly}
              className="font-mono"
              onChange={(e) => update({ canonical_event_type: e.target.value })}
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t("statusMapping.fields.stepKey")}</Label>
            <Input
              value={mapping.canonical_step_key}
              disabled={readOnly}
              className="font-mono"
              onChange={(e) => update({ canonical_step_key: e.target.value })}
            />
          </div>
        </div>

        <div className="flex items-center justify-between rounded-md border p-2.5">
          <Label className="cursor-pointer">
            {t("statusMapping.fields.terminal")}
          </Label>
          <Switch
            checked={mapping.is_terminal}
            disabled={readOnly}
            onCheckedChange={(checked) => update({ is_terminal: checked })}
          />
        </div>

        {warnings.length > 0 && (
          <ul className="space-y-1">
            {warnings.map((warning) => (
              <li
                key={warning.code}
                className={cn(
                  "flex items-start gap-1.5 text-xs",
                  warning.blocking ? "text-destructive" : "text-warning",
                )}
              >
                <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" />
                <span>{t(`statusMapping.warning.${warning.code}`)}</span>
              </li>
            ))}
          </ul>
        )}

        {!readOnly && (
          <div className="flex items-center justify-between">
            {hasBlocking && (
              <Badge variant="destructive">{t("statusMapping.blocksAutomation")}</Badge>
            )}
            <button
              type="button"
              onClick={onRemove}
              className="ml-auto inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t("statusMapping.removeRow")}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
