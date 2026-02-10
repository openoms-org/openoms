"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ORDER_STATUSES, ORDER_TRANSITIONS } from "@/lib/constants";

interface OrderStatusActionsProps {
  currentStatus: string;
  onTransition: (newStatus: string, force?: boolean) => void;
  isLoading?: boolean;
}

const DESTRUCTIVE_STATUSES = ["cancelled", "refunded"];

function getButtonVariant(status: string): "default" | "destructive" | "outline" {
  if (DESTRUCTIVE_STATUSES.includes(status)) return "destructive";
  if (["delivered", "completed", "confirmed"].includes(status)) return "default";
  return "outline";
}

export function OrderStatusActions({
  currentStatus,
  onTransition,
  isLoading = false,
}: OrderStatusActionsProps) {
  const [confirmDialog, setConfirmDialog] = useState<{
    status: string;
    force: boolean;
  } | null>(null);

  const normalTransitions = ORDER_TRANSITIONS[currentStatus] || [];

  // All other statuses (excluding current and normal transitions) for force mode
  const allStatuses = Object.keys(ORDER_STATUSES);
  const forceTransitions = allStatuses.filter(
    (s) => s !== currentStatus && !normalTransitions.includes(s)
  );

  const handleClick = (newStatus: string, force: boolean) => {
    if (force || DESTRUCTIVE_STATUSES.includes(newStatus)) {
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
    ? ORDER_STATUSES[confirmDialog.status]?.label || confirmDialog.status
    : "";

  return (
    <>
      {normalTransitions.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {normalTransitions.map((status) => {
            const config = ORDER_STATUSES[status];
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
            Wymus zmiane (niezgodne z flow)
          </p>
          <div className="flex flex-wrap items-center gap-2">
            {forceTransitions.map((status) => {
              const config = ORDER_STATUSES[status];
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

      <Dialog open={!!confirmDialog} onOpenChange={() => setConfirmDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {confirmDialog?.force
                ? "Wymuszona zmiana statusu"
                : "Potwierdzenie zmiany statusu"}
            </DialogTitle>
            <DialogDescription>
              {confirmDialog?.force
                ? `Ta zmiana jest niezgodna z normalnym flow zamowienia. Czy na pewno chcesz wymusic zmiane statusu na "${confirmLabel}"?`
                : `Czy na pewno chcesz zmienic status zamowienia na "${confirmLabel}"? Ta operacja moze byc nieodwracalna.`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmDialog(null)}>
              Anuluj
            </Button>
            <Button
              variant={confirmDialog?.force ? "destructive" : "default"}
              onClick={handleConfirm}
              disabled={isLoading}
            >
              {confirmDialog?.force ? "Wymus zmiane" : "Potwierdz"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
