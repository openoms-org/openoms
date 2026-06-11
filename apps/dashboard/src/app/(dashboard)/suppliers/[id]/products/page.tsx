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
  Eye,
  ShoppingBag,
  Loader2,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useSupplier,
  useSupplierProducts,
  useImportSupplierProducts,
  useImportSingleProduct,
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
import { SupplierProductLinkDialog } from "@/components/suppliers/supplier-product-link-dialog";
import type { SupplierProduct } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_LIMIT = 50;

export default function SupplierProductsPage() {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;

  const { data: supplier, isLoading: supplierLoading } = useSupplier(id);
  const importProducts = useImportSupplierProducts(id);
  const importSingle = useImportSingleProduct(id);
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
  const [linkTarget, setLinkTarget] = useState<SupplierProduct | null>(null);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);
  const [listingLoadingId, setListingLoadingId] = useState<string | null>(null);
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
          if (result.imported > 0) parts.push(`${t("imported")}: ${result.imported}`);
          if (result.skipped > 0) parts.push(`${t("skipped")}: ${result.skipped}`);
          if (result.errors?.length) parts.push(`${t("errors")}: ${result.errors.length}`);
          toast.success(parts.join(", ") || t("importZakonczony"));
          setSelectedIds(new Set());
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  }, [selectedIds, importProducts, t]);

  const handleBulkDelete = useCallback(() => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    bulkDelete.mutate(
      { supplier_product_ids: ids },
      {
        onSuccess: (result) => {
          toast.success(t("deletedProducts", { count: result.deleted }));
          setSelectedIds(new Set());
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  }, [selectedIds, bulkDelete, t]);

  const handleUnlink = useCallback(
    (spId: string) => {
      unlinkProduct.mutate(spId, {
        onSuccess: () => toast.success(t("productUnlinked")),
        onError: (error) => toast.error(getErrorMessage(error)),
      });
    },
    [unlinkProduct, t]
  );

  const handleDeleteSingle = useCallback(
    (spId: string) => {
      deleteProduct.mutate(spId, {
        onSuccess: () => toast.success(t("supplierProductDeleted")),
        onError: (error) => toast.error(getErrorMessage(error)),
      });
    },
    [deleteProduct, t]
  );

  const handleListOnMarketplace = useCallback(
    (sp: SupplierProduct) => {
      if (sp.product_id) {
        router.push(`/products/${sp.product_id}/listings?listing=new`);
        return;
      }
      setListingLoadingId(sp.id);
      importSingle.mutate(sp.id, {
        onSuccess: (product) => {
          setListingLoadingId(null);
          setDetailProduct(null);
          router.push(`/products/${product.id}/listings?listing=new`);
        },
        onError: (error) => {
          setListingLoadingId(null);
          toast.error(getErrorMessage(error));
        },
      });
    },
    [importSingle, router]
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
        header: t("product"),
        accessorKey: "name",
        cell: (row) => (
          <div className="min-w-0">
            <button
              className="font-medium truncate block text-left hover:underline cursor-pointer max-w-full"
              onClick={() => setDetailProduct(row)}
            >
              {row.name}
            </button>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              {getMetaString(row.metadata, "brand") && (
                <span>{getMetaString(row.metadata, "brand")}</span>
              )}
              {row.ean && (
                <span className="font-mono">{row.ean}</span>
              )}
            </div>
          </div>
        ),
      },
      {
        header: t("category"),
        accessorKey: "source_category",
        className: "max-w-[160px]",
        cell: (row) => (
          <span className="text-muted-foreground text-xs truncate block">
            {row.source_category || "---"}
          </span>
        ),
      },
      {
        header: t("netPrice"),
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
        header: t("stock"),
        accessorKey: "stock_quantity",
        className: "text-right",
        cell: (row) => <span className="text-right block">{row.stock_quantity}</span>,
      },
      {
        header: tc("status"),
        accessorKey: "product_id",
        cell: (row) =>
          row.product_id ? (
            <Badge variant="outline" className="gap-1">
              <Link2 className="h-3 w-3" />
              {t("linked")}
            </Badge>
          ) : (
            <Badge variant="secondary">{t("unlinked")}</Badge>
          ),
      },
      {
        header: "",
        accessorKey: "id",
        className: "w-[130px]",
        cell: (row) => (
          <div className="flex items-center gap-1 justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => setDetailProduct(row)}
              title={t("preview")}
            >
              <Eye className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => handleListOnMarketplace(row)}
              disabled={listingLoadingId === row.id}
              title={t("listOnMarketplace")}
            >
              {listingLoadingId === row.id ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <ShoppingBag className="h-3.5 w-3.5" />
              )}
            </Button>
            {row.product_id ? (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => handleUnlink(row.id)}
                title={t("unlinkProduct")}
              >
                <Unlink className="h-3.5 w-3.5" />
              </Button>
            ) : (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => setLinkTarget(row)}
                title={t("linkProduct")}
              >
                <Link2 className="h-3.5 w-3.5" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive hover:text-destructive"
              onClick={() => handleDeleteSingle(row.id)}
              title={tc("delete")}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ),
      },
    ],
    [handleUnlink, handleDeleteSingle, handleListOnMarketplace, listingLoadingId, t, tc]
  );

  if (supplierLoading) return <LoadingSkeleton />;
  if (!supplier) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        {t("notFound")}
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
              {t("supplierProductsTitle", { name: supplier.name })}
            </h1>
            <p className="text-muted-foreground text-sm">
              {t("productsInCatalog", { count: total })}
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
              placeholder={t("searchByNameEanSku")}
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
              <SelectValue placeholder={t("linkStatus")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{tc("all")}</SelectItem>
              <SelectItem value="unlinked">{t("unlinked")}</SelectItem>
              <SelectItem value="linked">{t("linked")}</SelectItem>
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
                <SelectValue placeholder={t("category")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("allCategories")}</SelectItem>
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
                  ? t("importing")
                  : t("addToProducts", { count: selectedUnlinked })}
              </Button>
              <Button
                variant="destructive"
                onClick={handleBulkDelete}
                disabled={bulkDelete.isPending}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                {bulkDelete.isPending
                  ? tc("deleting")
                  : `${tc("delete")} (${selectedIds.size})`}
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
          resizable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
          emptyMessage={t("noProductsRunSync")}
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
                      <span className="text-muted-foreground">{t("price")}:</span>{" "}
                      <span className="font-medium">
                        {detailProduct.price != null ? formatCurrency(detailProduct.price) : "---"}
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">{t("stock")}:</span>{" "}
                      <span className="font-medium">{detailProduct.stock_quantity}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">{t("category")}:</span>{" "}
                      <span className="font-medium">{detailProduct.source_category || "---"}</span>
                    </div>
                    {getMetaString(detailProduct.metadata, "brand") && (
                      <div>
                        <span className="text-muted-foreground">{t("brand")}:</span>{" "}
                        <span className="font-medium">
                          {getMetaString(detailProduct.metadata, "brand")}
                        </span>
                      </div>
                    )}
                    {detailProduct.metadata?.weight != null && (
                      <div>
                        <span className="text-muted-foreground">{t("weight")}:</span>{" "}
                        <span className="font-medium">
                          {String(detailProduct.metadata.weight)} kg
                        </span>
                      </div>
                    )}
                    {detailProduct.metadata?.retail_price != null && (
                      <div>
                        <span className="text-muted-foreground">{t("retailPrice")}:</span>{" "}
                        <span className="font-medium">
                          {formatCurrency(Number(detailProduct.metadata.retail_price))}
                        </span>
                      </div>
                    )}
                    {(detailProduct.metadata?.attributes as Record<string, string>)?.guarantee_months && (
                      <div>
                        <span className="text-muted-foreground">{t("guarantee")}:</span>{" "}
                        <span className="font-medium">
                          {(detailProduct.metadata.attributes as Record<string, string>).guarantee_months} {t("months")}
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
                      <span className="text-muted-foreground">{tc("status")}:</span>{" "}
                      {detailProduct.product_id ? (
                        <Badge variant="outline" className="gap-1">
                          <Link2 className="h-3 w-3" />
                          {t("linked")}
                        </Badge>
                      ) : (
                        <Badge variant="secondary">{t("unlinked")}</Badge>
                      )}
                    </div>
                    <div>
                      <span className="text-muted-foreground">{t("externalId")}:</span>{" "}
                      <span className="font-mono text-xs">{detailProduct.external_id}</span>
                    </div>
                  </div>

                  {/* Description */}
                  {getMetaString(detailProduct.metadata, "description") && (
                    <div>
                      <h4 className="text-sm font-medium mb-1">{tc("description")}</h4>
                      <div className="text-sm text-muted-foreground prose prose-sm max-w-none max-h-[200px] overflow-y-auto rounded border p-3 bg-muted/20 whitespace-pre-wrap">
                        {getMetaString(detailProduct.metadata, "description") || ""}
                      </div>
                    </div>
                  )}

                  {/* Actions */}
                  <div className="flex gap-2 pt-2 border-t">
                    <Button
                      size="sm"
                      onClick={() => handleListOnMarketplace(detailProduct)}
                      disabled={listingLoadingId === detailProduct.id}
                    >
                      {listingLoadingId === detailProduct.id ? (
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      ) : (
                        <ShoppingBag className="h-4 w-4 mr-2" />
                      )}
                      {t("listOnMarketplace")}
                    </Button>
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
                        {t("unlinkFromProduct")}
                      </Button>
                    ) : (
                      <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setLinkTarget(detailProduct);
                          setDetailProduct(null);
                        }}
                      >
                        <Link2 className="h-4 w-4 mr-2" />
                        {t("linkProduct")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          importProducts.mutate(
                            { supplier_product_ids: [detailProduct.id] },
                            {
                              onSuccess: (result) => {
                                if (result.imported > 0) {
                                  toast.success(t("importedToCatalog"));
                                } else {
                                  toast.info(t("productSkippedOrError"));
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
                        {t("importToCatalog")}
                      </Button>
                      </>
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
                      {tc("delete")}
                    </Button>
                  </div>
                </div>
              </>
            )}
          </DialogContent>
        </Dialog>

        {/* Link unlinked supplier product to an existing catalog product */}
        <SupplierProductLinkDialog
          key={linkTarget?.id ?? "none"}
          supplierId={id}
          supplierProduct={linkTarget}
          onClose={() => setLinkTarget(null)}
        />
      </div>
    </AdminGuard>
  );
}
