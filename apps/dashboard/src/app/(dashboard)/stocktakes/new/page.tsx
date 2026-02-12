"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWarehouses } from "@/hooks/use-warehouses";
import { useCreateStocktake } from "@/hooks/use-stocktakes";
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

export default function NewStocktakePage() {
  const router = useRouter();
  const { data: warehousesData, isLoading: warehousesLoading } = useWarehouses({
    limit: 100,
  });
  const createStocktake = useCreateStocktake();

  const [warehouseId, setWarehouseId] = useState("");
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");

  const warehouses = warehousesData?.items ?? [];

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!warehouseId || !name.trim()) {
      toast.error("Wypełnij wymagane pola");
      return;
    }

    createStocktake.mutate(
      {
        warehouse_id: warehouseId,
        name: name.trim(),
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: (data) => {
          toast.success("Inwentaryzacja została utworzona");
          router.push(`/stocktakes/${data.id}`);
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
        <h1 className="text-2xl font-bold tracking-tight">
          Nowa inwentaryzacja
        </h1>
        <p className="text-muted-foreground">
          Utwórz nową inwentaryzację dla wybranego magazynu
        </p>
      </div>

      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>Dane inwentaryzacji</CardTitle>
          <CardDescription>
            Po utworzeniu inwentaryzacji, do pozycji zostaną automatycznie
            dodane wszystkie produkty z wybranego magazynu wraz z ich aktualnymi
            stanami magazynowymi.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="warehouse">Magazyn *</Label>
              <Select value={warehouseId} onValueChange={setWarehouseId}>
                <SelectTrigger id="warehouse">
                  <SelectValue placeholder="Wybierz magazyn..." />
                </SelectTrigger>
                <SelectContent>
                  {warehouses.map((w) => (
                    <SelectItem key={w.id} value={w.id}>
                      {w.name}
                      {w.code ? ` (${w.code})` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="name">Nazwa *</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="np. Inwentaryzacja Q1 2025"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes">Uwagi</Label>
              <Textarea
                id="notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Opcjonalne uwagi..."
                rows={3}
              />
            </div>

            <div className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => router.push("/stocktakes")}
              >
                Anuluj
              </Button>
              <Button
                type="submit"
                disabled={
                  !warehouseId ||
                  !name.trim() ||
                  createStocktake.isPending ||
                  warehousesLoading
                }
              >
                {createStocktake.isPending
                  ? "Tworzenie..."
                  : "Utwórz inwentaryzację"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </AdminGuard>
  );
}
