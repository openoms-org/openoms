"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, Check, Eraser, Layers, Package, PackageOpen, Pencil, Plus, RefreshCw, Store, Trash2, Upload, X, Sparkles, Loader2, TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  useBundleComponents,
  useBundleStock,
  useAddBundleComponent,
  useUpdateBundleComponent,
  useRemoveBundleComponent,
} from "@/hooks/use-bundles";
import {
  usePushProductStock,
  useReconcileProductStock,
} from "@/hooks/use-stock-sync";
import { useProducts } from "@/hooks/use-products";
import { ProductForm } from "@/components/products/product-form";
import {
  useProduct,
  useUpdateProduct,
  useDeleteProduct,
} from "@/hooks/use-products";
import { useProductCategories } from "@/hooks/use-product-categories";
import { formatCurrency, formatDate } from "@/lib/utils";
import { ORDER_SOURCE_LABELS } from "@/lib/constants";
import { getErrorMessage } from "@/lib/api-client";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  useSuggestCategories,
  useGenerateDescription,
} from "@/hooks/use-ai";
import type { CreateProductRequest, AISuggestion, AIDescribeRequest } from "@/types/api";
import { normalizeProductImages } from "@/types/api";
import { useBGRemovalStatus, useRemoveProductImageBackground } from "@/hooks/use-bg-removal";
import { useRepricingLog } from "@/hooks/use-repricing";
import { isFeatureVisible } from "@/lib/readiness";
import { useTranslations } from "next-intl";

export default function ProductDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const t = useTranslations("products.detail");
  const tc = useTranslations("common");
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const [showAddComponentDialog, setShowAddComponentDialog] = useState(false);
  const [aiSuggestions, setAiSuggestions] = useState<AISuggestion | null>(null);
  const [showDescriptionDialog, setShowDescriptionDialog] = useState(false);
  const [aiShortDescription, setAiShortDescription] = useState("");
  const [aiLongDescription, setAiLongDescription] = useState("");
  const [showAIOptionsDialog, setShowAIOptionsDialog] = useState(false);
  const [aiStyle, setAiStyle] = useState<string>("professional");
  const [aiLanguage, setAiLanguage] = useState<string>("pl");
  const [aiLength, setAiLength] = useState<string>("medium");
  const [aiMarketplace, setAiMarketplace] = useState<string>("");

  const suggestCategories = useSuggestCategories();
  const generateDescription = useGenerateDescription();

  // Gate non-ready feature fetches behind surface visibility so ready pages do not
  // call gated endpoints (which return feature_not_available 404 in client-ready).
  const aiVisible = isFeatureVisible("ai");
  const repricingVisible = isFeatureVisible("repricing");

  const { data: product, isLoading } = useProduct(params.id);
  const { data: categoriesConfig } = useProductCategories();
  const { data: bgStatus } = useBGRemovalStatus({ enabled: aiVisible });
  const removeProductBg = useRemoveProductImageBackground(params.id);
  const updateProduct = useUpdateProduct(params.id);
  const deleteProduct = useDeleteProduct();

  const { data: bundleComponents, isLoading: isLoadingBundle } = useBundleComponents(params.id);
  const { data: bundleStockData } = useBundleStock(params.id, (bundleComponents?.length ?? 0) > 0);
  const addComponent = useAddBundleComponent(params.id);
  const updateComponent = useUpdateBundleComponent(params.id);
  const removeComponent = useRemoveBundleComponent(params.id);

  // Inline bundle-component quantity editing.
  const [editingComponentId, setEditingComponentId] = useState<string | null>(null);
  const [editingQuantity, setEditingQuantity] = useState<string>("");

  // Per-product stock-sync (feature-gated, beta). Hidden in client-ready surface.
  const stockSyncVisible = isFeatureVisible("stock_sync");
  const pushStock = usePushProductStock();
  const reconcileStock = useReconcileProductStock();

  const handleSaveComponentQuantity = (componentId: string) => {
    const qty = Number(editingQuantity);
    if (!Number.isFinite(qty) || qty < 1) {
      toast.error(t("componentQuantityInvalid"));
      return;
    }
    updateComponent.mutate(
      { componentId, data: { quantity: qty } },
      {
        onSuccess: () => {
          toast.success(t("componentQuantityUpdated"));
          setEditingComponentId(null);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };
  const { data: priceHistory } = useRepricingLog(
    { product_id: params.id, limit: 10 },
    { enabled: repricingVisible }
  );

  const handleUpdate = (data: CreateProductRequest) => {
    updateProduct.mutate(
      {
        name: data.name || undefined,
        sku: data.sku || undefined,
        ean: data.ean || undefined,
        price: data.price,
        stock_quantity: data.stock_quantity,
        source: data.source || undefined,
        description_short: data.description_short || undefined,
        description_long: data.description_long || undefined,
        weight: data.weight,
        width: data.width,
        height: data.height,
        depth: data.depth,
        image_url: data.image_url,
        images: data.images,
        tags: data.tags,
        category: data.category,
      },
      {
        onSuccess: () => {
          toast.success(t("updated"));
          setIsEditing(false);
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  const handleDelete = () => {
    deleteProduct.mutate(params.id, {
      onSuccess: () => {
        toast.success(t("deleted"));
        router.push("/products");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!product) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">{t("notFound")}</h1>
        <Button asChild variant="outline">
          <Link href="/products">{t("backToList")}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/products">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{product.name}</h1>
            <p className="text-muted-foreground">
              {t("createdAt", { date: formatDate(product.created_at) })}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {/* AI button group */}
          <Popover>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  if (!aiSuggestions) {
                    suggestCategories.mutate(params.id, {
                      onSuccess: (data) => setAiSuggestions(data),
                      onError: (error) => toast.error(getErrorMessage(error)),
                    });
                  }
                }}
              >
                {suggestCategories.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Sparkles className="h-4 w-4" />
                )}
                {t("suggestCategories")}
              </Button>
            </PopoverTrigger>
            {aiSuggestions && (
              <PopoverContent className="w-80">
                <div className="space-y-3">
                  <p className="text-sm font-medium">{t("suggestedCategories")}</p>
                  <div className="flex flex-wrap gap-1">
                    {aiSuggestions.categories.map((cat) => (
                      <button
                        key={cat}
                        className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary hover:bg-primary/20 cursor-pointer transition-colors"
                        onClick={() => {
                          updateProduct.mutate(
                            { category: cat },
                            {
                              onSuccess: () => {
                                toast.success(t("categoryApplied", { category: cat }));
                                setAiSuggestions(null);
                              },
                              onError: (error) => toast.error(getErrorMessage(error)),
                            }
                          );
                        }}
                      >
                        {cat}
                      </button>
                    ))}
                  </div>
                  {aiSuggestions.tags.length > 0 && (
                    <>
                      <p className="text-sm font-medium">{t("suggestedTags")}</p>
                      <div className="flex flex-wrap gap-1">
                        {aiSuggestions.tags.map((tag) => (
                          <button
                            key={tag}
                            className="rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium hover:bg-muted/80 cursor-pointer transition-colors"
                            onClick={() => {
                              const currentTags = product?.tags || [];
                              if (!currentTags.includes(tag)) {
                                updateProduct.mutate(
                                  { tags: [...currentTags, tag] },
                                  {
                                    onSuccess: () => toast.success(t("tagAdded", { tag })),
                                    onError: (error) => toast.error(getErrorMessage(error)),
                                  }
                                );
                              }
                            }}
                          >
                            + {tag}
                          </button>
                        ))}
                      </div>
                    </>
                  )}
                  <p className="text-xs text-muted-foreground">
                    {t("clickToApply")}
                  </p>
                </div>
              </PopoverContent>
            )}
          </Popover>

          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowAIOptionsDialog(true)}
          >
            <Sparkles className="h-4 w-4" />
            {t("generateAiDescription")}
          </Button>

          <Button variant="outline" size="sm" asChild>
            <Link href={`/products/${params.id}/listings`}>
              <Store className="h-4 w-4" />
              {t("marketplaceOffers")}
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href={`/products/${params.id}/variants`}>
              <Layers className="h-4 w-4" />
              {t("variants")}
            </Link>
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            <Pencil className="h-4 w-4" />
            {isEditing ? t("cancelEdit") : tc("edit")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="h-4 w-4" />
            {tc("delete")}
          </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("editProduct" as Parameters<typeof t>[0])}</CardTitle>
          </CardHeader>
          <CardContent>
            <ProductForm
              product={product}
              onSubmit={handleUpdate}
              isPending={updateProduct.isPending}
            />
          </CardContent>
        </Card>
      ) : (
        <>
        <Card>
          <CardHeader>
            <CardTitle>{t("photos")}</CardTitle>
          </CardHeader>
          <CardContent>
            {product.image_url ? (
              <div className="space-y-4">
                <div className="relative group inline-block">
                  <img
                    src={product.image_url}
                    alt={product.name}
                    className="max-w-sm rounded-lg border object-cover"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                  />
                  {bgStatus?.configured && (
                    <Button
                      variant="secondary"
                      size="sm"
                      className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity shadow-md"
                      disabled={removeProductBg.isPending}
                      onClick={() => {
                        removeProductBg.mutate(-1, {
                          onSuccess: () => toast.success(t("bgRemovedMain")),
                          onError: (error) => toast.error(error instanceof Error ? error.message : t("bgRemoveError")),
                        });
                      }}
                    >
                      {removeProductBg.isPending ? (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      ) : (
                        <Eraser className="mr-1 h-3 w-3" />
                      )}
                      {t("removeBackground")}
                    </Button>
                  )}
                </div>
                {product.images && product.images.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {normalizeProductImages(product.images).map((img, i) => (
                      <div key={i} className="relative group">
                        <img
                          src={img.url}
                          alt={img.alt || t("photoAlt", { index: i + 1 })}
                          className="h-20 w-20 rounded border object-cover"
                          onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                        />
                        {bgStatus?.configured && (
                          <Button
                            variant="secondary"
                            size="icon"
                            className="absolute -top-1 -right-1 h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity shadow-md"
                            disabled={removeProductBg.isPending}
                            onClick={() => {
                              removeProductBg.mutate(i, {
                                onSuccess: () => toast.success(t("bgRemovedPhoto", { index: i + 1 })),
                                onError: (error) => toast.error(error instanceof Error ? error.message : t("bgRemoveError")),
                              });
                            }}
                          >
                            {removeProductBg.isPending ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <Eraser className="h-3 w-3" />
                            )}
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="flex h-32 w-32 items-center justify-center rounded-lg border bg-muted">
                <Package className="h-12 w-12 text-muted-foreground" />
              </div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("productDetails")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <p className="text-sm text-muted-foreground">{tc("name")}</p>
                <p className="text-sm font-medium">{product.name}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="font-mono text-sm">{product.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">SKU</p>
                <p className="font-mono text-sm">{product.sku || "-"}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">EAN</p>
                <p className="font-mono text-sm">{product.ean || "-"}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{tc("price")}</p>
                <p className="text-sm font-medium">
                  {formatCurrency(product.price)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{tc("stockQuantity")}</p>
                <p
                  className={`text-sm font-medium ${
                    product.stock_quantity === 0
                      ? "text-destructive"
                      : product.stock_quantity < 10
                        ? "text-warning"
                        : ""
                  }`}
                >
                  {product.stock_quantity}
                </p>
                {stockSyncVisible && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={pushStock.isPending}
                      title={t("pushStockHint")}
                      onClick={() =>
                        pushStock.mutate(params.id, {
                          onSuccess: () => toast.success(t("stockPushed")),
                          onError: (error) => toast.error(getErrorMessage(error)),
                        })
                      }
                    >
                      <Upload className="mr-2 h-4 w-4" />
                      {t("pushStock")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={reconcileStock.isPending}
                      title={t("reconcileStockHint")}
                      onClick={() =>
                        reconcileStock.mutate(params.id, {
                          onSuccess: () => toast.success(t("stockReconciled")),
                          onError: (error) => toast.error(getErrorMessage(error)),
                        })
                      }
                    >
                      <RefreshCw className="mr-2 h-4 w-4" />
                      {t("reconcileStock")}
                    </Button>
                  </div>
                )}
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("source")}</p>
                <p className="text-sm">
                  {ORDER_SOURCE_LABELS[product.source] ?? product.source}
                </p>
              </div>
              {product.category && (() => {
                const cat = categoriesConfig?.categories?.find((c) => c.key === product.category);
                return (
                  <div>
                    <p className="text-sm text-muted-foreground">{tc("category")}</p>
                    <span
                      className="inline-block rounded-full px-2.5 py-0.5 text-xs font-medium mt-1"
                      style={{
                        backgroundColor: cat?.color ? `${cat.color}20` : undefined,
                        color: cat?.color,
                      }}
                    >
                      {cat?.label || product.category}
                    </span>
                  </div>
                );
              })()}
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("externalId")}
                </p>
                <p className="font-mono text-sm">
                  {product.external_id || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("createdDate")}</p>
                <p className="text-sm">{formatDate(product.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("lastUpdated")}
                </p>
                <p className="text-sm">{formatDate(product.updated_at)}</p>
              </div>
            </div>
            {product.description_short && (
              <div className="sm:col-span-2">
                <p className="text-sm text-muted-foreground">{t("shortDescription")}</p>
                <p className="mt-1 text-sm">{product.description_short}</p>
              </div>
            )}

            {product.description_long && (
              <div className="sm:col-span-2">
                <Separator />
                <div className="pt-4">
                  <p className="text-sm text-muted-foreground">{t("fullDescription")}</p>
                  <p className="mt-1 text-sm whitespace-pre-wrap">{product.description_long}</p>
                </div>
              </div>
            )}
            {(product.weight || product.width || product.height || product.depth) && (
              <>
                <Separator />
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-2">{t("dimensionsAndWeight")}</p>
                  <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                    {product.weight != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">{t("weight")}</p>
                        <p className="mt-1 font-medium">{product.weight} kg</p>
                      </div>
                    )}
                    {product.width != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">{t("width")}</p>
                        <p className="mt-1 font-medium">{product.width} cm</p>
                      </div>
                    )}
                    {product.height != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">{t("height")}</p>
                        <p className="mt-1 font-medium">{product.height} cm</p>
                      </div>
                    )}
                    {product.depth != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">{t("depth")}</p>
                        <p className="mt-1 font-medium">{product.depth} cm</p>
                      </div>
                    )}
                  </div>
                </div>
              </>
            )}
            {product.tags && product.tags.length > 0 && (
              <div className="pt-4">
                <p className="text-sm text-muted-foreground">{tc("tags")}</p>
                <div className="mt-1 flex flex-wrap gap-1">
                  {product.tags.map((tag) => (
                    <span key={tag} className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Bundle Toggle & Components */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span className="flex items-center gap-2">
                <PackageOpen className="h-4 w-4" />
                {t("productBundle")}
              </span>
              <div className="flex items-center gap-2">
                <Label htmlFor="is-bundle-toggle" className="text-sm font-normal text-muted-foreground">
                  {t("bundle")}
                </Label>
                <Switch
                  id="is-bundle-toggle"
                  checked={product.is_bundle}
                  onCheckedChange={(checked) => {
                    updateProduct.mutate(
                      { is_bundle: checked },
                      {
                        onSuccess: () => {
                          toast.success(checked ? t("markedAsBundle") : t("unmarkedAsBundle"));
                        },
                        onError: (error) => {
                          toast.error(getErrorMessage(error));
                        },
                      }
                    );
                  }}
                />
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {product.is_bundle ? (
              <div className="space-y-4">
                {bundleStockData && (
                  <div className="rounded-md bg-muted/50 p-3">
                    <p className="text-sm text-muted-foreground">{t("bundleStock")}</p>
                    <p className={`text-lg font-bold ${bundleStockData.stock === 0 ? "text-destructive" : ""}`}>
                      {bundleStockData.stock}
                    </p>
                  </div>
                )}
                {isLoadingBundle ? (
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-full" />
                    <Skeleton className="h-4 w-3/4" />
                  </div>
                ) : bundleComponents && bundleComponents.length > 0 ? (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("component")}</TableHead>
                        <TableHead>SKU</TableHead>
                        <TableHead className="text-right">{t("quantity")}</TableHead>
                        <TableHead className="text-right">{t("stock")}</TableHead>
                        <TableHead className="w-[50px]"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {bundleComponents.map((comp) => (
                        <TableRow key={comp.id}>
                          <TableCell className="font-medium">{comp.component_name}</TableCell>
                          <TableCell className="font-mono text-xs text-muted-foreground">
                            {comp.component_sku || "-"}
                          </TableCell>
                          <TableCell className="text-right">
                            {editingComponentId === comp.id ? (
                              <div className="flex items-center justify-end gap-1">
                                <Input
                                  type="number"
                                  min={1}
                                  value={editingQuantity}
                                  onChange={(e) => setEditingQuantity(e.target.value)}
                                  className="h-7 w-20 text-right"
                                  autoFocus
                                  onKeyDown={(e) => {
                                    if (e.key === "Enter") handleSaveComponentQuantity(comp.id);
                                    if (e.key === "Escape") setEditingComponentId(null);
                                  }}
                                />
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7"
                                  disabled={updateComponent.isPending}
                                  onClick={() => handleSaveComponentQuantity(comp.id)}
                                  aria-label={tc("save")}
                                >
                                  <Check className="h-4 w-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7"
                                  onClick={() => setEditingComponentId(null)}
                                  aria-label={tc("cancel")}
                                >
                                  <X className="h-4 w-4" />
                                </Button>
                              </div>
                            ) : (
                              <button
                                type="button"
                                className="inline-flex items-center gap-1 hover:underline"
                                onClick={() => {
                                  setEditingComponentId(comp.id);
                                  setEditingQuantity(String(comp.quantity));
                                }}
                                title={t("editQuantity")}
                              >
                                {comp.quantity}
                                <Pencil className="h-3 w-3 text-muted-foreground" />
                              </button>
                            )}
                          </TableCell>
                          <TableCell className="text-right">{comp.component_stock}</TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              onClick={() => {
                                removeComponent.mutate(comp.id, {
                                  onSuccess: () => toast.success(t("componentRemoved")),
                                  onError: (error) => toast.error(getErrorMessage(error)),
                                });
                              }}
                            >
                              <X className="h-4 w-4" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                ) : (
                  <div className="flex flex-col items-center justify-center py-8 text-center">
                    <PackageOpen className="h-8 w-8 text-muted-foreground/50 mb-2" />
                    <p className="text-sm text-muted-foreground">{t("noComponents")}</p>
                  </div>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowAddComponentDialog(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  {t("addComponent")}
                </Button>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("enableBundleHint")}
              </p>
            )}
          </CardContent>
        </Card>

        {/* Price History (Repricing Log) */}
        {priceHistory && priceHistory.items.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-4 w-4" />
                {t("priceHistory")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("oldPrice")}</TableHead>
                    <TableHead>{t("newPrice")}</TableHead>
                    <TableHead>{t("change")}</TableHead>
                    <TableHead>{t("reason")}</TableHead>
                    <TableHead>{tc("date")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {priceHistory.items.map((log) => {
                    const changePct =
                      log.old_price > 0
                        ? (
                            ((log.new_price - log.old_price) / log.old_price) *
                            100
                          ).toFixed(1)
                        : "0";
                    const isIncrease = log.new_price > log.old_price;
                    return (
                      <TableRow key={log.id}>
                        <TableCell>{formatCurrency(log.old_price)}</TableCell>
                        <TableCell>{formatCurrency(log.new_price)}</TableCell>
                        <TableCell>
                          <span
                            className={
                              isIncrease ? "text-green-600" : "text-red-600"
                            }
                          >
                            {isIncrease ? "+" : ""}
                            {changePct}%
                          </span>
                        </TableCell>
                        <TableCell className="max-w-[200px] truncate text-sm text-muted-foreground">
                          {log.reason || "-"}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatDate(log.applied_at)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
              <div className="mt-3 text-center">
                <Button variant="ghost" size="sm" asChild>
                  <Link href="/repricing">
                    {t("viewAllRepricing")}
                  </Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
        </>
      )}

      {/* AI Options Dialog */}
      <Dialog open={showAIOptionsDialog} onOpenChange={setShowAIOptionsDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiOptions.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("aiOptions.style")}</Label>
              <Select value={aiStyle} onValueChange={setAiStyle}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="professional">{t("aiOptions.professional")}</SelectItem>
                  <SelectItem value="promotional">{t("aiOptions.promotional")}</SelectItem>
                  <SelectItem value="casual">{t("aiOptions.casual")}</SelectItem>
                  <SelectItem value="seo">{t("aiOptions.seo")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("aiOptions.language")}</Label>
              <Select value={aiLanguage} onValueChange={setAiLanguage}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="pl">Polski</SelectItem>
                  <SelectItem value="en">English</SelectItem>
                  <SelectItem value="de">Deutsch</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("aiOptions.length")}</Label>
              <Select value={aiLength} onValueChange={setAiLength}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="short">{t("aiOptions.short")}</SelectItem>
                  <SelectItem value="medium">{t("aiOptions.medium")}</SelectItem>
                  <SelectItem value="long">{t("aiOptions.long")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("aiOptions.marketplace")}</Label>
              <Select value={aiMarketplace || "__none__"} onValueChange={(v) => setAiMarketplace(v === "__none__" ? "" : v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t("aiOptions.none")}</SelectItem>
                  <SelectItem value="allegro">Allegro</SelectItem>
                  <SelectItem value="amazon">Amazon</SelectItem>
                  <SelectItem value="ebay">eBay</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAIOptionsDialog(false)}>
              {tc("cancel")}
            </Button>
            <Button
              onClick={() => {
                const req: AIDescribeRequest = {
                  product_id: params.id,
                  style: aiStyle as AIDescribeRequest["style"],
                  language: aiLanguage as AIDescribeRequest["language"],
                  length: aiLength as AIDescribeRequest["length"],
                  marketplace: aiMarketplace as AIDescribeRequest["marketplace"] || undefined,
                };
                generateDescription.mutate(req, {
                  onSuccess: (data) => {
                    setAiShortDescription(data.short_description || "");
                    setAiLongDescription(data.long_description || data.description || "");
                    setShowAIOptionsDialog(false);
                    setShowDescriptionDialog(true);
                  },
                  onError: (error) => toast.error(getErrorMessage(error)),
                });
              }}
              disabled={generateDescription.isPending}
            >
              {generateDescription.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Sparkles className="mr-2 h-4 w-4" />
              )}
              {t("aiOptions.generate")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* AI Generated Description Dialog */}
      <Dialog open={showDescriptionDialog} onOpenChange={setShowDescriptionDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("aiResult.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {aiShortDescription && (
              <div>
                <p className="text-sm font-medium text-muted-foreground mb-1">{t("aiResult.shortDescription")}</p>
                <p className="text-sm whitespace-pre-wrap rounded-md border p-3">{aiShortDescription}</p>
              </div>
            )}
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-1">{t("aiResult.fullDescription")}</p>
              <p className="text-sm whitespace-pre-wrap rounded-md border p-3">{aiLongDescription}</p>
            </div>
          </div>
          <DialogFooter className="flex-col sm:flex-row gap-2">
            <Button variant="outline" onClick={() => setShowDescriptionDialog(false)}>
              {tc("cancel")}
            </Button>
            {aiShortDescription && (
              <Button
                variant="secondary"
                onClick={() => {
                  updateProduct.mutate(
                    { description_short: aiShortDescription },
                    {
                      onSuccess: () => toast.success(t("aiResult.shortUpdated")),
                      onError: (error) => toast.error(getErrorMessage(error)),
                    }
                  );
                }}
              >
                {t("aiResult.applyShort")}
              </Button>
            )}
            <Button
              onClick={() => {
                const update: Record<string, string> = { description_long: aiLongDescription };
                if (aiShortDescription) {
                  update.description_short = aiShortDescription;
                }
                updateProduct.mutate(update, {
                  onSuccess: () => {
                    toast.success(t("aiResult.descriptionUpdated"));
                    setShowDescriptionDialog(false);
                  },
                  onError: (error) => toast.error(getErrorMessage(error)),
                });
              }}
            >
              {t("aiResult.applyAll")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteProduct")}
        description={t("deleteProductConfirm", { name: product.name })}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteProduct.isPending}
      />

      <AddBundleComponentDialog
        open={showAddComponentDialog}
        onOpenChange={setShowAddComponentDialog}
        bundleProductId={params.id}
        onAdd={(componentId, quantity) => {
          addComponent.mutate(
            { component_product_id: componentId, quantity, position: 0 },
            {
              onSuccess: () => {
                toast.success(t("addComponentDialog.componentAdded"));
                setShowAddComponentDialog(false);
              },
              onError: (error) => {
                toast.error(getErrorMessage(error));
              },
            }
          );
        }}
        isPending={addComponent.isPending}
      />
    </div>
  );
}

function AddBundleComponentDialog({
  open,
  onOpenChange,
  bundleProductId,
  onAdd,
  isPending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  bundleProductId: string;
  onAdd: (componentId: string, quantity: number) => void;
  isPending: boolean;
}) {
  const t = useTranslations("products.detail.addComponentDialog");
  const tc = useTranslations("common");
  const [search, setSearch] = useState("");
  const [selectedProductId, setSelectedProductId] = useState<string>("");
  const [quantity, setQuantity] = useState(1);

  const { data: productsData } = useProducts({ name: search || undefined, limit: 20 });
  const products = (productsData?.items || []).filter((p) => p.id !== bundleProductId);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>{t("searchProduct")}</Label>
            <Input
              placeholder={t("searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          {products.length > 0 && (
            <div className="max-h-48 overflow-y-auto rounded-md border">
              {products.map((p) => (
                <div
                  key={p.id}
                  className={`cursor-pointer px-3 py-2 text-sm hover:bg-muted transition-colors ${selectedProductId === p.id ? "bg-primary/10 font-medium" : ""}`}
                  onClick={() => setSelectedProductId(p.id)}
                >
                  <span>{p.name}</span>
                  {p.sku && (
                    <span className="ml-2 text-xs text-muted-foreground">({p.sku})</span>
                  )}
                  <span className="ml-2 text-xs text-muted-foreground">
                    {tc("stockQuantity")}: {p.stock_quantity}
                  </span>
                </div>
              ))}
            </div>
          )}
          <div>
            <Label>{t("quantityInBundle")}</Label>
            <Input
              type="number"
              min={1}
              value={quantity}
              onChange={(e) => setQuantity(Math.max(1, Number(e.target.value)))}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc("cancel")}
          </Button>
          <Button
            onClick={() => onAdd(selectedProductId, quantity)}
            disabled={!selectedProductId || isPending}
          >
            {isPending ? t("adding") : t("addComponent" as Parameters<typeof t>[0])}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
