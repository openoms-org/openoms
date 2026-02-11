"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateWarehouseDocument } from "@/hooks/use-warehouse-documents";
import { useWarehouses } from "@/hooks/use-warehouses";
import { useSuppliers } from "@/hooks/use-suppliers";
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
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import Link from "next/link";
import type { CreateWarehouseDocItemRequest } from "@/types/api";

interface ItemRow {
  product_id: string;
  variant_id?: string;
  quantity: number;
  unit_price?: number;
  notes?: string;
}

export default function NewWarehouseDocumentPage() {
  const router = useRouter();
  const createDocument = useCreateWarehouseDocument();
  const { data: warehousesData } = useWarehouses({ limit: 100 });
  const { data: suppliersData } = useSuppliers({ limit: 100 });

  const [docType, setDocType] = useState<string>("");
  const [warehouseId, setWarehouseId] = useState<string>("");
  const [targetWarehouseId, setTargetWarehouseId] = useState<string>("");
  const [supplierId, setSupplierId] = useState<string>("");
  const [notes, setNotes] = useState<string>("");
  const [items, setItems] = useState<ItemRow[]>([
    { product_id: "", quantity: 1 },
  ]);

  const warehouses = warehousesData?.items ?? [];
  const suppliers = suppliersData?.items ?? [];

  const addItem = () => {
    setItems([...items, { product_id: "", quantity: 1 }]);
  };

  const removeItem = (index: number) => {
    if (items.length <= 1) return;
    setItems(items.filter((_, i) => i !== index));
  };

  const updateItem = (index: number, field: keyof ItemRow, value: string | number) => {
    const newItems = [...items];
    newItems[index] = { ...newItems[index], [field]: value };
    setItems(newItems);
  };

  const handleSubmit = () => {
    if (!docType || !warehouseId || items.some((i) => !i.product_id || i.quantity <= 0)) {
      toast.error("Wypelnij wszystkie wymagane pola");
      return;
    }

    const docItems: CreateWarehouseDocItemRequest[] = items.map((i) => ({
      product_id: i.product_id,
      variant_id: i.variant_id || undefined,
      quantity: i.quantity,
      unit_price: i.unit_price || undefined,
      notes: i.notes || undefined,
    }));

    createDocument.mutate(
      {
        document_type: docType as "PZ" | "WZ" | "MM",
        warehouse_id: warehouseId,
        target_warehouse_id: docType === "MM" ? targetWarehouseId || undefined : undefined,
        supplier_id: docType === "PZ" ? supplierId || undefined : undefined,
        notes: notes || undefined,
        items: docItems,
      },
      {
        onSuccess: (doc) => {
          toast.success(`Dokument ${doc.document_number} zostal utworzony`);
          router.push(`/settings/warehouse-documents/${doc.id}`);
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="mb-6">
        <Button variant="ghost" size="sm" asChild className="mb-4">
          <Link href="/settings/warehouse-documents">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Powrot do listy
          </Link>
        </Button>
        <h1 className="text-2xl font-bold tracking-tight">
          Nowy dokument magazynowy
        </h1>
        <p className="text-muted-foreground">
          Utworz dokument PZ, WZ lub MM
        </p>
      </div>

      <div className="max-w-2xl space-y-6">
        <div className="space-y-2">
          <Label>Typ dokumentu *</Label>
          <Select value={docType} onValueChange={setDocType}>
            <SelectTrigger>
              <SelectValue placeholder="Wybierz typ" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="PZ">PZ - Przyjecie zewnetrzne</SelectItem>
              <SelectItem value="WZ">WZ - Wydanie zewnetrzne</SelectItem>
              <SelectItem value="MM">MM - Przesuniecie miedzymagazynowe</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>
            {docType === "MM" ? "Magazyn zrodlowy *" : "Magazyn *"}
          </Label>
          <Select value={warehouseId} onValueChange={setWarehouseId}>
            <SelectTrigger>
              <SelectValue placeholder="Wybierz magazyn" />
            </SelectTrigger>
            <SelectContent>
              {warehouses.map((w) => (
                <SelectItem key={w.id} value={w.id}>
                  {w.name} {w.code ? `(${w.code})` : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {docType === "MM" && (
          <div className="space-y-2">
            <Label>Magazyn docelowy *</Label>
            <Select
              value={targetWarehouseId}
              onValueChange={setTargetWarehouseId}
            >
              <SelectTrigger>
                <SelectValue placeholder="Wybierz magazyn docelowy" />
              </SelectTrigger>
              <SelectContent>
                {warehouses
                  .filter((w) => w.id !== warehouseId)
                  .map((w) => (
                    <SelectItem key={w.id} value={w.id}>
                      {w.name} {w.code ? `(${w.code})` : ""}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {docType === "PZ" && (
          <div className="space-y-2">
            <Label>Dostawca</Label>
            <Select value={supplierId} onValueChange={setSupplierId}>
              <SelectTrigger>
                <SelectValue placeholder="Wybierz dostawce (opcjonalnie)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Brak</SelectItem>
                {suppliers.map((s) => (
                  <SelectItem key={s.id} value={s.id}>
                    {s.name} {s.code ? `(${s.code})` : ""}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <div className="space-y-2">
          <Label>Uwagi</Label>
          <Textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Opcjonalne uwagi do dokumentu"
            rows={3}
          />
        </div>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label className="text-base font-semibold">Pozycje *</Label>
            <Button variant="outline" size="sm" onClick={addItem}>
              <Plus className="h-4 w-4 mr-1" />
              Dodaj pozycje
            </Button>
          </div>

          {items.map((item, index) => (
            <div
              key={index}
              className="flex gap-3 items-end rounded-md border p-3"
            >
              <div className="flex-1 space-y-1">
                <Label className="text-xs">ID Produktu</Label>
                <Input
                  value={item.product_id}
                  onChange={(e) =>
                    updateItem(index, "product_id", e.target.value)
                  }
                  placeholder="UUID produktu"
                />
              </div>
              <div className="w-24 space-y-1">
                <Label className="text-xs">Ilosc</Label>
                <Input
                  type="number"
                  min={1}
                  value={item.quantity}
                  onChange={(e) =>
                    updateItem(index, "quantity", parseInt(e.target.value) || 1)
                  }
                />
              </div>
              <div className="w-28 space-y-1">
                <Label className="text-xs">Cena jedn.</Label>
                <Input
                  type="number"
                  step="0.01"
                  min={0}
                  value={item.unit_price ?? ""}
                  onChange={(e) =>
                    updateItem(
                      index,
                      "unit_price",
                      parseFloat(e.target.value) || 0
                    )
                  }
                  placeholder="0.00"
                />
              </div>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => removeItem(index)}
                disabled={items.length <= 1}
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          ))}
        </div>

        <div className="flex gap-3 pt-4">
          <Button variant="outline" asChild>
            <Link href="/settings/warehouse-documents">Anuluj</Link>
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={
              createDocument.isPending ||
              !docType ||
              !warehouseId ||
              items.some((i) => !i.product_id)
            }
          >
            {createDocument.isPending ? "Tworzenie..." : "Utworz dokument"}
          </Button>
        </div>
      </div>
    </AdminGuard>
  );
}
