"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
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

export function BulkActions({ selectedOrders, onClearSelection }: BulkActionsProps) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");
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
        toast.success(t("bulk.statusChanged", { count: result.succeeded }));
      } else {
        toast.warning(
          t("bulk.statusChangedPartial", { succeeded: result.succeeded, failed: result.failed })
        );
      }
      setTargetStatus("");
      onClearSelection();
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t("bulk.statusChangeError")
      );
    }
  };

  return (
    <>
      <div className="flex items-center gap-3 rounded-lg border bg-muted/50 p-3">
        <span className="text-sm font-medium">
          {t("bulk.selected", { count: selectedOrders.length })}
        </span>

        <Select value={targetStatus} onValueChange={setTargetStatus}>
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder={t("bulk.selectStatus")} />
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
                    {t("bulk.forceSeparator")}
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
            ? t("bulk.changing")
            : isForce
              ? t("bulk.forceChangeButton")
              : t("bulk.changeStatus")}
        </Button>

        <Button size="sm" variant="ghost" onClick={onClearSelection}>
          {t("bulk.deselect")}
        </Button>
      </div>

      <StatusTransitionDialog
        open={showConfirmDialog}
        onOpenChange={setShowConfirmDialog}
        title={isForce ? t("bulk.confirmForceTitle") : t("bulk.confirmTitle")}
        description={
          isForce
            ? t("bulk.confirmForceDescription", { count: selectedOrders.length, status: orderStatuses[targetStatus]?.label || targetStatus })
            : t("bulk.confirmDescription", { count: selectedOrders.length, status: orderStatuses[targetStatus]?.label || targetStatus })
        }
        isDestructive
        isPending={bulkTransition.isPending}
        onConfirm={executeBulkTransition}
        confirmLabel={bulkTransition.isPending ? t("bulk.changing") : t("bulk.confirmLabel")}
      />
    </>
  );
}
