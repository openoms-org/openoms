"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import {
  ArrowLeft,
  Factory,
  Package,
  Truck,
  Check,
  X,
  Send,
  Loader2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableFooter,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  useDropshipOrder,
  useUpdateDropshipStatus,
  useCancelDropshipOrder,
} from "@/hooks/use-dropship-orders";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { useTranslations } from "next-intl";

interface StatusTimelineStep {
  status: string;
  label: string;
  date?: string;
}

export default function DropshipOrderDetailPage() {
  const t = useTranslations("dropshipOrders");

  const DROPSHIP_STATUSES: Record<string, { label: string; color: string }> = {
    pending: { label: t("status.pending"), color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300" },
    sent: { label: t("status.sent"), color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300" },
    confirmed: { label: t("status.confirmed"), color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300" },
    shipped: { label: t("status.shipped"), color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
    delivered: { label: t("status.delivered"), color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300" },
    cancelled: { label: t("status.cancelled"), color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
  };
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [showShippedDialog, setShowShippedDialog] = useState(false);
  const [trackingNumber, setTrackingNumber] = useState("");
  const [carrier, setCarrier] = useState("");
  const [supplierRef, setSupplierRef] = useState("");

  const { data: order, isLoading } = useDropshipOrder(params.id);
  const updateStatus = useUpdateDropshipStatus(params.id);
  const cancelOrder = useCancelDropshipOrder();

  const handleStatusChange = async (
    status: string,
    extra: { tracking_number?: string; carrier?: string; supplier_reference?: string } = {}
  ) => {
    try {
      await updateStatus.mutateAsync({ status, ...extra });
      toast.success(t("statusChangedTo", { status: DROPSHIP_STATUSES[status]?.label || status }));
      setShowShippedDialog(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleCancel = async () => {
    try {
      await cancelOrder.mutateAsync(params.id);
      toast.success(t("orderCancelled"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!order) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">{t("orderNotFound")}</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/dropship-orders")}>
          {t("backToList")}
        </Button>
      </div>
    );
  }

  const timelineSteps: StatusTimelineStep[] = [
    { status: "pending", label: t("timeline.created"), date: order.created_at },
    { status: "sent", label: t("status.sent"), date: order.sent_at },
    { status: "confirmed", label: t("status.confirmed"), date: order.confirmed_at },
    { status: "shipped", label: t("status.shipped"), date: order.shipped_at },
    { status: "delivered", label: t("status.delivered"), date: order.delivered_at },
  ];

  const statusOrder = ["pending", "sent", "confirmed", "shipped", "delivered"];
  const currentIdx = statusOrder.indexOf(order.status);
  const isCancelled = order.status === "cancelled";

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/dropship-orders")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">
            Dropship {shortId(order.id)}
          </h1>
          <p className="text-muted-foreground">
            {t("supplier")}: {order.supplier_name} | {t("order.title")}:{" "}
            <Link href={`/orders/${order.order_id}`} className="text-primary hover:underline">
              {shortId(order.order_id)}
            </Link>
          </p>
        </div>
        <StatusBadge status={order.status} statusMap={DROPSHIP_STATUSES} />
      </div>

      {/* Status Timeline */}
      {!isCancelled && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              {timelineSteps.map((step, i) => {
                const isActive = statusOrder.indexOf(step.status) <= currentIdx;
                const isCurrent = step.status === order.status;
                return (
                  <div key={step.status} className="flex flex-1 items-center">
                    <div className="flex flex-col items-center gap-1">
                      <div
                        className={`flex h-8 w-8 items-center justify-center rounded-full border-2 text-xs font-medium ${
                          isActive
                            ? "border-primary bg-primary text-primary-foreground"
                            : "border-muted-foreground/30 text-muted-foreground"
                        } ${isCurrent ? "ring-2 ring-primary ring-offset-2" : ""}`}
                      >
                        {isActive ? <Check className="h-4 w-4" /> : i + 1}
                      </div>
                      <span className={`text-xs ${isActive ? "font-medium" : "text-muted-foreground"}`}>
                        {step.label}
                      </span>
                      {step.date && (
                        <span className="text-[10px] text-muted-foreground">
                          {formatDate(step.date)}
                        </span>
                      )}
                    </div>
                    {i < timelineSteps.length - 1 && (
                      <div
                        className={`mx-2 h-0.5 flex-1 ${
                          statusOrder.indexOf(timelineSteps[i + 1].status) <= currentIdx
                            ? "bg-primary"
                            : "bg-muted-foreground/20"
                        }`}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Actions */}
      {!isCancelled && order.status !== "delivered" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("actions")}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {order.status === "pending" && (
              <Button
                onClick={() => handleStatusChange("sent")}
                disabled={updateStatus.isPending}
              >
                <Send className="mr-2 h-4 w-4" />
                {t("markAsShipped")}
              </Button>
            )}
            {order.status === "sent" && (
              <>
                <Button
                  onClick={() => handleStatusChange("confirmed")}
                  disabled={updateStatus.isPending}
                >
                  <Check className="mr-2 h-4 w-4" />
                  {t("confirmedBySupplier")}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowShippedDialog(true)}
                  disabled={updateStatus.isPending}
                >
                  <Truck className="mr-2 h-4 w-4" />
                  {t("markAsShippedWithTracking")}
                </Button>
              </>
            )}
            {order.status === "confirmed" && (
              <Button
                onClick={() => setShowShippedDialog(true)}
                disabled={updateStatus.isPending}
              >
                <Truck className="mr-2 h-4 w-4" />
                {t("markAsInTransit")}
              </Button>
            )}
            {order.status === "shipped" && (
              <Button
                onClick={() => handleStatusChange("delivered")}
                disabled={updateStatus.isPending}
                className="bg-green-600 hover:bg-green-700"
              >
                <Package className="mr-2 h-4 w-4" />
                {t("markAsDelivered")}
              </Button>
            )}
            <Button
              variant="destructive"
              onClick={() => setShowCancelDialog(true)}
              disabled={cancelOrder.isPending}
            >
              <X className="mr-2 h-4 w-4" />
              {t("cancel")}
            </Button>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        {/* Details */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Factory className="h-4 w-4" />
              {t("details")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">{t("supplier")}</p>
                <p className="font-medium">
                  <Link href={`/suppliers/${order.supplier_id}`} className="text-primary hover:underline">
                    {order.supplier_name}
                  </Link>
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">{t("totalCost")}</p>
                <p className="font-medium">{formatCurrency(order.total_cost, order.currency)}</p>
              </div>
              {order.supplier_reference && (
                <div>
                  <p className="text-muted-foreground">{t("supplierReference")}</p>
                  <p className="font-medium">{order.supplier_reference}</p>
                </div>
              )}
              {order.tracking_number && (
                <div>
                  <p className="text-muted-foreground">{t("trackingNumber")}</p>
                  <p className="font-medium">{order.tracking_number}</p>
                </div>
              )}
              {order.carrier && (
                <div>
                  <p className="text-muted-foreground">{t("carrier")}</p>
                  <p className="font-medium">{order.carrier}</p>
                </div>
              )}
              {order.notes && (
                <div className="col-span-2">
                  <p className="text-muted-foreground">{t("notes")}</p>
                  <p className="mt-0.5">{order.notes}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Dates */}
        <Card>
          <CardHeader>
            <CardTitle>{t("dates")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">{t("timeline.created")}</p>
                <p className="font-medium">{formatDate(order.created_at)}</p>
              </div>
              {order.sent_at && (
                <div>
                  <p className="text-muted-foreground">{t("order.shipped")}</p>
                  <p className="font-medium">{formatDate(order.sent_at)}</p>
                </div>
              )}
              {order.confirmed_at && (
                <div>
                  <p className="text-muted-foreground">{t("status.confirmed")}</p>
                  <p className="font-medium">{formatDate(order.confirmed_at)}</p>
                </div>
              )}
              {order.shipped_at && (
                <div>
                  <p className="text-muted-foreground">{t("status.shipped")}</p>
                  <p className="font-medium">{formatDate(order.shipped_at)}</p>
                </div>
              )}
              {order.delivered_at && (
                <div>
                  <p className="text-muted-foreground">{t("status.delivered")}</p>
                  <p className="font-medium">{formatDate(order.delivered_at)}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Items */}
      <Card>
        <CardHeader>
          <CardTitle>{t("items")}</CardTitle>
        </CardHeader>
        <CardContent>
          {order.items && order.items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("product")}</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead className="text-right">{t("quantity")}</TableHead>
                  <TableHead className="text-right">{t("unitPrice")}</TableHead>
                  <TableHead className="text-right">{t("value")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {order.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">
                      {item.product_id ? (
                        <Link
                          href={`/products/${item.product_id}`}
                          className="text-primary hover:underline"
                        >
                          {item.product_name}
                        </Link>
                      ) : (
                        item.product_name
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{item.sku || "---"}</TableCell>
                    <TableCell className="text-right">{item.quantity}</TableCell>
                    <TableCell className="text-right">
                      {formatCurrency(item.unit_cost, order.currency)}
                    </TableCell>
                    <TableCell className="text-right font-medium">
                      {formatCurrency(item.quantity * item.unit_cost, order.currency)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
              <TableFooter>
                <TableRow>
                  <TableCell colSpan={4} className="font-semibold">
                    {t("total")}
                  </TableCell>
                  <TableCell className="text-right font-semibold">
                    {formatCurrency(order.total_cost, order.currency)}
                  </TableCell>
                </TableRow>
              </TableFooter>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">{t("noItems")}</p>
          )}
        </CardContent>
      </Card>

      {/* Shipped Dialog */}
      <Dialog open={showShippedDialog} onOpenChange={setShowShippedDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("markAsInTransit")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("trackingNumber")}</Label>
              <Input
                value={trackingNumber}
                onChange={(e) => setTrackingNumber(e.target.value)}
                placeholder={t("optionalTrackingNumber")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("carrier")}</Label>
              <Input
                value={carrier}
                onChange={(e) => setCarrier(e.target.value)}
                placeholder={t("carrierPlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("supplierReference")}</Label>
              <Input
                value={supplierRef}
                onChange={(e) => setSupplierRef(e.target.value)}
                placeholder={t("optionalSupplierOrderNumber")}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowShippedDialog(false)}>
              {t("cancel")}
            </Button>
            <Button
              disabled={updateStatus.isPending}
              onClick={() =>
                handleStatusChange("shipped", {
                  tracking_number: trackingNumber || undefined,
                  carrier: carrier || undefined,
                  supplier_reference: supplierRef || undefined,
                })
              }
            >
              {updateStatus.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Cancel Dialog */}
      <ConfirmDialog
        open={showCancelDialog}
        onOpenChange={setShowCancelDialog}
        title={t("cancelOrderTitle")}
        description={t("cancelOrderDescription")}
        confirmLabel={t("cancelOrderConfirm")}
        variant="destructive"
        onConfirm={handleCancel}
      />
    </div>
  );
}
