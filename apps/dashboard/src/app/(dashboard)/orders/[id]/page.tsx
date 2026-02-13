"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { Package, RotateCcw, Printer, FileText, Scissors, GitBranch, Headphones, Loader2, Plus, ExternalLink, Copy, Check, StickyNote, Save, Tag, Send, ChevronDown } from "lucide-react";
import { RateShopping } from "@/components/shipping/rate-shopping";
import { AllegroShipmentDialog } from "@/components/integrations/allegro-shipment-dialog";
import { useAllegroCarriers, useAllegroFulfillment, useAllegroTracking } from "@/hooks/use-allegro";
import { useOrder, useUpdateOrder, useDeleteOrder, useTransitionOrderStatus, useDuplicateOrder } from "@/hooks/use-orders";
import { useShipments } from "@/hooks/use-shipments";
import { useReturns } from "@/hooks/use-returns";
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
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ORDER_STATUSES, PAYMENT_STATUSES, SHIPMENT_STATUSES, RETURN_STATUSES, ORDER_PRIORITIES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { formatDate, formatCurrency, shortId, cn } from "@/lib/utils";
import { getErrorMessage, apiFetch } from "@/lib/api-client";
import type { CreateOrderRequest, UpdateOrderRequest } from "@/types/api";

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
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showSplitDialog, setShowSplitDialog] = useState(false);
  const [showCreateTicketDialog, setShowCreateTicketDialog] = useState(false);
  const [returnLinkCopied, setReturnLinkCopied] = useState(false);
  const [internalNotes, setInternalNotes] = useState("");
  const [internalNotesDirty, setInternalNotesDirty] = useState(false);
  const [showAllegroShipmentDialog, setShowAllegroShipmentDialog] = useState(false);
  const [showAllegroFulfillmentDialog, setShowAllegroFulfillmentDialog] = useState(false);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const { data: customFieldsConfig } = useCustomFields();

  const { data: order, isLoading } = useOrder(params.id);
  const updateOrder = useUpdateOrder(params.id);
  const deleteOrder = useDeleteOrder();
  const transitionStatus = useTransitionOrderStatus(params.id);

  const { data: shipmentsData, isLoading: isLoadingShipments } = useShipments({ order_id: params.id });
  const { data: returnsData, isLoading: isLoadingReturns } = useReturns({ order_id: params.id });
  const { data: orderGroups } = useOrderGroups(params.id);
  const splitOrder = useSplitOrder(params.id);
  const { data: ticketsData, isLoading: isLoadingTickets } = useOrderTickets(params.id);
  const createTicket = useCreateOrderTicket(params.id);
  const duplicateOrder = useDuplicateOrder();

  useEffect(() => {
    if (order && !internalNotesDirty) {
      setInternalNotes(order.internal_notes || "");
    }
  }, [order, internalNotesDirty]);

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
    <div className="mx-auto max-w-7xl space-y-6">
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
            onClick={async () => {
              try {
                const res = await apiFetch(`/v1/orders/${params.id}/print`);
                const blob = await res.blob();
                const url = URL.createObjectURL(blob);
                window.open(url, "_blank");
                setTimeout(() => URL.revokeObjectURL(url), 60000);
              } catch {
                toast.error("Nie udało się pobrać wydruku");
              }
            }}
          >
            <Printer className="mr-2 h-4 w-4" />
            Drukuj
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
                toast.error("Nie udało się pobrać listu przewozowego");
              }
            }}
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
              Zgłoś zwrot
            </Link>
          </Button>
          <Button
            variant="outline"
            onClick={() => {
              const returnUrl = `${window.location.origin}/return-request?order_id=${params.id}`;
              navigator.clipboard.writeText(returnUrl).then(() => {
                setReturnLinkCopied(true);
                toast.success("Link do formularza zwrotu skopiowany do schowka");
                setTimeout(() => setReturnLinkCopied(false), 2000);
              });
            }}
          >
            {returnLinkCopied ? (
              <Check className="mr-2 h-4 w-4" />
            ) : (
              <ExternalLink className="mr-2 h-4 w-4" />
            )}
            Link do zwrotu
          </Button>
          {order && order.status !== "merged" && order.status !== "split" && order.items && order.items.length >= 2 && (
            <Button variant="outline" onClick={() => setShowSplitDialog(true)}>
              <Scissors className="mr-2 h-4 w-4" />
              Podziel zamówienie
            </Button>
          )}
          <Button
            variant="outline"
            onClick={async () => {
              try {
                const newOrder = await duplicateOrder.mutateAsync(params.id);
                toast.success("Zamówienie zostało zduplikowane");
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
            Duplikuj zamówienie
          </Button>
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
                  <p className="text-sm text-muted-foreground">Priorytet</p>
                  <div className="mt-1">
                    <Select
                      value={order.priority || "normal"}
                      onValueChange={async (value) => {
                        try {
                          await updateOrder.mutateAsync({ priority: value as "urgent" | "high" | "normal" | "low" });
                          toast.success("Priorytet zamówienia został zmieniony");
                        } catch (error) {
                          toast.error(getErrorMessage(error));
                        }
                      }}
                    >
                      <SelectTrigger className="w-[140px]" size="sm">
                        <SelectValue>
                          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${ORDER_PRIORITIES[order.priority || "normal"]?.color || ""}`}>
                            {ORDER_PRIORITIES[order.priority || "normal"]?.label || "Normalny"}
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
            <CollapsibleSection
              title="Pozycje zamówienia"
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
            </CollapsibleSection>
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

          <CollapsibleSection
            title="Przesyłki"
            icon={Package}
            defaultOpen={!!(shipmentsData?.items && shipmentsData.items.length > 0)}
            badge={
              shipmentsData?.items && shipmentsData.items.length > 0 ? (
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  {shipmentsData.items.length}
                </span>
              ) : undefined
            }
          >
            {isLoadingShipments ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            ) : shipmentsData?.items && shipmentsData.items.length > 0 ? (
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
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Package className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">Brak przesyłek dla tego zamówienia.</p>
              </div>
            )}
          </CollapsibleSection>

          <RateShopping
            defaultToPostalCode={order.shipping_address?.postal_code}
            onSelectRate={(rate) => {
              router.push(
                `/shipments/new?order_id=${params.id}&carrier=${rate.carrier_code}`
              );
            }}
          />

          <CollapsibleSection
            title="Zwroty"
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
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <RotateCcw className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">Brak zwrotów dla tego zamówienia.</p>
              </div>
            )}
          </CollapsibleSection>

          {/* Helpdesk Tickets */}
          <CollapsibleSection
            title="Zgłoszenia"
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
                Utwórz zgłoszenie
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
                    <TableHead>Temat</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Utworzono</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {ticketsData.tickets.map((ticket) => (
                    <TableRow key={ticket.id}>
                      <TableCell className="font-mono text-sm">#{ticket.id}</TableCell>
                      <TableCell className="font-medium">{ticket.subject}</TableCell>
                      <TableCell>
                        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium">
                          {ticket.status === 2 ? "Otwarty" : ticket.status === 3 ? "Oczekujący" : ticket.status === 4 ? "Rozwiązany" : ticket.status === 5 ? "Zamknięty" : `Status ${ticket.status}`}
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
                <p className="text-sm text-muted-foreground">Brak zgłoszeń dla tego zamówienia.</p>
              </div>
            )}
          </CollapsibleSection>

          {/* Merge/Split History */}
          {(order.merged_into || order.split_from || (orderGroups && orderGroups.length > 0)) && (
            <CollapsibleSection
              title="Historia scalania/podziału"
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
              </div>
            </CollapsibleSection>
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

          {/* Notatki wewnętrzne */}
          <Card className="border-amber-300 dark:border-amber-700 bg-amber-50/50 dark:bg-amber-950/20">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
                <StickyNote className="h-4 w-4" />
                Notatki wewnętrzne
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={internalNotes}
                onChange={(e) => {
                  setInternalNotes(e.target.value);
                  setInternalNotesDirty(true);
                }}
                placeholder="Notatki widoczne tylko dla zespołu..."
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
                      toast.success("Notatki wewnętrzne zapisane");
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
                  Zapisz notatki
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

      {/* Allegro Shipment Dialog */}
      {order && order.source === "allegro" && order.external_id && (
        <AllegroShipmentDialog
          open={showAllegroShipmentDialog}
          onOpenChange={setShowAllegroShipmentDialog}
          order={order}
        />
      )}

      {/* Allegro Fulfillment Dialog */}
      {order && order.source === "allegro" && order.external_id && (
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
            toast.success("Zgłoszenie utworzone");
            setShowCreateTicketDialog(false);
          } catch (error) {
            toast.error(getErrorMessage(error));
          }
        }}
        isLoading={createTicket.isPending}
      />
    </div>
  );
}

function CreateTicketDialog({
  open,
  onOpenChange,
  customerEmail,
  onSubmit,
  isLoading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customerEmail: string;
  onSubmit: (data: { subject: string; description: string; email: string }) => void;
  isLoading: boolean;
}) {
  const [subject, setSubject] = useState("");
  const [description, setDescription] = useState("");
  const [email, setEmail] = useState(customerEmail);

  useEffect(() => {
    if (!open) {
      setSubject("");
      setDescription("");
      setEmail(customerEmail);
    }
  }, [open, customerEmail]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Utwórz zgłoszenie</DialogTitle>
          <DialogDescription>
            Utwórz zgłoszenie w systemie Freshdesk dla tego zamówienia.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>Email klienta</Label>
            <Input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="klient@example.com"
              type="email"
            />
          </div>
          <div>
            <Label>Temat</Label>
            <Input
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Temat zgłoszenia..."
            />
          </div>
          <div>
            <Label>Opis</Label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Opis problemu..."
              rows={4}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button
            onClick={() => onSubmit({ subject, description, email })}
            disabled={!subject || !description || !email || isLoading}
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : null}
            Utwórz zgłoszenie
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

function AllegroFulfillmentDialog({
  open,
  onOpenChange,
  orderId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
}) {
  const [fulfillmentStatus, setFulfillmentStatus] = useState("SENT");
  const [carrierId, setCarrierId] = useState("");
  const [waybill, setWaybill] = useState("");

  const { data: carriersData } = useAllegroCarriers();
  const fulfillmentMutation = useAllegroFulfillment(orderId);
  const trackingMutation = useAllegroTracking(orderId);

  useEffect(() => {
    if (!open) {
      setFulfillmentStatus("SENT");
      setCarrierId("");
      setWaybill("");
    }
  }, [open]);

  const handleSubmit = async () => {
    try {
      // Update fulfillment status
      await fulfillmentMutation.mutateAsync({ status: fulfillmentStatus });

      // Add tracking if carrier and waybill provided
      if (carrierId && waybill) {
        await trackingMutation.mutateAsync({ carrier_id: carrierId, waybill });
      }

      toast.success("Status realizacji Allegro zaktualizowany");
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
          <DialogTitle>Wyslij do Allegro</DialogTitle>
          <DialogDescription>
            Zaktualizuj status realizacji zamowienia na Allegro i dodaj numer przesylki.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>Status realizacji</Label>
            <Select value={fulfillmentStatus} onValueChange={setFulfillmentStatus}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="SENT">Wyslane (SENT)</SelectItem>
                <SelectItem value="PICKED_UP">Odebrane (PICKED_UP)</SelectItem>
                <SelectItem value="READY_FOR_SHIPMENT">Gotowe do wysylki</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Separator />
          <div>
            <Label>Dostawca (opcjonalnie)</Label>
            <Select value={carrierId} onValueChange={setCarrierId}>
              <SelectTrigger>
                <SelectValue placeholder="Wybierz dostawce..." />
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
            <Label>Numer przesylki (opcjonalnie)</Label>
            <Input
              value={waybill}
              onChange={(e) => setWaybill(e.target.value)}
              placeholder="np. 6280012345678"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Send className="mr-2 h-4 w-4" />
            )}
            Wyslij
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
