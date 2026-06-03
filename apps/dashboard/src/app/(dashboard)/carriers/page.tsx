"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { KeyRound, Truck, Trash2 } from "lucide-react";
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

export default function CarriersPage() {
  const router = useRouter();
  const t = useTranslations("carriers");
  const { carriers, isLoading, isError, refetch } = useIntegrationsByCategory();
  const deleteIntegration = useDeleteIntegration();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const visibleProviderKeys = new Set(
    getVisibleProviderKeys((carriers ?? []).map((integration) => integration.provider)),
  );
  const visibleCarriers = carriers?.filter((integration) =>
    visibleProviderKeys.has(integration.provider),
  );

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("carrierDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <PageHeader
        title="Kurierzy"
        description="Zarządzaj połączeniami z firmami kurierskimi"
        action={{ label: "Dodaj kuriera", href: "/carriers/new" }}
      />

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {!visibleCarriers || visibleCarriers.length === 0 ? (
        <EmptyState
          icon={Truck}
          title="Brak kurierów"
          description="Dodaj pierwszego kuriera, aby zacząć generować etykiety i śledzić przesyłki."
          action={{ label: "Dodaj kuriera", href: "/carriers/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kurier</TableHead>
                <TableHead>Etykieta</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Dane uwierzytelniające</TableHead>
                <TableHead>Ostatnia synchronizacja</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {visibleCarriers.map((integration) => (
                <TableRow
                  key={integration.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/carriers/${integration.id}`)}
                >
                  <TableCell className="font-medium">
                    <ProviderLogo
                      providerKey={integration.provider}
                      category="carrier"
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
                        Skonfigurowane
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="text-muted-foreground">
                        Brak
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
        title="Usuń kuriera"
        description="Czy na pewno chcesz usunąć tego kuriera? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteIntegration.isPending}
      />
    </AdminGuard>
  );
}
