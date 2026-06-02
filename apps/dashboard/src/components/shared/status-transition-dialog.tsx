"use client";

import { useTranslations } from "next-intl";
import { ActionDialog } from "@/components/shared/action-dialog";
import type { ReactNode } from "react";

interface StatusTransitionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: ReactNode;
  isDestructive?: boolean;
  isPending?: boolean;
  onConfirm: () => void;
  confirmLabel?: string;
  cancelLabel?: string;
}

export function StatusTransitionDialog({
  open,
  onOpenChange,
  title,
  description,
  isDestructive = false,
  isPending = false,
  onConfirm,
  confirmLabel,
  cancelLabel,
}: StatusTransitionDialogProps) {
  const t = useTranslations("common");

  return (
    <ActionDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      confirmLabel={confirmLabel ?? t("confirm")}
      cancelLabel={cancelLabel ?? t("cancel")}
      variant={isDestructive ? "destructive" : "default"}
      isPending={isPending}
      onConfirm={onConfirm}
    />
  );
}
