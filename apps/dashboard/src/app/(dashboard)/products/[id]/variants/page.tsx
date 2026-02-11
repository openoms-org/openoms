"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ArrowLeft, Plus, Pencil, Trash2, Layers } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useProduct } from "@/hooks/use-products";
import {
  useVariants,
  useCreateVariant,
  useUpdateVariant,
  useDeleteVariant,
} from "@/hooks/use-variants";
import { formatCurrency } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type {
  ProductVariant,
  CreateVariantRequest,
  UpdateVariantRequest,
} from "@/types/api";

interface AttributeEntry {
  key: string;
  value: string;
}

function AttributeEditor({
  attributes,
  onChange,
}: {
  attributes: AttributeEntry[];
  onChange: (attrs: AttributeEntry[]) => void;
}) {
  const addAttribute = () => {
    onChange([...attributes, { key: "", value: "" }]);
  };

  const removeAttribute = (index: number) => {
    onChange(attributes.filter((_, i) => i !== index));
  };

  const updateAttribute = (
    index: number,
    field: "key" | "value",
    val: string
  ) => {
    const updated = [...attributes];
    updated[index] = { ...updated[index], [field]: val };
    onChange(updated);
  };

  return (
    <div className="space-y-2">
      <Label>Atrybuty</Label>
      {attributes.map((attr, i) => (
        <div key={i} className="flex gap-2 items-center">
          <Input
            placeholder="Klucz (np. Kolor)"
            value={attr.key}
            onChange={(e) => updateAttribute(i, "key", e.target.value)}
            className="flex-1"
          />
          <Input
            placeholder="Wartość (np. Czerwony)"
            value={attr.value}
            onChange={(e) => updateAttribute(i, "value", e.target.value)}
            className="flex-1"
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => removeAttribute(i)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addAttribute}>
        <Plus className="h-3 w-3 mr-1" />
        Dodaj atrybut
      </Button>
    </div>
  );
}

function attributesToEntries(
  attrs: Record<string, string> | undefined
): AttributeEntry[] {
  if (!attrs || Object.keys(attrs).length === 0) return [];
  return Object.entries(attrs).map(([key, value]) => ({ key, value }));
}

function entriesToAttributes(
  entries: AttributeEntry[]
): Record<string, string> {
  const result: Record<string, string> = {};
  for (const entry of entries) {
    const k = entry.key.trim();
    if (k) {
      result[k] = entry.value;
    }
  }
  return result;
}

export default function ProductVariantsPage() {
  const params = useParams<{ id: string }>();
  const productId = params.id;

  const { data: product, isLoading: productLoading } = useProduct(productId);
  const { data: variantsData, isLoading: variantsLoading } = useVariants(
    productId,
    { limit: 100 }
  );
  const createVariant = useCreateVariant(productId);
  const deleteVariant = useDeleteVariant(productId);

  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingVariant, setEditingVariant] = useState<ProductVariant | null>(
    null
  );
  const [deleteVariantId, setDeleteVariantId] = useState<string | null>(null);

  // Create form state
  const [createForm, setCreateForm] = useState<{
    name: string;
    sku: string;
    ean: string;
    price_override: string;
    stock_quantity: string;
    weight: string;
    position: string;
    active: boolean;
    attributes: AttributeEntry[];
  }>({
    name: "",
    sku: "",
    ean: "",
    price_override: "",
    stock_quantity: "0",
    weight: "",
    position: "0",
    active: true,
    attributes: [],
  });

  const resetCreateForm = () => {
    setCreateForm({
      name: "",
      sku: "",
      ean: "",
      price_override: "",
      stock_quantity: "0",
      weight: "",
      position: "0",
      active: true,
      attributes: [],
    });
  };

  const handleCreate = () => {
    const data: CreateVariantRequest = {
      name: createForm.name,
      stock_quantity: parseInt(createForm.stock_quantity) || 0,
      active: createForm.active,
      position: parseInt(createForm.position) || 0,
      attributes: entriesToAttributes(createForm.attributes),
    };
    if (createForm.sku) data.sku = createForm.sku;
    if (createForm.ean) data.ean = createForm.ean;
    if (createForm.price_override)
      data.price_override = parseFloat(createForm.price_override);
    if (createForm.weight) data.weight = parseFloat(createForm.weight);

    createVariant.mutate(data, {
      onSuccess: () => {
        toast.success("Wariant został dodany");
        setShowCreateDialog(false);
        resetCreateForm();
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleDelete = () => {
    if (!deleteVariantId) return;
    deleteVariant.mutate(deleteVariantId, {
      onSuccess: () => {
        toast.success("Wariant został usunięty");
        setDeleteVariantId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  if (productLoading) {
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
          <Link href="/products">Wroc do listy</Link>
        </Button>
      </div>
    );
  }

  const variants = variantsData?.items ?? [];
  const hasVariants = variants.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/products/${productId}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{product.name}</h1>
            <p className="text-muted-foreground">Warianty produktu</p>
          </div>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Dodaj wariant
        </Button>
      </div>

      {variantsLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : !hasVariants ? (
        <EmptyState
          icon={Layers}
          title="Brak wariantow"
          description="Ten produkt nie posiada jeszcze wariantow. Dodaj warianty, aby zarzadzac rozmiarami, kolorami itp."
        />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>
              Warianty ({variantsData?.total ?? 0})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nazwa</TableHead>
                    <TableHead>SKU</TableHead>
                    <TableHead>EAN</TableHead>
                    <TableHead>Cena nadpisana</TableHead>
                    <TableHead>Stan magazynowy</TableHead>
                    <TableHead>Atrybuty</TableHead>
                    <TableHead>Pozycja</TableHead>
                    <TableHead>Aktywny</TableHead>
                    <TableHead className="w-[100px]">Akcje</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {variants.map((variant) => (
                    <TableRow key={variant.id}>
                      <TableCell className="font-medium">
                        {variant.name}
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {variant.sku || "---"}
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {variant.ean || "---"}
                      </TableCell>
                      <TableCell>
                        {variant.price_override != null
                          ? formatCurrency(variant.price_override)
                          : "---"}
                      </TableCell>
                      <TableCell>
                        <span
                          className={
                            variant.stock_quantity === 0
                              ? "text-destructive font-medium"
                              : variant.stock_quantity < 10
                                ? "text-yellow-600 dark:text-yellow-400 font-medium"
                                : ""
                          }
                        >
                          {variant.stock_quantity}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {variant.attributes &&
                            Object.entries(variant.attributes).map(
                              ([k, v]) => (
                                <Badge
                                  key={k}
                                  variant="secondary"
                                  className="text-xs"
                                >
                                  {k}: {v}
                                </Badge>
                              )
                            )}
                        </div>
                      </TableCell>
                      <TableCell>{variant.position}</TableCell>
                      <TableCell>
                        <Badge
                          variant={variant.active ? "default" : "outline"}
                        >
                          {variant.active ? "Tak" : "Nie"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setEditingVariant(variant)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteVariantId(variant.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Create Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Dodaj wariant</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="create-name">Nazwa *</Label>
              <Input
                id="create-name"
                value={createForm.name}
                onChange={(e) =>
                  setCreateForm({ ...createForm, name: e.target.value })
                }
                placeholder="np. Czerwony XL"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="create-sku">SKU</Label>
                <Input
                  id="create-sku"
                  value={createForm.sku}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, sku: e.target.value })
                  }
                />
              </div>
              <div>
                <Label htmlFor="create-ean">EAN</Label>
                <Input
                  id="create-ean"
                  value={createForm.ean}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, ean: e.target.value })
                  }
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="create-price">Cena nadpisana</Label>
                <Input
                  id="create-price"
                  type="number"
                  step="0.01"
                  min="0"
                  value={createForm.price_override}
                  onChange={(e) =>
                    setCreateForm({
                      ...createForm,
                      price_override: e.target.value,
                    })
                  }
                />
              </div>
              <div>
                <Label htmlFor="create-stock">Stan magazynowy</Label>
                <Input
                  id="create-stock"
                  type="number"
                  min="0"
                  value={createForm.stock_quantity}
                  onChange={(e) =>
                    setCreateForm({
                      ...createForm,
                      stock_quantity: e.target.value,
                    })
                  }
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="create-weight">Waga (kg)</Label>
                <Input
                  id="create-weight"
                  type="number"
                  step="0.001"
                  min="0"
                  value={createForm.weight}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, weight: e.target.value })
                  }
                />
              </div>
              <div>
                <Label htmlFor="create-position">Pozycja</Label>
                <Input
                  id="create-position"
                  type="number"
                  min="0"
                  value={createForm.position}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, position: e.target.value })
                  }
                />
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="create-active"
                checked={createForm.active}
                onCheckedChange={(checked) =>
                  setCreateForm({ ...createForm, active: checked })
                }
              />
              <Label htmlFor="create-active">Aktywny</Label>
            </div>
            <AttributeEditor
              attributes={createForm.attributes}
              onChange={(attributes) =>
                setCreateForm({ ...createForm, attributes })
              }
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCreateDialog(false)}
            >
              Anuluj
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!createForm.name || createVariant.isPending}
            >
              {createVariant.isPending ? "Dodawanie..." : "Dodaj"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      {editingVariant && (
        <EditVariantDialog
          productId={productId}
          variant={editingVariant}
          open={!!editingVariant}
          onOpenChange={(open) => {
            if (!open) setEditingVariant(null);
          }}
        />
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteVariantId}
        onOpenChange={(open) => {
          if (!open) setDeleteVariantId(null);
        }}
        title="Usun wariant"
        description="Czy na pewno chcesz usunac ten wariant? Ta operacja jest nieodwracalna."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteVariant.isPending}
      />
    </div>
  );
}

function EditVariantDialog({
  productId,
  variant,
  open,
  onOpenChange,
}: {
  productId: string;
  variant: ProductVariant;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const updateVariant = useUpdateVariant(productId, variant.id);

  const [form, setForm] = useState({
    name: variant.name,
    sku: variant.sku || "",
    ean: variant.ean || "",
    price_override:
      variant.price_override != null ? String(variant.price_override) : "",
    stock_quantity: String(variant.stock_quantity),
    weight: variant.weight != null ? String(variant.weight) : "",
    position: String(variant.position),
    active: variant.active,
    attributes: attributesToEntries(variant.attributes),
  });

  const handleUpdate = () => {
    const data: UpdateVariantRequest = {};
    if (form.name !== variant.name) data.name = form.name;
    if (form.sku !== (variant.sku || ""))
      data.sku = form.sku || undefined;
    if (form.ean !== (variant.ean || ""))
      data.ean = form.ean || undefined;

    const newPrice = form.price_override ? parseFloat(form.price_override) : undefined;
    if (newPrice !== variant.price_override) data.price_override = newPrice;

    const newStock = parseInt(form.stock_quantity) || 0;
    if (newStock !== variant.stock_quantity) data.stock_quantity = newStock;

    const newWeight = form.weight ? parseFloat(form.weight) : undefined;
    if (newWeight !== variant.weight) data.weight = newWeight;

    const newPos = parseInt(form.position) || 0;
    if (newPos !== variant.position) data.position = newPos;

    if (form.active !== variant.active) data.active = form.active;

    const attrs = entriesToAttributes(form.attributes);
    data.attributes = attrs;

    // Check if anything changed
    if (Object.keys(data).length === 0) {
      onOpenChange(false);
      return;
    }

    updateVariant.mutate(data, {
      onSuccess: () => {
        toast.success("Wariant został zaktualizowany");
        onOpenChange(false);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edytuj wariant</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label htmlFor="edit-name">Nazwa *</Label>
            <Input
              id="edit-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="edit-sku">SKU</Label>
              <Input
                id="edit-sku"
                value={form.sku}
                onChange={(e) => setForm({ ...form, sku: e.target.value })}
              />
            </div>
            <div>
              <Label htmlFor="edit-ean">EAN</Label>
              <Input
                id="edit-ean"
                value={form.ean}
                onChange={(e) => setForm({ ...form, ean: e.target.value })}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="edit-price">Cena nadpisana</Label>
              <Input
                id="edit-price"
                type="number"
                step="0.01"
                min="0"
                value={form.price_override}
                onChange={(e) =>
                  setForm({ ...form, price_override: e.target.value })
                }
              />
            </div>
            <div>
              <Label htmlFor="edit-stock">Stan magazynowy</Label>
              <Input
                id="edit-stock"
                type="number"
                min="0"
                value={form.stock_quantity}
                onChange={(e) =>
                  setForm({ ...form, stock_quantity: e.target.value })
                }
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="edit-weight">Waga (kg)</Label>
              <Input
                id="edit-weight"
                type="number"
                step="0.001"
                min="0"
                value={form.weight}
                onChange={(e) =>
                  setForm({ ...form, weight: e.target.value })
                }
              />
            </div>
            <div>
              <Label htmlFor="edit-position">Pozycja</Label>
              <Input
                id="edit-position"
                type="number"
                min="0"
                value={form.position}
                onChange={(e) =>
                  setForm({ ...form, position: e.target.value })
                }
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Switch
              id="edit-active"
              checked={form.active}
              onCheckedChange={(checked) =>
                setForm({ ...form, active: checked })
              }
            />
            <Label htmlFor="edit-active">Aktywny</Label>
          </div>
          <AttributeEditor
            attributes={form.attributes}
            onChange={(attributes) => setForm({ ...form, attributes })}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button
            onClick={handleUpdate}
            disabled={!form.name || updateVariant.isPending}
          >
            {updateVariant.isPending ? "Zapisywanie..." : "Zapisz"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
