"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { useOrder, useUpdateOrder, useDeleteOrder, useTransitionOrderStatus } from "@/hooks/use-orders";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ORDER_STATUSES, PAYMENT_STATUSES } from "@/lib/constants";
import { formatDate, formatCurrency } from "@/lib/utils";
import type { CreateOrderRequest } from "@/types/api";

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { data: order, isLoading } = useOrder(params.id);
  const updateOrder = useUpdateOrder(params.id);
  const deleteOrder = useDeleteOrder();
  const transitionStatus = useTransitionOrderStatus(params.id);

  const handleUpdate = async (data: CreateOrderRequest) => {
    try {
      await updateOrder.mutateAsync(data);
      toast.success("Zamowienie zostalo zaktualizowane");
      setIsEditing(false);
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas aktualizacji zamowienia"
      );
    }
  };

  const handleDelete = async () => {
    try {
      await deleteOrder.mutateAsync(params.id);
      toast.success("Zamowienie zostalo usuniete");
      router.push("/orders");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas usuwania zamowienia"
      );
    }
  };

  const handleTransition = async (newStatus: string, force?: boolean) => {
    try {
      await transitionStatus.mutateAsync({ status: newStatus, force });
      toast.success("Status zamowienia zostal zmieniony");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas zmiany statusu"
      );
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
        <p className="text-muted-foreground">Nie znaleziono zamowienia</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/orders")}>
          Wróc do listy
        </Button>
      </div>
    );
  }

  if (isEditing) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Edycja zamowienia</h1>
          <p className="text-muted-foreground mt-1">
            Zamowienie {order.id.slice(0, 8)}
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
            Zamowienie {order.id.slice(0, 8)}
          </h1>
          <p className="text-muted-foreground mt-1">
            Utworzone {formatDate(order.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setIsEditing(true)}>
            Edytuj
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            Usun
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Dane zamowienia</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Status</p>
                  <div className="mt-1">
                    <StatusBadge status={order.status} statusMap={ORDER_STATUSES} />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Zrodlo</p>
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
                  <p className="text-sm text-muted-foreground">Platnosc</p>
                  <div className="mt-1">
                    <StatusBadge status={order.payment_status} statusMap={PAYMENT_STATUSES} />
                  </div>
                </div>
                {order.payment_method && (
                  <div>
                    <p className="text-sm text-muted-foreground">Metoda platnosci</p>
                    <p className="mt-1 font-medium">{order.payment_method}</p>
                  </div>
                )}
                {order.paid_at && (
                  <div>
                    <p className="text-sm text-muted-foreground">Data oplacenia</p>
                    <p className="mt-1 font-medium">{formatDate(order.paid_at)}</p>
                  </div>
                )}
                {order.external_id && (
                  <div>
                    <p className="text-sm text-muted-foreground">ID zewnetrzne</p>
                    <p className="mt-1 font-mono text-sm">{order.external_id}</p>
                  </div>
                )}
              </div>

              {order.notes && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">Notatki</p>
                    <p className="mt-1 text-sm">{order.notes}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {order.items && order.items.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Pozycje zamowienia</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Produkt</TableHead>
                      <TableHead>SKU</TableHead>
                      <TableHead className="text-right">Ilosc</TableHead>
                      <TableHead className="text-right">Cena</TableHead>
                      <TableHead className="text-right">Wartosc</TableHead>
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
            </CardContent>
          </Card>

          <OrderTimeline orderId={params.id} />
        </div>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Usuwanie zamowienia</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz usunac to zamowienie? Ta operacja jest nieodwracalna.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteOrder.isPending}
            >
              {deleteOrder.isPending ? "Usuwanie..." : "Usun zamowienie"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
