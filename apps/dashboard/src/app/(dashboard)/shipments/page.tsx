"use client";

import { useState } from "react";
import Link from "next/link";
import { Plus, Truck, Printer } from "lucide-react";
import { toast } from "sonner";
import { EmptyState } from "@/components/shared/empty-state";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { ShipmentFilters } from "@/components/shipments/shipment-filters";
import { useShipments, useBatchLabels } from "@/hooks/use-shipments";
import { SHIPMENT_STATUSES } from "@/lib/constants";
import { formatDate, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { Shipment } from "@/types/api";

const DEFAULT_LIMIT = 20;

export default function ShipmentsPage() {
  const [filters, setFilters] = useState<{
    status?: string;
    provider?: string;
  }>({});
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const batchLabels = useBatchLabels();

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setPagination((prev) => ({ ...prev, offset: 0 }));
  };

  const { data, isLoading, isError, refetch } = useShipments({
    ...pagination,
    ...filters,
    sort_by: sortBy,
    sort_order: sortOrder,
  });

  const columns = [
    {
      header: "ID",
      accessorKey: "id" as const,
      cell: (shipment: Shipment) => (
        <Link
          href={`/shipments/${shipment.id}`}
          className="font-mono text-xs text-primary hover:underline"
        >
          {shortId(shipment.id)}
        </Link>
      ),
    },
    {
      header: "Zamówienie",
      accessorKey: "order_id" as const,
      cell: (shipment: Shipment) => (
        <Link
          href={`/orders/${shipment.order_id}`}
          className="font-mono text-xs text-primary hover:underline"
        >
          {shortId(shipment.order_id)}
        </Link>
      ),
    },
    {
      header: "Dostawca",
      accessorKey: "provider" as const,
      sortable: true,
      cell: (shipment: Shipment) => (
        <span className="uppercase text-sm">{shipment.provider}</span>
      ),
    },
    {
      header: "Status",
      accessorKey: "status" as const,
      sortable: true,
      cell: (shipment: Shipment) => (
        <StatusBadge status={shipment.status} statusMap={SHIPMENT_STATUSES} />
      ),
    },
    {
      header: "Numer śledzenia",
      accessorKey: "tracking_number" as const,
      cell: (shipment: Shipment) => (
        <span className="text-sm">
          {shipment.tracking_number ?? "-"}
        </span>
      ),
    },
    {
      header: "Data utworzenia",
      accessorKey: "created_at" as const,
      sortable: true,
      cell: (shipment: Shipment) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(shipment.created_at)}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Przesyłki</h1>
          <p className="text-muted-foreground">
            Zarządzaj przesyłkami zamówień
          </p>
        </div>
        <div className="flex items-center gap-2">
          {selectedIds.size > 0 && (
            <Button
              variant="outline"
              onClick={async () => {
                try {
                  await batchLabels.mutateAsync({
                    shipment_ids: Array.from(selectedIds),
                  });
                  toast.success("Etykiety zostaly pobrane");
                } catch (error) {
                  toast.error(getErrorMessage(error));
                }
              }}
              disabled={batchLabels.isPending}
            >
              <Printer className="h-4 w-4" />
              Drukuj etykiety ({selectedIds.size})
            </Button>
          )}
          <Button asChild>
            <Link href="/shipments/new">
              <Plus className="h-4 w-4" />
              Nowa przesyłka
            </Link>
          </Button>
        </div>
      </div>

      <ShipmentFilters
        filters={filters}
        onFilterChange={(newFilters) => {
          setFilters(newFilters);
          setPagination((prev) => ({ ...prev, offset: 0 }));
        }}
      />

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

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        isLoading={isLoading}
        emptyState={
          <EmptyState
            icon={Truck}
            title="Brak przesyłek"
            description="Nie znaleziono przesyłek do wyświetlenia."
            action={{ label: "Nowa przesyłka", href: "/shipments/new" }}
          />
        }
        sortBy={sortBy}
        sortOrder={sortOrder}
        onSort={handleSort}
        selectable
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
        rowId={(row) => row.id}
      />

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
    </div>
  );
}
