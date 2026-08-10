"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ClipboardCheck, Plus } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useStocktakes } from "@/hooks/use-stocktakes";
import { useAllWarehouses } from "@/hooks/use-warehouses";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { STOCKTAKE_STATUSES } from "@/lib/constants";
import { enumLabel } from "@/lib/i18n-fallback";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/status-badge";
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
import { useTranslations } from "next-intl";

export default function StocktakesPage() {
  const t = useTranslations("stocktakes");
  const ts = useTranslations("statuses");
  const router = useRouter();
  const [warehouseFilter, setWarehouseFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");

  const { data: warehousesData } = useAllWarehouses();
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
          <h1 className="text-2xl font-bold tracking-tight">{t("stocktake")}</h1>
          <p className="text-muted-foreground">
            {t("zarzadzajProcesamiInwentaryzacjiMagazynow")}
          </p>
        </div>
        <Button onClick={() => router.push("/stocktakes/new")}>
          <Plus className="h-4 w-4 mr-2" />
          {t("newStocktake")}
        </Button>
      </div>

      <div className="flex gap-4 mb-4">
        <Select value={warehouseFilter} onValueChange={setWarehouseFilter}>
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder={t("allWarehouses")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allWarehouses")}</SelectItem>
            {warehouses.map((w) => (
              <SelectItem key={w.id} value={w.id}>
                {w.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder={t("allStatuses")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allStatuses")}</SelectItem>
            {Object.keys(STOCKTAKE_STATUSES).map((key) => (
              <SelectItem key={key} value={key}>
                {enumLabel(ts, "stocktake", key)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

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
            {t("retry")}
          </Button>
        </div>
      )}

      {stocktakes.length === 0 ? (
        <EmptyState
          icon={ClipboardCheck}
          title={t("noStocktakes")}
          description={t("utworzNowaInwentaryzacjeAbyRozpoczacLiczenieStanow")}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("name")}</TableHead>
                <TableHead>{t("warehouse")}</TableHead>
                <TableHead>{t("status")}</TableHead>
                <TableHead>{t("items")}</TableHead>
                <TableHead>{t("rozbieznosci")}</TableHead>
                <TableHead>{t("created")}</TableHead>
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
                      <StatusBadge
                        status={stocktake.status}
                        statusMap={STOCKTAKE_STATUSES}
                        translationPrefix="stocktake"
                      />
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
