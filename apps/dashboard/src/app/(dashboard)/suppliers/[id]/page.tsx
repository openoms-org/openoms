"use client";

import { useState, useCallback, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { RefreshCw, ArrowLeft, Copy, ShieldOff, ExternalLink, Trash2, Check, AlertCircle, Search, Save, ChevronDown } from "lucide-react";
import type { ProductCategory, SupplierCategoryMapping, AllegroParameterMapping } from "@/types/api";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useSupplier,
  useUpdateSupplier,
  useSyncSupplier,
  useSupplierProducts,
  useSupplierPortalStatus,
  useGeneratePortalLink,
  useRevokePortalAccess,
  useSupplierCategoryMappings,
  useUpsertCategoryMapping,
  useDeleteCategoryMapping,
  useAllegroParameterMappings,
  useBulkUpsertAllegroMappings,
  useDeleteAllegroParameterMapping,
  useSupplierAttributes,
  useAllegroMappingCategories,
} from "@/hooks/use-suppliers";
import { useCategoryTree } from "@/hooks/use-categories";
import { useAllegroCategorySearch, useAllegroCategoryParams } from "@/hooks/use-allegro";
import type { AllegroCategoryParameter } from "@/hooks/use-allegro";
import { CategoryTreePicker } from "@/components/shared/category-tree-picker";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { SUPPLIER_STATUSES } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { useTranslations } from "next-intl";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";

function findCategoryById(categories: ProductCategory[], id: string): ProductCategory | undefined {
  for (const cat of categories) {
    if (cat.id === id) return cat;
    if (cat.children?.length) {
      const found = findCategoryById(cat.children, id);
      if (found) return found;
    }
  }
  return undefined;
}

export default function SupplierDetailPage() {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const { data: supplier, isLoading } = useSupplier(id);
  const updateSupplier = useUpdateSupplier(id);
  const syncSupplier = useSyncSupplier();
  const { data: productsData } = useSupplierProducts(id);

  const { data: portalStatus } = useSupplierPortalStatus(id);
  const generateLink = useGeneratePortalLink(id);
  const revokeAccess = useRevokePortalAccess(id);
  const [portalLink, setPortalLink] = useState<string | null>(null);

  const { data: categoryMappings } = useSupplierCategoryMappings(id);
  const upsertMapping = useUpsertCategoryMapping(id);
  const deleteMapping = useDeleteCategoryMapping(id);
  const { data: categoryTree } = useCategoryTree();

  const supplierKey = supplier?.id ?? null;
  const [name, setName] = useEffectSyncedState(supplier?.name ?? "", supplierKey);
  const [code, setCode] = useEffectSyncedState(supplier?.code || "", supplierKey);
  const [feedUrl, setFeedUrl] = useEffectSyncedState(supplier?.feed_url || "", supplierKey);
  const [feedFormat, setFeedFormat] = useEffectSyncedState(
    supplier?.feed_format ?? "iof",
    supplierKey,
  );
  const [syncInterval, setSyncInterval] = useEffectSyncedState(
    String(supplier?.sync_interval_minutes ?? 60),
    supplierKey,
  );
  const [status, setStatus] = useEffectSyncedState(
    supplier?.status ?? "active",
    supplierKey,
  );
  const [defaultCategoryId, setDefaultCategoryId] = useEffectSyncedState<
    string | undefined
  >(supplier?.default_category_id || undefined, supplierKey);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!supplier) {
    return <div className="text-center py-8 text-muted-foreground">{t("notFound")}</div>;
  }

  const handleUpdate = () => {
    updateSupplier.mutate(
      { name, code: code || undefined, feed_url: feedUrl || undefined, feed_format: feedFormat, sync_interval_minutes: parseInt(syncInterval, 10), status, default_category_id: defaultCategoryId },
      {
        onSuccess: () => toast.success(t("supplierUpdated")),
        onError: (error) =>
          toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleSync = () => {
    syncSupplier.mutate(id, {
      onSuccess: () => toast.success(t("synchronizacjaZakonczona")),
      onError: (error) =>
        toast.error(getErrorMessage(error)),
    });
  };

  const products = productsData?.items ?? [];

  return (
    <AdminGuard>
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/suppliers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{supplier.name}</h1>
          <p className="text-muted-foreground">
            {t("format")}: {supplier.feed_format.toUpperCase()} | {tc("createdAt")}: {formatDate(supplier.created_at)}
          </p>
        </div>
        <StatusBadge status={supplier.status} statusMap={SUPPLIER_STATUSES} />
        <Button onClick={handleSync} disabled={syncSupplier.isPending}>
          <RefreshCw className="h-4 w-4 mr-2" />
          {t("synchronize")}
        </Button>
      </div>

      {supplier.error_message && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {supplier.error_message}
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("supplierData")}</CardTitle>
            <CardDescription>{t("editSupplierInfo")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{tc("name")}</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="code">{tc("code")}</Label>
              <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="np. ABC123" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feedUrl">{t("feedUrl")}</Label>
              <Input id="feedUrl" value={feedUrl} onChange={(e) => setFeedUrl(e.target.value)} placeholder="https://example.com/feed.xml" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("format")}</Label>
                <Select value={feedFormat} onValueChange={setFeedFormat}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="iof">IOF</SelectItem>
                    <SelectItem value="csv">CSV</SelectItem>
                    <SelectItem value="xml">XML</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{tc("status")}</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">{tc("active")}</SelectItem>
                    <SelectItem value="inactive">{tc("inactive")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("syncIntervalLabel")}</Label>
              <Select value={syncInterval} onValueChange={setSyncInterval}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="5">{t("every5Min")}</SelectItem>
                  <SelectItem value="15">{t("every15Min")}</SelectItem>
                  <SelectItem value="30">{t("every30Min")}</SelectItem>
                  <SelectItem value="60">{t("every1Hour")}</SelectItem>
                  <SelectItem value="120">{t("every2Hours")}</SelectItem>
                  <SelectItem value="360">{t("every6Hours")}</SelectItem>
                  <SelectItem value="720">{t("every12Hours")}</SelectItem>
                  <SelectItem value="1440">{t("onceDaily")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("domyslnaKategoria")}</Label>
              <CategoryTreePicker
                value={defaultCategoryId}
                onChange={setDefaultCategoryId}
                placeholder={t("wybierzDomyslnaKategorie")}
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                {t("produktyBezMapowaniaKategoriiTrafiaDoTej")}
              </p>
            </div>
            <Button onClick={handleUpdate} disabled={updateSupplier.isPending} className="w-full">
              {updateSupplier.isPending ? tc("saving") : tc("saveChanges")}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{tc("details")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("lastSync")}</span>
              <span>{supplier.last_sync_at ? formatDate(supplier.last_sync_at) : t("never")}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("syncIntervalShort")}</span>
              <span>{supplier.sync_interval_minutes >= 60 ? `${supplier.sync_interval_minutes / 60}h` : `${supplier.sync_interval_minutes} min`}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("supplierProducts")}</span>
              <span>{productsData?.total ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("powiazaneZOms")}</span>
              <span>{products.filter((p) => p.product_id).length}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{tc("createdAt")}</span>
              <span>{formatDate(supplier.created_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{tc("updatedAt")}</span>
              <span>{formatDate(supplier.updated_at)}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("supplierPortal")}</CardTitle>
          <CardDescription>
            {t("supplierPortalDescription")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">
                {tc("status")}: {supplier.portal_enabled ? (
                  <Badge variant="default" className="ml-2">{tc("active")}</Badge>
                ) : (
                  <Badge variant="secondary" className="ml-2">{tc("inactive")}</Badge>
                )}
              </p>
              {portalStatus?.last_used_at && (
                <p className="text-xs text-muted-foreground">
                  {t("lastAccess")}: {formatDate(portalStatus.last_used_at)}
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                onClick={() => {
                  generateLink.mutate(undefined, {
                    onSuccess: (data) => {
                      setPortalLink(data.url);
                      toast.success(t("linkGenerated"));
                    },
                    onError: (error) => toast.error(getErrorMessage(error)),
                  });
                }}
                disabled={generateLink.isPending}
                size="sm"
              >
                <ExternalLink className="h-4 w-4 mr-2" />
                {generateLink.isPending ? t("generating") : t("generateLink")}
              </Button>
              {supplier.portal_enabled && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => {
                    revokeAccess.mutate(undefined, {
                      onSuccess: () => {
                        setPortalLink(null);
                        toast.success(t("accessRevoked"));
                      },
                      onError: (error) => toast.error(getErrorMessage(error)),
                    });
                  }}
                  disabled={revokeAccess.isPending}
                >
                  <ShieldOff className="h-4 w-4 mr-2" />
                  {revokeAccess.isPending ? t("revoking") : t("revokeAccess")}
                </Button>
              )}
            </div>
          </div>
          {portalLink && (
            <div className="rounded-md border bg-muted/50 p-3">
              <p className="text-xs text-muted-foreground mb-1">{t("portalLinkValid30Days")}</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-background rounded px-2 py-1 overflow-auto">
                  {portalLink}
                </code>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(portalLink);
                    toast.success(tc("copied"));
                  }}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Category Mappings */}
      <Card>
        <CardHeader>
          <CardTitle>{t("categoryMapping")}</CardTitle>
          <CardDescription>
            {t("powiazaniaMiedzyKategoriamiZFeedaDostawcyA")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!categoryMappings || categoryMappings.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              {t("brakMapowanKategoriiUruchomSynchronizacjeMapowania")}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("sourceCategory")}</TableHead>
                  <TableHead>{t("omsCategory")}</TableHead>
                  <TableHead>{tc("status")}</TableHead>
                  <TableHead className="w-[100px]">{tc("actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categoryMappings.map((mapping) => {
                  const targetCat = mapping.category_id
                    ? findCategoryById(categoryTree ?? [], mapping.category_id)
                    : null;
                  return (
                    <TableRow key={mapping.id}>
                      <TableCell className="font-medium">
                        {mapping.source_category}
                      </TableCell>
                      <TableCell>
                        <CategoryTreePicker
                          value={mapping.category_id}
                          onChange={(value) => {
                            upsertMapping.mutate(
                              {
                                source_category: mapping.source_category,
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
                        ) : mapping.auto_matched ? (
                          <Badge variant="secondary" className="gap-1">
                            <AlertCircle className="h-3 w-3" />
                            {t("autoMatched")}
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
                              title={tc("confirm")}
                              onClick={() => {
                                upsertMapping.mutate(
                                  {
                                    source_category: mapping.source_category,
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
                            title={tc("delete")}
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
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Allegro Parameter Mappings */}
      <AllegroParameterMappingSection supplierId={id} />

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>{t("supplierProducts")} ({productsData?.total ?? 0})</CardTitle>
            <CardDescription>{t("productsImportedFromFeed")}</CardDescription>
          </div>
          <Button
            variant="outline"
            onClick={() => router.push(`/suppliers/${id}/products`)}
          >
            {t("zarzadzajProduktami")}
          </Button>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold">{productsData?.total ?? 0}</p>
              <p className="text-xs text-muted-foreground">{t("total1")}</p>
            </div>
            <div>
              <p className="text-2xl font-bold">{products.filter((p) => p.product_id).length}</p>
              <p className="text-xs text-muted-foreground">{t("powiazane")}</p>
            </div>
            <div>
              <p className="text-2xl font-bold">{products.filter((p) => !p.product_id).length}</p>
              <p className="text-xs text-muted-foreground">{t("niepowiazane")}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
    </AdminGuard>
  );
}

// FIELD_SOURCES moved inside component to use translations

interface MappingRow {
  allegro_param_id: string;
  allegro_param_name: string;
  type: string;
  required: boolean;
  source_type: "attribute" | "field" | "static" | "";
  source_key: string;
  // id of the persisted mapping (if any), enabling single-row deletion.
  mapping_id?: string;
}

function AllegroParameterMappingSection({ supplierId }: { supplierId: string }) {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const FIELD_SOURCES = [
    { value: "ean", label: "EAN" },
    { value: "sku", label: "SKU" },
    { value: "name", label: tc("name") },
    { value: "brand", label: t("brand") },
    { value: "weight", label: t("weight") },
    { value: "price", label: tc("price") },
  ];
  const [categorySearch, setCategorySearch] = useState("");
  const [selectedCategoryId, setSelectedCategoryId] = useState<string | null>(null);
  const [selectedCategoryName, setSelectedCategoryName] = useState("");
  const [expanded, setExpanded] = useState(false);

  const { data: searchResults } = useAllegroCategorySearch(categorySearch);
  const { data: paramsData } = useAllegroCategoryParams(selectedCategoryId);
  const { data: existingMappings } = useAllegroParameterMappings(supplierId, selectedCategoryId);
  const { data: supplierAttributes } = useSupplierAttributes(supplierId);
  const { data: configuredCategories } = useAllegroMappingCategories(supplierId);
  const bulkUpsert = useBulkUpsertAllegroMappings(supplierId);
  const deleteMapping = useDeleteAllegroParameterMapping(supplierId);
  const mappingsReady = !selectedCategoryId || existingMappings !== undefined;

  const sourceMappingRows = useMemo<MappingRow[]>(() => {
    if (!mappingsReady) return [];
    if (!paramsData?.parameters?.length) return [];
    const existingMap = new Map<string, AllegroParameterMapping>();
    if (existingMappings) {
      for (const m of existingMappings) {
        existingMap.set(m.allegro_param_id, m);
      }
    }

    return paramsData.parameters.map((p) => {
      const existing = existingMap.get(p.id);
      return {
        allegro_param_id: p.id,
        allegro_param_name: p.name,
        type: p.type,
        required: p.required,
        source_type: existing?.source_type ?? "",
        source_key: existing?.source_key ?? "",
        mapping_id: existing?.id,
      };
    });
  }, [paramsData, existingMappings, mappingsReady]);
  const [mappingRows, setMappingRows] = useEffectSyncedState(
    sourceMappingRows,
    JSON.stringify(sourceMappingRows),
  );

  const updateRow = useCallback(
    (paramId: string, field: "source_type" | "source_key", value: string) => {
      setMappingRows((prev) =>
        prev.map((row) => {
          if (row.allegro_param_id !== paramId) return row;
          if (field === "source_type") {
            return { ...row, source_type: value as MappingRow["source_type"], source_key: "" };
          }
          return { ...row, [field]: value };
        }),
      );
    },
    [setMappingRows],
  );

  const handleSave = () => {
    if (!selectedCategoryId) return;
    const configured = mappingRows.filter((r) => r.source_type && r.source_key);
    if (configured.length === 0) {
      toast.error(t("configureAtLeastOneMapping"));
      return;
    }
    bulkUpsert.mutate(
      {
        allegro_category_id: selectedCategoryId,
        mappings: configured.map((r) => ({
          allegro_param_id: r.allegro_param_id,
          allegro_param_name: r.allegro_param_name,
          source_type: r.source_type as "attribute" | "field" | "static",
          source_key: r.source_key,
        })),
      },
      {
        onSuccess: () => toast.success(t("mappingsSaved")),
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  return (
    <Card>
      <CardHeader
        className="cursor-pointer"
        onClick={() => setExpanded((v) => !v)}
      >
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{t("allegroParamMapping")}</CardTitle>
            <CardDescription>
              {t("allegroParamMappingDesc")}
            </CardDescription>
          </div>
          <ChevronDown
            className={`h-5 w-5 text-muted-foreground transition-transform ${expanded ? "rotate-180" : ""}`}
          />
        </div>
      </CardHeader>
      {expanded && (
        <CardContent className="space-y-4">
          {/* Configured categories badges */}
          {configuredCategories && configuredCategories.length > 0 && (
            <div className="flex flex-wrap gap-1">
              <span className="text-xs text-muted-foreground mr-1 self-center">{t("configured")}:</span>
              {configuredCategories.map((catId) => (
                <Badge
                  key={catId}
                  variant={catId === selectedCategoryId ? "default" : "secondary"}
                  className="cursor-pointer text-xs"
                  onClick={() => {
                    setSelectedCategoryId(catId);
                    setSelectedCategoryName(catId);
                  }}
                >
                  {catId}
                </Badge>
              ))}
            </div>
          )}

          {/* Category search */}
          <div className="space-y-2">
            <Label>{t("allegroCategory")}</Label>
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("searchAllegroCategory")}
                value={categorySearch}
                onChange={(e) => setCategorySearch(e.target.value)}
                className="pl-9"
              />
            </div>
            {searchResults?.matchingCategories && searchResults.matchingCategories.length > 0 && categorySearch.length >= 2 && (
              <div className="border rounded-md max-h-48 overflow-y-auto">
                {searchResults.matchingCategories.map((cat) => {
                  const parts: string[] = [];
                  let current: typeof cat | null = cat;
                  while (current) {
                    parts.unshift(current.name);
                    current = current.parent ?? null;
                  }
                  return (
                    <button
                      key={cat.id}
                      type="button"
                      className="w-full text-left px-3 py-2 text-sm hover:bg-accent border-b last:border-b-0"
                      onClick={() => {
                        setSelectedCategoryId(cat.id);
                        setSelectedCategoryName(parts.join(" > "));
                        setCategorySearch("");
                      }}
                    >
                      <span className="text-muted-foreground text-xs">{parts.slice(0, -1).join(" > ")}</span>
                      {parts.length > 1 && <span className="text-muted-foreground text-xs"> &gt; </span>}
                      <span className="font-medium">{parts[parts.length - 1]}</span>
                    </button>
                  );
                })}
              </div>
            )}
            {selectedCategoryId && (
              <p className="text-xs text-muted-foreground">
                {t("selected")}: <span className="font-medium">{selectedCategoryName}</span> ({selectedCategoryId})
              </p>
            )}
          </div>

          {/* Parameter mapping table */}
          {selectedCategoryId && paramsData?.parameters && mappingsReady && (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("allegroParam")}</TableHead>
                    <TableHead className="w-[80px]">{tc("type")}</TableHead>
                    <TableHead className="w-[140px]">{t("source")}</TableHead>
                    <TableHead className="w-[200px]">{t("value")}</TableHead>
                    <TableHead className="w-[40px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {mappingRows.map((row) => (
                    <TableRow key={row.allegro_param_id}>
                      <TableCell>
                        <span className={row.required ? "font-medium" : ""}>
                          {row.allegro_param_name}
                        </span>
                        {row.required && (
                          <span className="text-destructive ml-1">*</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {row.type}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Select
                          value={row.source_type || undefined}
                          onValueChange={(v) => updateRow(row.allegro_param_id, "source_type", v)}
                        >
                          <SelectTrigger className="h-8 text-xs">
                            <SelectValue placeholder="—" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="attribute">{t("attribute")}</SelectItem>
                            <SelectItem value="field">{t("fieldLabel")}</SelectItem>
                            <SelectItem value="static">{t("staticValue")}</SelectItem>
                          </SelectContent>
                        </Select>
                      </TableCell>
                      <TableCell>
                        {row.source_type === "attribute" && (
                          <Select
                            value={row.source_key || undefined}
                            onValueChange={(v) => updateRow(row.allegro_param_id, "source_key", v)}
                          >
                            <SelectTrigger className="h-8 text-xs">
                              <SelectValue placeholder={t("selectAttribute")} />
                            </SelectTrigger>
                            <SelectContent>
                              {(supplierAttributes ?? []).map((attr) => (
                                <SelectItem key={attr} value={attr}>
                                  {attr}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        )}
                        {row.source_type === "field" && (
                          <Select
                            value={row.source_key || undefined}
                            onValueChange={(v) => updateRow(row.allegro_param_id, "source_key", v)}
                          >
                            <SelectTrigger className="h-8 text-xs">
                              <SelectValue placeholder={t("selectField")} />
                            </SelectTrigger>
                            <SelectContent>
                              {FIELD_SOURCES.map((f) => (
                                <SelectItem key={f.value} value={f.value}>
                                  {f.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        )}
                        {row.source_type === "static" && (
                          <Input
                            value={row.source_key}
                            onChange={(e) => updateRow(row.allegro_param_id, "source_key", e.target.value)}
                            placeholder={t("enterValue")}
                            className="h-8 text-xs"
                          />
                        )}
                      </TableCell>
                      <TableCell>
                        {row.mapping_id && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            disabled={deleteMapping.isPending}
                            title={tc("delete")}
                            onClick={() => {
                              const mappingId = row.mapping_id;
                              if (!mappingId) return;
                              deleteMapping.mutate(mappingId, {
                                onSuccess: () => toast.success(t("mappingDeleted")),
                                onError: (error) => toast.error(getErrorMessage(error)),
                              });
                            }}
                          >
                            <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="flex justify-end">
                <Button onClick={handleSave} disabled={bulkUpsert.isPending}>
                  <Save className="h-4 w-4 mr-2" />
                  {bulkUpsert.isPending ? tc("saving") : t("saveMappings")}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      )}
    </Card>
  );
}
