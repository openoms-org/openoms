"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Loader2, Download, Package, CheckCircle2 } from "lucide-react";
import { ActionDialog } from "@/components/shared/action-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  useAllegroDeliveryServices,
  useAllegroDeliveryProposals,
  useCreateAllegroShipment,
  downloadAllegroLabel,
} from "@/hooks/use-allegro";
import type { Order } from "@/types/api";
import { getErrorMessage } from "@/lib/api-client";
import { useTranslations } from "next-intl";

interface AllegroShipmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: Order;
}

type Step = "select-service" | "package-details" | "result";

export function AllegroShipmentDialog({
  open,
  onOpenChange,
  order,
}: AllegroShipmentDialogProps) {
  const t = useTranslations("integrations");
  const [step, setStep] = useState<Step>("select-service");
  const [selectedServiceId, setSelectedServiceId] = useState("");
  const [weight, setWeight] = useState("1");
  const [length, setLength] = useState("30");
  const [width, setWidth] = useState("20");
  const [height, setHeight] = useState("15");
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(
    null
  );
  const [isDownloading, setIsDownloading] = useState(false);

  const {
    data: deliveryData,
    isLoading: isLoadingServices,
    isError: isServicesError,
    error: servicesError,
  } = useAllegroDeliveryServices();
  const {
    data: proposals,
    isLoading: isLoadingProposals,
    isError: isProposalsError,
    error: proposalsError,
  } = useAllegroDeliveryProposals(order.external_id);
  const createShipment = useCreateAllegroShipment();

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setStep("select-service");
      setSelectedServiceId("");
      setWeight("1");
      setLength("30");
      setWidth("20");
      setHeight("15");
      setCreatedShipmentId(null);
    }
  }, [open]);

  const deliveryServices = deliveryData?.delivery_services ?? [];
  const suggested = proposals?.suggestedInput;
  const checkoutMethodId =
    typeof order.metadata?.delivery_method_id === "string"
      ? order.metadata.delivery_method_id
      : "";

  useEffect(() => {
    if (!open) return;
    const nextId = suggested?.deliveryMethodId || checkoutMethodId;
    if (nextId) {
      setSelectedServiceId(nextId);
    }
  }, [open, suggested?.deliveryMethodId, checkoutMethodId]);

  const handleCreateShipment = async () => {
    if (!selectedServiceId) {
      toast.error(t("selectDeliveryService"));
      return;
    }
    if (!suggested?.sender?.street || !(suggested.sender.postalCode || suggested.sender.zipCode)) {
      toast.error(t("selectDeliveryService"));
      return;
    }

    try {
      const result = await createShipment.mutateAsync({
        commandId: crypto.randomUUID(),
        order_id: order.id,
        input: {
          ...suggested,
          deliveryMethodId: selectedServiceId,
          credentialsId: suggested.credentialsId,
          packages: [
            {
              type: suggested.packages?.[0]?.type || "PACKAGE",
              weight: {
                value: parseFloat(weight) || 1,
                unit: "KILOGRAMS",
              },
              length: {
                value: parseFloat(length) || 30,
                unit: "CENTIMETER",
              },
              width: {
                value: parseFloat(width) || 20,
                unit: "CENTIMETER",
              },
              height: {
                value: parseFloat(height) || 15,
                unit: "CENTIMETER",
              },
            },
          ],
        },
      });

      setCreatedShipmentId(result.shipmentId);
      setStep("result");
      toast.success(t("allegroShipmentCreated"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDownloadLabel = async () => {
    if (!createdShipmentId) return;
    setIsDownloading(true);
    try {
      await downloadAllegroLabel(createdShipmentId);
      toast.success(t("labelDownloaded"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setIsDownloading(false);
    }
  };

  const title = (
    <span className="flex items-center gap-2">
      <Package className="h-5 w-5" />
      {t("allegroSendTitle")}
    </span>
  );

  const confirmLabel = (() => {
    if (step === "select-service") return t("next");
    if (step === "result") return t("close");
    if (createShipment.isPending) {
      return (
        <>
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          {t("creating")}
        </>
      );
    }
    return t("createShipment");
  })();

  const handleDialogConfirm = () => {
    if (step === "select-service") {
      setStep("package-details");
      return;
    }
    if (step === "result") {
      onOpenChange(false);
      return;
    }
    void handleCreateShipment();
  };

  return (
    <ActionDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={t("allegroCreateShipmentDesc")}
      confirmLabel={confirmLabel}
      cancelLabel={step === "package-details" ? t("back") : t("cancel")}
      confirmDisabled={
        (step === "select-service" &&
          (!selectedServiceId ||
            !suggested?.sender?.street ||
            !(suggested.sender.postalCode || suggested.sender.zipCode))) ||
        (step === "package-details" && createShipment.isPending)
      }
      hideCancel={step === "result"}
      onCancel={step === "package-details" ? () => setStep("select-service") : undefined}
      onConfirm={handleDialogConfirm}
      contentClassName="max-w-lg"
    >
      {step === "select-service" && (
        <div>
          <Label>{t("deliveryService")}</Label>
          {isLoadingProposals || isLoadingServices ? (
            <div className="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("loadingDeliveryServices")}
            </div>
          ) : isProposalsError ? (
            <p className="mt-2 text-sm text-destructive">
              {getErrorMessage(proposalsError)}
            </p>
          ) : !suggested?.sender?.street ? (
            <p className="mt-2 text-sm text-muted-foreground">
              {t("noDeliveryServices")}
            </p>
          ) : deliveryServices.length === 0 ? (
            <p className="mt-2 text-sm text-muted-foreground">
              {order.delivery_method || suggested.deliveryMethodId}
            </p>
          ) : (
            <Select
              value={selectedServiceId}
              onValueChange={setSelectedServiceId}
            >
              <SelectTrigger className="mt-1">
                <SelectValue placeholder={t("selectDeliveryServicePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {deliveryServices.map((svc) => (
                  <SelectItem key={svc.id} value={svc.id}>
                    {svc.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          {isServicesError && !isProposalsError && (
            <p className="mt-2 text-sm text-muted-foreground">
              {getErrorMessage(servicesError)}
            </p>
          )}
        </div>
      )}

      {step === "package-details" && (
        <>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>{t("weightKg")}</Label>
              <Input
                type="number"
                step="0.1"
                min="0.1"
                value={weight}
                onChange={(e) => setWeight(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t("addPackage.length")}</Label>
              <Input
                type="number"
                step="1"
                min="1"
                value={length}
                onChange={(e) => setLength(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t("addPackage.width")}</Label>
              <Input
                type="number"
                step="1"
                min="1"
                value={width}
                onChange={(e) => setWidth(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t("addPackage.height")}</Label>
              <Input
                type="number"
                step="1"
                min="1"
                value={height}
                onChange={(e) => setHeight(e.target.value)}
                className="mt-1"
              />
            </div>
          </div>

          <div className="rounded-md border bg-muted/50 p-3 text-sm">
            <p className="font-medium">{t("receiverData")}</p>
            <p className="text-muted-foreground">
              {order.shipping_address?.name || order.customer_name}
            </p>
            <p className="text-muted-foreground">
              {order.shipping_address?.street}
            </p>
            <p className="text-muted-foreground">
              {order.shipping_address?.postal_code} {order.shipping_address?.city}
            </p>
          </div>
        </>
      )}

      {step === "result" && createdShipmentId && (
        <>
          <div className="flex flex-col items-center gap-3 py-4">
            <CheckCircle2 className="h-12 w-12 text-green-500" />
            <p className="text-lg font-medium">{t("triggers.shipment.created")}</p>
            <p className="text-center text-sm text-muted-foreground">
              {t("shipmentIdLabel", { id: createdShipmentId })}
            </p>
          </div>

          <Button
            className="w-full"
            onClick={handleDownloadLabel}
            disabled={isDownloading}
          >
            {isDownloading ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Download className="mr-2 h-4 w-4" />
            )}
            {t("downloadLabel")}
          </Button>
        </>
      )}
    </ActionDialog>
  );
}
