"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useIntegration,
  useUpdateIntegration,
  useDeleteIntegration,
} from "@/hooks/use-integrations";
import { IntegrationForm } from "@/components/integrations/integration-form";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  INTEGRATION_STATUSES,
  INTEGRATION_PROVIDER_LABELS,
} from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
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
import { Skeleton } from "@/components/ui/skeleton";

export default function IntegrationDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { data: integration, isLoading } = useIntegration(params.id);
  const updateIntegration = useUpdateIntegration(params.id);
  const deleteIntegration = useDeleteIntegration();

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!integration) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Nie znaleziono integracji</h1>
        <Button asChild variant="outline">
          <Link href="/integrations">Wróć do listy</Link>
        </Button>
      </div>
    );
  }

  const providerLabel =
    INTEGRATION_PROVIDER_LABELS[integration.provider] ??
    integration.provider.charAt(0).toUpperCase() + integration.provider.slice(1);

  const handleStatusChange = (newStatus: string) => {
    updateIntegration.mutate(
      { status: newStatus as "active" | "inactive" | "error" },
      {
        onSuccess: () => {
          toast.success("Status integracji został zmieniony");
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Błąd podczas zmiany statusu"
          );
        },
      }
    );
  };

  const handleCredentialsUpdate = (data: {
    credentials: Record<string, unknown>;
    settings?: Record<string, unknown>;
  }) => {
    const payload: Record<string, unknown> = {};

    // Only send credentials if user actually filled in any field
    if (Object.keys(data.credentials).length > 0) {
      payload.credentials = data.credentials;
    }
    if (data.settings) {
      payload.settings = data.settings;
    }

    if (Object.keys(payload).length === 0) {
      toast.info("Nie wprowadzono żadnych zmian");
      return;
    }

    updateIntegration.mutate(payload, {
      onSuccess: () => {
        toast.success("Dane integracji zostały zaktualizowane");
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Błąd podczas aktualizacji danych"
        );
      },
    });
  };

  const handleDelete = () => {
    deleteIntegration.mutate(params.id, {
      onSuccess: () => {
        toast.success("Integracja została usunięta");
        router.push("/integrations");
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Błąd podczas usuwania integracji"
        );
      },
    });
  };

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{providerLabel}</h1>
            <p className="text-muted-foreground">
              Utworzona {formatDate(integration.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            Usuń
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Szczegóły</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">Dostawca</p>
                <p className="mt-1 font-medium">{providerLabel}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <div className="mt-1">
                  <StatusBadge
                    status={integration.status}
                    statusMap={INTEGRATION_STATUSES}
                  />
                </div>
              </div>
              {integration.label && (
                <div>
                  <p className="text-sm text-muted-foreground">Etykieta</p>
                  <p className="mt-1 font-medium">{integration.label}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">
                  Dane uwierzytelniające
                </p>
                <p className="mt-1 font-medium">
                  {integration.has_credentials ? "Skonfigurowane" : "Brak"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Ostatnia synchronizacja
                </p>
                <p className="mt-1 font-medium">
                  {integration.last_sync_at
                    ? formatDate(integration.last_sync_at)
                    : "---"}
                </p>
              </div>
              {integration.sync_cursor && (
                <div>
                  <p className="text-sm text-muted-foreground">Kursor synchronizacji</p>
                  <p className="mt-1 font-mono text-xs truncate">
                    {integration.sync_cursor}
                  </p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="mt-1 font-mono text-sm">{integration.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Ostatnia aktualizacja
                </p>
                <p className="mt-1 font-medium">
                  {formatDate(integration.updated_at)}
                </p>
              </div>
            </div>

            {integration.status === "error" && integration.error_message && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
                <p className="text-sm font-medium text-destructive">Błąd integracji</p>
                <p className="mt-1 text-sm text-destructive/80">
                  {integration.error_message}
                </p>
              </div>
            )}

            {integration.provider === "allegro" && (
              <div className="pt-2">
                <Button variant="outline" size="sm" asChild>
                  <Link href="/integrations/allegro">
                    <ExternalLink className="mr-2 h-4 w-4" />
                    Zarządzaj połączeniem Allegro
                  </Link>
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Zmień status</CardTitle>
            </CardHeader>
            <CardContent>
              <Select
                value={integration.status}
                onValueChange={handleStatusChange}
                disabled={updateIntegration.isPending}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Aktywna</SelectItem>
                  <SelectItem value="inactive">Nieaktywna</SelectItem>
                </SelectContent>
              </Select>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Aktualizuj dane integracji</CardTitle>
            </CardHeader>
            <CardContent>
              <IntegrationForm
                editProvider={integration.provider}
                existingSettings={integration.settings as Record<string, unknown> | undefined}
                isLoading={updateIntegration.isPending}
                onSubmit={handleCredentialsUpdate}
              />
            </CardContent>
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Usuń integrację"
        description="Czy na pewno chcesz usunąć tę integrację? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteIntegration.isPending}
      />
    </div>
    </AdminGuard>
  );
}
