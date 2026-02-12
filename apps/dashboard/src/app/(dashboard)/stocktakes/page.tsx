"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ClipboardCheck, Plus } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useStocktakes } from "@/hooks/use-stocktakes";
import { useWarehouses } from "@/hooks/use-warehouses";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Stocktake } from "@/types/api";

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

function StocktakeStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant="outline" className={statusVariants[status] || ""}>
      {statusLabels[status] || status}
    </Badge>
  );
}

export default function StocktakesPage() {
  const router = useRouter();
  const [warehouseFilter, setWarehouseFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");

  const { data: warehousesData } = useWarehouses({ limit: 100 });
  const warehouses = warehousesData?.items ?? [];

  const { data, isLoading, isError, refetch } = useStocktakes({
    warehouse_id: warehouseFilter && warehouseFilter !== "all" ? warehouseFilter : undefined,
    status: statusFilter && statusFilter !== "all" ? statusFilter : undefined,
    limit: 50,
  });

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const stocktakes = data?.items ?? [];

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Inwentaryzacja</h1>
          <p className="text-muted-foreground">
            Zarządzaj procesami inwentaryzacji magazynów
          </p>
        </div>
        <Button onClick={() => router.push("/stocktakes/new")}>
          <Plus className="h-4 w-4 mr-2" />
          Nowa inwentaryzacja
        </Button>
      </div>

      <div className="flex gap-4 mb-4">
        <Select value={warehouseFilter} onValueChange={setWarehouseFilter}>
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="Wszystkie magazyny" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie magazyny</SelectItem>
            {warehouses.map((w) => (
              <SelectItem key={w.id} value={w.id}>
                {w.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Wszystkie statusy" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie statusy</SelectItem>
            <SelectItem value="draft">Szkic</SelectItem>
            <SelectItem value="in_progress">W trakcie</SelectItem>
            <SelectItem value="completed">Zakończona</SelectItem>
            <SelectItem value="cancelled">Anulowana</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {stocktakes.length === 0 ? (
        <EmptyState
          icon={ClipboardCheck}
          title="Brak inwentaryzacji"
          description="Utwórz nową inwentaryzację, aby rozpocząć liczenie stanów magazynowych."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Magazyn</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Pozycje</TableHead>
                <TableHead>Rozbieżności</TableHead>
                <TableHead>Utworzono</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {stocktakes.map((stocktake: Stocktake) => {
                const warehouse = warehouses.find(
                  (w) => w.id === stocktake.warehouse_id
                );
                return (
                  <TableRow
                    key={stocktake.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => router.push(`/stocktakes/${stocktake.id}`)}
                  >
                    <TableCell className="font-medium">
                      {stocktake.name}
                    </TableCell>
                    <TableCell>{warehouse?.name || "---"}</TableCell>
                    <TableCell>
                      <StocktakeStatusBadge status={stocktake.status} />
                    </TableCell>
                    <TableCell>
                      {stocktake.stats ? (
                        <span>
                          {stocktake.stats.counted_items}/
                          {stocktake.stats.total_items}
                        </span>
                      ) : (
                        "---"
                      )}
                    </TableCell>
                    <TableCell>
                      {stocktake.stats?.discrepancies != null ? (
                        <span
                          className={
                            stocktake.stats.discrepancies > 0
                              ? "text-red-600 font-medium"
                              : ""
                          }
                        >
                          {stocktake.stats.discrepancies}
                        </span>
                      ) : (
                        "---"
                      )}
                    </TableCell>
                    <TableCell>{formatDate(stocktake.created_at)}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </AdminGuard>
  );
}
