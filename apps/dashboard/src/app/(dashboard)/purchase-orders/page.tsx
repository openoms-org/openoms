"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ClipboardList, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  usePurchaseOrders,
  useDeletePurchaseOrder,
} from "@/hooks/use-purchase-orders";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { StatusBadge } from "@/components/shared/status-badge";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { PURCHASE_ORDER_STATUSES } from "@/lib/constants";
import { Button } from "@/components/ui/button";
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
import { useTranslations } from "next-intl";

export default function PurchaseOrdersPage() {
  const t = useTranslations("purchaseOrders");
  const tc = useTranslations("common");
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const { data, isLoading, isError, refetch } = usePurchaseOrders({
    status: statusFilter === "all" ? undefined : statusFilter,
    limit: 50,
  });
  const deletePO = useDeletePurchaseOrder();
  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const orders = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deletePO.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("purchaseOrderDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <PageHeader
        title={t("purchaseOrders")}
        description={t("zarzadzajZamowieniamiZakupuDoDostawcow")}
        action={{ label: t("newOrder"), href: "/purchase-orders/new" }}
      />

      <div className="flex items-center gap-4 mb-4">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie statusy</SelectItem>
            <SelectItem value="draft">Szkic</SelectItem>
            <SelectItem value="sent">{t("order.shipped")}</SelectItem>
            <SelectItem value="partially_received">{t("purchaseOrder.partially_received")}</SelectItem>
            <SelectItem value="received">Odebrane</SelectItem>
            <SelectItem value="cancelled">Anulowane</SelectItem>
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

      {orders.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title={t("brakZamowienZakupu")}
          description={t("utworzPierwszeZamowienieZakupuAbyRozpoczacZamawian")}
          action={{ label: t("newOrder"), href: "/purchase-orders/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Numer PO</TableHead>
                <TableHead>Dostawca</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Kwota</TableHead>
                <TableHead>Oczekiwana dostawa</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((po) => (
                <TableRow
                  key={po.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/purchase-orders/${po.id}`)}
                >
                  <TableCell className="font-medium font-mono">
                    {po.po_number}
                  </TableCell>
                  <TableCell>{po.supplier_name}</TableCell>
                  <TableCell>
                    <StatusBadge
                      status={po.status}
                      statusMap={PURCHASE_ORDER_STATUSES}
                      translationPrefix="purchaseOrder"
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    {formatCurrency(po.total_amount, po.currency)}
                  </TableCell>
                  <TableCell>
                    {po.expected_delivery_date
                      ? formatDate(po.expected_delivery_date)
                      : "---"}
                  </TableCell>
                  <TableCell>{formatDate(po.created_at)}</TableCell>
                  <TableCell>
                    {po.status === "draft" && (
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(po.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("usunZamowienieZakupu")}
        description={t("czyNaPewnoChceszUsunacToZamowienieZakupuTaOperacja")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deletePO.isPending}
      />
    </AdminGuard>
  );
}
