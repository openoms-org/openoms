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
import { StatusBadge } from "@/components/shared/status-badge";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { getProviderDisplayName } from "@/lib/provider-info";
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

const KNOWN_PROVIDERS = ["allegro", "amazon", "shoper", "prestashop", "shopify"];

export default function MarketplacesPage() {
  const router = useRouter();
  const { marketplaces, isLoading, isError, refetch } = useIntegrationsByCategory();
  const deleteIntegration = useDeleteIntegration();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Marketplace zostal usuniety");
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
        description="Zarządzaj połączeniami z platformami sprzedażowymi"
        action={{ label: "Dodaj marketplace", href: "/marketplaces/new" }}
      />

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {!marketplaces || marketplaces.length === 0 ? (
        <EmptyState
          icon={Store}
          title="Brak marketplace"
          description="Dodaj pierwszą platformę sprzedażową, aby synchronizować zamówienia i produkty."
          action={{ label: "Dodaj marketplace", href: "/marketplaces/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Platforma</TableHead>
                <TableHead>Etykieta</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Dane uwierzytelniające</TableHead>
                <TableHead>Ostatnia synchronizacja</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {marketplaces.map((integration) => (
                <TableRow
                  key={integration.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => handleRowClick(integration.provider, integration.id)}
                >
                  <TableCell className="font-medium">
                    {getProviderDisplayName(integration.provider)}
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
        title="Usun marketplace"
        description="Czy na pewno chcesz usunac ten marketplace? Ta operacja jest nieodwracalna."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteIntegration.isPending}
      />
    </AdminGuard>
  );
}
