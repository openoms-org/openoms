"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useReturns } from "@/hooks/use-returns";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { RotateCcw } from "lucide-react";
import { RETURN_STATUSES } from "@/lib/constants";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { EmptyState } from "@/components/shared/empty-state";
import type { Return } from "@/types/api";

export default function ReturnsPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useReturns({
    status: statusFilter || undefined,
    limit,
    offset,
  });

  const columns: ColumnDef<Return>[] = [
    {
      header: "Status",
      accessorKey: "status",
      cell: (row) => <StatusBadge status={row.status} statusMap={RETURN_STATUSES} />,
    },
    {
      header: "Zamówienie",
      accessorKey: "order_id",
      cell: (row) => (
        <Link
          href={`/orders/${row.order_id}`}
          className="font-mono text-xs text-primary hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {shortId(row.order_id)}
        </Link>
      ),
    },
    {
      header: "Powód",
      accessorKey: "reason",
      cell: (row) => (
        <span className="text-sm">
          {row.reason.length > 50 ? `${row.reason.slice(0, 50)}...` : row.reason}
        </span>
      ),
    },
    {
      header: "Kwota zwrotu",
      accessorKey: "refund_amount",
      cell: (row) => formatCurrency(row.refund_amount),
    },
    {
      header: "Data",
      accessorKey: "created_at",
      cell: (row) => formatDate(row.created_at),
    },
  ];

  const handleStatusChange = (value: string) => {
    setStatusFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Zwroty</h1>
          <p className="text-muted-foreground mt-1">
            Zarządzaj zwrotami i reklamacjami
          </p>
        </div>
        <Button asChild>
          <Link href="/returns/new">Nowy zwrot</Link>
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="w-[200px]">
          <Select value={statusFilter || "all"} onValueChange={handleStatusChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie</SelectItem>
              <SelectItem value="requested">Zgłoszone</SelectItem>
              <SelectItem value="approved">Zatwierdzone</SelectItem>
              <SelectItem value="received">Odebrane</SelectItem>
              <SelectItem value="refunded">Zwrócone</SelectItem>
              <SelectItem value="rejected">Odrzucone</SelectItem>
              <SelectItem value="cancelled">Anulowane</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

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

      <div className="rounded-md border">
        <DataTable<Return>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyState={
            <EmptyState
              icon={RotateCcw}
              title="Brak zwrotów"
              description="Nie znaleziono zwrotów do wyświetlenia."
              action={{ label: "Nowy zwrot", href: "/returns/new" }}
            />
          }
          onRowClick={(row) => router.push(`/returns/${row.id}`)}
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
