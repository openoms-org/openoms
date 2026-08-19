"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { downloadShipmentLabel } from "@/hooks/use-shipments";
import { getErrorMessage } from "@/lib/api-client";

type ShipmentLabelDownloadProps = {
  shipmentId: string;
  children: React.ReactNode;
} & Omit<React.ComponentProps<typeof Button>, "onClick" | "type">;

export function ShipmentLabelDownload({
  shipmentId,
  children,
  disabled,
  ...buttonProps
}: ShipmentLabelDownloadProps) {
  const [pending, setPending] = useState(false);

  const handleClick = async () => {
    if (pending) return;
    setPending(true);
    try {
      await downloadShipmentLabel(shipmentId);
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setPending(false);
    }
  };

  return (
    <Button
      type="button"
      {...buttonProps}
      disabled={disabled || pending}
      onClick={handleClick}
    >
      {children}
    </Button>
  );
}
