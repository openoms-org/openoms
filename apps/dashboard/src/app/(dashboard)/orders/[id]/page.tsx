"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { Package, RotateCcw, Printer, FileText, Scissors, GitBranch } from "lucide-react";
import { useOrder, useUpdateOrder, useDeleteOrder, useTransitionOrderStatus } from "@/hooks/use-orders";
import { useShipments } from "@/hooks/use-shipments";
import { useReturns } from "@/hooks/use-returns";
import { useOrderGroups, useSplitOrder } from "@/hooks/use-order-groups";
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
import { OrderStatusActions } from "@/components/orders/order-status-actions";
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
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { ORDER_STATUSES, PAYMENT_STATUSES, SHIPMENT_STATUSES, RETURN_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { CreateOrderRequest, UpdateOrderRequest } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showSplitDialog, setShowSplitDialog] = useState(false);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const { data: customFieldsConfig } = useCustomFields();

  const { data: order, isLoading } = useOrder(params.id);
  const updateOrder = useUpdateOrder(params.id);
  const deleteOrder = useDeleteOrder();
  const transitionStatus = useTransitionOrderStatus(params.id);

  const { data: shipmentsData } = useShipments({ order_id: params.id });
  const { data: returnsData } = useReturns({ order_id: params.id });
  const { data: orderGroups } = useOrderGroups(params.id);
  const splitOrder = useSplitOrder(params.id);

  const handleUpdate = async (data: CreateOrderRequest) => {
    try {
      await updateOrder.mutateAsync(data as unknown as UpdateOrderRequest);
      toast.success("Zamówienie zostało zaktualizowane");
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteOrder.mutateAsync(params.id);
      toast.success("Zamówienie zostało usunięte");
      router.push("/orders");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleTransition = async (newStatus: string, force?: boolean) => {
    try {
      await transitionStatus.mutateAsync({ status: newStatus, force });
      toast.success("Status zamówienia został zmieniony");
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
        <p className="text-muted-foreground">Nie znaleziono zamówienia</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/orders")}>
          Wróć do listy
        </Button>
      </div>
    );
  }

  if (isEditing) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Edycja zamówienia</h1>
          <p className="text-muted-foreground mt-1">
            Zamówienie {shortId(order.id)}
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
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            Zamówienie {shortId(order.id)}
          </h1>
          <p className="text-muted-foreground mt-1">
            Utworzone {formatDate(order.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => window.open(`${API_URL}/v1/orders/${params.id}/print`, "_blank")}
          >
            <Printer className="mr-2 h-4 w-4" />
            Drukuj
          </Button>
          <Button
            variant="outline"
            onClick={() => window.open(`${API_URL}/v1/orders/${params.id}/packing-slip`, "_blank")}
          >
            <FileText className="mr-2 h-4 w-4" />
            List przewozowy
          </Button>
          <Button variant="outline" asChild>
            <Link href={`/shipments/new?order_id=${params.id}`}>
              <Package className="mr-2 h-4 w-4" />
              Utwórz przesyłkę
            </Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href={`/returns/new?order_id=${params.id}`}>
              <RotateCcw className="mr-2 h-4 w-4" />
              Zgłoś zwrot
            </Link>
          </Button>
          {order && order.status !== "merged" && order.status !== "split" && order.items && order.items.length >= 2 && (
            <Button variant="outline" onClick={() => setShowSplitDialog(true)}>
              <Scissors className="mr-2 h-4 w-4" />
              Podziel zamówienie
            </Button>
          )}
          <Button variant="outline" onClick={() => setIsEditing(true)}>
            Edytuj
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            Usuń
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Dane zamówienia</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Status</p>
                  <div className="mt-1">
                    <StatusBadge status={order.status} statusMap={orderStatuses} />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Źródło</p>
                  <p className="mt-1 font-medium">{order.source}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Kwota</p>
                  <p className="mt-1 font-medium">
                    {formatCurrency(order.total_amount, order.currency)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Waluta</p>
                  <p className="mt-1 font-medium">{order.currency}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Płatność</p>
                  <div className="mt-1">
                    <StatusBadge status={order.payment_status} statusMap={PAYMENT_STATUSES} />
                  </div>
                </div>
                {order.payment_method && (
                  <div>
                    <p className="text-sm text-muted-foreground">Metoda płatności</p>
                    <p className="mt-1 font-medium">{order.payment_method}</p>
                  </div>
                )}
                {order.paid_at && (
                  <div>
                    <p className="text-sm text-muted-foreground">Data opłacenia</p>
                    <p className="mt-1 font-medium">{formatDate(order.paid_at)}</p>
                  </div>
                )}
                {order.external_id && (
                  <div>
                    <p className="text-sm text-muted-foreground">ID zewnętrzne</p>
                    <p className="mt-1 font-mono text-sm">{order.external_id}</p>
                  </div>
                )}
              </div>

              {order.tags && order.tags.length > 0 && (
                <div>
                  <p className="text-sm text-muted-foreground">Tagi</p>
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
                    <p className="text-sm text-muted-foreground">Notatki</p>
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
                        Pola dodatkowe
                      </p>
                      <div className="grid grid-cols-2 gap-4">
                        {fieldsWithValues
                          .sort((a, b) => a.position - b.position)
                          .map((field) => {
                            const value = metadata![field.key];
                            let displayValue: string;
                            if (field.type === "checkbox") {
                              displayValue = value ? "Tak" : "Nie";
                            } else if (field.type === "date" && typeof value === "string") {
                              displayValue = new Date(value).toLocaleDateString("pl-PL");
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
            <Card>
              <CardHeader>
                <CardTitle>Pozycje zamówienia</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Produkt</TableHead>
                      <TableHead>SKU</TableHead>
                      <TableHead className="text-right">Ilość</TableHead>
                      <TableHead className="text-right">Cena</TableHead>
                      <TableHead className="text-right">Wartość</TableHead>
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
                        Razem
                      </TableCell>
                      <TableCell className="text-right font-bold">
                        {formatCurrency(order.total_amount, order.currency)}
                      </TableCell>
                    </TableRow>
                  </TableFooter>
                </Table>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Zmiana statusu</CardTitle>
            </CardHeader>
            <CardContent>
              <OrderStatusActions
                currentStatus={order.status}
                onTransition={handleTransition}
                isLoading={transitionStatus.isPending}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Przesyłki</CardTitle>
            </CardHeader>
            <CardContent>
              {shipmentsData?.items && shipmentsData.items.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Numer śledzenia</TableHead>
                      <TableHead>Dostawca</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Utworzono</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {shipmentsData.items.map((shipment) => (
                      <TableRow key={shipment.id}>
                        <TableCell>
                          <Link href={`/shipments/${shipment.id}`} className="font-medium text-primary hover:underline">
                            {shipment.tracking_number || shortId(shipment.id)}
                          </Link>
                        </TableCell>
                        <TableCell>{shipment.provider}</TableCell>
                        <TableCell>
                          <StatusBadge status={shipment.status} statusMap={SHIPMENT_STATUSES} />
                        </TableCell>
                        <TableCell>{formatDate(shipment.created_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">Brak przesyłek dla tego zamówienia.</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Zwroty</CardTitle>
            </CardHeader>
            <CardContent>
              {returnsData?.items && returnsData.items.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Powód</TableHead>
                      <TableHead>Kwota zwrotu</TableHead>
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
                <p className="text-sm text-muted-foreground">Brak zwrotów dla tego zamówienia.</p>
              )}
            </CardContent>
          </Card>

          {/* Merge/Split History */}
          {(order.merged_into || order.split_from || (orderGroups && orderGroups.length > 0)) && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <GitBranch className="h-4 w-4" />
                  Historia scalania/podziału
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {order.merged_into && (
                  <div className="rounded-md border bg-muted/50 p-3">
                    <p className="text-sm">
                      To zamówienie zostało scalone do:{" "}
                      <Link href={`/orders/${order.merged_into}`} className="font-medium text-primary hover:underline">
                        {shortId(order.merged_into)}
                      </Link>
                    </p>
                  </div>
                )}
                {order.split_from && (
                  <div className="rounded-md border bg-muted/50 p-3">
                    <p className="text-sm">
                      To zamówienie powstało z podziału:{" "}
                      <Link href={`/orders/${order.split_from}`} className="font-medium text-primary hover:underline">
                        {shortId(order.split_from)}
                      </Link>
                    </p>
                  </div>
                )}
                {orderGroups && orderGroups.map((group) => (
                  <div key={group.id} className="rounded-md border p-3 text-sm space-y-1">
                    <p className="font-medium">
                      {group.group_type === "merged" ? "Scalenie" : "Podział"} - {formatDate(group.created_at)}
                    </p>
                    <p className="text-muted-foreground">
                      {group.group_type === "merged" ? "Zamówienia źródłowe" : "Zamówienie źródłowe"}:{" "}
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
                      {group.group_type === "merged" ? "Zamówienie docelowe" : "Zamówienia docelowe"}:{" "}
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
                      <p className="text-muted-foreground">Notatka: {group.notes}</p>
                    )}
                  </div>
                ))}
              </CardContent>
            </Card>
          )}
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Dane klienta</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm text-muted-foreground">Nazwa</p>
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
                  <p className="text-sm text-muted-foreground">Telefon</p>
                  <p className="mt-1 text-sm">{order.customer_phone}</p>
                </div>
              )}
              {order.customer_id && (
                <div className="pt-2">
                  <Link
                    href={`/customers/${order.customer_id}`}
                    className="text-sm text-primary hover:underline font-medium"
                  >
                    Zobacz profil klienta
                  </Link>
                </div>
              )}
            </CardContent>
          </Card>

          <OrderTimeline orderId={params.id} />
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Usuwanie zamówienia"
        description="Czy na pewno chcesz usunąć to zamówienie? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń zamówienie"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteOrder.isPending}
      />

      {/* Split Order Dialog */}
      {order && order.items && (
        <SplitOrderDialog
          open={showSplitDialog}
          onOpenChange={setShowSplitDialog}
          order={order}
          onSplit={async (splits) => {
            try {
              await splitOrder.mutateAsync({ splits });
              toast.success("Zamówienie zostało podzielone");
              setShowSplitDialog(false);
            } catch (error) {
              toast.error(getErrorMessage(error));
            }
          }}
          isLoading={splitOrder.isPending}
        />
      )}
    </div>
  );
}

function SplitOrderDialog({
  open,
  onOpenChange,
  order,
  onSplit,
  isLoading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: { items?: { name: string; sku?: string; quantity: number; price: number }[]; currency: string };
  onSplit: (splits: { items: { name: string; sku?: string; quantity: number; price: number }[] }[]) => void;
  isLoading: boolean;
}) {
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
          <DialogTitle>Podziel zamówienie</DialogTitle>
          <DialogDescription>
            Przydziel pozycje do dwóch nowych zamówień. Każda pozycja trafi
            do zamówienia 1 lub 2.
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
                <Label className="text-xs">Zamówienie:</Label>
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
            <p className="font-medium">Zamówienie 1</p>
            <p className="text-muted-foreground">{split1Items.length} pozycji</p>
            <p className="font-medium">{formatCurrency(split1Total, order.currency)}</p>
          </div>
          <div>
            <p className="font-medium">Zamówienie 2</p>
            <p className="text-muted-foreground">{split2Items.length} pozycji</p>
            <p className="font-medium">{formatCurrency(split2Total, order.currency)}</p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isLoading || split1Items.length === 0 || split2Items.length === 0}
          >
            {isLoading ? "Dzielenie..." : "Podziel zamówienie"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
