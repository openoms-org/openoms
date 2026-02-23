"use client";

import { useState, useRef, useEffect, useMemo, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  ArrowLeft,
  Search,
  Link2,
  Download,
  Trash2,
  Unlink,
  Image as ImageIcon,
  X,
  Eye,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useSupplier,
  useSupplierProducts,
  useImportSupplierProducts,
  useSupplierSourceCategories,
  useBulkDeleteSupplierProducts,
  useUnlinkSupplierProduct,
  useDeleteSupplierProduct,
} from "@/hooks/use-suppliers";
import { formatCurrency } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { DensityToggle } from "@/components/shared/density-toggle";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import type { SupplierProduct } from "@/types/api";

const DEFAULT_LIMIT = 50;

export default function SupplierProductsPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;

  const { data: supplier, isLoading: supplierLoading } = useSupplier(id);
  const importProducts = useImportSupplierProducts(id);
  const bulkDelete = useBulkDeleteSupplierProducts(id);
  const unlinkProduct = useUnlinkSupplierProduct(id);
  const deleteProduct = useDeleteSupplierProduct(id);
  const { data: sourceCategories } = useSupplierSourceCategories(id);

  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [linkedFilter, setLinkedFilter] = useState<string>("all");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [detailProduct, setDetailProduct] = useState<SupplierProduct | null>(null);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const handleSearchChange = (value: string) => {
    setSearch(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setDebouncedSearch(value);
      setPagination((prev) => ({ ...prev, offset: 0 }));
    }, 300);
  };

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const linked = linkedFilter === "linked" ? true : linkedFilter === "unlinked" ? false : undefined;

  const { data, isLoading } = useSupplierProducts(id, {
    ...pagination,
    search: debouncedSearch || undefined,
    linked,
    category: categoryFilter !== "all" ? categoryFilter : undefined,
  });

  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const handleImport = useCallback(() => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    importProducts.mutate(
      { supplier_product_ids: ids },
      {
        onSuccess: (result) => {
          const parts: string[] = [];
          if (result.imported > 0) parts.push(`Zaimportowano: ${result.imported}`);
          if (result.skipped > 0) parts.push(`Pominięto: ${result.skipped}`);
          if (result.errors?.length) parts.push(`Błędy: ${result.errors.length}`);
          toast.success(parts.join(", ") || "Import zakończony");
          setSelectedIds(new Set());
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  }, [selectedIds, importProducts]);

  const handleBulkDelete = useCallback(() => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    bulkDelete.mutate(
      { supplier_product_ids: ids },
      {
        onSuccess: (result) => {
          toast.success(`Usunięto ${result.deleted} produktów`);
          setSelectedIds(new Set());
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  }, [selectedIds, bulkDelete]);

  const handleUnlink = useCallback(
    (spId: string) => {
      unlinkProduct.mutate(spId, {
        onSuccess: () => toast.success("Odłączono produkt"),
        onError: (error) => toast.error(getErrorMessage(error)),
      });
    },
    [unlinkProduct]
  );

  const handleDeleteSingle = useCallback(
    (spId: string) => {
      deleteProduct.mutate(spId, {
        onSuccess: () => toast.success("Usunięto produkt dostawcy"),
        onError: (error) => toast.error(getErrorMessage(error)),
      });
    },
    [deleteProduct]
  );

  // Extract metadata helpers
  const getMetaString = (meta: Record<string, unknown>, key: string): string | undefined => {
    const v = meta?.[key];
    return typeof v === "string" && v ? v : undefined;
  };

  const selectedUnlinked = useMemo(() => {
    if (selectedIds.size === 0) return 0;
    return items.filter((sp) => selectedIds.has(sp.id) && !sp.product_id).length;
  }, [selectedIds, items]);

  const columns: ColumnDef<SupplierProduct>[] = useMemo(
    () => [
      {
        header: "",
        accessorKey: "metadata",
        className: "w-[40px] px-0",
        cell: (row) => {
          const img = getMetaString(row.metadata, "image_url");
          return img ? (
            <button
              className="w-8 h-8 rounded border overflow-hidden bg-muted/30 flex-shrink-0 cursor-pointer"
              onClick={() => setDetailProduct(row)}
            >
              <img src={img} alt="" className="w-full h-full object-contain" />
            </button>
          ) : (
            <div className="w-8 h-8 rounded border bg-muted/30 flex items-center justify-center">
              <ImageIcon className="h-3.5 w-3.5 text-muted-foreground/50" />
            </div>
          );
        },
      },
      {
        header: "Nazwa",
        accessorKey: "name",
        cell: (row) => (
          <div className="max-w-[260px]">
            <button
              className="font-medium truncate block text-left hover:underline cursor-pointer"
              onClick={() => setDetailProduct(row)}
            >
              {row.name}
            </button>
            {getMetaString(row.metadata, "brand") && (
              <span className="text-xs text-muted-foreground">
                {getMetaString(row.metadata, "brand")}
              </span>
            )}
          </div>
        ),
      },
      {
        header: "EAN",
        accessorKey: "ean",
        cell: (row) => <span className="text-muted-foreground">{row.ean || "---"}</span>,
      },
      {
        header: "SKU",
        accessorKey: "sku",
        cell: (row) => <span className="text-muted-foreground">{row.sku || "---"}</span>,
      },
      {
        header: "Kategoria",
        accessorKey: "source_category",
        cell: (row) => (
          <span className="text-muted-foreground text-xs">
            {row.source_category || "---"}
          </span>
        ),
      },
      {
        header: "Cena netto",
        accessorKey: "price",
        className: "text-right",
        cell: (row) => (
          <div className="text-right">
            <span className="block">
              {row.price != null ? formatCurrency(row.price) : "---"}
            </span>
            {row.metadata?.retail_price != null && (
              <span className="text-xs text-muted-foreground">
                det. {formatCurrency(Number(row.metadata.retail_price))}
              </span>
            )}
          </div>
        ),
      },
      {
        header: "Stan",
        accessorKey: "stock_quantity",
        className: "text-right",
        cell: (row) => <span className="text-right block">{row.stock_quantity}</span>,
      },
      {
        header: "Status",
        accessorKey: "product_id",
        cell: (row) =>
          row.product_id ? (
            <Badge variant="outline" className="gap-1">
              <Link2 className="h-3 w-3" />
              Powiązany
            </Badge>
          ) : (
            <Badge variant="secondary">Niepowiązany</Badge>
          ),
      },
      {
        header: "",
        accessorKey: "id",
        className: "w-[100px]",
        cell: (row) => (
          <div className="flex items-center gap-1 justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => setDetailProduct(row)}
              title="Podgląd"
            >
              <Eye className="h-3.5 w-3.5" />
            </Button>
            {row.product_id && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => handleUnlink(row.id)}
                title="Odłącz"
              >
                <Unlink className="h-3.5 w-3.5" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive hover:text-destructive"
              onClick={() => handleDeleteSingle(row.id)}
              title="Usuń"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ),
      },
    ],
    [handleUnlink, handleDeleteSingle]
  );

  if (supplierLoading) return <LoadingSkeleton />;
  if (!supplier) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Dostawca nie znaleziony
      </div>
    );
  }

  return (
    <AdminGuard>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push(`/suppliers/${id}`)}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold tracking-tight">
              Produkty dostawcy: {supplier.name}
            </h1>
            <p className="text-muted-foreground text-sm">
              {total} produktów w katalogu
            </p>
          </div>
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Szukaj po nazwie, EAN, SKU..."
              className="pl-9"
            />
          </div>
          <Select
            value={linkedFilter}
            onValueChange={(value) => {
              setLinkedFilter(value);
              setPagination((prev) => ({ ...prev, offset: 0 }));
            }}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Status powiązania" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie</SelectItem>
              <SelectItem value="unlinked">Niepowiązane</SelectItem>
              <SelectItem value="linked">Powiązane</SelectItem>
            </SelectContent>
          </Select>
          {sourceCategories && sourceCategories.length > 0 && (
            <Select
              value={categoryFilter}
              onValueChange={(value) => {
                setCategoryFilter(value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
            >
              <SelectTrigger className="w-[220px]">
                <SelectValue placeholder="Kategoria" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Wszystkie kategorie</SelectItem>
                {sourceCategories.map((cat) => (
                  <SelectItem key={cat} value={cat}>
                    {cat}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <DensityToggle />
          {selectedIds.size > 0 && (
            <div className="flex items-center gap-2">
              <Button
                onClick={handleImport}
                disabled={importProducts.isPending || selectedUnlinked === 0}
              >
                <Download className="h-4 w-4 mr-2" />
                {importProducts.isPending
                  ? "Importowanie..."
                  : `Dodaj do produktów (${selectedUnlinked})`}
              </Button>
              <Button
                variant="destructive"
                onClick={handleBulkDelete}
                disabled={bulkDelete.isPending}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                {bulkDelete.isPending
                  ? "Usuwanie..."
                  : `Usuń (${selectedIds.size})`}
              </Button>
            </div>
          )}
        </div>

        {/* Table */}
        <DataTable
          columns={columns}
          data={items}
          isLoading={isLoading}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
          emptyMessage="Brak produktów. Uruchom synchronizację feeda."
        />

        {/* Pagination */}
        {total > 0 && (
          <DataTablePagination
            total={total}
            limit={pagination.limit}
            offset={pagination.offset}
            onPageChange={(offset) => setPagination((prev) => ({ ...prev, offset }))}
            onPageSizeChange={(limit) => setPagination({ limit, offset: 0 })}
          />
        )}

        {/* Detail Modal */}
        <Dialog open={!!detailProduct} onOpenChange={(open) => { if (!open) { setDetailProduct(null); setSelectedImageIndex(0); } }}>
          <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
            {detailProduct && (
              <>
                <DialogHeader>
                  <DialogTitle className="pr-8">{detailProduct.name}</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  {/* Images */}
                  {(() => {
                    const images = Array.isArray(detailProduct.metadata?.images)
                      ? (detailProduct.metadata.images as string[])
                      : getMetaString(detailProduct.metadata, "image_url")
                        ? [getMetaString(detailProduct.metadata, "image_url")!]
                        : [];
                    if (images.length === 0) return null;
                    return (
                      <div className="space-y-2">
                        <div className="rounded-lg border overflow-hidden bg-muted/30">
                          <img
                            src={images[selectedImageIndex] || images[0]}
                            alt={detailProduct.name}
                            className="w-full max-h-[300px] object-contain"
                          />
                        </div>
                        {images.length > 1 && (
                          <div className="flex gap-2 overflow-x-auto pb-1">
                            {images.map((url, i) => (
                              <button
                                key={i}
                                type="button"
                                onClick={() => setSelectedImageIndex(i)}
                                className={`flex-shrink-0 w-16 h-16 rounded-md border overflow-hidden bg-muted/30 ${
                                  (selectedImageIndex ?? 0) === i
                                    ? "ring-2 ring-primary"
                                    : "opacity-70 hover:opacity-100"
                                }`}
                              >
                                <img
                                  src={url}
                                  alt={`${detailProduct.name} ${i + 1}`}
                                  className="w-full h-full object-contain"
                                />
                              </button>
                            ))}
                          </div>
                        )}
                      </div>
                    );
                  })()}

                  {/* Basic Info */}
                  <div className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <span className="text-muted-foreground">EAN:</span>{" "}
                      <span className="font-medium">{detailProduct.ean || "---"}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">SKU:</span>{" "}
                      <span className="font-medium">{detailProduct.sku || "---"}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Cena:</span>{" "}
                      <span className="font-medium">
                        {detailProduct.price != null ? formatCurrency(detailProduct.price) : "---"}
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Stan:</span>{" "}
                      <span className="font-medium">{detailProduct.stock_quantity}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Kategoria:</span>{" "}
                      <span className="font-medium">{detailProduct.source_category || "---"}</span>
                    </div>
                    {getMetaString(detailProduct.metadata, "brand") && (
                      <div>
                        <span className="text-muted-foreground">Marka:</span>{" "}
                        <span className="font-medium">
                          {getMetaString(detailProduct.metadata, "brand")}
                        </span>
                      </div>
                    )}
                    {detailProduct.metadata?.weight != null && (
                      <div>
                        <span className="text-muted-foreground">Waga:</span>{" "}
                        <span className="font-medium">
                          {String(detailProduct.metadata.weight)} kg
                        </span>
                      </div>
                    )}
                    {detailProduct.metadata?.retail_price != null && (
                      <div>
                        <span className="text-muted-foreground">Cena detaliczna:</span>{" "}
                        <span className="font-medium">
                          {formatCurrency(Number(detailProduct.metadata.retail_price))}
                        </span>
                      </div>
                    )}
                    {(detailProduct.metadata?.attributes as Record<string, string>)?.guarantee_months && (
                      <div>
                        <span className="text-muted-foreground">Gwarancja:</span>{" "}
                        <span className="font-medium">
                          {(detailProduct.metadata.attributes as Record<string, string>).guarantee_months} mies.
                        </span>
                      </div>
                    )}
                    {(detailProduct.metadata?.attributes as Record<string, string>)?.tax_rate && (
                      <div>
                        <span className="text-muted-foreground">VAT:</span>{" "}
                        <span className="font-medium">
                          {(detailProduct.metadata.attributes as Record<string, string>).tax_rate}%
                        </span>
                      </div>
                    )}
                    <div>
                      <span className="text-muted-foreground">Status:</span>{" "}
                      {detailProduct.product_id ? (
                        <Badge variant="outline" className="gap-1">
                          <Link2 className="h-3 w-3" />
                          Powiązany
                        </Badge>
                      ) : (
                        <Badge variant="secondary">Niepowiązany</Badge>
                      )}
                    </div>
                    <div>
                      <span className="text-muted-foreground">ID zewnętrzny:</span>{" "}
                      <span className="font-mono text-xs">{detailProduct.external_id}</span>
                    </div>
                  </div>

                  {/* Description */}
                  {getMetaString(detailProduct.metadata, "description") && (
                    <div>
                      <h4 className="text-sm font-medium mb-1">Opis</h4>
                      <div
                        className="text-sm text-muted-foreground prose prose-sm max-w-none max-h-[200px] overflow-y-auto rounded border p-3 bg-muted/20"
                        dangerouslySetInnerHTML={{
                          __html: getMetaString(detailProduct.metadata, "description") || "",
                        }}
                      />
                    </div>
                  )}

                  {/* Actions */}
                  <div className="flex gap-2 pt-2 border-t">
                    {detailProduct.product_id ? (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          handleUnlink(detailProduct.id);
                          setDetailProduct(null);
                        }}
                      >
                        <Unlink className="h-4 w-4 mr-2" />
                        Odłącz od produktu
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        onClick={() => {
                          importProducts.mutate(
                            { supplier_product_ids: [detailProduct.id] },
                            {
                              onSuccess: (result) => {
                                if (result.imported > 0) {
                                  toast.success("Zaimportowano do katalogu");
                                } else {
                                  toast.info("Produkt pominięty lub błąd");
                                }
                                setDetailProduct(null);
                              },
                              onError: (error) => toast.error(getErrorMessage(error)),
                            }
                          );
                        }}
                        disabled={importProducts.isPending}
                      >
                        <Download className="h-4 w-4 mr-2" />
                        Importuj do katalogu
                      </Button>
                    )}
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => {
                        handleDeleteSingle(detailProduct.id);
                        setDetailProduct(null);
                      }}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Usuń
                    </Button>
                  </div>
                </div>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </AdminGuard>
  );
}
