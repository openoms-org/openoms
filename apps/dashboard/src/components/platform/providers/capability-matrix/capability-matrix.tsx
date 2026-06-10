"use client";

import { useMemo, useState } from "react";
import { Plus, Lock } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";
import { ApiClientError } from "@/lib/api-client";
import {
  useProviderCapabilities,
  useUpdateProviderCapabilities,
} from "@/hooks/use-platform-provider-config";
import { CapabilityRow } from "@/components/platform/providers/capability-matrix/capability-row";
import {
  emptyCapability,
  groupByDomain,
} from "@/components/platform/providers/capability-matrix/capability-helpers";
import {
  SUPPORT_STATUSES,
  type ProviderCapability,
  type SupportStatus,
} from "@/types/platform";

interface CapabilityMatrixProps {
  providerId: string;
  versionId: string;
  readOnly: boolean;
}

type StatusFilter = SupportStatus | "all";

export function CapabilityMatrix({
  providerId,
  versionId,
  readOnly,
}: CapabilityMatrixProps) {
  const t = useTranslations("platform");
  const { data, isLoading, isError, refetch } = useProviderCapabilities(
    providerId,
    versionId,
  );
  const updateCapabilities = useUpdateProviderCapabilities(providerId, versionId);

  const [capabilities, setCapabilities] = useEffectSyncedState<ProviderCapability[]>(
    data ?? [],
    versionId,
  );
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [filter, setFilter] = useState<StatusFilter>("all");

  const filtered = useMemo(
    () =>
      filter === "all"
        ? capabilities
        : capabilities.filter((c) => c.support_status === filter),
    [capabilities, filter],
  );

  // Group filtered rows by domain, but keep each row's original index so edits
  // map back to the unfiltered list.
  const grouped = useMemo(() => {
    const withIndex = capabilities.map((c, index) => ({ c, index }));
    const visible = withIndex.filter(
      ({ c }) => filter === "all" || c.support_status === filter,
    );
    const byDomain = groupByDomain(visible.map(({ c }) => c));
    const indexByCap = new Map(visible.map(({ c, index }) => [c, index]));
    return byDomain.map((group) => ({
      domain: group.domain,
      rows: group.capabilities.map((c) => ({
        capability: c,
        index: indexByCap.get(c)!,
      })),
    }));
  }, [capabilities, filter]);

  const updateRow = (index: number, next: ProviderCapability) => {
    setCapabilities((prev) => prev.map((c, i) => (i === index ? next : c)));
  };

  const removeRow = (index: number) => {
    setCapabilities((prev) => prev.filter((_, i) => i !== index));
    setExpanded((prev) => {
      const next = new Set<number>();
      for (const i of prev) {
        if (i < index) next.add(i);
        else if (i > index) next.add(i - 1);
      }
      return next;
    });
  };

  const toggleRow = (index: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  };

  const addCapability = () => {
    setCapabilities((prev) => [...prev, emptyCapability()]);
    setExpanded((prev) => new Set(prev).add(capabilities.length));
  };

  const handleSave = () => {
    const invalid = capabilities.find((c) => c.capability_key.trim() === "");
    if (invalid) {
      toast.error(t("capabilityMatrix.keyRequired"));
      return;
    }
    updateCapabilities.mutate(
      { capabilities },
      {
        onSuccess: () => toast.success(t("capabilityMatrix.saved")),
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <div className="rounded-md border border-destructive bg-destructive/10 p-4">
        <p className="text-sm text-destructive">{t("loadError")}</p>
        <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
          {t("retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {readOnly && (
        <div className="flex items-center gap-2 rounded-md border border-warning/40 bg-warning/10 p-3 text-sm text-warning">
          <Lock className="h-4 w-4" />
          {t("capabilityMatrix.readOnlyNotice")}
        </div>
      )}

      <Card>
        <CardHeader className="flex-row items-start justify-between gap-3">
          <div>
            <CardTitle>{t("capabilityMatrix.title")}</CardTitle>
            <CardDescription>{t("capabilityMatrix.description")}</CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Select value={filter} onValueChange={(v) => setFilter(v as StatusFilter)}>
              <SelectTrigger className="h-8 w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t("capabilityMatrix.filterAll")}
                </SelectItem>
                {SUPPORT_STATUSES.map((status) => (
                  <SelectItem key={status} value={status}>
                    {t(`capabilityMatrix.supportStatus.${status}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {!readOnly && (
              <Button type="button" variant="outline" size="sm" onClick={addCapability}>
                <Plus className="mr-1 h-3 w-3" />
                {t("capabilityMatrix.addRow")}
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          {filtered.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("capabilityMatrix.empty")}
            </p>
          ) : (
            grouped.map((group) => (
              <div key={group.domain} className="space-y-2">
                <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  {t.has(`capabilityMatrix.domain.${group.domain}`)
                    ? t(`capabilityMatrix.domain.${group.domain}`)
                    : group.domain}
                </h4>
                <div className="space-y-2">
                  {group.rows.map(({ capability, index }) => (
                    <CapabilityRow
                      key={index}
                      capability={capability}
                      expanded={expanded.has(index)}
                      readOnly={readOnly}
                      onToggle={() => toggleRow(index)}
                      onChange={(next) => updateRow(index, next)}
                      onRemove={() => removeRow(index)}
                    />
                  ))}
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      {!readOnly && (
        <div className="flex justify-end">
          <Button
            type="button"
            onClick={handleSave}
            disabled={updateCapabilities.isPending}
          >
            {updateCapabilities.isPending ? t("saving") : t("capabilityMatrix.save")}
          </Button>
        </div>
      )}
    </div>
  );
}
