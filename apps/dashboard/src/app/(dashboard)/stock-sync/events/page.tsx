"use client";

import { useState } from "react";
import { ArrowLeft, RefreshCw } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { EmptyState } from "@/components/shared/empty-state";
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
import { useStockSyncEvents } from "@/hooks/use-stock-sync";
import Link from "next/link";
import { useTranslations } from "next-intl";

const triggerTypeVariants: Record<string, string> = {
  order_placed:
    "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  order_cancelled:
    "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
  stock_adjusted:
    "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  manual:
    "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
  recount:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
};

export default function StockSyncEventsPage() {
  const t = useTranslations("stockSync");

  const triggerTypeLabels: Record<string, string> = {
    order_placed: t("triggerOrderPlaced"),
    order_cancelled: t("triggerOrderCancelled"),
    stock_adjusted: t("triggerStockAdjusted"),
    manual: t("triggerManualSync"),
    recount: t("triggerRecount"),
  };

  const [triggerFilter, setTriggerFilter] = useState<string>("");
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const { data, isLoading, refetch } = useStockSyncEvents({
    limit,
    offset,
    trigger_type: triggerFilter || undefined,
  });

  const events = data?.items ?? [];
  const total = data?.total ?? 0;
  const hasNext = offset + limit < total;
  const hasPrev = offset > 0;

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href="/stock-sync">
              <Button variant="ghost" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <div>
              <h1 className="text-2xl font-bold tracking-tight">
                {t("historiaZdarzenSynchronizacji")}
              </h1>
              <p className="text-muted-foreground">
                {t("dziennikZmianStanowMagazynowychISynchronizacjiZ")}
              </p>
            </div>
          </div>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t("refresh")}
          </Button>
        </div>

        {/* Filters */}
        <div className="flex gap-4">
          <Select
            value={triggerFilter}
            onValueChange={(v) => {
              setTriggerFilter(v === "all" ? "" : v);
              setOffset(0);
            }}
          >
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder={t("filterByEventType")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("allTypes")}</SelectItem>
              {Object.entries(triggerTypeLabels).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {isLoading ? (
          <LoadingSkeleton />
        ) : events.length === 0 ? (
          <EmptyState
            icon={RefreshCw}
            title={t("brakZdarzenSynchronizacji")}
            description={t("zdarzeniaPojawiaSiePoZmianachStanowMagazynowych")}
          />
        ) : (
          <>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("date")}</TableHead>
                    <TableHead>{t("eventType")}</TableHead>
                    <TableHead>{t("sku")}</TableHead>
                    <TableHead className="text-right">{t("previousStock")}</TableHead>
                    <TableHead className="text-right">{t("newStock")}</TableHead>
                    <TableHead className="text-right">{t("dostepny")}</TableHead>
                    <TableHead className="text-right">
                      {t("kanałyPowiadomione")}
                    </TableHead>
                    <TableHead className="text-right">{t("błedy1")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {events.map((event) => (
                    <TableRow key={event.id}>
                      <TableCell className="whitespace-nowrap">
                        {formatDate(event.created_at)}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="outline"
                          className={
                            triggerTypeVariants[event.trigger_type] || ""
                          }
                        >
                          {triggerTypeLabels[event.trigger_type] ||
                            event.trigger_type}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {event.sku || "-"}
                      </TableCell>
                      <TableCell className="text-right">
                        {event.old_quantity}
                      </TableCell>
                      <TableCell className="text-right">
                        <span
                          className={
                            event.new_quantity < event.old_quantity
                              ? "text-red-600"
                              : event.new_quantity > event.old_quantity
                                ? "text-green-600"
                                : ""
                          }
                        >
                          {event.new_quantity}
                        </span>
                      </TableCell>
                      <TableCell className="text-right font-semibold">
                        {event.available_quantity}
                      </TableCell>
                      <TableCell className="text-right">
                        {event.channels_notified}
                      </TableCell>
                      <TableCell className="text-right">
                        {event.channels_failed > 0 ? (
                          <span className="text-red-600 font-semibold">
                            {event.channels_failed}
                          </span>
                        ) : (
                          <span className="text-muted-foreground">0</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {t("showingRange", { from: offset + 1, to: Math.min(offset + limit, total), total })}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!hasPrev}
                  onClick={() => setOffset(Math.max(0, offset - limit))}
                >
                  {t("previous")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!hasNext}
                  onClick={() => setOffset(offset + limit)}
                >
                  {t("next")}
                </Button>
              </div>
            </div>
          </>
        )}
      </div>
    </AdminGuard>
  );
}
