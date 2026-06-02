"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Repeat, Trash2, Pause, Play, XCircle } from "lucide-react";
import { toast } from "sonner";
import {
  useRecurringOrders,
  useDeleteRecurringOrder,
  usePauseRecurringOrder,
  useResumeRecurringOrder,
  useCancelRecurringOrder,
} from "@/hooks/use-recurring-orders";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DensityToggle } from "@/components/shared/density-toggle";
import { getErrorMessage } from "@/lib/api-client";
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
import { useTranslations } from "next-intl";

const DEFAULT_LIMIT = 20;

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

export default function RecurringOrdersPage() {
  const t = useTranslations("recurringOrders");
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const { data, isLoading, isError, refetch } = useRecurringOrders({
    status: statusFilter || undefined,
    ...pagination,
  });
  const deleteRecurringOrder = useDeleteRecurringOrder();
  const pauseRecurringOrder = usePauseRecurringOrder();
  const resumeRecurringOrder = useResumeRecurringOrder();
  const cancelRecurringOrder = useCancelRecurringOrder();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const orders = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteRecurringOrder.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala usunieta");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handlePause = (id: string) => {
    pauseRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala wstrzymana");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const handleResume = (id: string) => {
    resumeRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala wznowiona");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  const handleCancel = (id: string) => {
    cancelRecurringOrder.mutate(id, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala anulowana");
        refetch();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    });
  };

  return (
    <>
      <PageHeader
        title="Subskrypcje"
        description="Zamowienia cykliczne dla stalych klientow B2B"
        action={{ label: "Nowa subskrypcja", href: "/recurring-orders/new" }}
      />

      <div className="flex items-center gap-2 mb-4">
        <Select
          value={statusFilter}
          onValueChange={(value) => {
            setStatusFilter(value === "all" ? "" : value);
            setPagination((prev) => ({ ...prev, offset: 0 }));
          }}
        >
          <SelectTrigger className="w-48">
            <SelectValue placeholder="Filtruj status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie statusy</SelectItem>
            <SelectItem value="active">Aktywna</SelectItem>
            <SelectItem value="paused">Wstrzymana</SelectItem>
            <SelectItem value="cancelled">Anulowana</SelectItem>
            <SelectItem value="completed">Zakonczona</SelectItem>
          </SelectContent>
        </Select>
        <div className="ml-auto">
          <DensityToggle />
        </div>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {orders.length === 0 ? (
        <EmptyState
          icon={Repeat}
          title="Brak subskrypcji"
          description={t("utworzPierwszaSubskrypcjeAbyAutomatycznieGenerowac")}
          action={{ label: "Nowa subskrypcja", href: "/recurring-orders/new" }}
        />
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Klient</TableHead>
                  <TableHead>Czestotliwosc</TableHead>
                  <TableHead>Nastepne zamowienie</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Utworzonych</TableHead>
                  <TableHead className="text-right">Maks.</TableHead>
                  <TableHead className="w-[140px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {orders.map((order) => (
                  <TableRow
                    key={order.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => router.push(`/recurring-orders/${order.id}`)}
                  >
                    <TableCell className="font-medium">
                      <div>{order.customer_name}</div>
                      {order.customer_email && (
                        <div className="text-xs text-muted-foreground">
                          {order.customer_email}
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      {frequencyLabels[order.frequency] || order.frequency}
                    </TableCell>
                    <TableCell>{formatDate(order.next_order_date)}</TableCell>
                    <TableCell>
                      <Badge
                        variant="secondary"
                        className={statusColors[order.status] || ""}
                      >
                        {statusLabels[order.status] || order.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      {order.total_orders_created}
                    </TableCell>
                    <TableCell className="text-right">
                      {order.max_orders ?? "---"}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                        {order.status === "active" && (
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            title="Wstrzymaj"
                            onClick={() => handlePause(order.id)}
                          >
                            <Pause className="h-4 w-4 text-yellow-600" />
                          </Button>
                        )}
                        {order.status === "paused" && (
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            title="Wznow"
                            onClick={() => handleResume(order.id)}
                          >
                            <Play className="h-4 w-4 text-green-600" />
                          </Button>
                        )}
                        {(order.status === "active" || order.status === "paused") && (
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            title="Anuluj"
                            onClick={() => handleCancel(order.id)}
                          >
                            <XCircle className="h-4 w-4 text-red-500" />
                          </Button>
                        )}
                        {order.total_orders_created === 0 && (
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            title="Usun"
                            onClick={() => setDeleteId(order.id)}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {data && (
            <DataTablePagination
              total={data.total}
              limit={data.limit}
              offset={data.offset}
              onPageChange={(offset) =>
                setPagination((prev) => ({ ...prev, offset }))
              }
              onPageSizeChange={(limit) =>
                setPagination({ limit, offset: 0 })
              }
            />
          )}
        </>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usun subskrypcje"
        description="Czy na pewno chcesz usunac te subskrypcje? Ta operacja jest nieodwracalna."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteRecurringOrder.isPending}
      />
    </>
  );
}
