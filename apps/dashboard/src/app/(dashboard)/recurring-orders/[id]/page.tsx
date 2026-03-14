"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Pause, Play, XCircle } from "lucide-react";
import { toast } from "sonner";
import {
  useRecurringOrder,
  usePauseRecurringOrder,
  useResumeRecurringOrder,
  useCancelRecurringOrder,
} from "@/hooks/use-recurring-orders";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { Button } from "@/components/ui/button";
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
import { useTranslations } from "next-intl";

const statusColors: Record<string, string> = {
  active: "bg-green-500/10 text-green-700 dark:text-green-400",
  paused: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400",
  cancelled: "bg-red-500/10 text-red-700 dark:text-red-400",
  completed: "bg-gray-500/10 text-gray-700 dark:text-gray-400",
};

const statusLabels: Record<string, string> = {
  active: "Aktywna",
  paused: "Wstrzymana",
  cancelled: "Anulowana",
  completed: "Zakonczona",
};

const frequencyLabels: Record<string, string> = {
  daily: "Codziennie",
  weekly: "Co tydzien",
  biweekly: "Co 2 tygodnie",
  monthly: "Co miesiac",
  quarterly: "Co kwartal",
};

export default function RecurringOrderDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const t = useTranslations("recurringOrders");
  const { id } = use(params);
  const router = useRouter();
  const { data: order, isLoading, refetch } = useRecurringOrder(id);
  const pauseRecurringOrder = usePauseRecurringOrder();
  const resumeRecurringOrder = useResumeRecurringOrder();
  const cancelRecurringOrder = useCancelRecurringOrder();

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!order) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">Nie znaleziono subskrypcji</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/recurring-orders")}>
          {t("wrocDoListy")}
        </Button>
      </div>
    );
  }

  const handlePause = () => {
    pauseRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala wstrzymana");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const handleResume = () => {
    resumeRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala wznowiona");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const handleCancel = () => {
    cancelRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala anulowana");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const items = order.items ?? [];
  const totalAmount = items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0);

  return (
    <div className="mx-auto max-w-4xl">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/recurring-orders")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">
            {order.customer_name}
          </h1>
          <p className="text-muted-foreground text-sm">
            Subskrypcja {frequencyLabels[order.frequency] || order.frequency}
          </p>
        </div>
        <Badge
          variant="secondary"
          className={statusColors[order.status] || ""}
        >
          {statusLabels[order.status] || order.status}
        </Badge>
        <div className="flex items-center gap-2">
          {order.status === "active" && (
            <Button variant="outline" size="sm" onClick={handlePause}>
              <Pause className="h-4 w-4 mr-1" />
              Wstrzymaj
            </Button>
          )}
          {order.status === "paused" && (
            <Button variant="outline" size="sm" onClick={handleResume}>
              <Play className="h-4 w-4 mr-1" />
              Wznow
            </Button>
          )}
          {(order.status === "active" || order.status === "paused") && (
            <Button variant="destructive" size="sm" onClick={handleCancel}>
              <XCircle className="h-4 w-4 mr-1" />
              Anuluj
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6 mb-6">
        <Card>
          <CardHeader>
            <CardTitle>Szczegoly</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Klient</dt>
                <dd className="font-medium">{order.customer_name}</dd>
              </div>
              {order.customer_email && (
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">E-mail</dt>
                  <dd>{order.customer_email}</dd>
                </div>
              )}
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Czestotliwosc</dt>
                <dd>{frequencyLabels[order.frequency] || order.frequency}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Interwale (dni)</dt>
                <dd>{order.interval_days}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Utworzona</dt>
                <dd>{formatDate(order.created_at)}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Harmonogram</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Nastepne zamowienie</dt>
                <dd className="font-medium">{formatDate(order.next_order_date)}</dd>
              </div>
              {order.last_order_date && (
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Ostatnie zamowienie</dt>
                  <dd>{formatDate(order.last_order_date)}</dd>
                </div>
              )}
              {order.end_date && (
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Data zakonczenia</dt>
                  <dd>{formatDate(order.end_date)}</dd>
                </div>
              )}
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Utworzone zamowienia</dt>
                <dd className="font-medium">
                  {order.total_orders_created}
                  {order.max_orders ? ` / ${order.max_orders}` : ""}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Kwota za cykl</dt>
                <dd className="font-medium">{formatCurrency(totalAmount)}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Pozycje zamowienia</CardTitle>
          <CardDescription>Produkty zamawiane w kazdym cyklu</CardDescription>
        </CardHeader>
        <CardContent>
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground">Brak pozycji</p>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>SKU</TableHead>
                    <TableHead>Produkt</TableHead>
                    <TableHead className="text-right">Ilosc</TableHead>
                    <TableHead className="text-right">Cena jedn.</TableHead>
                    <TableHead className="text-right">Wartosc</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-mono text-sm">{item.sku}</TableCell>
                      <TableCell>{item.product_name}</TableCell>
                      <TableCell className="text-right">{item.quantity}</TableCell>
                      <TableCell className="text-right">
                        {formatCurrency(item.unit_price)}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {formatCurrency(item.quantity * item.unit_price)}
                      </TableCell>
                    </TableRow>
                  ))}
                  <TableRow>
                    <TableCell colSpan={4} className="text-right font-semibold">
                      Razem
                    </TableCell>
                    <TableCell className="text-right font-semibold">
                      {formatCurrency(totalAmount)}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {order.notes && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Notatki</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm whitespace-pre-wrap">{order.notes}</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
