"use client";

import { useState, useCallback, useMemo, useEffect } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  closestCorners,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useDroppable } from "@dnd-kit/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient, getErrorMessage } from "@/lib/api-client";
import { useOrderStatuses, statusesToMap, COLOR_PRESETS } from "@/hooks/use-order-statuses";
import { ORDER_STATUSES } from "@/lib/constants";
import { KanbanCard } from "./kanban-card";
import { Skeleton } from "@/components/ui/skeleton";
import { Package } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import type { Order, ListResponse } from "@/types/api";

interface KanbanBoardProps {
  filters: {
    status?: string;
    source?: string;
    search?: string;
    payment_status?: string;
    tag?: string;
  };
}

const COLUMN_PAGE_SIZE = 20;

/** Hook: fetches orders for a single status column */
function useColumnOrders(
  status: string,
  filters: KanbanBoardProps["filters"],
  enabled: boolean
) {
  const params = new URLSearchParams();
  params.set("status", status);
  params.set("limit", String(COLUMN_PAGE_SIZE));
  params.set("sort_by", "created_at");
  params.set("sort_order", "desc");
  if (filters.source) params.set("source", filters.source);
  if (filters.search) params.set("search", filters.search);
  if (filters.payment_status) params.set("payment_status", filters.payment_status);
  if (filters.tag) params.set("tag", filters.tag);

  return useQuery({
    queryKey: ["orders-kanban", status, filters],
    queryFn: () =>
      apiClient<ListResponse<Order>>(`/v1/orders?${params.toString()}`),
    enabled,
  });
}

/** A single Kanban column */
function KanbanColumn({
  statusDef,
  statusKey,
  colorClass,
  filters,
  activeOrder,
  overColumnId,
}: {
  statusDef: { label: string; color: string };
  statusKey: string;
  colorClass: string;
  filters: KanbanBoardProps["filters"];
  activeOrder: Order | null;
  overColumnId: string | null;
}) {
  const [loadedPages, setLoadedPages] = useState(1);
  const { data, isLoading } = useColumnOrders(statusKey, filters, true);
  const queryClient = useQueryClient();

  const orders = data?.items || [];
  const total = data?.total || 0;
  const hasMore = orders.length < total && loadedPages * COLUMN_PAGE_SIZE < total;

  const { setNodeRef, isOver } = useDroppable({
    id: `column-${statusKey}`,
    data: { status: statusKey },
  });

  const isHighlighted = isOver || overColumnId === statusKey;

  const handleLoadMore = useCallback(async () => {
    const nextPage = loadedPages + 1;
    const nextOffset = loadedPages * COLUMN_PAGE_SIZE;

    const params = new URLSearchParams();
    params.set("status", statusKey);
    params.set("limit", String(COLUMN_PAGE_SIZE));
    params.set("offset", String(nextOffset));
    params.set("sort_by", "created_at");
    params.set("sort_order", "desc");
    if (filters.source) params.set("source", filters.source);
    if (filters.search) params.set("search", filters.search);
    if (filters.payment_status) params.set("payment_status", filters.payment_status);
    if (filters.tag) params.set("tag", filters.tag);

    try {
      const moreData = await apiClient<ListResponse<Order>>(
        `/v1/orders?${params.toString()}`
      );

      // Merge into existing query cache
      queryClient.setQueryData<ListResponse<Order>>(
        ["orders-kanban", statusKey, filters],
        (prev) => {
          if (!prev) return moreData;
          return {
            ...prev,
            items: [...prev.items, ...moreData.items],
          };
        }
      );
      setLoadedPages(nextPage);
    } catch {
      toast.error("Nie udało się załadować więcej zamówień");
    }
  }, [loadedPages, statusKey, filters, queryClient]);

  // Reset pagination when filters change
  useEffect(() => {
    setLoadedPages(1);
  }, [filters]);

  return (
    <div
      className={cn(
        "flex h-full w-[300px] min-w-[280px] shrink-0 flex-col rounded-lg border bg-muted/30",
        "dark:bg-muted/10 dark:border-border",
        isHighlighted && "ring-2 ring-primary/40 bg-primary/5 dark:bg-primary/10"
      )}
    >
      {/* Column header */}
      <div className="flex items-center gap-2 rounded-t-lg border-b px-3 py-2.5">
        <span
          className={cn(
            "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
            colorClass
          )}
        >
          {statusDef.label}
        </span>
        <span className="text-xs text-muted-foreground">
          {isLoading ? "..." : total}
        </span>
      </div>

      {/* Column body */}
      <div
        ref={setNodeRef}
        className="flex-1 overflow-y-auto p-2 space-y-2"
        style={{ maxHeight: "calc(100vh - 300px)" }}
      >
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-24 w-full rounded-lg" />
            ))}
          </div>
        ) : orders.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Package className="h-8 w-8 text-muted-foreground/40 mb-2" />
            <p className="text-xs text-muted-foreground">Brak zamówień</p>
          </div>
        ) : (
          <SortableContext
            items={orders.map((o) => o.id)}
            strategy={verticalListSortingStrategy}
          >
            {orders.map((order) => (
              <KanbanCard key={order.id} order={order} />
            ))}
          </SortableContext>
        )}

        {hasMore && !isLoading && (
          <button
            onClick={handleLoadMore}
            className="w-full rounded-md border border-dashed py-2 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
          >
            Pokaż więcej
          </button>
        )}
      </div>
    </div>
  );
}

export function KanbanBoard({ filters }: KanbanBoardProps) {
  const { data: statusConfig, isLoading: statusLoading } = useOrderStatuses();
  const queryClient = useQueryClient();

  const [activeOrder, setActiveOrder] = useState<Order | null>(null);
  const [overColumnId, setOverColumnId] = useState<string | null>(null);

  // Get the ordered list of statuses from config, or fall back to defaults
  const statusEntries = useMemo(() => {
    if (statusConfig) {
      return statusConfig.statuses
        .sort((a, b) => a.position - b.position)
        .map((s) => ({
          key: s.key,
          label: s.label,
          color: s.color,
          colorClass: COLOR_PRESETS[s.color] || COLOR_PRESETS.gray,
        }));
    }
    // Fallback: use default ORDER_STATUSES
    return Object.entries(ORDER_STATUSES).map(([key, val]) => ({
      key,
      label: val.label,
      color: "gray",
      colorClass: val.color,
    }));
  }, [statusConfig]);

  // If a status filter is active, only show that column
  const visibleStatuses = useMemo(() => {
    if (filters.status) {
      return statusEntries.filter((s) => s.key === filters.status);
    }
    return statusEntries;
  }, [statusEntries, filters.status]);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    const order = event.active.data.current?.order as Order | undefined;
    if (order) {
      setActiveOrder(order);
    }
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    const overId = event.over?.id;
    if (!overId) {
      setOverColumnId(null);
      return;
    }

    // Check if hovering over a column directly
    const overIdStr = String(overId);
    if (overIdStr.startsWith("column-")) {
      setOverColumnId(overIdStr.replace("column-", ""));
      return;
    }

    // If hovering over a card, find the column it belongs to
    const overData = event.over?.data.current;
    if (overData?.order) {
      setOverColumnId((overData.order as Order).status);
    }
  }, []);

  const handleDragEnd = useCallback(
    async (event: DragEndEvent) => {
      setActiveOrder(null);
      setOverColumnId(null);

      const { active, over } = event;
      if (!over) return;

      const draggedOrder = active.data.current?.order as Order | undefined;
      if (!draggedOrder) return;

      // Determine target status
      let targetStatus: string | null = null;
      const overIdStr = String(over.id);

      if (overIdStr.startsWith("column-")) {
        targetStatus = overIdStr.replace("column-", "");
      } else {
        // Dropped onto a card — get its status
        const overOrder = over.data.current?.order as Order | undefined;
        if (overOrder) {
          targetStatus = overOrder.status;
        }
      }

      if (!targetStatus || targetStatus === draggedOrder.status) return;

      const sourceStatus = draggedOrder.status;
      const orderId = draggedOrder.id;

      // Optimistic update: move order to new column
      queryClient.setQueryData<ListResponse<Order>>(
        ["orders-kanban", sourceStatus, filters],
        (prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            items: prev.items.filter((o) => o.id !== orderId),
            total: prev.total - 1,
          };
        }
      );

      const updatedOrder = { ...draggedOrder, status: targetStatus };
      queryClient.setQueryData<ListResponse<Order>>(
        ["orders-kanban", targetStatus, filters],
        (prev) => {
          if (!prev) {
            return { items: [updatedOrder], total: 1, limit: COLUMN_PAGE_SIZE, offset: 0 };
          }
          return {
            ...prev,
            items: [updatedOrder, ...prev.items],
            total: prev.total + 1,
          };
        }
      );

      try {
        await apiClient<Order>(`/v1/orders/${orderId}/status`, {
          method: "POST",
          body: JSON.stringify({ status: targetStatus }),
        });

        const statusLabel =
          statusEntries.find((s) => s.key === targetStatus)?.label || targetStatus;
        toast.success(`Zmieniono status zamówienia na: ${statusLabel}`);

        // Invalidate to get fresh data from server
        queryClient.invalidateQueries({ queryKey: ["orders-kanban", sourceStatus, filters] });
        queryClient.invalidateQueries({ queryKey: ["orders-kanban", targetStatus, filters] });
        // Also invalidate the table view queries
        queryClient.invalidateQueries({ queryKey: ["orders"] });
      } catch (error) {
        // Revert optimistic update
        queryClient.setQueryData<ListResponse<Order>>(
          ["orders-kanban", targetStatus, filters],
          (prev) => {
            if (!prev) return prev;
            return {
              ...prev,
              items: prev.items.filter((o) => o.id !== orderId),
              total: prev.total - 1,
            };
          }
        );

        queryClient.setQueryData<ListResponse<Order>>(
          ["orders-kanban", sourceStatus, filters],
          (prev) => {
            if (!prev) {
              return { items: [draggedOrder], total: 1, limit: COLUMN_PAGE_SIZE, offset: 0 };
            }
            return {
              ...prev,
              items: [draggedOrder, ...prev.items],
              total: prev.total + 1,
            };
          }
        );

        toast.error(getErrorMessage(error));
      }
    },
    [filters, queryClient, statusEntries]
  );

  const handleDragCancel = useCallback(() => {
    setActiveOrder(null);
    setOverColumnId(null);
  }, []);

  if (statusLoading) {
    return (
      <div className="flex gap-4 overflow-x-auto pb-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="flex h-[400px] w-[300px] min-w-[280px] shrink-0 flex-col rounded-lg border bg-muted/30"
          >
            <div className="border-b px-3 py-2.5">
              <Skeleton className="h-5 w-24" />
            </div>
            <div className="flex-1 p-2 space-y-2">
              <Skeleton className="h-24 w-full rounded-lg" />
              <Skeleton className="h-24 w-full rounded-lg" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={handleDragCancel}
    >
      <div className="flex gap-4 overflow-x-auto pb-4">
        {visibleStatuses.map((status) => (
          <KanbanColumn
            key={status.key}
            statusKey={status.key}
            statusDef={{ label: status.label, color: status.colorClass }}
            colorClass={status.colorClass}
            filters={filters}
            activeOrder={activeOrder}
            overColumnId={overColumnId}
          />
        ))}
      </div>

      <DragOverlay>
        {activeOrder ? (
          <div className="w-[270px]">
            <KanbanCard order={activeOrder} isDragOverlay />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
