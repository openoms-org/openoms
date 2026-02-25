"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Check, Trash2, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useIntegration,
  useUpdateIntegration,
  useDeleteIntegration,
} from "@/hooks/use-integrations";
import {
  useMarketplaceCategoryMappings,
  useUpsertMarketplaceCategoryMapping,
  useDeleteMarketplaceCategoryMapping,
} from "@/hooks/use-marketplace-category-mappings";
import { IntegrationForm } from "@/components/integrations/integration-form";
import { CategoryTreePicker } from "@/components/shared/category-tree-picker";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import {
  INTEGRATION_STATUSES,
  INTEGRATION_PROVIDER_LABELS,
  PROVIDER_CATEGORIES,
} from "@/lib/constants";
import { MarketplaceShipmentSettings } from "@/components/integrations/marketplace-shipment-settings";
import { formatDate } from "@/lib/utils";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

const DEDICATED_PAGES: Record<string, string> = {
  allegro: "/marketplaces/allegro",
  amazon: "/marketplaces/amazon",
};

export default function IntegrationDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { data: integration, isLoading } = useIntegration(params.id);
  const updateIntegration = useUpdateIntegration(params.id);
  const deleteIntegration = useDeleteIntegration();

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // Redirect to dedicated page for providers that have one
  if (integration && DEDICATED_PAGES[integration.provider]) {
    router.replace(DEDICATED_PAGES[integration.provider]);
    return null;
  }

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

  const isMarketplace = PROVIDER_CATEGORIES.marketplace.providers.includes(integration.provider);

  // Dynamic back-link based on provider category
  const getBackLink = (provider: string): string => {
    if (PROVIDER_CATEGORIES.carrier.providers.includes(provider)) return "/carriers";
    if (PROVIDER_CATEGORIES.marketplace.providers.includes(provider)) return "/marketplaces";
    if (PROVIDER_CATEGORIES.invoicing.providers.includes(provider)) return "/invoicing";
    return "/integrations";
  };
  const backLink = getBackLink(integration.provider);

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
        router.push(backLink);
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
            <Link href={backLink}>
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

      {isMarketplace && (
        <MarketplaceShipmentSettings
          provider={integration.provider}
          settings={(integration.settings ?? {}) as Record<string, unknown>}
          onSave={(newSettings) => {
            updateIntegration.mutate(
              { settings: newSettings },
              {
                onSuccess: () => {
                  toast.success("Ustawienia przesyłek zostały zapisane");
                },
                onError: (error) => {
                  toast.error(
                    error instanceof Error
                      ? error.message
                      : "Błąd podczas zapisywania ustawień przesyłek"
                  );
                },
              }
            );
          }}
          isLoading={updateIntegration.isPending}
        />
      )}

      {isMarketplace && (
        <MarketplaceCategoryMappingSection integrationId={params.id} />
      )}

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

function MarketplaceCategoryMappingSection({ integrationId }: { integrationId: string }) {
  const { data: categoryMappings } = useMarketplaceCategoryMappings(integrationId);
  const upsertMapping = useUpsertMarketplaceCategoryMapping(integrationId);
  const deleteMapping = useDeleteMarketplaceCategoryMapping(integrationId);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Mapowanie kategorii marketplace</CardTitle>
        <CardDescription>
          Powiązania między kategoriami z marketplace a kategoriami OMS
        </CardDescription>
      </CardHeader>
      <CardContent>
        {!categoryMappings || categoryMappings.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Brak mapowań. Import ofert z marketplace utworzy mapowania automatycznie.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kategoria marketplace</TableHead>
                <TableHead>ID zewnętrzne</TableHead>
                <TableHead>Kategoria OMS</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[100px]">Akcje</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {categoryMappings.map((mapping) => (
                <TableRow key={mapping.id}>
                  <TableCell className="font-medium">
                    {mapping.external_category_name || mapping.external_category_id}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {mapping.external_category_id}
                  </TableCell>
                  <TableCell>
                    <CategoryTreePicker
                      value={mapping.category_id}
                      onChange={(value) => {
                        upsertMapping.mutate(
                          {
                            external_category_id: mapping.external_category_id,
                            external_category_name: mapping.external_category_name,
                            category_id: value,
                            confirmed: true,
                          },
                          {
                            onSuccess: () => toast.success("Mapowanie zaktualizowane"),
                            onError: (error) => toast.error(getErrorMessage(error)),
                          }
                        );
                      }}
                      placeholder="Przypisz kategorię..."
                      className="w-[220px]"
                    />
                  </TableCell>
                  <TableCell>
                    {mapping.confirmed ? (
                      <Badge variant="default" className="gap-1">
                        <Check className="h-3 w-3" />
                        Potwierdzone
                      </Badge>
                    ) : mapping.auto_created ? (
                      <Badge variant="secondary" className="gap-1">
                        <AlertCircle className="h-3 w-3" />
                        Auto-utworzone
                      </Badge>
                    ) : (
                      <Badge variant="outline">Nieprzypisane</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      {!mapping.confirmed && mapping.category_id && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title="Potwierdź"
                          onClick={() => {
                            upsertMapping.mutate(
                              {
                                external_category_id: mapping.external_category_id,
                                external_category_name: mapping.external_category_name,
                                category_id: mapping.category_id,
                                confirmed: true,
                              },
                              {
                                onSuccess: () => toast.success("Mapowanie potwierdzone"),
                                onError: (error) => toast.error(getErrorMessage(error)),
                              }
                            );
                          }}
                        >
                          <Check className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0"
                        title="Usuń"
                        onClick={() => {
                          deleteMapping.mutate(mapping.id, {
                            onSuccess: () => toast.success("Mapowanie usunięte"),
                            onError: (error) => toast.error(getErrorMessage(error)),
                          });
                        }}
                      >
                        <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
