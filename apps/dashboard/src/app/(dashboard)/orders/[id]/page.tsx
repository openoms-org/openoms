"use client";

import { useState, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Package, RotateCcw, Printer, FileText, Scissors, GitBranch, Headphones, Loader2, Plus, ExternalLink, Copy, Check, StickyNote, Save, Tag, Send, ChevronDown, Factory } from "lucide-react";
import { RateShopping } from "@/components/shipping/rate-shopping";
import { AllegroShipmentDialog } from "@/components/integrations/allegro-shipment-dialog";
import { useAllegroCarriers, useAllegroFulfillment, useAllegroTracking } from "@/hooks/use-allegro";
import { useOrder, useUpdateOrder, useDeleteOrder, useTransitionOrderStatus, useDuplicateOrder } from "@/hooks/use-orders";
import { useOrderShipments, useCreateOrderShipment } from "@/hooks/use-shipments";
import { useReturns } from "@/hooks/use-returns";
import { useOrderDropshipOrders, useAutoRouteDropship } from "@/hooks/use-dropship-orders";
import { useOrderGroups, useSplitOrder } from "@/hooks/use-order-groups";
import { useOrderTickets, useCreateOrderTicket } from "@/hooks/use-helpdesk";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { OrderTimeline } from "@/components/orders/order-timeline";
import { OrderForm } from "@/components/orders/order-form";
import { mapCreateOrderRequestToUpdateOrderRequest } from "@/components/orders/order-request-mappers";
import { OrderStatusActions } from "@/components/orders/order-status-actions";
import { OrderInvoicesSection } from "@/components/orders/order-invoices-section";
import { StatusBadge } from "@/components/shared/status-badge";
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
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ORDER_STATUSES, PAYMENT_STATUSES, SHIPMENT_STATUSES, RETURN_STATUSES, ORDER_PRIORITIES, SHIPMENT_PROVIDERS } from "@/lib/constants";
import { getSelectableShipmentProviders, isFeatureVisible } from "@/lib/readiness";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { formatDate, formatCurrency, shortId, cn, sanitizeUrl } from "@/lib/utils";
import { getErrorMessage, apiFetch } from "@/lib/api-client";
import type { CreateOrderRequest } from "@/types/api";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";

function CollapsibleSection({
  title,
  icon: Icon,
  defaultOpen = true,
  children,
  badge,
  headerAction,
}: {
  title: string;
  icon?: React.ComponentType<{ className?: string }>;
  defaultOpen?: boolean;
  children: React.ReactNode;
  badge?: React.ReactNode;
  headerAction?: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <Card>
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-6 py-4 text-left"
      >
        <div className="flex items-center gap-2">
          {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
          <h3 className="font-semibold">{title}</h3>
          {badge}
        </div>
        <div className="flex items-center gap-2">
          {headerAction && (
            <span onClick={(e) => e.stopPropagation()}>
              {headerAction}
            </span>
          )}
          <ChevronDown
            className={cn(
              "h-4 w-4 text-muted-foreground transition-transform",
              open && "rotate-180"
            )}
          />
        </div>
      </button>
      {open && <CardContent>{children}</CardContent>}
    </Card>
  );
}

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const { data: order, isLoading } = useOrder(params.id);
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showSplitDialog, setShowSplitDialog] = useState(false);
  const [showCreateTicketDialog, setShowCreateTicketDialog] = useState(false);
  const [returnLinkCopied, setReturnLinkCopied] = useState(false);
  const [internalNotes, setInternalNotes] = useEffectSyncedState(
    order?.internal_notes || "",
    order?.id ?? null,
  );
  const [internalNotesDirty, setInternalNotesDirty] = useState(false);
  const [showAllegroShipmentDialog, setShowAllegroShipmentDialog] = useState(false);
  const [showAllegroFulfillmentDialog, setShowAllegroFulfillmentDialog] = useState(false);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const { data: customFieldsConfig } = useCustomFields();

  const updateOrder = useUpdateOrder(params.id);
  const deleteOrder = useDeleteOrder();
  const transitionStatus = useTransitionOrderStatus(params.id);

  const { data: orderShipments, isLoading: isLoadingShipments } = useOrderShipments(params.id);
  const createOrderShipment = useCreateOrderShipment(params.id);
  const [showAddPackageDialog, setShowAddPackageDialog] = useState(false);
  const { data: returnsData, isLoading: isLoadingReturns } = useReturns({ order_id: params.id });
  const { data: orderGroups } = useOrderGroups(params.id);
  const splitOrder = useSplitOrder(params.id);
  const { data: ticketsData, isLoading: isLoadingTickets } = useOrderTickets(params.id);
  const createTicket = useCreateOrderTicket(params.id);
  const duplicateOrder = useDuplicateOrder();
  // Dropship is a beta feature; only fetch/render it when visible in the active
  // surface, otherwise the gated endpoints return feature_not_available 404.
  const dropshipVisible = isFeatureVisible("dropship");
  const { data: dropshipOrders, isLoading: isLoadingDropship } = useOrderDropshipOrders(
    params.id,
    { enabled: dropshipVisible }
  );
  const autoRouteDropship = useAutoRouteDropship();
  // Invoicing is a controlled feature; only mount the invoice section (which
  // fetches /orders/{id}/invoices) when visible in the active surface.
  const invoicingVisible = isFeatureVisible("invoicing");

  // Calculate total order weight from items for rate shopping
  const items = order?.items;
  const orderWeight = useMemo(() => {
    if (!items?.length) return undefined;
    let total = 0;
    for (const item of items) {
      if (item.weight && item.weight > 0 && item.quantity > 0) {
        total += item.weight * item.quantity;
      }
    }
    return total > 0 ? Math.round(total * 1000) / 1000 : undefined;
  }, [items]);

  const handleUpdate = async (data: CreateOrderRequest) => {
    try {
      await updateOrder.mutateAsync(mapCreateOrderRequestToUpdateOrderRequest(data));
      toast.success(t("detail.orderUpdated"));
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteOrder.mutateAsync(params.id);
      toast.success(t("detail.orderDeleted"));
      router.push("/orders");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleTransition = async (newStatus: string, force?: boolean) => {
    try {
      await transitionStatus.mutateAsync({ status: newStatus, force });
      toast.success(t("detail.statusChanged"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
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
        <p className="text-muted-foreground">{t("detail.notFound")}</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/orders")}>
          {t("detail.backToList")}
        </Button>
      </div>
    );
  }

  if (isEditing) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">{t("editOrder")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("detail.orderNumber", { id: shortId(order.id) })}
          </p>
        </div>
        <div className="max-w-2xl">
          <OrderForm
            order={order}
            onSubmit={handleUpdate}
            isSubmitting={updateOrder.isPending}
            onCancel={() => setIsEditing(false)}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t("detail.orderNumber", { id: shortId(order.id) })}
          </h1>
          <p className="text-muted-foreground mt-1">
            {t("detail.createdOn", { date: formatDate(order.created_at) })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={async () => {
              try {
                const res = await apiFetch(`/v1/orders/${params.id}/print`);
                const blob = await res.blob();
                const url = URL.createObjectURL(blob);
                window.open(url, "_blank");
                setTimeout(() => URL.revokeObjectURL(url), 60000);
              } catch {
                toast.error(t("detail.printError"));
              }
            }}
          >
            <Printer className="mr-2 h-4 w-4" />
            {t("detail.print")}
          </Button>
          <Button
            variant="outline"
            onClick={async () => {
              try {
                const res = await apiFetch(`/v1/orders/${params.id}/packing-slip`);
                const blob = await res.blob();
                const url = URL.createObjectURL(blob);
                window.open(url, "_blank");
                setTimeout(() => URL.revokeObjectURL(url), 60000);
              } catch {
                toast.error(t("detail.packingSlipError"));
              }
            }}
          >
            <FileText className="mr-2 h-4 w-4" />
            {t("detail.packingSlip")}
          </Button>
          <Button variant="outline" onClick={() => setShowAddPackageDialog(true)}>
            <Package className="mr-2 h-4 w-4" />
            {t("detail.addPackage")}
          </Button>
          {order.source === "allegro" && order.external_id && (
            <>
              <Button variant="outline" onClick={() => setShowAllegroFulfillmentDialog(true)}>
                <Send className="mr-2 h-4 w-4" />
                Wyslij do Allegro
              </Button>
              <Button variant="outline" onClick={() => setShowAllegroShipmentDialog(true)}>
                <Tag className="mr-2 h-4 w-4" />
                Etykieta Allegro
              </Button>
            </>
          )}
          <Button variant="outline" asChild>
            <Link href={`/returns/new?order_id=${params.id}`}>
              <RotateCcw className="mr-2 h-4 w-4" />
              {t("detail.reportReturn")}
            </Link>
          </Button>
          <Button
            variant="outline"
            onClick={() => {
              const returnUrl = `${window.location.origin}/return-request?order_id=${params.id}`;
              navigator.clipboard.writeText(returnUrl).then(() => {
                setReturnLinkCopied(true);
                toast.success(t("detail.returnLinkCopied"));
                setTimeout(() => setReturnLinkCopied(false), 2000);
              });
            }}
          >
            {returnLinkCopied ? (
              <Check className="mr-2 h-4 w-4" />
            ) : (
              <ExternalLink className="mr-2 h-4 w-4" />
            )}
            {t("detail.returnLink")}
          </Button>
          {order.status !== "merged" && order.status !== "split" && order.items && order.items.length >= 2 && (
            <Button variant="outline" onClick={() => setShowSplitDialog(true)}>
              <Scissors className="mr-2 h-4 w-4" />
              {t("detail.splitOrder")}
            </Button>
          )}
          <Button
            variant="outline"
            onClick={async () => {
              try {
                const newOrder = await duplicateOrder.mutateAsync(params.id);
                toast.success(t("detail.duplicated"));
                router.push(`/orders/${newOrder.id}`);
              } catch (error) {
                toast.error(getErrorMessage(error));
              }
            }}
            disabled={duplicateOrder.isPending}
          >
            {duplicateOrder.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Copy className="mr-2 h-4 w-4" />
            )}
            {t("detail.duplicateOrder")}
          </Button>
          <Button variant="outline" onClick={() => setIsEditing(true)}>
            {tc("edit")}
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            {tc("delete")}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{t("detail.orderData")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">{tc("status")}</p>
                  <div className="mt-1">
                    <StatusBadge status={order.status} statusMap={orderStatuses} />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("form.priority")}</p>
                  <div className="mt-1">
                    <Select
                      value={order.priority || "normal"}
                      onValueChange={async (value) => {
                        try {
                          await updateOrder.mutateAsync({ priority: value as "urgent" | "high" | "normal" | "low" });
                          toast.success(t("detail.priorityChanged"));
                        } catch (error) {
                          toast.error(getErrorMessage(error));
                        }
                      }}
                    >
                      <SelectTrigger className="w-[140px]" size="sm">
                        <SelectValue>
                          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${ORDER_PRIORITIES[order.priority || "normal"]?.color || ""}`}>
                            {ORDER_PRIORITIES[order.priority || "normal"]?.label || t("detail.orderData")}
                          </span>
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {Object.entries(ORDER_PRIORITIES).map(([key, { label, color }]) => (
                          <SelectItem key={key} value={key}>
                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${color}`}>
                              {label}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("columns.source")}</p>
                  <p className="mt-1 font-medium">{order.source}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("columns.amount")}</p>
                  <p className="mt-1 font-medium">
                    {formatCurrency(order.total_amount, order.currency)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("form.currency")}</p>
                  <p className="mt-1 font-medium">{order.currency}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("columns.payment")}</p>
                  <div className="mt-1">
                    <StatusBadge status={order.payment_status} statusMap={PAYMENT_STATUSES} />
                  </div>
                </div>
                {order.payment_method && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("form.paymentMethod")}</p>
                    <p className="mt-1 font-medium">{order.payment_method}</p>
                  </div>
                )}
                {order.paid_at && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("detail.paidAt")}</p>
                    <p className="mt-1 font-medium">{formatDate(order.paid_at)}</p>
                  </div>
                )}
                {order.external_id && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("detail.externalId")}</p>
                    <p className="mt-1 font-mono text-sm">{order.external_id}</p>
                  </div>
                )}
              </div>

              {order.tags && order.tags.length > 0 && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("columns.tags")}</p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {order.tags.map((tag) => (
                      <span key={tag} className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {order.notes && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">{tc("notes")}</p>
                    <p className="mt-1 text-sm">{order.notes}</p>
                  </div>
                </>
              )}

              {(() => {
                const metadata = order.metadata as Record<string, unknown> | undefined;
                const fields = customFieldsConfig?.fields || [];
                const fieldsWithValues = fields.filter(
                  (f) =>
                    metadata?.[f.key] !== undefined &&
                    metadata?.[f.key] !== null &&
                    metadata?.[f.key] !== ""
                );
                if (fieldsWithValues.length === 0) return null;
                return (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-2">
                        {t("form.customFields")}
                      </p>
                      <div className="grid grid-cols-2 gap-4">
                        {fieldsWithValues
                          .sort((a, b) => a.position - b.position)
                          .map((field) => {
                            const value = metadata![field.key];
                            let displayValue: string;
                            if (field.type === "checkbox") {
                              displayValue = value ? t("customFieldYes") : t("customFieldNo");
                            } else if (field.type === "date" && typeof value === "string") {
                              displayValue = new Date(value).toLocaleDateString();
                            } else {
                              displayValue = String(value);
                            }
                            return (
                              <div key={field.key}>
                                <p className="text-sm text-muted-foreground">{field.label}</p>
                                <p className="mt-1 font-medium">{displayValue}</p>
                              </div>
                            );
                          })}
                      </div>
                    </div>
                  </>
                );
              })()}
            </CardContent>
          </Card>

          {order.items && order.items.length > 0 && (
            <CollapsibleSection
              title={t("form.orderItems")}
              icon={Package}
              defaultOpen={true}
              badge={
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {order.items.length}
                </span>
              }
            >
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("detail.product")}</TableHead>
                    <TableHead>SKU</TableHead>
                    <TableHead className="text-right">{tc("quantity")}</TableHead>
                    <TableHead className="text-right">{tc("price")}</TableHead>
                    <TableHead className="text-right">{t("detail.value")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {order.items.map((item, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-medium">{item.name}</TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {item.sku || "—"}
                      </TableCell>
                      <TableCell className="text-right">{item.quantity}</TableCell>
                      <TableCell className="text-right">
                        {formatCurrency(item.price, order.currency)}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {formatCurrency(item.price * item.quantity, order.currency)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
                <TableFooter>
                  <TableRow>
                    <TableCell colSpan={4} className="font-medium">
                      {t("detail.total")}
                    </TableCell>
                    <TableCell className="text-right font-bold">
                      {formatCurrency(order.total_amount, order.currency)}
                    </TableCell>
                  </TableRow>
                </TableFooter>
              </Table>
            </CollapsibleSection>
          )}

          <Card>
            <CardHeader>
              <CardTitle>{t("detail.statusChange")}</CardTitle>
            </CardHeader>
            <CardContent>
              <OrderStatusActions
                currentStatus={order.status}
                onTransition={handleTransition}
                isPending={transitionStatus.isPending}
              />
            </CardContent>
          </Card>

          <CollapsibleSection
            title={t("detail.packages")}
            icon={Package}
            defaultOpen={!!(orderShipments && orderShipments.length > 0)}
            badge={
              orderShipments && orderShipments.length > 0 ? (
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {orderShipments.length}
                </span>
              ) : undefined
            }
            headerAction={
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowAddPackageDialog(true)}
              >
                <Plus className="h-4 w-4" />
                {t("detail.addPackage")}
              </Button>
            }
          >
            {isLoadingShipments ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : orderShipments && orderShipments.length > 0 ? (
              <div className="space-y-4">
                {orderShipments.map((shipment) => (
                  <div
                    key={shipment.id}
                    className="rounded-lg border bg-card p-4 space-y-3"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                          {shipment.package_number}
                        </span>
                        <div>
                          <Link
                            href={`/shipments/${shipment.id}`}
                            className="font-medium text-primary hover:underline"
                          >
                            {t("detail.packageOf", { num: shipment.package_number, total: orderShipments.length })}
                          </Link>
                          <p className="text-xs text-muted-foreground">
                            {shipment.provider.toUpperCase()} — {formatDate(shipment.created_at)}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={shipment.status} statusMap={SHIPMENT_STATUSES} />
                        <Button variant="outline" size="sm" asChild>
                          <Link href={`/shipments/${shipment.id}`}>
                            {tc("details")}
                          </Link>
                        </Button>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
                      <div>
                        <p className="text-muted-foreground">{t("detail.trackingNumber")}</p>
                        <p className="font-medium">
                          {shipment.tracking_number || "—"}
                        </p>
                      </div>
                      {shipment.weight != null && (
                        <div>
                          <p className="text-muted-foreground">{t("detail.weight")}</p>
                          <p className="font-medium">{shipment.weight} kg</p>
                        </div>
                      )}
                      {(shipment.length != null || shipment.width != null || shipment.height != null) && (
                        <div>
                          <p className="text-muted-foreground">{t("detail.dimensions")}</p>
                          <p className="font-medium">
                            {shipment.length ?? "—"} x {shipment.width ?? "—"} x {shipment.height ?? "—"}
                          </p>
                        </div>
                      )}
                      {shipment.label_url && (
                        <div>
                          <p className="text-muted-foreground">{t("detail.label")}</p>
                          <a
                            href={sanitizeUrl(shipment.label_url)}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-medium text-primary hover:underline"
                          >
                            {tc("download")}
                          </a>
                        </div>
                      )}
                    </div>

                    {shipment.carbon_kg != null && (
                      <div className="text-sm flex items-center gap-1.5 text-green-700 dark:text-green-400">
                        <span>CO2: {shipment.carbon_kg.toFixed(3)} kg</span>
                        {shipment.distance_km != null && (
                          <span className="text-muted-foreground">({shipment.distance_km.toFixed(0)} km)</span>
                        )}
                      </div>
                    )}
                    {shipment.notes && (
                      <div className="text-sm">
                        <p className="text-muted-foreground">{tc("notes")}</p>
                        <p className="mt-0.5">{shipment.notes}</p>
                      </div>
                    )}
                  </div>
                ))}
                {/* Total carbon for all shipments */}
                {orderShipments.some((s) => s.carbon_kg != null) && (
                  <div className="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900 dark:bg-green-950/30 mt-2">
                    <p className="text-sm font-medium text-green-700 dark:text-green-400">
                      {t("detail.carbonFootprint", { amount: orderShipments
                        .reduce((sum, s) => sum + (s.carbon_kg ?? 0), 0)
                        .toFixed(3) })}
                    </p>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Package className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">{t("detail.noPackages")}</p>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-3"
                  onClick={() => setShowAddPackageDialog(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  {t("detail.addFirstPackage")}
                </Button>
              </div>
            )}
          </CollapsibleSection>

          <RateShopping
            defaultToPostalCode={order.shipping_address?.postal_code}
            defaultWeight={orderWeight}
            onSelectRate={(rate) => {
              router.push(
                `/shipments/new?order_id=${params.id}&carrier=${rate.carrier_code}`
              );
            }}
          />

          <CollapsibleSection
            title={t("detail.returns")}
            icon={RotateCcw}
            defaultOpen={!!(returnsData?.items && returnsData.items.length > 0)}
            badge={
              returnsData?.items && returnsData.items.length > 0 ? (
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {returnsData.items.length}
                </span>
              ) : undefined
            }
          >
            {isLoadingReturns ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : returnsData?.items && returnsData.items.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{tc("status")}</TableHead>
                    <TableHead>{t("detail.reason")}</TableHead>
                    <TableHead>{t("detail.refundAmount")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {returnsData.items.map((ret) => (
                    <TableRow key={ret.id}>
                      <TableCell>
                        <Link href={`/returns/${ret.id}`} className="font-medium text-primary hover:underline">
                          {shortId(ret.id)}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={ret.status} statusMap={RETURN_STATUSES} />
                      </TableCell>
                      <TableCell className="max-w-[200px] truncate">{ret.reason}</TableCell>
                      <TableCell>{formatCurrency(ret.refund_amount)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <RotateCcw className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">{t("detail.noReturns")}</p>
              </div>
            )}
          </CollapsibleSection>

          {/* Dropshipping — beta feature, only rendered when visible in the active surface */}
          {dropshipVisible && (
          <CollapsibleSection
            title="Dropshipping"
            icon={Factory}
            defaultOpen={!!(dropshipOrders && dropshipOrders.length > 0)}
            badge={
              dropshipOrders && dropshipOrders.length > 0 ? (
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {dropshipOrders.length}
                </span>
              ) : undefined
            }
            headerAction={
              <Button
                variant="outline"
                size="sm"
                disabled={autoRouteDropship.isPending}
                onClick={async () => {
                  try {
                    const result = await autoRouteDropship.mutateAsync(params.id);
                    if (result.length === 0) {
                      toast.info(t("detail.noDropshipProducts"));
                    } else {
                      toast.success(t("detail.dropshipCreated", { count: result.length }));
                    }
                  } catch (error) {
                    toast.error(getErrorMessage(error));
                  }
                }}
              >
                {autoRouteDropship.isPending ? (
                  <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                ) : (
                  <Factory className="mr-1 h-4 w-4" />
                )}
                {t("detail.forwardToSuppliers")}
              </Button>
            }
          >
            {isLoadingDropship ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : dropshipOrders && dropshipOrders.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t("detail.supplier")}</TableHead>
                    <TableHead>{tc("status")}</TableHead>
                    <TableHead>{t("detail.cost")}</TableHead>
                    <TableHead>{t("detail.trackingNumber")}</TableHead>
                    <TableHead>{tc("createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dropshipOrders.map((d) => (
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
                        <StatusBadge
                          status={d.status}
                          statusMap={{
                            pending: { label: t("dropship.pending"), color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300" },
                            sent: { label: t("dropship.sent"), color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300" },
                            confirmed: { label: t("dropship.confirmed"), color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300" },
                            shipped: { label: t("dropship.shipped"), color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
                            delivered: { label: t("dropship.delivered"), color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300" },
                            cancelled: { label: t("dropship.cancelled"), color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
                          }}
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
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Factory className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">
                  {t("detail.noDropshipOrders")}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("detail.dropshipHint")}
                </p>
              </div>
            )}
          </CollapsibleSection>
          )}

          {/* Invoices — manual issuance, guarded against duplicating auto-created invoices */}
          {invoicingVisible && <OrderInvoicesSection order={order} />}

          {/* Helpdesk Tickets */}
          <CollapsibleSection
            title={t("detail.tickets")}
            icon={Headphones}
            defaultOpen={!!(ticketsData?.tickets && ticketsData.tickets.length > 0)}
            badge={
              ticketsData?.tickets && ticketsData.tickets.length > 0 ? (
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {ticketsData.tickets.length}
                </span>
              ) : undefined
            }
            headerAction={
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateTicketDialog(true)}
              >
                <Plus className="h-4 w-4" />
                {t("detail.createTicket")}
              </Button>
            }
          >
            {isLoadingTickets ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : ticketsData?.tickets && ticketsData.tickets.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t("detail.subject")}</TableHead>
                    <TableHead>{tc("status")}</TableHead>
                    <TableHead>{tc("createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {ticketsData.tickets.map((ticket) => (
                    <TableRow key={ticket.id}>
                      <TableCell className="font-mono text-sm">#{ticket.id}</TableCell>
                      <TableCell className="font-medium">{ticket.subject}</TableCell>
                      <TableCell>
                        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium">
                          {ticket.status === 2 ? t("detail.ticketStatusOpen") : ticket.status === 3 ? t("detail.ticketStatusPending") : ticket.status === 4 ? t("detail.ticketStatusResolved") : ticket.status === 5 ? t("detail.ticketStatusClosed") : `Status ${ticket.status}`}
                        </span>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(ticket.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Headphones className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">{t("detail.noTickets")}</p>
              </div>
            )}
          </CollapsibleSection>

          {/* Merge/Split History */}
          {(order.merged_into || order.split_from || (orderGroups && orderGroups.length > 0)) && (
            <CollapsibleSection
              title={t("detail.mergeHistory")}
              icon={GitBranch}
              defaultOpen={true}
              badge={
                orderGroups && orderGroups.length > 0 ? (
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                    {orderGroups.length}
                  </span>
                ) : undefined
              }
            >
              <div className="space-y-3">
                {order.merged_into && (
                  <div className="rounded-md border bg-muted/50 p-3">
                    <p className="text-sm">
                      {t("detail.mergedInto")}{" "}
                      <Link href={`/orders/${order.merged_into}`} className="font-medium text-primary hover:underline">
                        {shortId(order.merged_into)}
                      </Link>
                    </p>
                  </div>
                )}
                {order.split_from && (
                  <div className="rounded-md border bg-muted/50 p-3">
                    <p className="text-sm">
                      {t("detail.splitFrom")}{" "}
                      <Link href={`/orders/${order.split_from}`} className="font-medium text-primary hover:underline">
                        {shortId(order.split_from)}
                      </Link>
                    </p>
                  </div>
                )}
                {orderGroups && orderGroups.map((group) => (
                  <div key={group.id} className="rounded-md border p-3 text-sm space-y-1">
                    <p className="font-medium">
                      {group.group_type === "merged" ? t("detail.mergeAction") : t("detail.splitAction")} - {formatDate(group.created_at)}
                    </p>
                    <p className="text-muted-foreground">
                      {group.group_type === "merged" ? t("detail.sourceOrders") : t("detail.sourceOrder")}:{" "}
                      {group.source_order_ids.map((id, i) => (
                        <span key={id}>
                          {i > 0 && ", "}
                          <Link href={`/orders/${id}`} className="text-primary hover:underline">
                            {shortId(id)}
                          </Link>
                        </span>
                      ))}
                    </p>
                    <p className="text-muted-foreground">
                      {group.group_type === "merged" ? t("detail.targetOrder") : t("detail.targetOrders")}:{" "}
                      {group.target_order_ids.map((id, i) => (
                        <span key={id}>
                          {i > 0 && ", "}
                          <Link href={`/orders/${id}`} className="text-primary hover:underline">
                            {shortId(id)}
                          </Link>
                        </span>
                      ))}
                    </p>
                    {group.notes && (
                      <p className="text-muted-foreground">{t("detail.note")}: {group.notes}</p>
                    )}
                  </div>
                ))}
              </div>
            </CollapsibleSection>
          )}
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{t("detail.customerData")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm text-muted-foreground">{t("detail.customerName")}</p>
                <p className="mt-1 font-medium">{order.customer_name}</p>
              </div>
              {order.customer_email && (
                <div>
                  <p className="text-sm text-muted-foreground">Email</p>
                  <p className="mt-1 text-sm">{order.customer_email}</p>
                </div>
              )}
              {order.customer_phone && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("form.customerPhone")}</p>
                  <p className="mt-1 text-sm">{order.customer_phone}</p>
                </div>
              )}
              {order.customer_id && (
                <div className="pt-2">
                  <Link
                    href={`/customers/${order.customer_id}`}
                    className="text-sm text-primary hover:underline font-medium"
                  >
                    {t("detail.viewCustomerProfile")}
                  </Link>
                </div>
              )}
            </CardContent>
          </Card>

          
          <Card className="border-amber-300 dark:border-amber-700 bg-amber-50/50 dark:bg-amber-950/20">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
                <StickyNote className="h-4 w-4" />
                {t("detail.internalNotes")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={internalNotes}
                onChange={(e) => {
                  setInternalNotes(e.target.value);
                  setInternalNotesDirty(true);
                }}
                placeholder={t("detail.internalNotesPlaceholder")}
                rows={4}
                className="border-amber-300 dark:border-amber-700 bg-white dark:bg-amber-950/30"
              />
              {internalNotesDirty && (
                <Button
                  size="sm"
                  className="mt-2"
                  onClick={async () => {
                    try {
                      await updateOrder.mutateAsync({ internal_notes: internalNotes });
                      toast.success(t("detail.internalNotesSaved"));
                      setInternalNotesDirty(false);
                    } catch (error) {
                      toast.error(getErrorMessage(error));
                    }
                  }}
                  disabled={updateOrder.isPending}
                >
                  {updateOrder.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  {t("detail.saveNotes")}
                </Button>
              )}
            </CardContent>
          </Card>

          <OrderTimeline orderId={params.id} />
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("detail.deleteTitle")}
        description={t("detail.deleteDescription")}
        confirmLabel={t("detail.deleteConfirm")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteOrder.isPending}
      />

      {/* Split Order Dialog */}
      {order.items && (
        <SplitOrderDialog
          open={showSplitDialog}
          onOpenChange={setShowSplitDialog}
          order={order}
          onSplit={async (splits) => {
            try {
              await splitOrder.mutateAsync({ splits });
              toast.success(t("detail.orderSplit"));
              setShowSplitDialog(false);
            } catch (error) {
              toast.error(getErrorMessage(error));
            }
          }}
          isPending={splitOrder.isPending}
        />
      )}

      {/* Allegro Shipment Dialog */}
      {order.source === "allegro" && order.external_id && (
        <AllegroShipmentDialog
          open={showAllegroShipmentDialog}
          onOpenChange={setShowAllegroShipmentDialog}
          order={order}
        />
      )}

      {/* Allegro Fulfillment Dialog */}
      {order.source === "allegro" && order.external_id && (
        <AllegroFulfillmentDialog
          open={showAllegroFulfillmentDialog}
          onOpenChange={setShowAllegroFulfillmentDialog}
          orderId={params.id}
        />
      )}

      {/* Create Ticket Dialog */}
      <CreateTicketDialog
        open={showCreateTicketDialog}
        onOpenChange={setShowCreateTicketDialog}
        customerEmail={order?.customer_email || ""}
        onSubmit={async (data) => {
          try {
            await createTicket.mutateAsync(data);
            toast.success(t("detail.ticketCreated"));
            setShowCreateTicketDialog(false);
          } catch (error) {
            toast.error(getErrorMessage(error));
          }
        }}
        isPending={createTicket.isPending}
      />

      {/* Add Package Dialog */}
      <AddPackageDialog
        open={showAddPackageDialog}
        onOpenChange={setShowAddPackageDialog}
        onSubmit={async (data) => {
          try {
            await createOrderShipment.mutateAsync(data);
            toast.success(t("detail.packageAdded"));
            setShowAddPackageDialog(false);
          } catch (error) {
            toast.error(getErrorMessage(error));
          }
        }}
        isPending={createOrderShipment.isPending}
      />
    </div>
  );
}

function CreateTicketDialog({
  open,
  onOpenChange,
  customerEmail,
  onSubmit,
  isPending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customerEmail: string;
  onSubmit: (data: { subject: string; description: string; email: string }) => void;
  isPending: boolean;
}) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const [subject, setSubject] = useEffectSyncedState("", open);
  const [description, setDescription] = useEffectSyncedState("", open);
  const [email, setEmail] = useEffectSyncedState(
    customerEmail,
    `${open}:${customerEmail}`,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("createTicket.title")}</DialogTitle>
          <DialogDescription>
            {t("createTicket.description")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>{t("createTicket.customerEmail")}</Label>
            <Input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="klient@example.com"
              type="email"
            />
          </div>
          <div>
            <Label>{t("createTicket.subject")}</Label>
            <Input
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder={t("createTicket.subjectPlaceholder")}
            />
          </div>
          <div>
            <Label>{t("createTicket.descriptionLabel")}</Label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("createTicket.descriptionPlaceholder")}
              rows={4}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc("cancel")}
          </Button>
          <Button
            onClick={() => onSubmit({ subject, description, email })}
            disabled={!subject || !description || !email || isPending}
          >
            {isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : null}
            {t("detail.createTicket")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SplitOrderDialog({
  open,
  onOpenChange,
  order,
  onSplit,
  isPending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: { items?: { name: string; sku?: string; quantity: number; price: number }[]; currency: string };
  onSplit: (splits: { items: { name: string; sku?: string; quantity: number; price: number }[] }[]) => void;
  isPending: boolean;
}) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const items = order.items || [];
  const [allocation, setAllocation] = useState<number[]>(() => items.map(() => 1));

  const handleAllocChange = (index: number, split: number) => {
    setAllocation((prev) => {
      const next = [...prev];
      next[index] = split;
      return next;
    });
  };

  const handleSubmit = () => {
    const split1Items = items.filter((_, i) => allocation[i] === 1);
    const split2Items = items.filter((_, i) => allocation[i] === 2);

    if (split1Items.length === 0 || split2Items.length === 0) {
      return;
    }

    onSplit([{ items: split1Items }, { items: split2Items }]);
  };

  const split1Items = items.filter((_, i) => allocation[i] === 1);
  const split2Items = items.filter((_, i) => allocation[i] === 2);
  const split1Total = split1Items.reduce((s, item) => s + item.price * item.quantity, 0);
  const split2Total = split2Items.reduce((s, item) => s + item.price * item.quantity, 0);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("split.title")}</DialogTitle>
          <DialogDescription>
            {t("split.description")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          {items.map((item, i) => (
            <div key={i} className="flex items-center justify-between gap-4 rounded-md border p-3">
              <div className="flex-1">
                <p className="text-sm font-medium">{item.name}</p>
                <p className="text-xs text-muted-foreground">
                  {item.quantity} x {formatCurrency(item.price, order.currency)}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Label className="text-xs">{t("split.orderLabel")}</Label>
                <select
                  className="rounded border px-2 py-1 text-sm"
                  value={allocation[i]}
                  onChange={(e) => handleAllocChange(i, Number(e.target.value))}
                >
                  <option value={1}>1</option>
                  <option value={2}>2</option>
                </select>
              </div>
            </div>
          ))}
        </div>
        <div className="grid grid-cols-2 gap-4 rounded-md bg-muted/50 p-3 text-sm">
          <div>
            <p className="font-medium">{t("split.orderNum", { num: 1 })}</p>
            <p className="text-muted-foreground">{t("split.itemCount", { count: split1Items.length })}</p>
            <p className="font-medium">{formatCurrency(split1Total, order.currency)}</p>
          </div>
          <div>
            <p className="font-medium">{t("split.orderNum", { num: 2 })}</p>
            <p className="text-muted-foreground">{t("split.itemCount", { count: split2Items.length })}</p>
            <p className="font-medium">{formatCurrency(split2Total, order.currency)}</p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc("cancel")}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isPending || split1Items.length === 0 || split2Items.length === 0}
          >
            {isPending ? t("split.splitting") : t("split.splitButton")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AddPackageDialog({
  open,
  onOpenChange,
  onSubmit,
  isPending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: {
    provider: string;
    tracking_number?: string;
    weight?: number;
    length?: number;
    width?: number;
    height?: number;
    notes?: string;
  }) => void;
  isPending: boolean;
}) {
  const [provider, setProvider] = useEffectSyncedState("manual", open);
  const [trackingNumber, setTrackingNumber] = useEffectSyncedState("", open);
  const [weight, setWeight] = useEffectSyncedState("", open);
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const [length, setLength] = useEffectSyncedState("", open);
  const [width, setWidth] = useEffectSyncedState("", open);
  const [height, setHeight] = useEffectSyncedState("", open);
  const [notes, setNotes] = useEffectSyncedState("", open);
  const selectableProviders = getSelectableShipmentProviders(SHIPMENT_PROVIDERS);

  const handleSubmit = () => {
    const data: Parameters<typeof onSubmit>[0] = { provider };
    if (trackingNumber) data.tracking_number = trackingNumber;
    if (weight) data.weight = parseFloat(weight);
    if (length) data.length = parseFloat(length);
    if (width) data.width = parseFloat(width);
    if (height) data.height = parseFloat(height);
    if (notes) data.notes = notes;
    onSubmit(data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("addPackage.title")}</DialogTitle>
          <DialogDescription>
            {t("addPackage.description")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>{t("addPackage.provider")}</Label>
            <Select value={provider} onValueChange={setProvider}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {selectableProviders.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p.toUpperCase()}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{t("addPackage.trackingNumber")}</Label>
            <Input
              value={trackingNumber}
              onChange={(e) => setTrackingNumber(e.target.value)}
              placeholder={t("addPackage.trackingPlaceholder")}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>{t("addPackage.weight")}</Label>
              <Input
                type="number"
                step="0.001"
                min="0"
                value={weight}
                onChange={(e) => setWeight(e.target.value)}
                placeholder="np. 2.5"
              />
            </div>
            <div>
              <Label>{t("addPackage.length")}</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={length}
                onChange={(e) => setLength(e.target.value)}
                placeholder="np. 30"
              />
            </div>
            <div>
              <Label>{t("addPackage.width")}</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={width}
                onChange={(e) => setWidth(e.target.value)}
                placeholder="np. 20"
              />
            </div>
            <div>
              <Label>{t("addPackage.height")}</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={height}
                onChange={(e) => setHeight(e.target.value)}
                placeholder="np. 15"
              />
            </div>
          </div>
          <div>
            <Label>{t("addPackage.notes")}</Label>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t("addPackage.notesPlaceholder")}
              rows={2}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc("cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={!provider || isPending}>
            {isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Plus className="mr-2 h-4 w-4" />
            )}
            {t("detail.addPackage")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AllegroFulfillmentDialog({
  open,
  onOpenChange,
  orderId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
}) {
  const [fulfillmentStatus, setFulfillmentStatus] = useEffectSyncedState("SENT", open);
  const [carrierId, setCarrierId] = useEffectSyncedState("", open);
  const [waybill, setWaybill] = useEffectSyncedState("", open);

  const { data: carriersData } = useAllegroCarriers();
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const fulfillmentMutation = useAllegroFulfillment(orderId);
  const trackingMutation = useAllegroTracking(orderId);

  const handleSubmit = async () => {
    try {
      // Update fulfillment status
      await fulfillmentMutation.mutateAsync({ status: fulfillmentStatus });

      // Add tracking if carrier and waybill provided
      if (carrierId && waybill) {
        await trackingMutation.mutateAsync({ carrier_id: carrierId, waybill });
      }

      toast.success(t("allegro.fulfillmentUpdated"));
      onOpenChange(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const isSubmitting = fulfillmentMutation.isPending || trackingMutation.isPending;
  const carriers = carriersData?.carriers || [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("allegro.sendTitle")}</DialogTitle>
          <DialogDescription>
            {t("allegro.sendDescription")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>{t("allegro.fulfillmentStatus")}</Label>
            <Select value={fulfillmentStatus} onValueChange={setFulfillmentStatus}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="SENT">{t("allegro.sent")}</SelectItem>
                <SelectItem value="PICKED_UP">{t("allegro.pickedUp")}</SelectItem>
                <SelectItem value="READY_FOR_SHIPMENT">{t("allegro.readyForShipment")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Separator />
          <div>
            <Label>{t("allegro.providerOptional")}</Label>
            <Select value={carrierId} onValueChange={setCarrierId}>
              <SelectTrigger>
                <SelectValue placeholder={t("allegro.selectProvider")} />
              </SelectTrigger>
              <SelectContent>
                {carriers.map((carrier) => (
                  <SelectItem key={carrier.id} value={carrier.id}>
                    {carrier.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{t("allegro.trackingOptional")}</Label>
            <Input
              value={waybill}
              onChange={(e) => setWaybill(e.target.value)}
              placeholder="e.g. 6280012345678"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc("cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Send className="mr-2 h-4 w-4" />
            )}
            {t("allegro.send")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
