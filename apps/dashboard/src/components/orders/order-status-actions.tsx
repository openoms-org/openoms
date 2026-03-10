"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { StatusTransitionDialog } from "@/components/shared/status-transition-dialog";
import { ORDER_STATUSES, ORDER_TRANSITIONS } from "@/lib/constants";
import { isDestructiveOrderStatus } from "@/lib/order-utils";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";

interface OrderStatusActionsProps {
  currentStatus: string;
  onTransition: (newStatus: string, force?: boolean) => void;
  isLoading?: boolean;
}

function getButtonVariant(status: string): "default" | "destructive" | "outline" {
  if (isDestructiveOrderStatus(status)) return "destructive";
  if (["delivered", "completed", "confirmed"].includes(status)) return "default";
  return "outline";
}

export function OrderStatusActions({
  currentStatus,
  onTransition,
  isLoading = false,
}: OrderStatusActionsProps) {
  const t = useTranslations("orders");
  const [confirmDialog, setConfirmDialog] = useState<{
    status: string;
    force: boolean;
  } | null>(null);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const orderTransitions = statusConfig?.transitions ?? ORDER_TRANSITIONS;

  const normalTransitions = orderTransitions[currentStatus] || [];

  const allStatuses = Object.keys(orderStatuses);
  const forceTransitions = allStatuses.filter(
    (s) => s !== currentStatus && !normalTransitions.includes(s)
  );

  const handleClick = (newStatus: string, force: boolean) => {
    if (force || isDestructiveOrderStatus(newStatus)) {
      setConfirmDialog({ status: newStatus, force });
    } else {
      onTransition(newStatus, false);
    }
  };

  const handleConfirm = () => {
    if (confirmDialog) {
      onTransition(confirmDialog.status, confirmDialog.force);
      setConfirmDialog(null);
    }
  };

  const confirmLabel = confirmDialog
    ? orderStatuses[confirmDialog.status]?.label || confirmDialog.status
    : "";

  return (
    <>
      {normalTransitions.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {normalTransitions.map((status) => {
            const config = orderStatuses[status];
            return (
              <Button
                key={status}
                variant={getButtonVariant(status)}
                size="sm"
                onClick={() => handleClick(status, false)}
                disabled={isLoading}
              >
                {config?.label || status}
              </Button>
            );
          })}
        </div>
      )}

      {forceTransitions.length > 0 && (
        <>
          {normalTransitions.length > 0 && <Separator className="my-3" />}
          <p className="text-xs text-muted-foreground mb-2">
            {t("statusActions.forceChange")}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            {forceTransitions.map((status) => {
              const config = orderStatuses[status];
              return (
                <Button
                  key={status}
                  variant="ghost"
                  size="sm"
                  className="text-muted-foreground"
                  onClick={() => handleClick(status, true)}
                  disabled={isLoading}
                >
                  {config?.label || status}
                </Button>
              );
            })}
          </div>
        </>
      )}

      <StatusTransitionDialog
        open={!!confirmDialog}
        onOpenChange={() => setConfirmDialog(null)}
        title={
          confirmDialog?.force
            ? t("statusActions.forceTitle")
            : t("statusActions.confirmTitle")
        }
        description={
          confirmDialog?.force
            ? t("statusActions.forceDescription", { status: confirmLabel })
            : t("statusActions.confirmDescription", { status: confirmLabel })
        }
        isDestructive={confirmDialog?.force ?? false}
        isPending={isLoading}
        onConfirm={handleConfirm}
        confirmLabel={confirmDialog?.force ? t("statusActions.forceButton") : t("statusActions.confirmButton")}
      />
    </>
  );
}
