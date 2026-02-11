"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, Plug, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useIntegrations, useDeleteIntegration } from "@/hooks/use-integrations";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { StatusBadge } from "@/components/shared/status-badge";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
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

export default function IntegrationsPage() {
  const router = useRouter();
  const { data: integrations, isLoading, isError, refetch } = useIntegrations();
  const deleteIntegration = useDeleteIntegration();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Integracja została usunięta");
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
        title="Integracje"
        description="Zarządzaj połączeniami z zewnętrznymi serwisami"
        action={{ label: "Nowa integracja", href: "/integrations/new" }}
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

      {!integrations || integrations.length === 0 ? (
        <EmptyState
          icon={Plug}
          title="Brak integracji"
          description="Dodaj pierwszą integrację, aby połączyć się z zewnętrznymi serwisami."
          action={{ label: "Nowa integracja", href: "/integrations/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Dostawca</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Dane uwierzytelniające</TableHead>
                <TableHead>Ostatnia synchronizacja</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {integrations.map((integration) => (
                <TableRow
                  key={integration.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/integrations/${integration.id}`)}
                >
                  <TableCell className="font-medium">
                    {integration.provider.charAt(0).toUpperCase() +
                      integration.provider.slice(1)}
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={integration.status}
                      statusMap={INTEGRATION_STATUSES}
                    />
                  </TableCell>
                  <TableCell>
                    {integration.has_credentials ? (
                      <Badge variant="outline" className="gap-1 border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-300">
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
        title="Usuń integrację"
        description="Czy na pewno chcesz usunąć tę integrację? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteIntegration.isPending}
      />
    </AdminGuard>
  );
}
