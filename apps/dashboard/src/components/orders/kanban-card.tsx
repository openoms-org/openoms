"use client";

import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useRouter } from "next/navigation";
import { GripVertical, AlertCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { ORDER_SOURCE_LABELS } from "@/lib/constants";
import { formatCurrency } from "@/lib/utils";
import { formatDistanceToNow } from "date-fns";
import { pl } from "date-fns/locale";
import type { Order } from "@/types/api";
import { cn } from "@/lib/utils";

interface KanbanCardProps {
  order: Order;
  isDragOverlay?: boolean;
}

export function KanbanCard({ order, isDragOverlay }: KanbanCardProps) {
  const router = useRouter();

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: order.id,
    data: { order },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const isUrgent = order.tags?.some(
    (tag) => tag.toLowerCase() === "pilne" || tag.toLowerCase() === "urgent"
  );

  const relativeDate = formatDistanceToNow(new Date(order.created_at), {
    addSuffix: true,
    locale: pl,
  });

  const sourceLabel = ORDER_SOURCE_LABELS[order.source] || order.source;

  const handleClick = (e: React.MouseEvent) => {
    // Don't navigate if we're dragging
    if (isDragging) return;
    // Don't navigate if clicking on the drag handle
    const target = e.target as HTMLElement;
    if (target.closest("[data-drag-handle]")) return;
    router.push(`/orders/${order.id}`);
  };

  return (
    <div
      ref={isDragOverlay ? undefined : setNodeRef}
      style={isDragOverlay ? undefined : style}
      className={cn(
        "group flex items-start gap-2 rounded-lg border bg-card p-3 shadow-sm transition-colors hover:border-primary/30 cursor-pointer",
        "dark:bg-card dark:border-border dark:hover:border-primary/40",
        isDragging && "opacity-30",
        isDragOverlay && "shadow-lg ring-2 ring-primary/20 rotate-2"
      )}
      onClick={handleClick}
    >
      {/* Drag handle */}
      <button
        data-drag-handle
        className="mt-0.5 shrink-0 cursor-grab touch-none text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 active:cursor-grabbing"
        {...(isDragOverlay ? {} : { ...attributes, ...listeners })}
      >
        <GripVertical className="h-4 w-4" />
      </button>

      {/* Card content */}
      <div className="min-w-0 flex-1 space-y-1.5">
        {/* Top row: order number + urgent indicator */}
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-bold text-foreground">
            #{order.id.substring(0, 8).toUpperCase()}
          </span>
          {isUrgent && (
            <AlertCircle className="h-4 w-4 shrink-0 text-red-500" />
          )}
        </div>

        {/* Customer name */}
        <p className="truncate text-sm text-muted-foreground">
          {order.customer_name}
        </p>

        {/* Bottom row: amount, source, date */}
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-sm font-medium text-foreground">
            {formatCurrency(order.total_amount, order.currency)}
          </span>
          <Badge
            variant="secondary"
            className="text-[10px] px-1.5 py-0"
          >
            {sourceLabel}
          </Badge>
        </div>

        <p className="text-xs text-muted-foreground">{relativeDate}</p>
      </div>
    </div>
  );
}
