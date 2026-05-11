"use client";

import type { KeyboardEvent, ReactNode } from "react";
import { useTranslations } from "next-intl";
import { AlertCircle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
type DialogSize = NonNullable<React.ComponentProps<typeof DialogContent>["size"]>;

interface ActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  confirmLabel?: ReactNode;
  cancelLabel?: ReactNode;
  variant?: "default" | "destructive";
  isLoading?: boolean;
  confirmDisabled?: boolean;
  onConfirm: () => void;
  onCancel?: () => void;
  error?: ReactNode;
  children?: ReactNode;
  contentClassName?: string;
  size?: DialogSize;
  hideCancel?: boolean;
  hideConfirm?: boolean;
}

function isFormControl(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  return Boolean(target.closest("button,input,textarea,select,[contenteditable=true]"));
}

export function ActionDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  variant = "default",
  isLoading = false,
  confirmDisabled = false,
  onConfirm,
  onCancel,
  error,
  children,
  contentClassName,
  size = "md",
  hideCancel = false,
  hideConfirm = false,
}: ActionDialogProps) {
  const t = useTranslations("common");
  const isConfirmDisabled = isLoading || confirmDisabled;

  const handleCancel = () => {
    if (isLoading) return;
    if (onCancel) {
      onCancel();
      return;
    }
    onOpenChange(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Enter" || isConfirmDisabled || hideConfirm) return;
    if (isFormControl(event.target)) return;

    event.preventDefault();
    onConfirm();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={contentClassName}
        size={size}
        onKeyDown={handleKeyDown}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>

        {children && <div className="space-y-4 py-2">{children}</div>}

        {error && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <div>{error}</div>
          </div>
        )}

        {(!hideCancel || !hideConfirm) && (
          <DialogFooter>
            {!hideCancel && (
              <Button
                type="button"
                variant="outline"
                onClick={handleCancel}
                disabled={isLoading}
              >
                {cancelLabel ?? t("cancel")}
              </Button>
            )}
            {!hideConfirm && (
              <Button
                type="button"
                variant={variant}
                onClick={onConfirm}
                disabled={isConfirmDisabled}
              >
                {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
                {isLoading ? t("processing") : (confirmLabel ?? t("confirm"))}
              </Button>
            )}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

export type { ActionDialogProps };
