"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { SHIPMENT_TRANSITIONS, SHIPMENT_STATUSES } from "@/lib/constants";

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
  const [confirmStatus, setConfirmStatus] = useState<string | null>(null);

  const availableTransitions = SHIPMENT_TRANSITIONS[currentStatus] ?? [];

  if (availableTransitions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Brak dostepnych zmian statusu.
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

      <Dialog
        open={!!confirmStatus}
        onOpenChange={(open) => !open && setConfirmStatus(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Potwierdzenie zmiany statusu</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz zmienic status przesylki na{" "}
              <strong>
                {confirmStatus
                  ? SHIPMENT_STATUSES[confirmStatus]?.label ?? confirmStatus
                  : ""}
              </strong>
              ? Ta operacja moze byc nieodwracalna.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmStatus(null)}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirm}
              disabled={isLoading}
            >
              Potwierdz
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
