"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, Layers, Package, PackageOpen, Pencil, Plus, Store, Trash2, X, Sparkles, Loader2 } from "lucide-react";
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
  useRemoveBundleComponent,
} from "@/hooks/use-bundles";
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
  useImproveDescription,
  useTranslateDescription,
} from "@/hooks/use-ai";
import type { CreateProductRequest, AISuggestion, AIDescribeRequest } from "@/types/api";

export default function ProductDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
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
  const improveDescription = useImproveDescription();
  const translateDescription = useTranslateDescription();

  const { data: product, isLoading } = useProduct(params.id);
  const { data: categoriesConfig } = useProductCategories();
  const updateProduct = useUpdateProduct(params.id);
  const deleteProduct = useDeleteProduct();

  const { data: bundleComponents, isLoading: isLoadingBundle } = useBundleComponents(params.id);
  const { data: bundleStockData } = useBundleStock(params.id);
  const addComponent = useAddBundleComponent(params.id);
  const removeComponent = useRemoveBundleComponent(params.id);

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
          toast.success("Produkt został zaktualizowany");
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
        toast.success("Produkt został usunięty");
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
        <h1 className="text-2xl font-bold">Nie znaleziono produktu</h1>
        <Button asChild variant="outline">
          <Link href="/products">Wróć do listy</Link>
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
              Utworzony {formatDate(product.created_at)}
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
                Sugeruj kategorie
              </Button>
            </PopoverTrigger>
            {aiSuggestions && (
              <PopoverContent className="w-80">
                <div className="space-y-3">
                  <p className="text-sm font-medium">Sugerowane kategorie</p>
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
                                toast.success(`Kategoria "${cat}" zastosowana`);
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
                      <p className="text-sm font-medium">Sugerowane tagi</p>
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
                                    onSuccess: () => toast.success(`Tag "${tag}" dodany`),
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
                    Kliknij aby zastosować sugestię
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
            Generuj opis AI
          </Button>

          <Button variant="outline" size="sm" asChild>
            <Link href={`/products/${params.id}/listings`}>
              <Store className="h-4 w-4" />
              Oferty marketplace
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href={`/products/${params.id}/variants`}>
              <Layers className="h-4 w-4" />
              Warianty
            </Link>
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            <Pencil className="h-4 w-4" />
            {isEditing ? "Anuluj edycję" : "Edytuj"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="h-4 w-4" />
            Usuń
          </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>Edycja produktu</CardTitle>
          </CardHeader>
          <CardContent>
            <ProductForm
              product={product}
              onSubmit={handleUpdate}
              isLoading={updateProduct.isPending}
            />
          </CardContent>
        </Card>
      ) : (
        <>
        <Card>
          <CardHeader>
            <CardTitle>Zdjęcia</CardTitle>
          </CardHeader>
          <CardContent>
            {product.image_url ? (
              <div className="space-y-4">
                <img
                  src={product.image_url}
                  alt={product.name}
                  className="max-w-sm rounded-lg border object-cover"
                  onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                />
                {product.images && product.images.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {product.images.map((img, i) => (
                      <img
                        key={i}
                        src={img.url}
                        alt={img.alt || `Zdjęcie ${i + 1}`}
                        className="h-20 w-20 rounded border object-cover"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                      />
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
            <CardTitle>Szczegóły produktu</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <p className="text-sm text-muted-foreground">Nazwa</p>
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
                <p className="text-sm text-muted-foreground">Cena</p>
                <p className="text-sm font-medium">
                  {formatCurrency(product.price)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Stan magazynowy</p>
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
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Źródło</p>
                <p className="text-sm">
                  {ORDER_SOURCE_LABELS[product.source] ?? product.source}
                </p>
              </div>
              {product.category && (() => {
                const cat = categoriesConfig?.categories?.find((c) => c.key === product.category);
                return (
                  <div>
                    <p className="text-sm text-muted-foreground">Kategoria</p>
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
                  Identyfikator zewnętrzny
                </p>
                <p className="font-mono text-sm">
                  {product.external_id || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Data utworzenia</p>
                <p className="text-sm">{formatDate(product.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Ostatnia aktualizacja
                </p>
                <p className="text-sm">{formatDate(product.updated_at)}</p>
              </div>
            </div>
            {product.description_short && (
              <div className="sm:col-span-2">
                <p className="text-sm text-muted-foreground">Krótki opis</p>
                <p className="mt-1 text-sm">{product.description_short}</p>
              </div>
            )}

            {product.description_long && (
              <div className="sm:col-span-2">
                <Separator />
                <div className="pt-4">
                  <p className="text-sm text-muted-foreground">Opis</p>
                  <p className="mt-1 text-sm whitespace-pre-wrap">{product.description_long}</p>
                </div>
              </div>
            )}
            {(product.weight || product.width || product.height || product.depth) && (
              <>
                <Separator />
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-2">Wymiary i waga</p>
                  <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                    {product.weight != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">Waga</p>
                        <p className="mt-1 font-medium">{product.weight} kg</p>
                      </div>
                    )}
                    {product.width != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">Szerokość</p>
                        <p className="mt-1 font-medium">{product.width} cm</p>
                      </div>
                    )}
                    {product.height != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">Wysokość</p>
                        <p className="mt-1 font-medium">{product.height} cm</p>
                      </div>
                    )}
                    {product.depth != null && (
                      <div>
                        <p className="text-sm text-muted-foreground">Głębokość</p>
                        <p className="mt-1 font-medium">{product.depth} cm</p>
                      </div>
                    )}
                  </div>
                </div>
              </>
            )}
            {product.tags && product.tags.length > 0 && (
              <div className="pt-4">
                <p className="text-sm text-muted-foreground">Tagi</p>
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
                Zestaw produktów
              </span>
              <div className="flex items-center gap-2">
                <Label htmlFor="is-bundle-toggle" className="text-sm font-normal text-muted-foreground">
                  Zestaw
                </Label>
                <Switch
                  id="is-bundle-toggle"
                  checked={product.is_bundle}
                  onCheckedChange={(checked) => {
                    updateProduct.mutate(
                      { is_bundle: checked },
                      {
                        onSuccess: () => {
                          toast.success(checked ? "Produkt oznaczony jako zestaw" : "Produkt nie jest już zestawem");
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
                    <p className="text-sm text-muted-foreground">Stan zestawu (kalkulowany)</p>
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
                        <TableHead>Komponent</TableHead>
                        <TableHead>SKU</TableHead>
                        <TableHead className="text-right">Ilość</TableHead>
                        <TableHead className="text-right">Stan</TableHead>
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
                          <TableCell className="text-right">{comp.quantity}</TableCell>
                          <TableCell className="text-right">{comp.component_stock}</TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                              onClick={() => {
                                removeComponent.mutate(comp.id, {
                                  onSuccess: () => toast.success("Komponent usunięty z zestawu"),
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
                    <p className="text-sm text-muted-foreground">Brak komponentów w zestawie.</p>
                  </div>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowAddComponentDialog(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Dodaj komponent
                </Button>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Włącz przełącznik &quot;Zestaw&quot; aby zarządzać komponentami zestawu.
              </p>
            )}
          </CardContent>
        </Card>
        </>
      )}

      {/* AI Options Dialog */}
      <Dialog open={showAIOptionsDialog} onOpenChange={setShowAIOptionsDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Generuj opis AI</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Styl</Label>
              <Select value={aiStyle} onValueChange={setAiStyle}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="professional">Profesjonalny</SelectItem>
                  <SelectItem value="promotional">Promocyjny</SelectItem>
                  <SelectItem value="casual">Swobodny</SelectItem>
                  <SelectItem value="seo">SEO</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Język</Label>
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
              <Label>Długość</Label>
              <Select value={aiLength} onValueChange={setAiLength}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="short">Krótki</SelectItem>
                  <SelectItem value="medium">Średni</SelectItem>
                  <SelectItem value="long">Długi</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Marketplace</Label>
              <Select value={aiMarketplace || "__none__"} onValueChange={(v) => setAiMarketplace(v === "__none__" ? "" : v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">Brak</SelectItem>
                  <SelectItem value="allegro">Allegro</SelectItem>
                  <SelectItem value="amazon">Amazon</SelectItem>
                  <SelectItem value="ebay">eBay</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAIOptionsDialog(false)}>
              Anuluj
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
              Generuj
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* AI Generated Description Dialog */}
      <Dialog open={showDescriptionDialog} onOpenChange={setShowDescriptionDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Wygenerowany opis AI</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {aiShortDescription && (
              <div>
                <p className="text-sm font-medium text-muted-foreground mb-1">Krótki opis</p>
                <p className="text-sm whitespace-pre-wrap rounded-md border p-3">{aiShortDescription}</p>
              </div>
            )}
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-1">Pełny opis</p>
              <p className="text-sm whitespace-pre-wrap rounded-md border p-3">{aiLongDescription}</p>
            </div>
          </div>
          <DialogFooter className="flex-col sm:flex-row gap-2">
            <Button variant="outline" onClick={() => setShowDescriptionDialog(false)}>
              Anuluj
            </Button>
            {aiShortDescription && (
              <Button
                variant="secondary"
                onClick={() => {
                  updateProduct.mutate(
                    { description_short: aiShortDescription },
                    {
                      onSuccess: () => toast.success("Krótki opis zaktualizowany"),
                      onError: (error) => toast.error(getErrorMessage(error)),
                    }
                  );
                }}
              >
                Zastosuj krótki opis
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
                    toast.success("Opis produktu zaktualizowany");
                    setShowDescriptionDialog(false);
                  },
                  onError: (error) => toast.error(getErrorMessage(error)),
                });
              }}
            >
              Zastosuj wszystko
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Usuń produkt"
        description={`Czy na pewno chcesz usunąć produkt "${product.name}"? Ta operacja jest nieodwracalna.`}
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteProduct.isPending}
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
                toast.success("Komponent dodany do zestawu");
                setShowAddComponentDialog(false);
              },
              onError: (error) => {
                toast.error(getErrorMessage(error));
              },
            }
          );
        }}
        isLoading={addComponent.isPending}
      />
    </div>
  );
}

function AddBundleComponentDialog({
  open,
  onOpenChange,
  bundleProductId,
  onAdd,
  isLoading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  bundleProductId: string;
  onAdd: (componentId: string, quantity: number) => void;
  isLoading: boolean;
}) {
  const [search, setSearch] = useState("");
  const [selectedProductId, setSelectedProductId] = useState<string>("");
  const [quantity, setQuantity] = useState(1);

  const { data: productsData } = useProducts({ name: search || undefined, limit: 20 });
  const products = (productsData?.items || []).filter((p) => p.id !== bundleProductId);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Dodaj komponent do zestawu</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>Szukaj produktu</Label>
            <Input
              placeholder="Wpisz nazwę produktu..."
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
                    Stan: {p.stock_quantity}
                  </span>
                </div>
              ))}
            </div>
          )}
          <div>
            <Label>Ilość w zestawie</Label>
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
            Anuluj
          </Button>
          <Button
            onClick={() => onAdd(selectedProductId, quantity)}
            disabled={!selectedProductId || isLoading}
          >
            {isLoading ? "Dodawanie..." : "Dodaj komponent"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
