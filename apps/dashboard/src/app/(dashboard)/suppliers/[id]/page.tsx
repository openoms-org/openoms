"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { RefreshCw, ArrowLeft, Copy, ShieldOff, ExternalLink, Trash2, Check, AlertCircle } from "lucide-react";
import type { ProductCategory, SupplierCategoryMapping } from "@/types/api";
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
} from "@/hooks/use-suppliers";
import { useCategoryTree } from "@/hooks/use-categories";
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

  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [feedFormat, setFeedFormat] = useState("iof");
  const [syncInterval, setSyncInterval] = useState("60");
  const [status, setStatus] = useState("active");
  const [defaultCategoryId, setDefaultCategoryId] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (supplier) {
      setName(supplier.name);
      setCode(supplier.code || "");
      setFeedUrl(supplier.feed_url || "");
      setFeedFormat(supplier.feed_format);
      setSyncInterval(String(supplier.sync_interval_minutes ?? 60));
      setStatus(supplier.status);
      setDefaultCategoryId(supplier.default_category_id || undefined);
    }
  }, [supplier]);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!supplier) {
    return <div className="text-center py-8 text-muted-foreground">Dostawca nie znaleziony</div>;
  }

  const handleUpdate = () => {
    updateSupplier.mutate(
      { name, code: code || undefined, feed_url: feedUrl || undefined, feed_format: feedFormat, sync_interval_minutes: parseInt(syncInterval, 10), status, default_category_id: defaultCategoryId },
      {
        onSuccess: () => toast.success("Dostawca zaktualizowany"),
        onError: (error) =>
          toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleSync = () => {
    syncSupplier.mutate(id, {
      onSuccess: () => toast.success("Synchronizacja zakończona"),
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
            Format: {supplier.feed_format.toUpperCase()} | Utworzono: {formatDate(supplier.created_at)}
          </p>
        </div>
        <StatusBadge status={supplier.status} statusMap={SUPPLIER_STATUSES} />
        <Button onClick={handleSync} disabled={syncSupplier.isPending}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Synchronizuj
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
            <CardTitle>Dane dostawcy</CardTitle>
            <CardDescription>Edytuj informacje o dostawcy</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nazwa</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="code">Kod</Label>
              <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="np. ABC123" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feedUrl">URL feeda</Label>
              <Input id="feedUrl" value={feedUrl} onChange={(e) => setFeedUrl(e.target.value)} placeholder="https://example.com/feed.xml" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Format</Label>
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
                <Label>Status</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Aktywny</SelectItem>
                    <SelectItem value="inactive">Nieaktywny</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Interwał synchronizacji</Label>
              <Select value={syncInterval} onValueChange={setSyncInterval}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="5">Co 5 minut</SelectItem>
                  <SelectItem value="15">Co 15 minut</SelectItem>
                  <SelectItem value="30">Co 30 minut</SelectItem>
                  <SelectItem value="60">Co 1 godzinę</SelectItem>
                  <SelectItem value="120">Co 2 godziny</SelectItem>
                  <SelectItem value="360">Co 6 godzin</SelectItem>
                  <SelectItem value="720">Co 12 godzin</SelectItem>
                  <SelectItem value="1440">Raz dziennie</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Domyślna kategoria</Label>
              <CategoryTreePicker
                value={defaultCategoryId}
                onChange={setDefaultCategoryId}
                placeholder="Wybierz domyślną kategorię..."
                className="w-full"
              />
              <p className="text-xs text-muted-foreground">
                Produkty bez mapowania kategorii trafią do tej kategorii
              </p>
            </div>
            <Button onClick={handleUpdate} disabled={updateSupplier.isPending} className="w-full">
              {updateSupplier.isPending ? "Zapisywanie..." : "Zapisz zmiany"}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Informacje</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Ostatnia synchronizacja</span>
              <span>{supplier.last_sync_at ? formatDate(supplier.last_sync_at) : "Nigdy"}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Interwał synca</span>
              <span>{supplier.sync_interval_minutes >= 60 ? `${supplier.sync_interval_minutes / 60}h` : `${supplier.sync_interval_minutes} min`}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Produkty dostawcy</span>
              <span>{productsData?.total ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Powiązane z OMS</span>
              <span>{products.filter((p) => p.product_id).length}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Utworzono</span>
              <span>{formatDate(supplier.created_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Zaktualizowano</span>
              <span>{formatDate(supplier.updated_at)}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Portal dostawcy</CardTitle>
          <CardDescription>
            Zewnetrzny portal, przez ktory dostawca moze potwierdzac zamowienia, oznaczac wysylke i komunikowac sie
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">
                Status: {supplier.portal_enabled ? (
                  <Badge variant="default" className="ml-2">Aktywny</Badge>
                ) : (
                  <Badge variant="secondary" className="ml-2">Nieaktywny</Badge>
                )}
              </p>
              {portalStatus?.last_used_at && (
                <p className="text-xs text-muted-foreground">
                  Ostatni dostep: {formatDate(portalStatus.last_used_at)}
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                onClick={() => {
                  generateLink.mutate(undefined, {
                    onSuccess: (data) => {
                      setPortalLink(data.url);
                      toast.success("Link wygenerowany");
                    },
                    onError: (error) => toast.error(getErrorMessage(error)),
                  });
                }}
                disabled={generateLink.isPending}
                size="sm"
              >
                <ExternalLink className="h-4 w-4 mr-2" />
                {generateLink.isPending ? "Generowanie..." : "Generuj link"}
              </Button>
              {supplier.portal_enabled && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => {
                    revokeAccess.mutate(undefined, {
                      onSuccess: () => {
                        setPortalLink(null);
                        toast.success("Dostep odwolany");
                      },
                      onError: (error) => toast.error(getErrorMessage(error)),
                    });
                  }}
                  disabled={revokeAccess.isPending}
                >
                  <ShieldOff className="h-4 w-4 mr-2" />
                  {revokeAccess.isPending ? "Odwolywanie..." : "Odwolaj dostep"}
                </Button>
              )}
            </div>
          </div>
          {portalLink && (
            <div className="rounded-md border bg-muted/50 p-3">
              <p className="text-xs text-muted-foreground mb-1">Link portalu (wazny 30 dni):</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-background rounded px-2 py-1 overflow-auto">
                  {portalLink}
                </code>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(portalLink);
                    toast.success("Skopiowano do schowka");
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
          <CardTitle>Mapowanie kategorii</CardTitle>
          <CardDescription>
            Powiązania między kategoriami z feeda dostawcy a kategoriami OMS
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!categoryMappings || categoryMappings.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              Brak mapowań kategorii. Uruchom synchronizację — mapowania zostaną utworzone automatycznie.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kategoria źródłowa</TableHead>
                  <TableHead>Kategoria OMS</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-[100px]">Akcje</TableHead>
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
                        ) : mapping.auto_matched ? (
                          <Badge variant="secondary" className="gap-1">
                            <AlertCircle className="h-3 w-3" />
                            Auto-dopasowanie
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
                                    source_category: mapping.source_category,
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
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Produkty dostawcy ({productsData?.total ?? 0})</CardTitle>
            <CardDescription>Produkty zaimportowane z feeda dostawcy</CardDescription>
          </div>
          <Button
            variant="outline"
            onClick={() => router.push(`/suppliers/${id}/products`)}
          >
            Zarządzaj produktami
          </Button>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold">{productsData?.total ?? 0}</p>
              <p className="text-xs text-muted-foreground">Łącznie</p>
            </div>
            <div>
              <p className="text-2xl font-bold">{products.filter((p) => p.product_id).length}</p>
              <p className="text-xs text-muted-foreground">Powiązane</p>
            </div>
            <div>
              <p className="text-2xl font-bold">{products.filter((p) => !p.product_id).length}</p>
              <p className="text-xs text-muted-foreground">Niepowiązane</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
    </AdminGuard>
  );
}
