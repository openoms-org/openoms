"use client";

import { useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  Play,
  CheckCircle2,
  XCircle,
  Trash2,
  ArrowLeft,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  useStocktake,
  useStocktakeItems,
  useStartStocktake,
  useCompleteStocktake,
  useCancelStocktake,
  useDeleteStocktake,
  useRecordCount,
} from "@/hooks/use-stocktakes";
import { useWarehouses } from "@/hooks/use-warehouses";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
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
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Tabs,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import type { StocktakeItem } from "@/types/api";

const statusLabels: Record<string, string> = {
  draft: "Szkic",
  in_progress: "W trakcie",
  completed: "Zakończona",
  cancelled: "Anulowana",
};

const statusVariants: Record<string, string> = {
  draft: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
  in_progress:
    "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  completed:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  cancelled: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
};

export default function StocktakeDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const { data: stocktake, isLoading, isError } = useStocktake(id);
  const { data: warehousesData } = useWarehouses({ limit: 100 });
  const warehouses = warehousesData?.items ?? [];

  const [itemFilter, setItemFilter] = useState<
    "all" | "uncounted" | "discrepancies"
  >("all");
  const [offset, setOffset] = useState(0);
  const limit = 50;

  const {
    data: itemsData,
    isLoading: itemsLoading,
  } = useStocktakeItems(id, { filter: itemFilter, limit, offset });

  const startStocktake = useStartStocktake();
  const completeStocktake = useCompleteStocktake();
  const cancelStocktake = useCancelStocktake();
  const deleteStocktake = useDeleteStocktake();
  const recordCount = useRecordCount(id);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showCompleteConfirm, setShowCompleteConfirm] = useState(false);
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  // Inline edit state
  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");

  const handleStart = useCallback(() => {
    startStocktake.mutate(id, {
      onSuccess: () => toast.success("Inwentaryzacja została rozpoczęta"),
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  }, [id, startStocktake]);

  const handleComplete = useCallback(() => {
    completeStocktake.mutate(id, {
      onSuccess: () => {
        toast.success("Inwentaryzacja została zakończona");
        setShowCompleteConfirm(false);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
        setShowCompleteConfirm(false);
      },
    });
  }, [id, completeStocktake]);

  const handleCancel = useCallback(() => {
    cancelStocktake.mutate(id, {
      onSuccess: () => {
        toast.success("Inwentaryzacja została anulowana");
        setShowCancelConfirm(false);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
        setShowCancelConfirm(false);
      },
    });
  }, [id, cancelStocktake]);

  const handleDelete = useCallback(() => {
    deleteStocktake.mutate(id, {
      onSuccess: () => {
        toast.success("Inwentaryzacja została usunięta");
        router.push("/stocktakes");
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  }, [id, deleteStocktake, router]);

  const handleCountSubmit = useCallback(
    (itemId: string, value: string) => {
      const qty = parseInt(value, 10);
      if (isNaN(qty) || qty < 0) {
        toast.error("Podaj poprawną ilość");
        return;
      }

      recordCount.mutate(
        { itemId, data: { counted_quantity: qty } },
        {
          onSuccess: () => {
            setEditingItemId(null);
            setEditValue("");
          },
          onError: (error) => toast.error(getErrorMessage(error)),
        }
      );
    },
    [recordCount]
  );

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError || !stocktake) {
    return (
      <AdminGuard>
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            Nie znaleziono inwentaryzacji.
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => router.push("/stocktakes")}
          >
            Powrót do listy
          </Button>
        </div>
      </AdminGuard>
    );
  }

  const warehouse = warehouses.find((w) => w.id === stocktake.warehouse_id);
  const stats = stocktake.stats;
  const items = itemsData?.items ?? [];
  const totalItems = itemsData?.total ?? 0;
  const isInProgress = stocktake.status === "in_progress";
  const isDraft = stocktake.status === "draft";
  const isCompleted = stocktake.status === "completed";

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <Button
              variant="ghost"
              size="sm"
              className="mb-2 -ml-2"
              onClick={() => router.push("/stocktakes")}
            >
              <ArrowLeft className="h-4 w-4 mr-1" />
              Powrót
            </Button>
            <h1 className="text-2xl font-bold tracking-tight">
              {stocktake.name}
            </h1>
            <div className="flex items-center gap-3 mt-1">
              <Badge
                variant="outline"
                className={statusVariants[stocktake.status] || ""}
              >
                {statusLabels[stocktake.status] || stocktake.status}
              </Badge>
              <span className="text-muted-foreground">
                {warehouse?.name || "---"}
              </span>
              <span className="text-muted-foreground">
                {formatDate(stocktake.created_at)}
              </span>
            </div>
          </div>

          <div className="flex gap-2">
            {isDraft && (
              <>
                <Button onClick={handleStart} disabled={startStocktake.isPending}>
                  <Play className="h-4 w-4 mr-2" />
                  {startStocktake.isPending ? "Uruchamianie..." : "Rozpocznij"}
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => setShowDeleteConfirm(true)}
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Usuń
                </Button>
              </>
            )}
            {isInProgress && (
              <>
                <Button
                  onClick={() => setShowCompleteConfirm(true)}
                  disabled={
                    completeStocktake.isPending ||
                    (stats != null &&
                      stats.counted_items < stats.total_items)
                  }
                >
                  <CheckCircle2 className="h-4 w-4 mr-2" />
                  Zakończ
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowCancelConfirm(true)}
                >
                  <XCircle className="h-4 w-4 mr-2" />
                  Anuluj
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Stats */}
        {stats && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Pozycje</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{stats.total_items}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Zliczone</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats.counted_items}/{stats.total_items}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Rozbieżności</CardDescription>
              </CardHeader>
              <CardContent>
                <p
                  className={`text-2xl font-bold ${
                    stats.discrepancies > 0 ? "text-red-600" : ""
                  }`}
                >
                  {stats.discrepancies}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Nadwyżki</CardDescription>
              </CardHeader>
              <CardContent>
                <p
                  className={`text-2xl font-bold ${
                    stats.surplus_count > 0 ? "text-green-600" : ""
                  }`}
                >
                  {stats.surplus_count}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Niedobory</CardDescription>
              </CardHeader>
              <CardContent>
                <p
                  className={`text-2xl font-bold ${
                    stats.shortage_count > 0 ? "text-red-600" : ""
                  }`}
                >
                  {stats.shortage_count}
                </p>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Notes */}
        {stocktake.notes && (
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Uwagi</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm">{stocktake.notes}</p>
            </CardContent>
          </Card>
        )}

        {/* Items */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Pozycje</h2>
            <Tabs
              value={itemFilter}
              onValueChange={(v) => {
                setItemFilter(v as typeof itemFilter);
                setOffset(0);
              }}
            >
              <TabsList>
                <TabsTrigger value="all">Wszystkie</TabsTrigger>
                <TabsTrigger value="uncounted">Niezliczone</TabsTrigger>
                <TabsTrigger value="discrepancies">Rozbieżności</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {itemsLoading ? (
            <LoadingSkeleton />
          ) : items.length === 0 ? (
            <p className="text-muted-foreground text-center py-8">
              Brak pozycji{" "}
              {itemFilter === "uncounted"
                ? "do zliczenia"
                : itemFilter === "discrepancies"
                ? "z rozbieżnościami"
                : ""}
            </p>
          ) : (
            <>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Produkt</TableHead>
                      <TableHead>SKU</TableHead>
                      <TableHead className="text-right">Oczekiwane</TableHead>
                      <TableHead className="text-right">Zliczone</TableHead>
                      <TableHead className="text-right">Różnica</TableHead>
                      <TableHead>Uwagi</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item: StocktakeItem) => (
                      <TableRow key={item.id}>
                        <TableCell className="font-medium">
                          {item.product_name || "---"}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {item.product_sku || "---"}
                        </TableCell>
                        <TableCell className="text-right">
                          {item.expected_quantity}
                        </TableCell>
                        <TableCell className="text-right">
                          {isInProgress ? (
                            editingItemId === item.id ? (
                              <div className="flex items-center justify-end gap-1">
                                <Input
                                  type="number"
                                  min={0}
                                  className="w-20 h-8 text-right"
                                  value={editValue}
                                  onChange={(e) => setEditValue(e.target.value)}
                                  onKeyDown={(e) => {
                                    if (e.key === "Enter") {
                                      handleCountSubmit(item.id, editValue);
                                    }
                                    if (e.key === "Escape") {
                                      setEditingItemId(null);
                                    }
                                  }}
                                  autoFocus
                                  disabled={recordCount.isPending}
                                />
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  className="h-8 px-2"
                                  onClick={() =>
                                    handleCountSubmit(item.id, editValue)
                                  }
                                  disabled={recordCount.isPending}
                                >
                                  OK
                                </Button>
                              </div>
                            ) : (
                              <button
                                className="px-2 py-1 rounded hover:bg-muted transition-colors text-right w-full cursor-pointer"
                                onClick={() => {
                                  setEditingItemId(item.id);
                                  setEditValue(
                                    item.counted_quantity != null
                                      ? String(item.counted_quantity)
                                      : ""
                                  );
                                }}
                              >
                                {item.counted_quantity != null ? (
                                  item.counted_quantity
                                ) : (
                                  <span className="text-muted-foreground italic">
                                    kliknij
                                  </span>
                                )}
                              </button>
                            )
                          ) : item.counted_quantity != null ? (
                            item.counted_quantity
                          ) : (
                            <span className="text-muted-foreground">---</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          {item.counted_quantity != null ? (
                            <span
                              className={
                                item.difference > 0
                                  ? "text-green-600 font-medium"
                                  : item.difference < 0
                                  ? "text-red-600 font-medium"
                                  : ""
                              }
                            >
                              {item.difference > 0
                                ? `+${item.difference}`
                                : item.difference}
                            </span>
                          ) : (
                            <span className="text-muted-foreground">---</span>
                          )}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {item.notes || ""}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* Pagination */}
              {totalItems > limit && (
                <div className="flex items-center justify-between mt-4">
                  <p className="text-sm text-muted-foreground">
                    Wyświetlono {offset + 1}-{Math.min(offset + limit, totalItems)}{" "}
                    z {totalItems}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={offset === 0}
                      onClick={() => setOffset(Math.max(0, offset - limit))}
                    >
                      Poprzednia
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={offset + limit >= totalItems}
                      onClick={() => setOffset(offset + limit)}
                    >
                      Następna
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="Usuń inwentaryzację"
        description="Czy na pewno chcesz usunąć tę inwentaryzację? Operacji nie można cofnąć."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteStocktake.isPending}
      />

      {/* Complete confirmation */}
      <Dialog open={showCompleteConfirm} onOpenChange={setShowCompleteConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Zakończ inwentaryzację</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz zakończyć tę inwentaryzację? Zostaną
              automatycznie wygenerowane dokumenty korygujące (PZ dla nadwyżek,
              WZ dla niedoborów) i zaktualizowane stany magazynowe.
            </DialogDescription>
          </DialogHeader>
          {stats && stats.discrepancies > 0 && (
            <div className="rounded-md border p-4 bg-muted/50">
              <p className="text-sm font-medium mb-2">Podsumowanie rozbieżności:</p>
              <ul className="text-sm space-y-1">
                <li className="text-green-600">
                  Nadwyżki: {stats.surplus_count} pozycji
                </li>
                <li className="text-red-600">
                  Niedobory: {stats.shortage_count} pozycji
                </li>
              </ul>
            </div>
          )}
          {stats && stats.discrepancies === 0 && (
            <p className="text-sm text-muted-foreground">
              Nie wykryto żadnych rozbieżności. Stany magazynowe nie zostaną zmienione.
            </p>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCompleteConfirm(false)}
            >
              Anuluj
            </Button>
            <Button
              onClick={handleComplete}
              disabled={completeStocktake.isPending}
            >
              {completeStocktake.isPending
                ? "Kończenie..."
                : "Zakończ inwentaryzację"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Cancel confirmation */}
      <ConfirmDialog
        open={showCancelConfirm}
        onOpenChange={setShowCancelConfirm}
        title="Anuluj inwentaryzację"
        description="Czy na pewno chcesz anulować tę inwentaryzację? Zliczone dane zostaną zachowane, ale stany magazynowe nie będą zmienione."
        confirmLabel="Anuluj inwentaryzację"
        variant="destructive"
        onConfirm={handleCancel}
        isLoading={cancelStocktake.isPending}
      />
    </AdminGuard>
  );
}
