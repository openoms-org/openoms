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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ORDER_TRANSITIONS, ORDER_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import type { Order } from "@/types/api";

interface BulkActionsProps {
  selectedOrders: Order[];
  onClearSelection: () => void;
}

const DESTRUCTIVE_STATUSES = ["cancelled", "refunded"];

export function BulkActions({ selectedOrders, onClearSelection }: BulkActionsProps) {
  const [targetStatus, setTargetStatus] = useState<string>("");
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const bulkTransition = useBulkTransitionStatus();

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const orderTransitions = statusConfig?.transitions ?? ORDER_TRANSITIONS;

  // Compute common valid transitions across all selected orders
  const commonTransitions = selectedOrders.reduce<string[]>((acc, order, index) => {
    const transitions = orderTransitions[order.status] || [];
    if (index === 0) return [...transitions];
    return acc.filter((t) => transitions.includes(t));
  }, []);

  // All other statuses for force mode (excluding statuses that are already in common transitions)
  const selectedStatuses = new Set(selectedOrders.map((o) => o.status));
  const forceTransitions = Object.keys(orderStatuses).filter(
    (s) => !commonTransitions.includes(s) && !selectedStatuses.has(s)
  );

  const isForce = !commonTransitions.includes(targetStatus) && targetStatus !== "";

  const handleAction = () => {
    if (!targetStatus) return;
    if (isForce || DESTRUCTIVE_STATUSES.includes(targetStatus)) {
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
        toast.success(`Zmieniono status ${result.succeeded} zamowien`);
      } else {
        toast.warning(
          `Zmieniono: ${result.succeeded}, bledy: ${result.failed}`
        );
      }
      setTargetStatus("");
      onClearSelection();
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas zmiany statusu"
      );
    }
  };

  return (
    <>
      <div className="flex items-center gap-3 rounded-lg border bg-muted/50 p-3">
        <span className="text-sm font-medium">
          Zaznaczono {selectedOrders.length} zamowien
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
                    ── Wymus zmiane ──
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
              ? "Wymus zmiane"
              : "Zmien status"}
        </Button>

        <Button size="sm" variant="ghost" onClick={onClearSelection}>
          Odznacz
        </Button>
      </div>

      <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {isForce ? "Wymuszona zmiana statusu" : "Potwierdzenie"}
            </DialogTitle>
            <DialogDescription>
              {isForce
                ? `Ta zmiana jest niezgodna z normalnym flow. Czy na pewno chcesz wymusic zmiane statusu ${selectedOrders.length} zamowien na "${orderStatuses[targetStatus]?.label || targetStatus}"?`
                : `Czy na pewno chcesz zmienic status ${selectedOrders.length} zamowien na "${orderStatuses[targetStatus]?.label || targetStatus}"?`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowConfirmDialog(false)}>
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={executeBulkTransition}
              disabled={bulkTransition.isPending}
            >
              {bulkTransition.isPending ? "Zmieniam..." : "Potwierdz"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
