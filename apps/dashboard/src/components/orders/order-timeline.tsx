"use client";

import { useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { pl } from "date-fns/locale";
import {
  Plus,
  ArrowRightLeft,
  Pencil,
  Trash2,
  Clock,
  ChevronDown,
} from "lucide-react";
import { useOrderAudit } from "@/hooks/use-orders";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ORDER_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { cn } from "@/lib/utils";
import type { AuditLogEntry } from "@/types/api";

const ACTION_LABELS: Record<string, string> = {
  "order.created": "Utworzono zamówienie",
  "order.updated": "Zaktualizowano zamówienie",
  "order.deleted": "Usunięto zamówienie",
  "order.status_changed": "Zmieniono status",
};

function getActionIcon(action: string) {
  switch (action) {
    case "order.created":
      return <Plus className="h-4 w-4" />;
    case "order.status_changed":
      return <ArrowRightLeft className="h-4 w-4" />;
    case "order.updated":
      return <Pencil className="h-4 w-4" />;
    case "order.deleted":
      return <Trash2 className="h-4 w-4" />;
    default:
      return <Clock className="h-4 w-4" />;
  }
}

function EntryChanges({ entry, orderStatuses }: { entry: AuditLogEntry; orderStatuses: Record<string, { label: string; color: string }> }) {
  if (!entry.changes || Object.keys(entry.changes).length === 0) {
    return null;
  }

  if (entry.action === "order.status_changed") {
    const from = entry.changes.from;
    const to = entry.changes.to;
    if (from && to) {
      return (
        <p className="text-xs text-muted-foreground mt-1">
          {orderStatuses[from]?.label || from} &rarr; {orderStatuses[to]?.label || to}
        </p>
      );
    }
  }

  return (
    <div className="text-xs text-muted-foreground mt-1">
      {Object.entries(entry.changes).map(([key, value]) => (
        <p key={key}>
          {key}: {value}
        </p>
      ))}
    </div>
  );
}

function TimelineEntry({ entry, orderStatuses }: { entry: AuditLogEntry; orderStatuses: Record<string, { label: string; color: string }> }) {
  const label = ACTION_LABELS[entry.action] || entry.action;
  const relativeTime = formatDistanceToNow(new Date(entry.created_at), {
    addSuffix: true,
    locale: pl,
  });

  return (
    <div className="relative flex gap-3 pb-6 last:pb-0">
      <div className="relative flex flex-col items-center">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border bg-background">
          {getActionIcon(entry.action)}
        </div>
        <div className="absolute top-8 bottom-0 w-px bg-border last:hidden" />
      </div>
      <div className="flex-1 pt-1">
        <p className="text-sm font-medium">{label}</p>
        {entry.user_name && (
          <p className="text-xs text-muted-foreground">{entry.user_name}</p>
        )}
        <EntryChanges entry={entry} orderStatuses={orderStatuses} />
        <p className="text-xs text-muted-foreground mt-1">{relativeTime}</p>
      </div>
    </div>
  );
}

interface OrderTimelineProps {
  orderId: string;
}

export function OrderTimeline({ orderId }: OrderTimelineProps) {
  const { data: entries, isLoading } = useOrderAudit(orderId);
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const hasEntries = !!(entries && entries.length > 0);
  const [open, setOpen] = useState(hasEntries);

  return (
    <Card>
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-6 py-4 text-left"
      >
        <div className="flex items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <h3 className="font-semibold">Historia zamówienia</h3>
          {hasEntries && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
              {entries.length}
            </span>
          )}
        </div>
        <ChevronDown
          className={cn(
            "h-4 w-4 text-muted-foreground transition-transform",
            open && "rotate-180"
          )}
        />
      </button>
      {open && (
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : !entries || entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <Clock className="h-8 w-8 text-muted-foreground/50 mb-2" />
              <p className="text-sm text-muted-foreground">Brak historii</p>
            </div>
          ) : (
            <div>
              {entries.map((entry) => (
                <TimelineEntry key={entry.id} entry={entry} orderStatuses={orderStatuses} />
              ))}
            </div>
          )}
        </CardContent>
      )}
    </Card>
  );
}
