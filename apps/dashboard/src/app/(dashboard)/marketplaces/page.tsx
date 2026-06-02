"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, Store, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useIntegrationsByCategory, useDeleteIntegration } from "@/hooks/use-integrations";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ProviderLogo } from "@/components/shared/provider-logo";
import { StatusBadge } from "@/components/shared/status-badge";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { getVisibleProviderKeys } from "@/lib/readiness";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTranslations } from "next-intl";

const KNOWN_PROVIDERS = ["allegro", "amazon", "olx", "shoper", "prestashop", "shopify"];

export default function MarketplacesPage() {
  const t = useTranslations("marketplaces");
  const router = useRouter();
  const { marketplaces, isLoading, isError, refetch } = useIntegrationsByCategory();
  const deleteIntegration = useDeleteIntegration();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const visibleProviderKeys = new Set(
    getVisibleProviderKeys((marketplaces ?? []).map((integration) => integration.provider)),
  );
  const visibleMarketplaces = marketplaces?.filter((integration) =>
    visibleProviderKeys.has(integration.provider),
  );

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("marketplaceDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleRowClick = (provider: string, id: string) => {
    if (KNOWN_PROVIDERS.includes(provider)) {
      router.push(`/marketplaces/${provider}`);
    } else {
      router.push(`/integrations/${id}`);
    }
  };

  return (
    <AdminGuard>
      <PageHeader
        title="Marketplace"
        description={t("manageSalesPlatformConnections")}
        action={{ label: t("addMarketplace"), href: "/marketplaces/new" }}
      />

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("retry")}
          </Button>
        </div>
      )}

      {!visibleMarketplaces || visibleMarketplaces.length === 0 ? (
        <EmptyState
          icon={Store}
          title={t("noMarketplaces")}
          description={t("dodajPierwszaPlatformeSprzedazowaAbySynchronizowac")}
          action={{ label: t("addMarketplace"), href: "/marketplaces/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("platform")}</TableHead>
                <TableHead>{t("label")}</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>{t("authCredentials")}</TableHead>
                <TableHead>{t("lastSync")}</TableHead>
                <TableHead>{t("createdAt")}</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleMarketplaces.map((integration) => (
                <TableRow
                  key={integration.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => handleRowClick(integration.provider, integration.id)}
                >
                  <TableCell className="font-medium">
                    <ProviderLogo
                      providerKey={integration.provider}
                      category="marketplace"
                      size="sm"
                    />
                  </TableCell>
                  <TableCell>
                    {integration.label || "---"}
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={integration.status}
                      statusMap={INTEGRATION_STATUSES}
                    />
                  </TableCell>
                  <TableCell>
                    {integration.has_credentials ? (
                      <Badge variant="success" className="gap-1">
                        <KeyRound className="h-3 w-3" />
                        {t("configured")}
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="text-muted-foreground">
                        {t("none")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {integration.last_sync_at
                      ? formatDate(integration.last_sync_at)
                      : "---"}
                  </TableCell>
                  <TableCell>{formatDate(integration.created_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(integration.id);
                      }}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("deleteMarketplace")}
        description={t("deleteMarketplaceConfirmation")}
        confirmLabel={t("deleteAction")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteIntegration.isPending}
      />
    </AdminGuard>
  );
}
