"use client";

import { useMemo, useState } from "react";
import { Plus, Lock, Info } from "lucide-react";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";
import { ApiClientError } from "@/lib/api-client";
import {
  useProviderGaps,
  useProviderStatusMappings,
  useUpdateProviderStatusMappings,
} from "@/hooks/use-platform-provider-config";
import { GapsInbox } from "@/components/platform/providers/status-mapping/gaps-inbox";
import { MappingRow } from "@/components/platform/providers/status-mapping/mapping-row";
import {
  computeMappingWarnings,
  emptyMapping,
  rowKey,
  warningsByRow,
} from "@/components/platform/providers/status-mapping/status-mapping-helpers";
import {
  STATUS_DOMAINS,
  type ProviderStatusMapping,
  type StatusDomain,
} from "@/types/platform";

interface StatusMappingWorkbenchProps {
  providerId: string;
  versionId: string;
  readOnly: boolean;
}

// Domains the backend persists today. The design spec also lists package /
// invoice / return tabs; those are surfaced as not-yet-supported so operators
// don't enter data the server would reject.
const UNSUPPORTED_DOMAINS = ["package", "invoice", "return"] as const;

export function StatusMappingWorkbench({
  providerId,
  versionId,
  readOnly,
}: StatusMappingWorkbenchProps) {
  const t = useTranslations("platform");
  const { data, isLoading, isError, refetch } = useProviderStatusMappings(
    providerId,
    versionId,
  );
  const { data: gaps = [] } = useProviderGaps(providerId, versionId);
  const updateMappings = useUpdateProviderStatusMappings(providerId, versionId);

  const [mappings, setMappings] = useEffectSyncedState<ProviderStatusMapping[]>(
    data ?? [],
    versionId,
  );
  const [activeDomain, setActiveDomain] = useState<StatusDomain>("order");

  const warnings = useMemo(
    () => computeMappingWarnings(mappings, gaps),
    [mappings, gaps],
  );
  const warningIndex = useMemo(() => warningsByRow(warnings), [warnings]);
  const blockingCount = warnings.filter((w) => w.blocking).length;

  const updateRow = (index: number, next: ProviderStatusMapping) => {
    setMappings((prev) => prev.map((m, i) => (i === index ? next : m)));
  };

  const removeRow = (index: number) => {
    setMappings((prev) => prev.filter((_, i) => i !== index));
  };

  const addRow = (domain: StatusDomain) => {
    setMappings((prev) => [...prev, emptyMapping(domain)]);
  };

  const handleSave = () => {
    const invalid = mappings.find((m) => m.raw_status.trim() === "");
    if (invalid) {
      toast.error(t("statusMapping.rawRequired"));
      return;
    }
    updateMappings.mutate(
      { status_mappings: mappings },
      {
        onSuccess: () => toast.success(t("statusMapping.saved")),
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
          {t("statusMapping.readOnlyNotice")}
        </div>
      )}

      <GapsInbox providerId={providerId} versionId={versionId} readOnly={readOnly} />

      <Card>
        <CardHeader>
          <CardTitle>{t("statusMapping.title")}</CardTitle>
          <CardDescription>{t("statusMapping.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs
            value={activeDomain}
            onValueChange={(v) => setActiveDomain(v as StatusDomain)}
          >
            <TabsList>
              {STATUS_DOMAINS.map((domain) => (
                <TabsTrigger key={domain} value={domain}>
                  {t(`statusMapping.domain.${domain}`)}
                </TabsTrigger>
              ))}
              {UNSUPPORTED_DOMAINS.map((domain) => (
                <TabsTrigger key={domain} value={domain} disabled>
                  {t(`statusMapping.domain.${domain}`)}
                </TabsTrigger>
              ))}
            </TabsList>

            {STATUS_DOMAINS.map((domain) => {
              const rows = mappings
                .map((mapping, index) => ({ mapping, index }))
                .filter(({ mapping }) => mapping.status_domain === domain);
              return (
                <TabsContent key={domain} value={domain} className="space-y-3 pt-4">
                  {rows.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      {t("statusMapping.empty")}
                    </p>
                  ) : (
                    rows.map(({ mapping, index }) => (
                      <MappingRow
                        key={index}
                        mapping={mapping}
                        warnings={
                          warningIndex.get(
                            rowKey(mapping.status_domain, mapping.raw_status),
                          ) ?? []
                        }
                        readOnly={readOnly}
                        onChange={(next) => updateRow(index, next)}
                        onRemove={() => removeRow(index)}
                      />
                    ))
                  )}
                  {!readOnly && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => addRow(domain)}
                    >
                      <Plus className="mr-1 h-3 w-3" />
                      {t("statusMapping.addRow")}
                    </Button>
                  )}
                </TabsContent>
              );
            })}

            {UNSUPPORTED_DOMAINS.map((domain) => (
              <TabsContent key={domain} value={domain} className="pt-4">
                <div className="flex items-center gap-2 rounded-md border p-3 text-sm text-muted-foreground">
                  <Info className="h-4 w-4" />
                  {t("statusMapping.domainNotSupported")}
                </div>
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>

      {!readOnly && (
        <div className="flex items-center justify-end gap-3">
          {blockingCount > 0 && (
            <span className="text-sm text-destructive">
              {t("statusMapping.warningCount", { count: blockingCount })}
            </span>
          )}
          <Button
            type="button"
            onClick={handleSave}
            disabled={updateMappings.isPending}
          >
            {updateMappings.isPending ? t("saving") : t("statusMapping.save")}
          </Button>
        </div>
      )}
    </div>
  );
}
