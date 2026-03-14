"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { StatusTransitionDialog } from "@/components/shared/status-transition-dialog";
import { SHIPMENT_TRANSITIONS, SHIPMENT_STATUSES } from "@/lib/constants";
import { useTranslations } from "next-intl";

interface ShipmentStatusActionsProps {
  currentStatus: string;
  onTransition: (status: string) => void;
  isLoading?: boolean;
}

const DESTRUCTIVE_STATUSES = ["failed"];

export function ShipmentStatusActions({
  currentStatus,
  onTransition,
  isLoading,
}: ShipmentStatusActionsProps) {
  const t = useTranslations("shipments");
  const [confirmStatus, setConfirmStatus] = useState<string | null>(null);

  const availableTransitions = SHIPMENT_TRANSITIONS[currentStatus] ?? [];

  if (availableTransitions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        {t("brakDostepnychZmianStatusu")}
      </p>
    );
  }

  const handleClick = (status: string) => {
    if (DESTRUCTIVE_STATUSES.includes(status)) {
      setConfirmStatus(status);
    } else {
      onTransition(status);
    }
  };

  const handleConfirm = () => {
    if (confirmStatus) {
      onTransition(confirmStatus);
      setConfirmStatus(null);
    }
  };

  return (
    <>
      <div className="flex flex-wrap gap-2">
        {availableTransitions.map((status) => {
          const statusInfo = SHIPMENT_STATUSES[status];
          const isDestructive = DESTRUCTIVE_STATUSES.includes(status);

          return (
            <Button
              key={status}
              variant={isDestructive ? "destructive" : "outline"}
              size="sm"
              onClick={() => handleClick(status)}
              disabled={isLoading}
            >
              {statusInfo?.label ?? status}
            </Button>
          );
        })}
      </div>

      <StatusTransitionDialog
        open={!!confirmStatus}
        onOpenChange={(open) => !open && setConfirmStatus(null)}
        title={t("statusChangeConfirmation")}
        description={
          <>
            {t("confirmShipmentStatusChange")}{" "}
            <strong>
              {confirmStatus
                ? SHIPMENT_STATUSES[confirmStatus]?.label ?? confirmStatus
                : ""}
            </strong>
            {t("taOperacjaMozeBycNieodwracalna")}
          </>
        }
        isDestructive
        isPending={isLoading}
        onConfirm={handleConfirm}
      />
    </>
  );
}
