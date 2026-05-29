"use client";

import { useState, useMemo, useRef, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Package, Plus, Search, Sparkles, Loader2, Download, Upload, ShoppingBag, Eye, Image as ImageIcon, Link2, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useBulkCategorize } from "@/hooks/use-ai";
import { apiFetch } from "@/lib/api-client";
import { downloadBlob } from "@/lib/download";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTable, type EditableColumnConfig } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { EmptyState } from "@/components/shared/empty-state";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DensityToggle } from "@/components/shared/density-toggle";
import { useProducts, useDeleteProduct } from "@/hooks/use-products";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAllSuppliers, useAllSupplierProducts } from "@/hooks/use-suppliers";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/api-client";
import { useCategoryTree } from "@/hooks/use-categories";
import { useRedownloadImages } from "@/hooks/use-image-redownload";
import { CategoryTreePicker } from "@/components/shared/category-tree-picker";
import { ProductListToolbar } from "@/components/products/product-list-toolbar";
import { formatCurrency, formatDate } from "@/lib/utils";
import { ORDER_SOURCE_LABELS } from "@/lib/constants";
import { apiClient } from "@/lib/api-client";
import { useQueryClient } from "@tanstack/react-query";
import type { Product, ProductCategory, SupplierProductWithSupplier } from "@/types/api";
import { isFeatureVisible } from "@/lib/readiness";
import { useTranslations } from "next-intl";

const DEFAULT_LIMIT = 20;

const MARKETPLACE_LABELS: Record<string, string> = {
  allegro: "Allegro",
  woocommerce: "WooCommerce",
  amazon: "Amazon",
  ebay: "eBay",
};

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

export default function ProductsPage() {
  const t = useTranslations("products");
  const tl = useTranslations("products.list");
  const [activeTab, setActiveTab] = useState("my-products");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground">
            {tl("manageCatalog")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <DensityToggle />
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="my-products">{tl("myProducts")}</TabsTrigger>
          <TabsTrigger value="supplier-catalog">{tl("supplierCatalog")}</TabsTrigger>
        </TabsList>

        <TabsContent value="my-products" className="space-y-4 mt-4">
          <MyProductsTab />
        </TabsContent>

        <TabsContent value="supplier-catalog" className="space-y-4 mt-4">
          <SupplierCatalogTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ─── My Products Tab (existing functionality) ───

function MyProductsTab() {
  const t = useTranslations("products.list");
  const tc = useTranslations("common");
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [localTagFilter, setLocalTagFilter] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const tagDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const [categoryIdFilter, setCategoryIdFilter] = useState<string | undefined>(undefined);
  const [supplierFilter, setSupplierFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [marketplaceFilter, setMarketplaceFilter] = useState("");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [selectedProducts, setSelectedProducts] = useState<Set<string>>(new Set());
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showBulkDelete, setShowBulkDelete] = useState(false);
  const [bulkDeleting, setBulkDeleting] = useState(false);
  const bulkCategorize = useBulkCategorize();
  const deleteProduct = useDeleteProduct();
  const redownloadImages = useRedownloadImages();

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setPagination((prev) => ({ ...prev, offset: 0 }));
  };

  const handleTagFilterChange = (value: string) => {
    setLocalTagFilter(value);
    if (tagDebounceRef.current) {
      clearTimeout(tagDebounceRef.current);
    }
    tagDebounceRef.current = setTimeout(() => {
      setTagFilter(value);
      setPagination((prev) => ({ ...prev, offset: 0 }));
    }, 300);
  };

  useEffect(() => {
    return () => {
      if (tagDebounceRef.current) {
        clearTimeout(tagDebounceRef.current);
      }
    };
  }, []);

  // Suppliers is a controlled feature; only fetch when it is visible in the active
  // surface, otherwise the gated /v1/suppliers endpoint returns 404 in client-ready.
  const suppliersVisible = isFeatureVisible("suppliers");

  const { data: categoryTree } = useCategoryTree();
  const { data: suppliersData } = useAllSuppliers({}, { enabled: suppliersVisible });

  const { data, isLoading, isError, refetch } = useProducts({
    ...pagination,
    search: search || undefined,
    name: search || undefined,
    tag: tagFilter || undefined,
    category_id: categoryIdFilter || undefined,
    supplier_id: supplierFilter || undefined,
    source: sourceFilter || undefined,
    marketplace: marketplaceFilter || undefined,
    sort_by: sortBy,
    sort_order: sortOrder,
  });

  const editableColumns = useMemo<EditableColumnConfig<Product>[]>(
    () => [
      {
        accessorKey: "price",
        type: "number",
        onSave: async (row, value) => {
          await apiClient<Product>(`/v1/products/${row.id}`, {
            method: "PATCH",
            body: JSON.stringify({ price: Number(value) }),
          });
          queryClient.invalidateQueries({ queryKey: ["products"] });
        },
      },
      {
        accessorKey: "stock_quantity",
        type: "number",
        onSave: async (row, value) => {
          await apiClient<Product>(`/v1/products/${row.id}`, {
            method: "PATCH",
            body: JSON.stringify({ stock_quantity: Number(value) }),
          });
          queryClient.invalidateQueries({ queryKey: ["products"] });
        },
      },
    ],
    [queryClient]
  );

  const columns = [
    {
      header: "",
      accessorKey: "image_url" as const,
      cell: (product: Product) => (
        product.image_url ? (
          <img
            src={product.image_url}
            alt={product.name}
            className="h-10 w-10 sm:h-12 sm:w-12 rounded-md object-cover"
            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
          />
        ) : (
          <div className="flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-md bg-muted">
            <Package className="h-5 w-5 text-muted-foreground" />
          </div>
        )
      ),
    },
    {
      header: t("name"),
      accessorKey: "name" as const,
      sortable: true,
      cell: (product: Product) => (
        <Link
          href={`/products/${product.id}`}
          className="font-medium text-primary hover:underline"
        >
          {product.name}
        </Link>
      ),
    },
    {
      header: "SKU",
      accessorKey: "sku" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="font-mono text-sm">{product.sku || "-"}</span>
      ),
    },
    {
      header: t("price"),
      accessorKey: "price" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="text-sm">{formatCurrency(product.price)}</span>
      ),
    },
    {
      header: t("stock"),
      accessorKey: "stock_quantity" as const,
      sortable: true,
      cell: (product: Product) => (
        <span
          className={`text-sm font-medium ${
            product.stock_quantity === 0
              ? "text-destructive"
              : product.stock_quantity < 10
                ? "text-warning"
                : ""
          }`}
        >
          {product.stock_quantity}
        </span>
      ),
    },
    {
      header: t("source"),
      accessorKey: "source" as const,
      cell: (product: Product) => (
        <span className="text-sm">
          {product.source === "supplier" && product.supplier_name
            ? product.supplier_name
            : (ORDER_SOURCE_LABELS[product.source] ?? product.source)}
        </span>
      ),
    },
    {
      header: "Marketplace",
      accessorKey: "marketplace_providers" as const,
      cell: (product: Product) => {
        const providers = product.marketplace_providers;
        if (!providers?.length) return <span className="text-xs text-muted-foreground">—</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {providers.map((p) => (
              <Badge key={p} variant="outline" className="text-xs">
                {MARKETPLACE_LABELS[p] ?? p}
              </Badge>
            ))}
          </div>
        );
      },
    },
    {
      header: t("category"),
      accessorKey: "category" as const,
      cell: (product: Product) => {
        const cat = product.category_id ? findCategoryById(categoryTree ?? [], product.category_id) : null;
        if (cat) {
          return (
            <span
              className="rounded-full px-2 py-0.5 text-xs font-medium"
              style={{
                backgroundColor: `${cat.color}20`,
                color: cat.color,
              }}
            >
              {cat.name}
            </span>
          );
        }
        if (product.category) {
          return (
            <span className="rounded-full px-2 py-0.5 text-xs font-medium text-muted-foreground bg-muted">
              {product.category}
            </span>
          );
        }
        return null;
      },
    },
    {
      header: t("tags"),
      accessorKey: "tags" as const,
      cell: (product: Product) => (
        <div className="flex flex-wrap gap-1">
          {product.tags?.map((tag) => (
            <span key={tag} className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
              {tag}
            </span>
          ))}
        </div>
      ),
    },
    {
      header: t("createdAt"),
      accessorKey: "created_at" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(product.created_at)}
        </span>
      ),
    },
    {
      header: "",
      accessorKey: "id" as const,
      className: "w-[50px]",
      cell: (product: Product) => (
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            setDeleteId(product.id);
          }}
        >
          <Trash2 className="h-4 w-4 text-destructive" />
        </Button>
      ),
    },
  ];

  return (
    <>
      <ProductListToolbar
        filters={
          <>
            <div className="relative w-full min-w-0 sm:w-[280px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t("searchPlaceholder")}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPagination((prev) => ({ ...prev, offset: 0 }));
                }}
                className="pl-9"
              />
            </div>
            <Input
              placeholder={t("filterByTag")}
              value={localTagFilter}
              onChange={(e) => handleTagFilterChange(e.target.value)}
              className="w-full sm:w-[160px]"
            />
            <CategoryTreePicker
              value={categoryIdFilter}
              onChange={(value) => {
                setCategoryIdFilter(value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
              placeholder={t("categoryPlaceholder")}
              className="w-full sm:w-[200px]"
            />
            <Select
              value={supplierFilter}
              onValueChange={(value) => {
                setSupplierFilter(value === "__all__" ? "" : value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
            >
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder={t("supplierPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{t("allSuppliers")}</SelectItem>
                {suppliersData?.items?.map((s) => (
                  <SelectItem key={s.id} value={s.id}>
                    {s.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={sourceFilter}
              onValueChange={(value) => {
                setSourceFilter(value === "__all__" ? "" : value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
            >
              <SelectTrigger className="w-full sm:w-[160px]">
                <SelectValue placeholder={t("sourcePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{t("allSources")}</SelectItem>
                {Object.entries(ORDER_SOURCE_LABELS).map(([key, label]) => (
                  <SelectItem key={key} value={key}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={marketplaceFilter}
              onValueChange={(value) => {
                setMarketplaceFilter(value === "__all__" ? "" : value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
            >
              <SelectTrigger className="w-full sm:w-[170px]">
                <SelectValue placeholder="Marketplace..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{t("allMarketplaces")}</SelectItem>
                <SelectItem value="none">{t("notListed")}</SelectItem>
                {Object.entries(MARKETPLACE_LABELS).map(([key, label]) => (
                  <SelectItem key={key} value={key}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        }
        actions={
          <>
            {selectedProducts.size > 0 && (
              <>
                <Button
                  variant="outline"
                  onClick={() => {
                    bulkCategorize.mutate(Array.from(selectedProducts), {
                      onSuccess: (data) => {
                        const succeeded = data.results.filter((r) => !r.error).length;
                        const failed = data.results.filter((r) => r.error).length;
                        toast.success(t("autoCategorizeResult", { succeeded, failed }));
                        setSelectedProducts(new Set());
                      },
                      onError: (error) => {
                        toast.error(error instanceof Error ? error.message : t("autoCategorizeError"));
                      },
                    });
                  }}
                  disabled={bulkCategorize.isPending}
                >
                  {bulkCategorize.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Sparkles className="h-4 w-4" />
                  )}
                  {t("autoCategorize")} ({selectedProducts.size})
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => setShowBulkDelete(true)}
                >
                  <Trash2 className="h-4 w-4" />
                  {t("deleteCount", { count: selectedProducts.size })}
                </Button>
              </>
            )}
            <Button
              variant="outline"
              onClick={() => {
                redownloadImages.mutate(undefined, {
                  onSuccess: (data) => {
                    toast.success(
                      t("photosDownloaded", { downloaded: data.downloaded, skipped: data.skipped, failed: data.failed })
                    );
                  },
                  onError: (error) => {
                    toast.error(getErrorMessage(error));
                  },
                });
              }}
              disabled={redownloadImages.isPending}
            >
              {redownloadImages.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <ImageIcon className="h-4 w-4" />
              )}
              {t("downloadPhotos")}
            </Button>
            <Button
              variant="outline"
              onClick={async () => {
                try {
                  const res = await apiFetch("/v1/products/export");
                  const blob = await res.blob();
                  downloadBlob(blob, `products_${new Date().toISOString().slice(0, 10)}.csv`);
                  toast.success(t("csvExportStarted"));
                } catch {
                  toast.error(t("csvExportError"));
                }
              }}
            >
              <Download className="h-4 w-4" />
              {t("exportCsv")}
            </Button>
            <Button variant="outline" asChild>
              <Link href="/products/import">
                <Upload className="h-4 w-4" />
                {t("importCsv")}
              </Link>
            </Button>
            <Button asChild>
              <Link href="/products/new">
                <Plus className="h-4 w-4" />
                {t("addProduct")}
              </Link>
            </Button>
          </>
        }
      />

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("tryAgain")}
          </Button>
        </div>
      )}

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        isLoading={isLoading}
        emptyState={
          <EmptyState
            icon={Package}
            title={t("noProducts")}
            description={t("noProductsHint")}
            action={{ label: t("importFromAllegro"), href: "/marketplaces/new" }}
            secondaryAction={{ label: t("addProduct"), href: "/products/new" }}
          />
        }
        sortBy={sortBy}
        sortOrder={sortOrder}
        onSort={handleSort}
        editableColumns={editableColumns}
        selectable
        selectedIds={selectedProducts}
        onSelectionChange={setSelectedProducts}
      />

      {data && (
        <DataTablePagination
          total={data.total}
          limit={data.limit}
          offset={data.offset}
          onPageChange={(offset) =>
            setPagination((prev) => ({ ...prev, offset }))
          }
          onPageSizeChange={(limit) =>
            setPagination({ limit, offset: 0 })
          }
        />
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={tc("delete")}
        description={t("deleteConfirm")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={() => {
          if (!deleteId) return;
          deleteProduct.mutate(deleteId, {
            onSuccess: () => {
              toast.success(t("productDeleted"));
              setDeleteId(null);
            },
            onError: (error) => {
              toast.error(getErrorMessage(error));
            },
          });
        }}
        isLoading={deleteProduct.isPending}
      />

      <ConfirmDialog
        open={showBulkDelete}
        onOpenChange={(open) => !open && setShowBulkDelete(false)}
        title={t("deleteProducts", { count: selectedProducts.size })}
        description={t("bulkDeleteConfirm")}
        confirmLabel={bulkDeleting ? t("deleting") : t("deleteAll")}
        variant="destructive"
        onConfirm={async () => {
          setBulkDeleting(true);
          const ids = Array.from(selectedProducts);
          let succeeded = 0;
          let failed = 0;
          for (const id of ids) {
            try {
              await apiFetch(`/v1/products/${id}`, { method: "DELETE" });
              succeeded++;
            } catch {
              failed++;
            }
          }
          setBulkDeleting(false);
          setShowBulkDelete(false);
          setSelectedProducts(new Set());
          if (failed === 0) {
            toast.success(t("bulkDeleteResult", { succeeded }));
          } else {
            toast.warning(t("bulkDeletePartial", { succeeded, failed }));
          }
        }}
        isLoading={bulkDeleting}
      />
    </>
  );
}

// ─── Supplier Catalog Tab ───

function SupplierCatalogTab() {
  const t = useTranslations("products.list");
  const tc = useTranslations("common");
  const router = useRouter();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [supplierFilter, setSupplierFilter] = useState("");
  const [pagination, setPagination] = useState({ limit: 50, offset: 0 });
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [importingId, setImportingId] = useState<string | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [detailProduct, setDetailProduct] = useState<SupplierProductWithSupplier | null>(null);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);
  const [bulkImporting, setBulkImporting] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPagination((prev) => ({ ...prev, offset: 0 }));
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Supplier products view depends on the controlled "suppliers" feature; skip both
  // gated fetches when the feature is not visible in the active surface.
  const suppliersVisible = isFeatureVisible("suppliers");

  const { data: suppliersData } = useAllSuppliers({}, { enabled: suppliersVisible });

  const { data, isLoading } = useAllSupplierProducts(
    {
      search: debouncedSearch || undefined,
      supplier_id: supplierFilter || undefined,
      sort_by: sortBy,
      sort_order: sortOrder,
      limit: pagination.limit,
      offset: pagination.offset,
    },
    { enabled: suppliersVisible }
  );

  const items = data?.items ?? [];

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setPagination((prev) => ({ ...prev, offset: 0 }));
  };

  const handleListOnMarketplace = useCallback(async (sp: SupplierProductWithSupplier) => {
    if (sp.product_id) {
      router.push(`/products/${sp.product_id}/listings?listing=new`);
      return;
    }
    setImportingId(sp.id);
    try {
      const product = await apiClient<Product>(
        `/v1/suppliers/${sp.supplier_id}/products/${sp.id}/import-single`,
        { method: "POST" }
      );
      toast.success(t("productImported"));
      router.push(`/products/${product.id}/listings?listing=new`);
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setImportingId(null);
    }
  }, [router, t]);

  const handleImportOnly = useCallback(async (sp: SupplierProductWithSupplier) => {
    if (sp.product_id) {
      toast.info(t("alreadyInCatalog"));
      return;
    }
    setImportingId(sp.id);
    try {
      await apiClient<Product>(
        `/v1/suppliers/${sp.supplier_id}/products/${sp.id}/import-single`,
        { method: "POST" }
      );
      toast.success(t("productImported"));
      queryClient.invalidateQueries({ queryKey: ["supplier-products"] });
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setImportingId(null);
    }
  }, [queryClient, t]);

  const handleBulkImport = useCallback(async () => {
    const selected = items.filter((sp) => selectedIds.has(sp.id) && !sp.product_id);
    if (selected.length === 0) {
      toast.info(t("noUnimportedSelected"));
      return;
    }
    setBulkImporting(true);
    let imported = 0;
    let errors = 0;
    for (const sp of selected) {
      try {
        await apiClient<Product>(
          `/v1/suppliers/${sp.supplier_id}/products/${sp.id}/import-single`,
          { method: "POST" }
        );
        imported++;
      } catch {
        errors++;
      }
    }
    setBulkImporting(false);
    setSelectedIds(new Set());
    queryClient.invalidateQueries({ queryKey: ["supplier-products"] });
    queryClient.invalidateQueries({ queryKey: ["products"] });
    const parts: string[] = [];
    if (imported > 0) parts.push(t("bulkImportResult", { imported }));
    if (errors > 0) parts.push(t("bulkImportErrors", { errors }));
    toast.success(parts.join(", "));
  }, [selectedIds, items, queryClient, t]);

  const selectedUnimported = useMemo(() => {
    return items.filter((sp) => selectedIds.has(sp.id) && !sp.product_id).length;
  }, [selectedIds, items]);

  const getMetaString = (meta: Record<string, unknown>, key: string): string | undefined => {
    const v = meta?.[key];
    return typeof v === "string" && v ? v : undefined;
  };

  const columns = [
    {
      header: "",
      accessorKey: "metadata" as const,
      className: "w-[50px] px-0",
      cell: (sp: SupplierProductWithSupplier) => {
        const imgUrl = getMetaString(sp.metadata, "image_url");
        return imgUrl ? (
          <button
            className="w-8 h-8 rounded border overflow-hidden bg-muted/30 flex-shrink-0 cursor-pointer"
            onClick={() => { setDetailProduct(sp); setSelectedImageIndex(0); }}
          >
            <img src={imgUrl} alt="" className="w-full h-full object-contain" />
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
      accessorKey: "name" as const,
      sortable: true,
      cell: (sp: SupplierProductWithSupplier) => {
        const brand = getMetaString(sp.metadata, "brand");
        return (
          <div className="min-w-0">
            <button
              className="font-medium truncate block text-left hover:underline cursor-pointer max-w-full"
              onClick={() => { setDetailProduct(sp); setSelectedImageIndex(0); }}
            >
              {sp.name}
            </button>
            <p className="text-xs text-muted-foreground truncate">
              {[brand, sp.ean].filter(Boolean).join(" · ")}
            </p>
          </div>
        );
      },
    },
    {
      header: t("category"),
      accessorKey: "source_category" as const,
      cell: (sp: SupplierProductWithSupplier) => (
        <span className="text-sm text-muted-foreground truncate max-w-[160px] block">
          {sp.source_category || "-"}
        </span>
      ),
    },
    {
      header: t("price"),
      accessorKey: "price" as const,
      sortable: true,
      className: "text-right",
      cell: (sp: SupplierProductWithSupplier) => (
        <div className="text-right">
          <span className="block text-sm">{sp.price ? formatCurrency(sp.price) : "-"}</span>
          {sp.metadata?.retail_price != null && (
            <span className="text-xs text-muted-foreground">
              det. {formatCurrency(Number(sp.metadata.retail_price))}
            </span>
          )}
        </div>
      ),
    },
    {
      header: t("stock"),
      accessorKey: "stock_quantity" as const,
      sortable: true,
      className: "text-right",
      cell: (sp: SupplierProductWithSupplier) => (
        <span
          className={`text-sm font-medium ${
            sp.stock_quantity === 0 ? "text-destructive" : ""
          }`}
        >
          {sp.stock_quantity}
        </span>
      ),
    },
    {
      header: t("supplier"),
      accessorKey: "supplier_name" as const,
      cell: (sp: SupplierProductWithSupplier) => (
        <span className="text-sm">{sp.supplier_name}</span>
      ),
    },
    {
      header: "Status",
      accessorKey: "product_id" as const,
      cell: (sp: SupplierProductWithSupplier) => (
        sp.product_id ? (
          <Badge variant="outline" className="gap-1">
            <Link2 className="h-3 w-3" />
            {t("inCatalog")}
          </Badge>
        ) : (
          <Badge variant="secondary">{t("notImported")}</Badge>
        )
      ),
    },
    {
      header: "",
      accessorKey: "id" as const,
      className: "w-[120px]",
      cell: (sp: SupplierProductWithSupplier) => (
        <div className="flex items-center gap-1 justify-end" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            title={t("preview")}
            onClick={() => { setDetailProduct(sp); setSelectedImageIndex(0); }}
          >
            <Eye className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            title={t("listOnMarketplace")}
            disabled={importingId === sp.id}
            onClick={() => handleListOnMarketplace(sp)}
          >
            {importingId === sp.id ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <ShoppingBag className="h-3.5 w-3.5" />
            )}
          </Button>
          {!sp.product_id && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              title={t("importToCatalog")}
              disabled={importingId === sp.id}
              onClick={() => handleImportOnly(sp)}
            >
              <Download className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <>
      <ProductListToolbar
        filters={
          <>
            <div className="relative w-full min-w-0 sm:w-[280px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t("searchPlaceholder")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select
              value={supplierFilter}
              onValueChange={(value) => {
                setSupplierFilter(value === "__all__" ? "" : value);
                setPagination((prev) => ({ ...prev, offset: 0 }));
              }}
            >
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder={t("supplierPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{t("allSuppliers")}</SelectItem>
                {suppliersData?.items?.map((s) => (
                  <SelectItem key={s.id} value={s.id}>
                    {s.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        }
        actions={
          selectedIds.size > 0 ? (
            <Button
              onClick={handleBulkImport}
              disabled={bulkImporting || selectedUnimported === 0}
            >
              {bulkImporting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              {bulkImporting
                ? t("importing")
                : t("importCount", { count: selectedUnimported })}
            </Button>
          ) : null
        }
      />

      <DataTable
        columns={columns}
        data={items}
        isLoading={isLoading}
        emptyState={
          <EmptyState
            icon={Package}
            title={t("noSupplierProducts")}
            description={t("noSupplierProductsHint")}
            action={{ label: t("addSupplier"), href: "/suppliers/new" }}
          />
        }
        sortBy={sortBy}
        sortOrder={sortOrder}
        onSort={handleSort}
        resizable
        selectable
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
        rowId={(row) => row.id}
      />

      {data && (
        <DataTablePagination
          total={data.total}
          limit={data.limit}
          offset={data.offset}
          onPageChange={(offset) =>
            setPagination((prev) => ({ ...prev, offset }))
          }
          onPageSizeChange={(limit) =>
            setPagination({ limit, offset: 0 })
          }
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
                                selectedImageIndex === i
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
                  <div>
                    <span className="text-muted-foreground">Dostawca:</span>{" "}
                    <span className="font-medium">{detailProduct.supplier_name}</span>
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
                  <div>
                    <span className="text-muted-foreground">Status:</span>{" "}
                    {detailProduct.product_id ? (
                      <Badge variant="outline" className="gap-1">
                        <Link2 className="h-3 w-3" />
                        {t("inCatalog")}
                      </Badge>
                    ) : (
                      <Badge variant="secondary">{t("notImported")}</Badge>
                    )}
                  </div>
                </div>

                {/* Description */}
                {getMetaString(detailProduct.metadata, "description") && (
                  <div>
                    <h4 className="text-sm font-medium mb-1">Opis</h4>
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
                    disabled={importingId === detailProduct.id}
                  >
                    {importingId === detailProduct.id ? (
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    ) : (
                      <ShoppingBag className="h-4 w-4 mr-2" />
                    )}
                    Wystaw na marketplace
                  </Button>
                  {!detailProduct.product_id && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        handleImportOnly(detailProduct);
                        setDetailProduct(null);
                      }}
                      disabled={importingId === detailProduct.id}
                    >
                      <Download className="h-4 w-4 mr-2" />
                      Importuj do katalogu
                    </Button>
                  )}
                  {detailProduct.product_id && (
                    <Button variant="outline" size="sm" asChild>
                      <Link href={`/products/${detailProduct.product_id}`}>
                        <Eye className="h-4 w-4 mr-2" />
                        Zobacz produkt
                      </Link>
                    </Button>
                  )}
                </div>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
