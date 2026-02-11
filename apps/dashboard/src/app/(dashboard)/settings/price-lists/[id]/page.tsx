"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  usePriceList,
  useUpdatePriceList,
  usePriceListItems,
  useCreatePriceListItem,
  useDeletePriceListItem,
} from "@/hooks/use-price-lists";
import { useProducts } from "@/hooks/use-products";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const discountTypeLabels: Record<string, string> = {
  percentage: "Procentowy",
  fixed: "Kwotowy",
  override: "Cena nadpisana",
};

export default function PriceListDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const { data: priceList, isLoading } = usePriceList(id);
  const updatePriceList = useUpdatePriceList(id);
  const { data: itemsData, isLoading: itemsLoading } = usePriceListItems(id, {
    limit: 100,
  });
  const createItem = useCreatePriceListItem(id);
  const deleteItem = useDeletePriceListItem(id);
  const { data: productsData } = useProducts({ limit: 100 });

  const [showAddProduct, setShowAddProduct] = useState(false);
  const [selectedProductId, setSelectedProductId] = useState("");
  const [newDiscount, setNewDiscount] = useState("");
  const [newPrice, setNewPrice] = useState("");
  const [newMinQty, setNewMinQty] = useState("1");
  const [deleteItemId, setDeleteItemId] = useState<string | null>(null);

  // Inline editing state
  const [editName, setEditName] = useState<string | null>(null);
  const [editDescription, setEditDescription] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!priceList) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">Nie znaleziono cennika</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => router.push("/settings/price-lists")}
        >
          Powrot do listy
        </Button>
      </div>
    );
  }

  const items = itemsData?.items ?? [];
  const products = productsData?.items ?? [];

  const handleToggleActive = () => {
    updatePriceList.mutate(
      { active: !priceList.active },
      {
        onSuccess: () => {
          toast.success(
            priceList.active ? "Cennik dezaktywowany" : "Cennik aktywowany"
          );
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleSaveName = () => {
    if (editName === null || editName === priceList.name) {
      setEditName(null);
      return;
    }
    updatePriceList.mutate(
      { name: editName },
      {
        onSuccess: () => {
          toast.success("Nazwa zaktualizowana");
          setEditName(null);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleSaveDescription = () => {
    if (editDescription === null) return;
    updatePriceList.mutate(
      { description: editDescription },
      {
        onSuccess: () => {
          toast.success("Opis zaktualizowany");
          setEditDescription(null);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleAddProduct = () => {
    if (!selectedProductId) return;
    const data: {
      product_id: string;
      discount?: number;
      price?: number;
      min_quantity?: number;
    } = {
      product_id: selectedProductId,
    };
    if (newDiscount) data.discount = parseFloat(newDiscount);
    if (newPrice) data.price = parseFloat(newPrice);
    if (newMinQty) data.min_quantity = parseInt(newMinQty, 10);

    createItem.mutate(data, {
      onSuccess: () => {
        toast.success("Produkt dodany do cennika");
        setShowAddProduct(false);
        setSelectedProductId("");
        setNewDiscount("");
        setNewPrice("");
        setNewMinQty("1");
      },
      onError: (err) => toast.error(getErrorMessage(err)),
    });
  };

  const handleDeleteItem = () => {
    if (!deleteItemId) return;
    deleteItem.mutate(deleteItemId, {
      onSuccess: () => {
        toast.success("Pozycja usunieta z cennika");
        setDeleteItemId(null);
      },
      onError: (err) => toast.error(getErrorMessage(err)),
    });
  };

  const getProductName = (productId: string) => {
    const product = products.find((p) => p.id === productId);
    return product?.name ?? productId.slice(0, 8);
  };

  return (
    <>
      <div className="flex items-center gap-4 mb-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.push("/settings/price-lists")}
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          Powrot
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Szczegoly cennika</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Nazwa</Label>
              {editName !== null ? (
                <div className="flex gap-2">
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                  />
                  <Button size="sm" onClick={handleSaveName}>
                    Zapisz
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditName(null)}
                  >
                    Anuluj
                  </Button>
                </div>
              ) : (
                <p
                  className="text-sm cursor-pointer hover:text-primary"
                  onClick={() => setEditName(priceList.name)}
                >
                  {priceList.name}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label>Opis</Label>
              {editDescription !== null ? (
                <div className="flex gap-2">
                  <Input
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                  />
                  <Button size="sm" onClick={handleSaveDescription}>
                    Zapisz
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditDescription(null)}
                  >
                    Anuluj
                  </Button>
                </div>
              ) : (
                <p
                  className="text-sm cursor-pointer hover:text-primary"
                  onClick={() =>
                    setEditDescription(priceList.description ?? "")
                  }
                >
                  {priceList.description || "---"}
                </p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Typ rabatu</Label>
                <p className="text-sm mt-1">
                  {discountTypeLabels[priceList.discount_type] ??
                    priceList.discount_type}
                </p>
              </div>
              <div>
                <Label>Waluta</Label>
                <p className="text-sm mt-1">{priceList.currency}</p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Utworzono</Label>
                <p className="text-sm mt-1">
                  {formatDate(priceList.created_at)}
                </p>
              </div>
              <div>
                <Label>Zaktualizowano</Label>
                <p className="text-sm mt-1">
                  {formatDate(priceList.updated_at)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <Label>Aktywny</Label>
              <Switch
                checked={priceList.active}
                onCheckedChange={handleToggleActive}
              />
            </div>
            {priceList.is_default && (
              <Badge variant="default">Domyslny cennik</Badge>
            )}
            {priceList.valid_from && (
              <div>
                <Label>Wazny od</Label>
                <p className="text-sm">{formatDate(priceList.valid_from)}</p>
              </div>
            )}
            {priceList.valid_to && (
              <div>
                <Label>Wazny do</Label>
                <p className="text-sm">{formatDate(priceList.valid_to)}</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Items */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Pozycje cennika</CardTitle>
          <Dialog open={showAddProduct} onOpenChange={setShowAddProduct}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-2" />
                Dodaj produkt
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Dodaj produkt do cennika</DialogTitle>
                <DialogDescription>
                  Wybierz produkt i ustaw rabat lub cene
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Produkt</Label>
                  <Select
                    value={selectedProductId}
                    onValueChange={setSelectedProductId}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Wybierz produkt..." />
                    </SelectTrigger>
                    <SelectContent>
                      {products.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.name} {p.sku ? `(${p.sku})` : ""}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {priceList.discount_type === "override" ? (
                  <div className="space-y-2">
                    <Label>Cena ({priceList.currency})</Label>
                    <Input
                      type="number"
                      step="0.01"
                      value={newPrice}
                      onChange={(e) => setNewPrice(e.target.value)}
                      placeholder="0.00"
                    />
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Label>
                      Rabat (
                      {priceList.discount_type === "percentage"
                        ? "%"
                        : priceList.currency}
                      )
                    </Label>
                    <Input
                      type="number"
                      step="0.01"
                      value={newDiscount}
                      onChange={(e) => setNewDiscount(e.target.value)}
                      placeholder="0"
                    />
                  </div>
                )}

                <div className="space-y-2">
                  <Label>Minimalna ilosc</Label>
                  <Input
                    type="number"
                    min="1"
                    value={newMinQty}
                    onChange={(e) => setNewMinQty(e.target.value)}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setShowAddProduct(false)}
                >
                  Anuluj
                </Button>
                <Button
                  onClick={handleAddProduct}
                  disabled={!selectedProductId || createItem.isPending}
                >
                  {createItem.isPending ? "Dodawanie..." : "Dodaj"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent>
          {itemsLoading ? (
            <LoadingSkeleton />
          ) : items.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">
              Brak pozycji w cenniku. Dodaj produkty, aby ustawic ceny.
            </p>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Produkt</TableHead>
                    <TableHead>
                      {priceList.discount_type === "override"
                        ? "Cena"
                        : "Rabat"}
                    </TableHead>
                    <TableHead>Min. ilosc</TableHead>
                    <TableHead>Dodano</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium">
                        {getProductName(item.product_id)}
                      </TableCell>
                      <TableCell>
                        {priceList.discount_type === "override"
                          ? item.price != null
                            ? `${item.price.toFixed(2)} ${priceList.currency}`
                            : "---"
                          : item.discount != null
                          ? priceList.discount_type === "percentage"
                            ? `${item.discount}%`
                            : `${item.discount.toFixed(2)} ${priceList.currency}`
                          : "---"}
                      </TableCell>
                      <TableCell>{item.min_quantity}</TableCell>
                      <TableCell>{formatDate(item.created_at)}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => setDeleteItemId(item.id)}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!deleteItemId}
        onOpenChange={(open) => !open && setDeleteItemId(null)}
        title="Usun pozycje"
        description="Czy na pewno chcesz usunac te pozycje z cennika?"
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDeleteItem}
        isLoading={deleteItem.isPending}
      />
    </>
  );
}
