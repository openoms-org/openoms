"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Plus, Trash2, Search } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreatePurchaseOrder, useSendPurchaseOrder } from "@/hooks/use-purchase-orders";
import { useSuppliers } from "@/hooks/use-suppliers";
import { useWarehouses } from "@/hooks/use-warehouses";
import { useProducts } from "@/hooks/use-products";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { CreatePurchaseOrderItemReq } from "@/types/api";

interface ItemRow extends CreatePurchaseOrderItemReq {
  _key: string;
}

export default function NewPurchaseOrderPage() {
  const router = useRouter();
  const createPO = useCreatePurchaseOrder();
  const sendPO = useSendPurchaseOrder();
  const { data: suppliersData } = useSuppliers({ limit: 100 });
  const { data: warehousesData } = useWarehouses({ limit: 100 });

  const [supplierID, setSupplierID] = useState("");
  const [supplierName, setSupplierName] = useState("");
  const [warehouseID, setWarehouseID] = useState("");
  const [expectedDate, setExpectedDate] = useState("");
  const [currency, setCurrency] = useState("PLN");
  const [notes, setNotes] = useState("");
  const [items, setItems] = useState<ItemRow[]>([]);

  // Product search dialog
  const [showProductSearch, setShowProductSearch] = useState(false);
  const [productSearch, setProductSearch] = useState("");
  const { data: productsData } = useProducts({
    name: productSearch || undefined,
    limit: 20,
  });

  const suppliers = suppliersData?.items ?? [];
  const warehouses = warehousesData?.items ?? [];

  const handleSupplierChange = (value: string) => {
    setSupplierID(value);
    const supplier = suppliers.find((s) => s.id === value);
    if (supplier) {
      setSupplierName(supplier.name);
    }
  };

  const addItem = (product?: { id: string; name: string; sku?: string; price?: number }) => {
    const newItem: ItemRow = {
      _key: crypto.randomUUID(),
      product_id: product?.id,
      sku: product?.sku || "",
      product_name: product?.name || "",
      quantity: 1,
      unit_cost: product?.price || 0,
      notes: "",
    };
    setItems([...items, newItem]);
    setShowProductSearch(false);
    setProductSearch("");
  };

  const updateItem = (key: string, field: string, value: string | number) => {
    setItems(items.map((item) =>
      item._key === key ? { ...item, [field]: value } : item
    ));
  };

  const removeItem = (key: string) => {
    setItems(items.filter((item) => item._key !== key));
  };

  const totalAmount = items.reduce(
    (sum, item) => sum + item.quantity * item.unit_cost,
    0
  );

  const handleSubmit = (sendAfterCreate: boolean) => {
    if (!supplierName.trim()) {
      toast.error("Nazwa dostawcy jest wymagana");
      return;
    }
    if (items.length === 0) {
      toast.error("Dodaj przynajmniej jedną pozycję");
      return;
    }

    const payload = {
      supplier_id: supplierID || undefined,
      supplier_name: supplierName,
      warehouse_id: warehouseID || undefined,
      expected_delivery_date: expectedDate || undefined,
      currency,
      notes,
      items: items.map(({ _key, ...item }) => item),
    };

    createPO.mutate(payload, {
      onSuccess: (po) => {
        if (sendAfterCreate) {
          sendPO.mutate(po.id, {
            onSuccess: () => {
              toast.success("Zamówienie zakupu zostało utworzone i wysłane");
              router.push(`/purchase-orders/${po.id}`);
            },
            onError: (error) => {
              toast.error(getErrorMessage(error));
              router.push(`/purchase-orders/${po.id}`);
            },
          });
        } else {
          toast.success("Zamówienie zakupu zostało utworzone");
          router.push(`/purchase-orders/${po.id}`);
        }
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-5xl">
        <div className="flex items-center gap-4 mb-6">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push("/purchase-orders")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold tracking-tight">
            Nowe zamówienie zakupu
          </h1>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Dostawca</CardTitle>
              <CardDescription>Wybierz dostawcę lub wpisz nazwę</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Dostawca z listy</Label>
                <Select value={supplierID} onValueChange={handleSupplierChange}>
                  <SelectTrigger>
                    <SelectValue placeholder="Wybierz dostawcę..." />
                  </SelectTrigger>
                  <SelectContent>
                    {suppliers.map((s) => (
                      <SelectItem key={s.id} value={s.id}>
                        {s.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="supplierName">Nazwa dostawcy *</Label>
                <Input
                  id="supplierName"
                  value={supplierName}
                  onChange={(e) => setSupplierName(e.target.value)}
                  placeholder="Nazwa dostawcy"
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Szczegóły</CardTitle>
              <CardDescription>
                Dodatkowe informacje o zamówieniu
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Magazyn docelowy</Label>
                <Select value={warehouseID} onValueChange={setWarehouseID}>
                  <SelectTrigger>
                    <SelectValue placeholder="Wybierz magazyn..." />
                  </SelectTrigger>
                  <SelectContent>
                    {warehouses.map((w) => (
                      <SelectItem key={w.id} value={w.id}>
                        {w.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="expectedDate">Oczekiwana dostawa</Label>
                  <Input
                    id="expectedDate"
                    type="date"
                    value={expectedDate}
                    onChange={(e) => setExpectedDate(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Waluta</Label>
                  <Select value={currency} onValueChange={setCurrency}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="PLN">PLN</SelectItem>
                      <SelectItem value="EUR">EUR</SelectItem>
                      <SelectItem value="USD">USD</SelectItem>
                      <SelectItem value="GBP">GBP</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="notes">Uwagi</Label>
                <Textarea
                  id="notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Notatki do zamówienia..."
                  rows={3}
                />
              </div>
            </CardContent>
          </Card>
        </div>

        <Card className="mt-6">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Pozycje zamówienia</CardTitle>
                <CardDescription>
                  Dodaj produkty do zamówienia zakupu
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowProductSearch(true)}
                >
                  <Search className="h-4 w-4 mr-2" />
                  Z katalogu
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => addItem()}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Dodaj pozycję
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {items.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">
                Dodaj pozycje do zamówienia klikając przyciski powyżej.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Produkt</TableHead>
                    <TableHead>SKU</TableHead>
                    <TableHead className="w-[100px]">Ilość</TableHead>
                    <TableHead className="w-[120px]">Cena jedn.</TableHead>
                    <TableHead className="text-right w-[120px]">Wartość</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item._key}>
                      <TableCell>
                        <Input
                          value={item.product_name}
                          onChange={(e) =>
                            updateItem(item._key, "product_name", e.target.value)
                          }
                          placeholder="Nazwa produktu"
                          className="h-8"
                        />
                      </TableCell>
                      <TableCell>
                        <Input
                          value={item.sku}
                          onChange={(e) =>
                            updateItem(item._key, "sku", e.target.value)
                          }
                          placeholder="SKU"
                          className="h-8"
                        />
                      </TableCell>
                      <TableCell>
                        <Input
                          type="number"
                          min={1}
                          value={item.quantity}
                          onChange={(e) =>
                            updateItem(
                              item._key,
                              "quantity",
                              parseInt(e.target.value) || 1
                            )
                          }
                          className="h-8"
                        />
                      </TableCell>
                      <TableCell>
                        <Input
                          type="number"
                          min={0}
                          step={0.01}
                          value={item.unit_cost}
                          onChange={(e) =>
                            updateItem(
                              item._key,
                              "unit_cost",
                              parseFloat(e.target.value) || 0
                            )
                          }
                          className="h-8"
                        />
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {(item.quantity * item.unit_cost).toFixed(2)}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => removeItem(item._key)}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {items.length > 0 && (
              <div className="flex justify-end pt-4 border-t mt-4">
                <div className="text-right">
                  <p className="text-sm text-muted-foreground">Suma:</p>
                  <p className="text-lg font-bold">
                    {totalAmount.toFixed(2)} {currency}
                  </p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="flex gap-2 mt-6 justify-end">
          <Button
            variant="outline"
            onClick={() => router.push("/purchase-orders")}
          >
            Anuluj
          </Button>
          <Button
            variant="secondary"
            disabled={createPO.isPending}
            onClick={() => handleSubmit(false)}
          >
            {createPO.isPending ? "Tworzenie..." : "Zapisz jako szkic"}
          </Button>
          <Button
            disabled={createPO.isPending || sendPO.isPending}
            onClick={() => handleSubmit(true)}
          >
            {createPO.isPending || sendPO.isPending
              ? "Tworzenie..."
              : "Zapisz i wyślij"}
          </Button>
        </div>
      </div>

      <Dialog open={showProductSearch} onOpenChange={setShowProductSearch}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Wybierz produkt z katalogu</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                value={productSearch}
                onChange={(e) => setProductSearch(e.target.value)}
                placeholder="Szukaj produktu..."
                className="pl-9"
              />
            </div>
            <div className="max-h-64 overflow-auto border rounded-md">
              {productsData?.items && productsData.items.length > 0 ? (
                productsData.items.map((product) => (
                  <button
                    key={product.id}
                    type="button"
                    onClick={() =>
                      addItem({
                        id: product.id,
                        name: product.name,
                        sku: product.sku,
                        price: product.price,
                      })
                    }
                    className="w-full text-left px-3 py-2 text-sm hover:bg-muted/50 transition-colors"
                  >
                    <div className="font-medium">{product.name}</div>
                    <div className="text-xs text-muted-foreground">
                      SKU: {product.sku || "---"} | Cena:{" "}
                      {product.price != null ? `${product.price} PLN` : "---"}
                    </div>
                  </button>
                ))
              ) : (
                <p className="text-sm text-muted-foreground text-center py-4">
                  {productSearch
                    ? "Brak wyników"
                    : "Wpisz nazwę, aby wyszukać"}
                </p>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </AdminGuard>
  );
}
