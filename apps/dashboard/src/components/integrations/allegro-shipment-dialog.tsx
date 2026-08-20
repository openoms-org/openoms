"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Loader2, Download, Package, CheckCircle2, ExternalLink } from "lucide-react";
import { ActionDialog } from "@/components/shared/action-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  useAllegroDeliveryServices,
  useAllegroDeliveryProposals,
  useCreateAllegroShipment,
  useImportExistingWzAShipment,
  downloadAllegroLabel,
  useAllegroAccount,
} from "@/hooks/use-allegro";
import { downloadShipmentLabel } from "@/hooks/use-shipments";
import { sanitizeUrl } from "@/lib/utils";
import type { Order } from "@/types/api";
import { getErrorMessage } from "@/lib/api-client";
import { useTranslations } from "next-intl";
import {
  checkoutMethodLabel,
  resolveWzACreateDeliveryMethod,
} from "@/components/integrations/allegro-wza-method";
import { allegroSalesCenterCreateShipmentURL } from "@/components/integrations/allegro-sales-center";
import {
  type ImportedWzAShipment,
  wzaImportDialogError,
  wzaLabelDownloadTarget,
} from "@/components/integrations/allegro-wza-import";

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
  const [weight, setWeight] = useState("1");
  const [length, setLength] = useState("30");
  const [width, setWidth] = useState("20");
  const [height, setHeight] = useState("15");
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(
    null
  );
  const [importedShipment, setImportedShipment] = useState<ImportedWzAShipment | null>(
    null
  );
  const [importError, setImportError] = useState<string | null>(null);
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
  const importExisting = useImportExistingWzAShipment(order.id);
  const accountQuery = useAllegroAccount({ enabled: open });

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setStep("select-service");
      setWeight("1");
      setLength("30");
      setWidth("20");
      setHeight("15");
      setCreatedShipmentId(null);
      setImportedShipment(null);
      setImportError(null);
    }
  }, [open]);

  const deliveryServices = deliveryData?.delivery_services ?? [];
  const suggested = proposals?.suggestedInput;
  const checkoutMethodId =
    typeof order.metadata?.delivery_method_id === "string"
      ? order.metadata.delivery_method_id
      : "";
  const checkoutMethodName =
    typeof order.metadata?.delivery_method_name === "string"
      ? order.metadata.delivery_method_name
      : order.delivery_method || "";
  const methodDecision = resolveWzACreateDeliveryMethod({
    proposedDeliveryMethodId: suggested?.deliveryMethodId,
    checkoutMethodId,
    checkoutMethodName,
  });
  const proposedMethodId = methodDecision.ok ? methodDecision.deliveryMethodId : "";
  const salesCenterUrl = allegroSalesCenterCreateShipmentURL({
    checkoutFormId: order.external_id,
    sellerId: accountQuery.data?.user.id,
    sandbox: accountQuery.data?.sandbox === true,
  });

  const handleCreateShipment = async () => {
    if (!methodDecision.ok) {
      toast.error(t("noWzAMethodForCheckout", { method: checkoutMethodLabel(methodDecision) || t("unknownCheckoutMethod") }));
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
          deliveryMethodId: methodDecision.deliveryMethodId,
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

  const handleImportExisting = async () => {
    setImportError(null);
    try {
      const result = await importExisting.mutateAsync();
      const failure = wzaImportDialogError({ shipments: result.shipments ?? [] });
      if (failure) {
        const message =
          failure.kind === "empty" ? t("noExistingWzAShipment") : failure.message;
        setImportError(message);
        toast.error(message);
        return;
      }
      const first = result.shipments[0];
      if (!first) {
        const message = t("noExistingWzAShipment");
        setImportError(message);
        toast.error(message);
        return;
      }
      setImportedShipment(first);
      setCreatedShipmentId(first.allegro_shipment_id || first.id);
      setStep("result");
      toast.success(t("wzaImported"));
    } catch (error) {
      const failure = wzaImportDialogError({ error });
      const message =
        failure?.kind === "empty"
          ? t("noExistingWzAShipment")
          : failure?.kind === "request"
            ? failure.message
            : getErrorMessage(error);
      setImportError(message);
      toast.error(message);
    }
  };

  const handleDownloadLabel = async () => {
    const target = importedShipment
      ? wzaLabelDownloadTarget(importedShipment)
      : createdShipmentId
        ? { kind: "allegro" as const, shipmentId: createdShipmentId }
        : { kind: "none" as const };
    if (target.kind === "none") return;
    setIsDownloading(true);
    try {
      if (target.kind === "oms") {
        await downloadShipmentLabel(target.shipmentId);
      } else {
        await downloadAllegroLabel(target.shipmentId);
      }
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
          (!methodDecision.ok ||
            !suggested?.sender?.street ||
            !(suggested.sender.postalCode || suggested.sender.zipCode))) ||
        (step === "package-details" && createShipment.isPending)
      }
      hideCancel={step === "result"}
      onCancel={step === "package-details" ? () => setStep("select-service") : undefined}
      onConfirm={handleDialogConfirm}
      contentClassName="max-w-lg"
      error={importError}
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
          ) : !methodDecision.ok ? (
            <div className="mt-2 space-y-3">
              <p className="text-sm text-muted-foreground">
                {t("noWzAMethodForCheckout", {
                  method: checkoutMethodLabel(methodDecision) || t("unknownCheckoutMethod"),
                })}
              </p>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => void handleImportExisting()}
                disabled={importExisting.isPending}
              >
                {importExisting.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Download className="mr-2 h-4 w-4" />
                )}
                {importExisting.isPending ? t("importingWzA") : t("importExistingWzA")}
              </Button>
              {salesCenterUrl ? (
                <Button variant="outline" size="sm" asChild>
                  <a
                    href={sanitizeUrl(salesCenterUrl)}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <ExternalLink className="mr-2 h-4 w-4" />
                    {t("openSalesCenterCreateShipment")}
                  </a>
                </Button>
              ) : null}
            </div>
          ) : !suggested?.sender?.street ? (
            <p className="mt-2 text-sm text-muted-foreground">
              {t("noDeliveryServices")}
            </p>
          ) : (
            <div className="mt-2 space-y-3">
              <p className="text-sm">
                {deliveryServices.find((svc) => svc.id === proposedMethodId)?.name ||
                  checkoutMethodName ||
                  proposedMethodId}
              </p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void handleImportExisting()}
                disabled={importExisting.isPending}
              >
                {importExisting.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Download className="mr-2 h-4 w-4" />
                )}
                {importExisting.isPending ? t("importingWzA") : t("importExistingWzA")}
              </Button>
            </div>
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

      {step === "result" && (createdShipmentId || importedShipment) && (
        <>
          <div className="flex flex-col items-center gap-3 py-4">
            <CheckCircle2 className="h-12 w-12 text-green-500" />
            <p className="text-lg font-medium">
              {importedShipment ? t("wzaImported") : t("triggers.shipment.created")}
            </p>
            {importedShipment?.waybill ? (
              <p className="text-center text-sm font-medium">
                {t("wzaWaybillLabel", { waybill: importedShipment.waybill })}
              </p>
            ) : null}
            <p className="text-center text-sm text-muted-foreground">
              {t("shipmentIdLabel", {
                id: importedShipment?.allegro_shipment_id || createdShipmentId || importedShipment?.id || "",
              })}
            </p>
          </div>

          <Button
            className="w-full"
            onClick={handleDownloadLabel}
            disabled={
              isDownloading ||
              (importedShipment
                ? wzaLabelDownloadTarget(importedShipment).kind === "none"
                : !createdShipmentId)
            }
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
