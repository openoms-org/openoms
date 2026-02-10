"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Check, X, Plug, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useIntegrations, useDeleteIntegration } from "@/hooks/use-integrations";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { StatusBadge } from "@/components/shared/status-badge";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
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
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: integrations, isLoading } = useIntegrations();
  const deleteIntegration = useDeleteIntegration();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Integracja zostala usunieta");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : "Błąd usuwania integracji");
      },
    });
  };

  return (
    <>
      <PageHeader
        title="Integracje"
        description="Zarządzaj polaczeniami z zewnetrznymi serwisami"
        action={{ label: "Nowa integracja", href: "/integrations/new" }}
      />

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
                <TableHead>Dane uwierzytelniajace</TableHead>
                <TableHead>Ostatnia synchronizacja</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {integrations.map((integration) => (
                <TableRow
                  key={integration.id}
                  className="cursor-pointer"
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
                      <Check className="h-4 w-4 text-green-600" />
                    ) : (
                      <X className="h-4 w-4 text-muted-foreground" />
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
                      onClick={() => setDeleteId(integration.id)}
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
    </>
  );
}
