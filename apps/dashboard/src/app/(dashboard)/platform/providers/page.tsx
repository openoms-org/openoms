"use client";

import { useRouter } from "next/navigation";
import { Boxes } from "lucide-react";
import { useTranslations } from "next-intl";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { Button } from "@/components/ui/button";
import { PlatformAccessGate } from "@/components/platform/platform-access-gate";
import { PublicationStateBadge } from "@/components/platform/publication-state-badge";
import { usePlatformMe, useProviderDefinitions, hasPlatformAccess } from "@/hooks/use-platform-providers";
import type { ProviderDefinition, ProviderPublicationState } from "@/types/platform";

function ProviderRegistry() {
  const t = useTranslations("platform");
  const router = useRouter();
  const { data: me } = usePlatformMe();
  const { data, isLoading, isError, refetch } = useProviderDefinitions({
    enabled: hasPlatformAccess(me),
  });

  const definitions = data ?? [];

  const columns: ColumnDef<ProviderDefinition>[] = [
    {
      header: t("registry.columns.key"),
      accessorKey: "provider_key",
      cell: (row) => <span className="font-mono text-sm">{row.provider_key}</span>,
    },
    {
      header: t("registry.columns.name"),
      accessorKey: "display_name",
      cell: (row) => <span className="font-medium">{row.display_name}</span>,
    },
    {
      header: t("registry.columns.type"),
      accessorKey: "provider_type",
    },
    {
      header: t("registry.columns.latestState"),
      accessorKey: "latest_published_version_id",
      cell: (row) => <LatestStateCell definition={row} />,
    },
  ];

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
    <div className="rounded-md border">
      <DataTable
        columns={columns}
        data={definitions}
        isLoading={isLoading}
        onRowClick={(row) => router.push(`/platform/providers/${row.id}`)}
        emptyState={
          <EmptyState
            icon={Boxes}
            title={t("registry.emptyTitle")}
            description={t("registry.emptyDescription")}
          />
        }
      />
    </div>
  );
}

/**
 * The registry list only carries the latest published version *id*, not its
 * state, so when present we render a neutral "published" marker; otherwise a
 * dash. The full version timeline (with states) lives on the detail page.
 */
function LatestStateCell({ definition }: { definition: ProviderDefinition }) {
  if (!definition.latest_published_version_id) {
    return <span className="text-muted-foreground">—</span>;
  }
  const published: ProviderPublicationState = "available";
  return <PublicationStateBadge state={published} />;
}

export default function PlatformProvidersPage() {
  const t = useTranslations("platform");

  return (
    <>
      <PageHeader title={t("registry.title")} description={t("registry.description")} />
      <PlatformAccessGate>
        <ProviderRegistry />
      </PlatformAccessGate>
    </>
  );
}
