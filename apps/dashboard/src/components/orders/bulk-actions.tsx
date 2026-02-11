"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useBulkTransitionStatus } from "@/hooks/use-orders";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { StatusTransitionDialog } from "@/components/shared/status-transition-dialog";
import { ORDER_TRANSITIONS, ORDER_STATUSES } from "@/lib/constants";
import { isDestructiveOrderStatus } from "@/lib/order-utils";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import type { Order } from "@/types/api";

interface BulkActionsProps {
  selectedOrders: Order[];
  onClearSelection: () => void;
}

function pluralOrders(count: number): string {
  if (count === 1) return "zamówienie";
  const lastTwo = count % 100;
  const lastOne = count % 10;
  if (lastTwo >= 10 && lastTwo <= 20) return "zamówień";
  if (lastOne >= 2 && lastOne <= 4) return "zamówienia";
  return "zamówień";
}

export function BulkActions({ selectedOrders, onClearSelection }: BulkActionsProps) {
  const [targetStatus, setTargetStatus] = useState<string>("");
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const bulkTransition = useBulkTransitionStatus();

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const orderTransitions = statusConfig?.transitions ?? ORDER_TRANSITIONS;

  const commonTransitions = selectedOrders.reduce<string[]>((acc, order, index) => {
    const transitions = orderTransitions[order.status] || [];
    if (index === 0) return [...transitions];
    return acc.filter((t) => transitions.includes(t));
  }, []);

  const selectedStatuses = new Set(selectedOrders.map((o) => o.status));
  const forceTransitions = Object.keys(orderStatuses).filter(
    (s) => !commonTransitions.includes(s) && !selectedStatuses.has(s)
  );

  const isForce = !commonTransitions.includes(targetStatus) && targetStatus !== "";

  const handleAction = () => {
    if (!targetStatus) return;
    if (isForce || isDestructiveOrderStatus(targetStatus)) {
      setShowConfirmDialog(true);
      return;
    }
    executeBulkTransition();
  };

  const executeBulkTransition = async () => {
    setShowConfirmDialog(false);
    try {
      const result = await bulkTransition.mutateAsync({
        order_ids: selectedOrders.map((o) => o.id),
        status: targetStatus,
        force: isForce,
      });
      if (result.failed === 0) {
        toast.success(`Zmieniono status ${result.succeeded} zamówień`);
      } else {
        toast.warning(
          `Zmieniono: ${result.succeeded}, błędy: ${result.failed}`
        );
      }
      setTargetStatus("");
      onClearSelection();
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Błąd podczas zmiany statusu"
      );
    }
  };

  return (
    <>
      <div className="flex items-center gap-3 rounded-lg border bg-muted/50 p-3">
        <span className="text-sm font-medium">
          Zaznaczono {selectedOrders.length} {pluralOrders(selectedOrders.length)}
        </span>

        <Select value={targetStatus} onValueChange={setTargetStatus}>
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="Wybierz status" />
          </SelectTrigger>
          <SelectContent>
            {commonTransitions.length > 0 && (
              <>
                {commonTransitions.map((status) => (
                  <SelectItem key={status} value={status}>
                    {orderStatuses[status]?.label || status}
                  </SelectItem>
                ))}
              </>
            )}
            {forceTransitions.length > 0 && (
              <>
                {commonTransitions.length > 0 && (
                  <SelectItem value="__separator" disabled>
                    ── Wymuś zmianę ──
                  </SelectItem>
                )}
                {forceTransitions.map((status) => (
                  <SelectItem key={status} value={status}>
                    {orderStatuses[status]?.label || status}
                  </SelectItem>
                ))}
              </>
            )}
          </SelectContent>
        </Select>

        <Button
          size="sm"
          variant={isForce ? "destructive" : "default"}
          onClick={handleAction}
          disabled={!targetStatus || bulkTransition.isPending}
        >
          {bulkTransition.isPending
            ? "Zmieniam..."
            : isForce
              ? "Wymuś zmianę"
              : "Zmień status"}
        </Button>

        <Button size="sm" variant="ghost" onClick={onClearSelection}>
          Odznacz
        </Button>
      </div>

      <StatusTransitionDialog
        open={showConfirmDialog}
        onOpenChange={setShowConfirmDialog}
        title={isForce ? "Wymuszona zmiana statusu" : "Potwierdzenie"}
        description={
          isForce
            ? `Ta zmiana jest niezgodna z normalnym flow. Czy na pewno chcesz wymusić zmianę statusu ${selectedOrders.length} zamówień na "${orderStatuses[targetStatus]?.label || targetStatus}"?`
            : `Czy na pewno chcesz zmienić status ${selectedOrders.length} zamówień na "${orderStatuses[targetStatus]?.label || targetStatus}"?`
        }
        isDestructive
        isPending={bulkTransition.isPending}
        onConfirm={executeBulkTransition}
        confirmLabel={bulkTransition.isPending ? "Zmieniam..." : "Potwierdź"}
      />
    </>
  );
}
