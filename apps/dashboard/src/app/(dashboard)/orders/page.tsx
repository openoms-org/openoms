"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useOrders, exportOrdersCSV } from "@/hooks/use-orders";
import { DataTable, type ColumnDef, type EditableColumnConfig } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { OrderFilters } from "@/components/orders/order-filters";
import { BulkActions } from "@/components/orders/bulk-actions";
import { Button } from "@/components/ui/button";
import { ORDER_STATUSES, PAYMENT_STATUSES, ORDER_SOURCE_LABELS } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { Download, ShoppingCart, Merge } from "lucide-react";
import { EmptyState } from "@/components/shared/empty-state";
import { apiClient } from "@/lib/api-client";
import { getErrorMessage } from "@/lib/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { useMergeOrders } from "@/hooks/use-order-groups";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Order } from "@/types/api";

export default function OrdersPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const [filters, setFilters] = useState<{ status?: string; source?: string; search?: string; payment_status?: string; tag?: string }>({});
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [showMergeDialog, setShowMergeDialog] = useState(false);
  const mergeOrders = useMergeOrders();

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setOffset(0);
  };

  const { data, isLoading, isError, refetch } = useOrders({
    ...filters,
    limit,
    offset,
    sort_by: sortBy,
    sort_order: sortOrder,
  });

  const selectedOrders = (data?.items || []).filter((o) => selectedIds.has(o.id));

  const statusOptions = useMemo(() => {
    return Object.entries(orderStatuses).map(([key, val]) => ({
      value: key,
      label: val.label,
    }));
  }, [orderStatuses]);

  const editableColumns = useMemo<EditableColumnConfig<Order>[]>(
    () => [
      {
        accessorKey: "status",
        type: "select",
        options: statusOptions,
        onSave: async (row, value) => {
          await apiClient<Order>(`/v1/orders/${row.id}/status`, {
            method: "POST",
            body: JSON.stringify({ status: value as string }),
          });
          queryClient.invalidateQueries({ queryKey: ["orders"] });
        },
      },
      {
        accessorKey: "notes",
        type: "text",
        onSave: async (row, value) => {
          await apiClient<Order>(`/v1/orders/${row.id}`, {
            method: "PATCH",
            body: JSON.stringify({ notes: value as string }),
          });
          queryClient.invalidateQueries({ queryKey: ["orders"] });
        },
      },
    ],
    [statusOptions, queryClient]
  );

  const columns: ColumnDef<Order>[] = [
    {
      header: "ID",
      accessorKey: "id",
      cell: (row) => (
        <span className="font-mono text-xs">{shortId(row.id)}</span>
      ),
    },
    {
      header: "Klient",
      accessorKey: "customer_name",
      sortable: true,
    },
    {
      header: "Źródło",
      accessorKey: "source",
      cell: (row) => ORDER_SOURCE_LABELS[row.source] || row.source,
      sortable: true,
    },
    {
      header: "Status",
      accessorKey: "status",
      cell: (row) => <StatusBadge status={row.status} statusMap={orderStatuses} />,
      sortable: true,
    },
    {
      header: "Kwota",
      accessorKey: "total_amount",
      cell: (row) => formatCurrency(row.total_amount, row.currency),
      sortable: true,
    },
    {
      header: "Płatność",
      accessorKey: "payment_status",
      cell: (row) => <StatusBadge status={row.payment_status} statusMap={PAYMENT_STATUSES} />,
      sortable: true,
    },
    {
      header: "Notatki",
      accessorKey: "notes",
      cell: (row) => (
        <span className="text-sm text-muted-foreground truncate max-w-[200px] inline-block">
          {row.notes || "—"}
        </span>
      ),
    },
    {
      header: "Tagi",
      accessorKey: "tags" as const,
      cell: (row: Order) => (
        <div className="flex flex-wrap gap-1">
          {row.tags?.map((tag) => (
            <span key={tag} className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
              {tag}
            </span>
          ))}
        </div>
      ),
    },
    {
      header: "Data",
      accessorKey: "created_at",
      cell: (row) => formatDate(row.created_at),
      sortable: true,
    },
  ];

  const handleFilterChange = (newFilters: { status?: string; source?: string; search?: string; payment_status?: string; tag?: string }) => {
    setFilters(newFilters);
    setOffset(0);
    setSelectedIds(new Set());
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
    setSelectedIds(new Set());
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
    setSelectedIds(new Set());
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Zamówienia</h1>
          <p className="text-muted-foreground mt-1">
            Zarządzaj zamówieniami w systemie
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => exportOrdersCSV({ ...filters, limit: 10000, offset: 0 })}>
            <Download className="mr-2 h-4 w-4" />
            Eksportuj CSV
          </Button>
          <Button asChild>
            <Link href="/orders/new">Nowe zamówienie</Link>
          </Button>
        </div>
      </div>

      <OrderFilters filters={filters} onFilterChange={handleFilterChange} />

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {selectedIds.size > 0 && (
        <div className="space-y-3">
          <BulkActions
            selectedOrders={selectedOrders}
            onClearSelection={() => setSelectedIds(new Set())}
          />
          {selectedIds.size >= 2 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowMergeDialog(true)}
            >
              <Merge className="mr-2 h-4 w-4" />
              Scal zamówienia ({selectedIds.size})
            </Button>
          )}
        </div>
      )}

      <div className="rounded-md border">
        <DataTable<Order>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyState={
            <EmptyState
              icon={ShoppingCart}
              title="Brak zamówień"
              description="Nie znaleziono zamówień do wyświetlenia."
              action={{ label: "Nowe zamówienie", href: "/orders/new" }}
            />
          }
          onRowClick={(row) => router.push(`/orders/${row.id}`)}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
          sortBy={sortBy}
          sortOrder={sortOrder}
          onSort={handleSort}
          editableColumns={editableColumns}
        />
      </div>

      {data && (
        <DataTablePagination
          total={data.total}
          limit={limit}
          offset={offset}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      )}

      <Dialog open={showMergeDialog} onOpenChange={setShowMergeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Scal zamówienia</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz scalić {selectedIds.size} zamówień w jedno?
              Pozycje ze wszystkich zamówień zostaną połączone, a kwoty
              zsumowane. Oryginalne zamówienia otrzymają status
              &quot;merged&quot;.
            </DialogDescription>
          </DialogHeader>
          <div className="text-sm text-muted-foreground">
            <p className="font-medium mb-1">Wybrane zamówienia:</p>
            <ul className="list-disc pl-5 space-y-1">
              {selectedOrders.map((o) => (
                <li key={o.id}>
                  {shortId(o.id)} - {o.customer_name} ({formatCurrency(o.total_amount, o.currency)})
                </li>
              ))}
            </ul>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowMergeDialog(false)}>
              Anuluj
            </Button>
            <Button
              onClick={async () => {
                try {
                  await mergeOrders.mutateAsync({
                    order_ids: Array.from(selectedIds),
                  });
                  toast.success("Zamówienia zostały scalone");
                  setShowMergeDialog(false);
                  setSelectedIds(new Set());
                } catch (error) {
                  toast.error(getErrorMessage(error));
                }
              }}
              disabled={mergeOrders.isPending}
            >
              {mergeOrders.isPending ? "Scalanie..." : "Scal zamówienia"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
