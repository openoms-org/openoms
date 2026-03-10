"use client";

import { useState } from "react";
import Link from "next/link";
import { Factory } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { useDropshipOrders } from "@/hooks/use-dropship-orders";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { useTranslations } from "next-intl";

export default function DropshipOrdersPage() {
  const t = useTranslations("dropshipOrders");

  const DROPSHIP_STATUSES: Record<string, { label: string; color: string }> = {
    pending: { label: t("status.pending"), color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300" },
    sent: { label: t("status.sent"), color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300" },
    confirmed: { label: t("status.confirmed"), color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300" },
    shipped: { label: t("status.shipped"), color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
    delivered: { label: t("status.delivered"), color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300" },
    cancelled: { label: t("status.cancelled"), color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
  };
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [page, setPage] = useState(0);
  const limit = 20;

  const { data, isLoading } = useDropshipOrders({
    status: statusFilter || undefined,
    limit,
    offset: page * limit,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("title")}</h1>
          <p className="text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            setStatusFilter(v === "__all__" ? "" : v);
            setPage(0);
          }}
        >
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder={t("filterByStatus")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t("allStatuses")}</SelectItem>
            {Object.entries(DROPSHIP_STATUSES).map(([key, { label }]) => (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Factory className="h-5 w-5" />
            {t("dropshipOrders")}
            {data && (
              <span className="text-sm font-normal text-muted-foreground">
                ({data.total})
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : data && data.items.length > 0 ? (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t("supplier")}</TableHead>
                    <TableHead>{t("order.title")}</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>{t("cost")}</TableHead>
                    <TableHead>{t("trackingNumber")}</TableHead>
                    <TableHead>{t("createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.items.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell>
                        <Link
                          href={`/dropship-orders/${d.id}`}
                          className="font-medium text-primary hover:underline"
                        >
                          {shortId(d.id)}
                        </Link>
                      </TableCell>
                      <TableCell className="font-medium">
                        {d.supplier_name}
                      </TableCell>
                      <TableCell>
                        <Link
                          href={`/orders/${d.order_id}`}
                          className="text-primary hover:underline"
                        >
                          {shortId(d.order_id)}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          status={d.status}
                          statusMap={DROPSHIP_STATUSES}
                        />
                      </TableCell>
                      <TableCell>
                        {formatCurrency(d.total_cost, d.currency)}
                      </TableCell>
                      <TableCell>
                        {d.tracking_number || "---"}
                      </TableCell>
                      <TableCell>{formatDate(d.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="mt-4 flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  {t("showing", { from: page * limit + 1, to: Math.min((page + 1) * limit, data.total), total: data.total })}
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page === 0}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    {t("previous")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={(page + 1) * limit >= data.total}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    {t("next")}
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Factory className="h-12 w-12 text-muted-foreground/50 mb-3" />
              <p className="text-muted-foreground">
                {t("emptyTitle")}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                {t("emptyDescription")}
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
