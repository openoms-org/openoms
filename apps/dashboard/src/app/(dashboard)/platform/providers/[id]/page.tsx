"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Boxes } from "lucide-react";
import { useTranslations } from "next-intl";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PlatformAccessGate } from "@/components/platform/platform-access-gate";
import { PublicationStateBadge } from "@/components/platform/publication-state-badge";
import { VersionConfig } from "@/components/platform/providers/version-config";
import {
  usePlatformMe,
  useProviderDefinition,
  useProviderVersions,
  hasPlatformAccess,
} from "@/hooks/use-platform-providers";
import { formatDate } from "@/lib/utils";
import type { ProviderDefinition, ProviderVersion } from "@/types/platform";

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 py-2 sm:flex-row sm:items-start sm:gap-4">
      <dt className="w-48 shrink-0 text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm text-foreground">{children}</dd>
    </div>
  );
}

function TagList({ values }: { values: string[] }) {
  if (!values || values.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {values.map((value) => (
        <Badge key={value} variant="outline">
          {value}
        </Badge>
      ))}
    </div>
  );
}

function OverviewCard({ definition }: { definition: ProviderDefinition }) {
  const t = useTranslations("platform");

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("detail.metadataTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="divide-y">
          <MetaRow label={t("detail.fields.providerKey")}>
            <span className="font-mono">{definition.provider_key}</span>
          </MetaRow>
          <MetaRow label={t("detail.fields.providerType")}>{definition.provider_type}</MetaRow>
          <MetaRow label={t("detail.fields.owner")}>
            {definition.owner || <span className="text-muted-foreground">—</span>}
          </MetaRow>
          <MetaRow label={t("detail.fields.regions")}>
            <TagList values={definition.regions} />
          </MetaRow>
          <MetaRow label={t("detail.fields.businessDomains")}>
            <TagList values={definition.business_domains} />
          </MetaRow>
          <MetaRow label={t("detail.fields.notes")}>
            {definition.notes || <span className="text-muted-foreground">—</span>}
          </MetaRow>
          <MetaRow label={t("detail.fields.createdAt")}>
            {formatDate(definition.created_at)}
          </MetaRow>
          <MetaRow label={t("detail.fields.updatedAt")}>
            {formatDate(definition.updated_at)}
          </MetaRow>
        </dl>
      </CardContent>
    </Card>
  );
}

function VersionsCard({
  versions,
  isLoading,
  isError,
  refetch,
}: {
  versions: ProviderVersion[];
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
}) {
  const t = useTranslations("platform");

  const columns: ColumnDef<ProviderVersion>[] = [
    {
      header: t("detail.versionsColumns.version"),
      accessorKey: "version",
      cell: (row) => <span className="font-mono text-sm">{row.version}</span>,
    },
    {
      header: t("detail.versionsColumns.state"),
      accessorKey: "publication_state",
      cell: (row) => <PublicationStateBadge state={row.publication_state} />,
    },
    {
      header: t("detail.versionsColumns.changelog"),
      accessorKey: "changelog",
      cell: (row) =>
        row.changelog || <span className="text-muted-foreground">—</span>,
    },
    {
      header: t("detail.versionsColumns.createdAt"),
      accessorKey: "created_at",
      cell: (row) => formatDate(row.created_at),
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("detail.versionsTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isError ? (
          <div className="rounded-md border border-destructive bg-destructive/10 p-4">
            <p className="text-sm text-destructive">{t("loadError")}</p>
            <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
              {t("retry")}
            </Button>
          </div>
        ) : (
          <DataTable
            columns={columns}
            data={versions}
            isLoading={isLoading}
            emptyMessage={t("detail.versionsEmpty")}
          />
        )}
      </CardContent>
    </Card>
  );
}

function ProviderDetail({ id }: { id: string }) {
  const t = useTranslations("platform");
  const { data: me } = usePlatformMe();
  const hasAccess = hasPlatformAccess(me);
  const { data: definition, isLoading, isError, refetch } = useProviderDefinition(id, {
    enabled: hasAccess,
  });
  const versionsQuery = useProviderVersions(id, { enabled: hasAccess });
  const versions = versionsQuery.data ?? [];

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError || !definition) {
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
    <>
      <PageHeader
        title={definition.display_name}
        description={definition.provider_key}
        meta={
          definition.latest_published_version_id ? (
            <PublicationStateBadge state="available" />
          ) : (
            <Badge variant="outline">{t("detail.noPublishedVersion")}</Badge>
          )
        }
      />
      <div className="space-y-6">
        <OverviewCard definition={definition} />
        <VersionsCard
          versions={versions}
          isLoading={versionsQuery.isLoading}
          isError={versionsQuery.isError}
          refetch={versionsQuery.refetch}
        />
        <VersionConfig providerId={id} versions={versions} me={me} />
      </div>
    </>
  );
}

export default function PlatformProviderDetailPage() {
  const t = useTranslations("platform");
  const params = useParams<{ id: string }>();
  const id = params.id;

  return (
    <>
      <div className="mb-4">
        <Button asChild variant="ghost" size="sm">
          <Link href="/platform/providers">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("detail.backToRegistry")}
          </Link>
        </Button>
      </div>
      <PlatformAccessGate>
        {id ? (
          <ProviderDetail id={id} />
        ) : (
          <EmptyState
            icon={Boxes}
            title={t("detail.notFoundTitle")}
            description={t("detail.notFoundDescription")}
          />
        )}
      </PlatformAccessGate>
    </>
  );
}
