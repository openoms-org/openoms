"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Send, PackageCheck, Ban, Pencil } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  usePurchaseOrder,
  useSendPurchaseOrder,
  useReceivePurchaseOrderItems,
  useCancelPurchaseOrder,
} from "@/hooks/use-purchase-orders";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { PURCHASE_ORDER_STATUSES } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
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
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import type { PurchaseOrderItem, ReceiveItemEntry } from "@/types/api";
import { useTranslations } from "next-intl";

export default function PurchaseOrderDetailPage() {
  const t = useTranslations("purchaseOrders");
  const tc = useTranslations("common");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const { data: po, isLoading } = usePurchaseOrder(id);
  const sendPO = useSendPurchaseOrder();
  const receiveItems = useReceivePurchaseOrderItems(id);
  const cancelPO = useCancelPurchaseOrder();

  const [showReceiveDialog, setShowReceiveDialog] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [receiveEntries, setReceiveEntries] = useState<
    Record<string, number>
  >({});

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!po) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        {t("zamowienieZakupuNieZnalezione")}
      </div>
    );
  }

  const items = po.items ?? [];

  const handleSend = () => {
    sendPO.mutate(id, {
      onSuccess: () => toast.success(t("zamowienieZostaloWyslaneDo")),
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const openReceiveDialog = () => {
    const initial: Record<string, number> = {};
    for (const item of items) {
      const remaining = item.quantity_ordered - item.quantity_received;
      if (remaining > 0) {
        initial[item.id] = remaining;
      }
    }
    setReceiveEntries(initial);
    setShowReceiveDialog(true);
  };

  const handleReceive = () => {
    const entries: ReceiveItemEntry[] = Object.entries(receiveEntries)
      .filter(([, qty]) => qty > 0)
      .map(([item_id, quantity_received]) => ({ item_id, quantity_received }));

    if (entries.length === 0) {
      toast.error(t("wprowadzIlosciDoPrzyjecia"));
      return;
    }

    receiveItems.mutate(
      { items: entries },
      {
        onSuccess: () => {
          toast.success(t("itemsReceived"));
          setShowReceiveDialog(false);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleCancel = () => {
    cancelPO.mutate(id, {
      onSuccess: () => {
        toast.success(t("orderCancelled"));
        setShowCancelDialog(false);
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const canSend = po.status === "draft";
  const canReceive = po.status === "sent" || po.status === "partially_received";
  const canCancel =
    po.status !== "cancelled" && po.status !== "received";

  const receivedTotal = items.reduce(
    (sum, item) => sum + item.quantity_received * item.unit_cost,
    0
  );

  return (
    <AdminGuard>
      <div className="mx-auto max-w-6xl space-y-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push("/purchase-orders")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold tracking-tight font-mono">
              {po.po_number}
            </h1>
            <p className="text-muted-foreground">
              {po.supplier_name} | Utworzono: {formatDate(po.created_at)}
            </p>
          </div>
          <StatusBadge status={po.status} statusMap={PURCHASE_ORDER_STATUSES} />
          <div className="flex gap-2">
            {canSend && (
              <Button
                variant="outline"
                onClick={() => router.push(`/purchase-orders/new?edit=${id}`)}
              >
                <Pencil className="h-4 w-4 mr-2" />
                {tc("edit")}
              </Button>
            )}
            {canSend && (
              <Button onClick={handleSend} disabled={sendPO.isPending}>
                <Send className="h-4 w-4 mr-2" />
                {t("allegro.send")}
              </Button>
            )}
            {canReceive && (
              <Button onClick={openReceiveDialog} variant="default">
                <PackageCheck className="h-4 w-4 mr-2" />
                {t("receiveGoods")}
              </Button>
            )}
            {canCancel && (
              <Button
                variant="destructive"
                onClick={() => setShowCancelDialog(true)}
              >
                <Ban className="h-4 w-4 mr-2" />
                Anuluj
              </Button>
            )}
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Dostawca</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Nazwa</span>
                <span className="font-medium">{po.supplier_name}</span>
              </div>
              {po.supplier_id && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">ID dostawcy</span>
                  <span className="font-mono text-xs">{po.supplier_id}</span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Podsumowanie</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t("wartoscZamowienia")}</span>
                <span className="font-medium">
                  {formatCurrency(po.total_amount, po.currency)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Waluta</span>
                <span>{po.currency}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Pozycji</span>
                <span>{items.length}</span>
              </div>
              {receivedTotal > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t("przyjetoZa")}</span>
                  <span className="font-medium text-green-600">
                    {formatCurrency(receivedTotal, po.currency)}
                  </span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Informacje</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Oczekiwana dostawa</span>
                <span>
                  {po.expected_delivery_date
                    ? formatDate(po.expected_delivery_date)
                    : "---"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Utworzono</span>
                <span>{formatDate(po.created_at)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Zaktualizowano</span>
                <span>{formatDate(po.updated_at)}</span>
              </div>
            </CardContent>
          </Card>
        </div>

        {po.notes && (
          <Card>
            <CardHeader>
              <CardTitle>Uwagi</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm whitespace-pre-wrap">{po.notes}</p>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle>{t("pozycjeZamowienia")}{items.length})</CardTitle>
            <CardDescription>
              {t("listaProduktowWZamowieniuZakupu")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {items.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">
                {t("brakPozycjiWZamowieniu")}
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Produkt</TableHead>
                    <TableHead>SKU</TableHead>
                    <TableHead className="text-right">Cena jedn.</TableHead>
                    <TableHead className="text-center">{t("zamowiono")}</TableHead>
                    <TableHead className="text-center">{t("przyjeto")}</TableHead>
                    <TableHead className="text-center">{t("postep")}</TableHead>
                    <TableHead className="text-right">{t("detail.value")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium">
                        {item.product_name}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {item.sku || "---"}
                      </TableCell>
                      <TableCell className="text-right">
                        {item.unit_cost.toFixed(2)}
                      </TableCell>
                      <TableCell className="text-center">
                        {item.quantity_ordered}
                      </TableCell>
                      <TableCell className="text-center">
                        {item.quantity_received}
                      </TableCell>
                      <TableCell className="text-center">
                        <ProgressBadge item={item} />
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {item.total_cost.toFixed(2)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Receive Items Dialog */}
      <Dialog open={showReceiveDialog} onOpenChange={setShowReceiveDialog}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("receiveGoods")}</DialogTitle>
            <DialogDescription>
              {t("receiveDescription", { number: po.po_number })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 max-h-80 overflow-auto">
            {items
              .filter(
                (item) => item.quantity_received < item.quantity_ordered
              )
              .map((item) => {
                const remaining =
                  item.quantity_ordered - item.quantity_received;
                return (
                  <div
                    key={item.id}
                    className="flex items-center gap-3 p-2 rounded border"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {item.product_name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {t("ordered")}: {item.quantity_ordered} | {t("received")}:{" "}
                        {item.quantity_received} | {t("remaining")}: {remaining}
                      </p>
                    </div>
                    <div className="w-20">
                      <Label className="sr-only">{t("quantity")}</Label>
                      <Input
                        type="number"
                        min={0}
                        max={remaining}
                        value={receiveEntries[item.id] ?? 0}
                        onChange={(e) =>
                          setReceiveEntries({
                            ...receiveEntries,
                            [item.id]: Math.min(
                              parseInt(e.target.value) || 0,
                              remaining
                            ),
                          })
                        }
                        className="h-8"
                      />
                    </div>
                  </div>
                );
              })}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowReceiveDialog(false)}
            >
              Anuluj
            </Button>
            <Button
              onClick={handleReceive}
              disabled={receiveItems.isPending}
            >
              {receiveItems.isPending ? "Przyjmowanie..." : "Przyjmij"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Cancel Confirm Dialog */}
      <ConfirmDialog
        open={showCancelDialog}
        onOpenChange={setShowCancelDialog}
        title={t("anulujZamowienieZakupu")}
        description={t("czyNaPewnoChceszAnulowacToZamowienieZakupuPrzyjete")}
        confirmLabel={t("anulujZamowienie")}
        variant="destructive"
        onConfirm={handleCancel}
        isPending={cancelPO.isPending}
      />
    </AdminGuard>
  );
}

function ProgressBadge({ item }: { item: PurchaseOrderItem }) {
  const t = useTranslations("purchaseOrders");

  if (item.quantity_received === 0) {
    return (
      <Badge variant="outline" className="text-xs">
        {t("progressAwaiting")}
      </Badge>
    );
  }
  if (item.quantity_received >= item.quantity_ordered) {
    return (
      <Badge className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 text-xs">
        {t("progressComplete")}
      </Badge>
    );
  }
  return (
    <Badge className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 text-xs">
      {item.quantity_received}/{item.quantity_ordered}
    </Badge>
  );
}
