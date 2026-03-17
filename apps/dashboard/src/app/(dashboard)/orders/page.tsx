"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useOrders, exportOrdersCSV } from "@/hooks/use-orders";
import { DataTable, type ColumnDef, type EditableColumnConfig } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { OrderFilters } from "@/components/orders/order-filters";
import { BulkActions } from "@/components/orders/bulk-actions";
import { KanbanBoard } from "@/components/orders/kanban-board";
import { Button } from "@/components/ui/button";
import { ORDER_STATUSES, PAYMENT_STATUSES, ORDER_SOURCE_LABELS, ORDER_PRIORITIES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { formatDate, formatCurrency, shortId, cn } from "@/lib/utils";
import { Download, ShoppingCart, Merge, Printer, LayoutGrid, Columns3 } from "lucide-react";
import { EmptyState } from "@/components/shared/empty-state";
import { apiClient } from "@/lib/api-client";
import { getErrorMessage } from "@/lib/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { useMergeOrders } from "@/hooks/use-order-groups";
import { useBatchLabels } from "@/hooks/use-shipments";
import { toast } from "sonner";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { DensityToggle } from "@/components/shared/density-toggle";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Order } from "@/types/api";

type ViewMode = "table" | "kanban";

function useViewMode(): [ViewMode, (mode: ViewMode) => void] {
  const [view, setViewState] = useState<ViewMode>(() => {
    if (typeof window === "undefined") return "table";
    const saved = localStorage.getItem("orders-view-mode");
    return saved === "kanban" || saved === "table" ? saved : "table";
  });

  const setView = (mode: ViewMode) => {
    setViewState(mode);
    localStorage.setItem("orders-view-mode", mode);
  };

  return [view, setView];
}

export default function OrdersPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const t = useTranslations("orders");
  const tc = useTranslations("common");
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const [filters, setFilters] = useState<{ status?: string; source?: string; search?: string; payment_status?: string; tag?: string; priority?: string }>({});
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [showMergeDialog, setShowMergeDialog] = useState(false);
  const [viewMode, setViewMode] = useViewMode();
  const mergeOrders = useMergeOrders();
  const batchLabels = useBatchLabels();

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
      header: t("columns.id"),
      accessorKey: "id",
      cell: (row) => (
        <span className="font-mono text-xs">{shortId(row.id)}</span>
      ),
    },
    {
      header: t("columns.customer"),
      accessorKey: "customer_name",
      sortable: true,
    },
    {
      header: t("columns.source"),
      accessorKey: "source",
      cell: (row) => ORDER_SOURCE_LABELS[row.source] || row.source,
      sortable: true,
      className: "hidden lg:table-cell",
    },
    {
      header: t("columns.status"),
      accessorKey: "status",
      cell: (row) => <StatusBadge status={row.status} statusMap={orderStatuses} translationPrefix="order" />,
      sortable: true,
    },
    {
      header: t("columns.priority"),
      accessorKey: "priority",
      cell: (row) => {
        const priority = row.priority || "normal";
        const config = ORDER_PRIORITIES[priority];
        if (!config || priority === "normal") return null;
        return (
          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${config.color}`}>
            {config.label}
          </span>
        );
      },
      sortable: true,
      className: "hidden lg:table-cell",
    },
    {
      header: t("columns.amount"),
      accessorKey: "total_amount",
      cell: (row) => formatCurrency(row.total_amount, row.currency),
      sortable: true,
    },
    {
      header: t("columns.payment"),
      accessorKey: "payment_status",
      cell: (row) => <StatusBadge status={row.payment_status} statusMap={PAYMENT_STATUSES} translationPrefix="payment" />,
      sortable: true,
      className: "hidden lg:table-cell",
    },
    {
      header: t("columns.notes"),
      accessorKey: "notes",
      cell: (row) => (
        <span className="text-sm text-muted-foreground truncate max-w-[200px] inline-block">
          {row.notes || "\u2014"}
        </span>
      ),
      className: "hidden md:table-cell",
    },
    {
      header: t("columns.tags"),
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
      className: "hidden lg:table-cell",
    },
    {
      header: t("columns.date"),
      accessorKey: "created_at",
      cell: (row) => formatDate(row.created_at),
      sortable: true,
      className: "hidden md:table-cell",
    },
  ];

  const handleFilterChange = (newFilters: { status?: string; source?: string; search?: string; payment_status?: string; tag?: string; priority?: string }) => {
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
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("subtitle")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <DensityToggle />
          {/* View mode switcher */}
          <TooltipProvider>
            <div className="flex items-center rounded-md border bg-muted p-0.5">
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    onClick={() => setViewMode("table")}
                    className={cn(
                      "inline-flex items-center justify-center rounded-sm px-2.5 py-1.5 text-sm font-medium transition-colors",
                      viewMode === "table"
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    <LayoutGrid className="h-4 w-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>{tc("tableView")}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    onClick={() => setViewMode("kanban")}
                    className={cn(
                      "inline-flex items-center justify-center rounded-sm px-2.5 py-1.5 text-sm font-medium transition-colors",
                      viewMode === "kanban"
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    <Columns3 className="h-4 w-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>{tc("kanbanView")}</TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>

          <Button variant="outline" onClick={() => exportOrdersCSV({ ...filters, limit: 10000, offset: 0 })}>
            <Download className="mr-2 h-4 w-4" />
            {tc("exportCsv")}
          </Button>
          <Button asChild>
            <Link href="/orders/new">{t("newOrder")}</Link>
          </Button>
        </div>
      </div>

      <OrderFilters filters={filters} onFilterChange={handleFilterChange} />

      {isError && viewMode === "table" && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {tc("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {tc("retry")}
          </Button>
        </div>
      )}

      {viewMode === "table" && selectedIds.size > 0 && (
        <div className="space-y-3">
          <BulkActions
            selectedOrders={selectedOrders}
            onClearSelection={() => setSelectedIds(new Set())}
          />
          <div className="flex gap-2">
            {selectedIds.size >= 2 && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowMergeDialog(true)}
              >
                <Merge className="mr-2 h-4 w-4" />
                {t("merge.button", { count: selectedIds.size })}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={async () => {
                try {
                  // Fetch shipments for selected orders and batch download labels
                  const orderIds = Array.from(selectedIds);
                  const responses = await Promise.all(
                    orderIds.map((orderId) =>
                      apiClient<{ items: { id: string; label_url?: string }[] }>(
                        `/v1/shipments?order_id=${orderId}&limit=100`
                      ).catch(() => ({ items: [] }))
                    )
                  );
                  const shipmentIds = responses
                    .flatMap((r) => r.items)
                    .filter((s) => s.label_url)
                    .map((s) => s.id);

                  if (shipmentIds.length === 0) {
                    toast.error(t("labels.noLabels"));
                    return;
                  }

                  await batchLabels.mutateAsync({ shipment_ids: shipmentIds });
                  toast.success(t("labels.downloaded", { count: shipmentIds.length }));
                } catch (error) {
                  toast.error(getErrorMessage(error));
                }
              }}
              disabled={batchLabels.isPending}
            >
              <Printer className="mr-2 h-4 w-4" />
              {t("labels.generate", { count: selectedIds.size })}
            </Button>
          </div>
        </div>
      )}

      {/* Kanban view */}
      {viewMode === "kanban" && (
        <KanbanBoard filters={filters} />
      )}

      {/* Table view */}
      {viewMode === "table" && (
        <>
          <div className="rounded-md border">
            <DataTable<Order>
              columns={columns}
              data={data?.items || []}
              isLoading={isLoading}
              emptyState={
                <EmptyState
                  icon={ShoppingCart}
                  title={t("empty.title")}
                  description={t("empty.description")}
                  action={{ label: t("empty.connectAllegro"), href: "/integrations" }}
                  secondaryAction={{ label: t("empty.addOrder"), href: "/orders/new" }}
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
        </>
      )}

      <Dialog open={showMergeDialog} onOpenChange={setShowMergeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("merge.title")}</DialogTitle>
            <DialogDescription>
              {t("merge.description", { count: selectedIds.size })}
            </DialogDescription>
          </DialogHeader>
          <div className="text-sm text-muted-foreground">
            <p className="font-medium mb-1">{t("merge.selectedOrders")}</p>
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
              {tc("cancel")}
            </Button>
            <Button
              onClick={async () => {
                try {
                  await mergeOrders.mutateAsync({
                    order_ids: Array.from(selectedIds),
                  });
                  toast.success(t("merge.success"));
                  setShowMergeDialog(false);
                  setSelectedIds(new Set());
                } catch (error) {
                  toast.error(getErrorMessage(error));
                }
              }}
              disabled={mergeOrders.isPending}
            >
              {mergeOrders.isPending ? t("merge.merging") : t("merge.title")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
