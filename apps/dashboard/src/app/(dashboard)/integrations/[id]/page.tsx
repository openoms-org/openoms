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
import { ProviderLogo } from "@/components/shared/provider-logo";
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
import { useTranslations } from "next-intl";

const DEDICATED_PAGES: Record<string, string> = {
  allegro: "/marketplaces/allegro",
  amazon: "/marketplaces/amazon",
};

export default function IntegrationDetailPage() {
  const t = useTranslations("integrations");
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
        <h1 className="text-2xl font-bold">{t("integrationNotFound")}</h1>
        <Button asChild variant="outline">
          <Link href="/integrations">{t("detail.backToList")}</Link>
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
          toast.success(t("integrationStatusChanged"));
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("bulk.statusChangeError")
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
      toast.info(t("noChangesIntroduced"));
      return;
    }

    updateIntegration.mutate(payload, {
      onSuccess: () => {
        toast.success(t("integrationDataUpdated"));
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("dataUpdateError")
        );
      },
    });
  };

  const handleDelete = () => {
    deleteIntegration.mutate(params.id, {
      onSuccess: () => {
        toast.success(t("integrationDeleted"));
        router.push(backLink);
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("integrationDeleteError")
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
          <ProviderLogo
            providerKey={integration.provider}
            fallbackName={providerLabel}
            size="lg"
          />
          <div>
            <h1 className="text-2xl font-bold">{providerLabel}</h1>
            <p className="text-muted-foreground">
              {t("createdOn", { date: formatDate(integration.created_at) })}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            {t("delete")}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("details")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">{t("provider")}</p>
                <div className="mt-1">
                  <ProviderLogo
                    providerKey={integration.provider}
                    fallbackName={providerLabel}
                    size="sm"
                  />
                </div>
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
                  <p className="text-sm text-muted-foreground">{t("label")}</p>
                  <p className="mt-1 font-medium">{integration.label}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("authCredentials")}
                </p>
                <p className="mt-1 font-medium">
                  {integration.has_credentials ? t("configured") : t("notConfigured")}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("lastSync")}
                </p>
                <p className="mt-1 font-medium">
                  {integration.last_sync_at
                    ? formatDate(integration.last_sync_at)
                    : "---"}
                </p>
              </div>
              {integration.sync_cursor && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("syncCursor")}</p>
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
                  {t("lastUpdated")}
                </p>
                <p className="mt-1 font-medium">
                  {formatDate(integration.updated_at)}
                </p>
              </div>
            </div>

            {integration.status === "error" && integration.error_message && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
                <p className="text-sm font-medium text-destructive">{t("integrationError")}</p>
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
              <CardTitle>{t("bulk.changeStatus")}</CardTitle>
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
                  <SelectItem value="active">{t("statusActive")}</SelectItem>
                  <SelectItem value="inactive">{t("statusInactive")}</SelectItem>
                </SelectContent>
              </Select>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("updateIntegrationData")}</CardTitle>
            </CardHeader>
            <CardContent>
              <IntegrationForm
                editProvider={integration.provider}
                existingSettings={integration.settings as Record<string, unknown> | undefined}
                isPending={updateIntegration.isPending}
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
                  toast.success(t("shipmentSettingsSaved"));
                },
                onError: (error) => {
                  toast.error(
                    error instanceof Error
                      ? error.message
                      : t("shipmentSettingsSaveError")
                  );
                },
              }
            );
          }}
          isPending={updateIntegration.isPending}
        />
      )}

      {isMarketplace && (
        <MarketplaceCategoryMappingSection integrationId={params.id} />
      )}

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteIntegration")}
        description={t("deleteIntegrationConfirm")}
        confirmLabel={t("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteIntegration.isPending}
      />
    </div>
    </AdminGuard>
  );
}

function MarketplaceCategoryMappingSection({ integrationId }: { integrationId: string }) {
  const t = useTranslations("integrations");
  const { data: categoryMappings } = useMarketplaceCategoryMappings(integrationId);
  const upsertMapping = useUpsertMarketplaceCategoryMapping(integrationId);
  const deleteMapping = useDeleteMarketplaceCategoryMapping(integrationId);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("categoryMapping")}</CardTitle>
        <CardDescription>
          {t("categoryMappingDescription")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {!categoryMappings || categoryMappings.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            {t("noCategoryMappings")}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("marketplaceCategory")}</TableHead>
                <TableHead>{t("detail.externalId")}</TableHead>
                <TableHead>{t("omsCategory")}</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[100px]">{t("actions")}</TableHead>
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
                            onSuccess: () => toast.success(t("mappingUpdated")),
                            onError: (error) => toast.error(getErrorMessage(error)),
                          }
                        );
                      }}
                      placeholder={t("assignCategory")}
                      className="w-[220px]"
                    />
                  </TableCell>
                  <TableCell>
                    {mapping.confirmed ? (
                      <Badge variant="default" className="gap-1">
                        <Check className="h-3 w-3" />
                        {t("confirmed")}
                      </Badge>
                    ) : mapping.auto_created ? (
                      <Badge variant="secondary" className="gap-1">
                        <AlertCircle className="h-3 w-3" />
                        {t("autoCreated")}
                      </Badge>
                    ) : (
                      <Badge variant="outline">{t("unassigned")}</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      {!mapping.confirmed && mapping.category_id && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title={t("confirm")}
                          onClick={() => {
                            upsertMapping.mutate(
                              {
                                external_category_id: mapping.external_category_id,
                                external_category_name: mapping.external_category_name,
                                category_id: mapping.category_id,
                                confirmed: true,
                              },
                              {
                                onSuccess: () => toast.success(t("mappingConfirmed")),
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
                        title={t("delete")}
                        onClick={() => {
                          deleteMapping.mutate(mapping.id, {
                            onSuccess: () => toast.success(t("mappingDeleted")),
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
