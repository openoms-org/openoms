"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useOrders, exportOrdersCSV } from "@/hooks/use-orders";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { OrderFilters } from "@/components/orders/order-filters";
import { BulkActions } from "@/components/orders/bulk-actions";
import { Button } from "@/components/ui/button";
import { ORDER_STATUSES, PAYMENT_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { formatDate, formatCurrency } from "@/lib/utils";
import { Download } from "lucide-react";
import type { Order } from "@/types/api";

const SOURCE_LABELS: Record<string, string> = {
  manual: "Ręczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

export default function OrdersPage() {
  const router = useRouter();
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const [filters, setFilters] = useState<{ status?: string; source?: string; search?: string; payment_status?: string; tag?: string }>({});
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setOffset(0);
  };

  const { data, isLoading } = useOrders({
    ...filters,
    limit,
    offset,
    sort_by: sortBy,
    sort_order: sortOrder,
  });

  const selectedOrders = (data?.items || []).filter((o) => selectedIds.has(o.id));

  const columns: ColumnDef<Order>[] = [
    {
      header: "ID",
      accessorKey: "id",
      cell: (row) => (
        <span className="font-mono text-xs">{row.id.slice(0, 8)}</span>
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
      cell: (row) => SOURCE_LABELS[row.source] || row.source,
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

      {selectedIds.size > 0 && (
        <BulkActions
          selectedOrders={selectedOrders}
          onClearSelection={() => setSelectedIds(new Set())}
        />
      )}

      <div className="rounded-md border">
        <DataTable<Order>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyMessage="Brak zamówień do wyświetlenia"
          onRowClick={(row) => router.push(`/orders/${row.id}`)}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
          sortBy={sortBy}
          sortOrder={sortOrder}
          onSort={handleSort}
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
    </div>
  );
}
