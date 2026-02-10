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
import { formatDate, formatCurrency } from "@/lib/utils";
import { Download } from "lucide-react";
import type { Order } from "@/types/api";

const SOURCE_LABELS: Record<string, string> = {
  manual: "Reczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

export default function OrdersPage() {
  const router = useRouter();
  const [filters, setFilters] = useState<{ status?: string; source?: string; search?: string; payment_status?: string }>({});
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const { data, isLoading } = useOrders({
    ...filters,
    limit,
    offset,
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
    },
    {
      header: "Zrodlo",
      accessorKey: "source",
      cell: (row) => SOURCE_LABELS[row.source] || row.source,
    },
    {
      header: "Status",
      accessorKey: "status",
      cell: (row) => <StatusBadge status={row.status} statusMap={ORDER_STATUSES} />,
    },
    {
      header: "Kwota",
      accessorKey: "total_amount",
      cell: (row) => formatCurrency(row.total_amount, row.currency),
    },
    {
      header: "Platnosc",
      accessorKey: "payment_status",
      cell: (row) => <StatusBadge status={row.payment_status} statusMap={PAYMENT_STATUSES} />,
    },
    {
      header: "Data",
      accessorKey: "created_at",
      cell: (row) => formatDate(row.created_at),
    },
  ];

  const handleFilterChange = (newFilters: { status?: string; source?: string; search?: string; payment_status?: string }) => {
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
          <h1 className="text-2xl font-bold">Zamowienia</h1>
          <p className="text-muted-foreground mt-1">
            Zarzadzaj zamowieniami w systemie
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => exportOrdersCSV({ ...filters, limit: 10000, offset: 0 })}>
            <Download className="mr-2 h-4 w-4" />
            Eksportuj CSV
          </Button>
          <Button asChild>
            <Link href="/orders/new">Nowe zamowienie</Link>
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
          emptyMessage="Brak zamowien do wyswietlenia"
          onRowClick={(row) => router.push(`/orders/${row.id}`)}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
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
