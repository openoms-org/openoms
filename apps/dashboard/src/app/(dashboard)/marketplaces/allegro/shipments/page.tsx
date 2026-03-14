"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Plus,
  Download,
  Trash2,
  Truck,
  FileText,
  Clock,
  Package,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroDeliveryServices,
  useCreateAllegroShipment,
  useCancelAllegroShipment,
  useAllegroPickupProposals,
  useScheduleAllegroPickup,
  downloadAllegroLabel,
  downloadAllegroProtocol,
} from "@/hooks/use-allegro";
import type {
  AllegroCreateShipmentCommand,
  AllegroCreateShipmentResponse,
  AllegroDeliveryService,
  AllegroPickupProposal,
  AllegroPickupTimeWindow,
} from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { useTranslations } from "next-intl";

// ---- Local shipment tracking (client-side) ----

interface LocalShipment {
  commandId: string;
  shipmentId: string;
  status: string;
  deliveryServiceId: string;
  deliveryServiceName: string;
  receiverName: string;
  receiverCity: string;
  createdAt: string;
}

function statusBadgeVariant(status: string) {
  switch (status) {
    case "CREATED":
    case "NEW":
      return "default" as const;
    case "CANCELLED":
      return "destructive" as const;
    case "LABEL_DOWNLOADED":
      return "info" as const;
    case "PICKUP_SCHEDULED":
      return "success" as const;
    default:
      return "secondary" as const;
  }
}

function statusLabel(status: string, t: (key: string) => string) {
  switch (status) {
    case "CREATED":
    case "NEW":
      return t("allegroShipments.statusCreated");
    case "CANCELLED":
      return t("allegroShipments.statusCancelled");
    case "LABEL_DOWNLOADED":
      return t("allegroShipments.statusLabelDownloaded");
    case "PICKUP_SCHEDULED":
      return t("allegroShipments.statusPickupScheduled");
    default:
      return status;
  }
}

export default function AllegroShipmentsPage() {
  const t = useTranslations("marketplaces");
  const [shipments, setShipments] = useState<LocalShipment[]>([]);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [pickupDialogOpen, setPickupDialogOpen] = useState(false);
  const [protocolDialogOpen, setProtocolDialogOpen] = useState(false);
  const [selectedShipmentIds, setSelectedShipmentIds] = useState<Set<string>>(
    new Set()
  );

  const addShipment = useCallback((s: LocalShipment) => {
    setShipments((prev) => [s, ...prev]);
  }, []);

  const removeShipment = useCallback((shipmentId: string) => {
    setShipments((prev) => prev.filter((s) => s.shipmentId !== shipmentId));
  }, []);

  const updateShipmentStatus = useCallback(
    (shipmentId: string, status: string) => {
      setShipments((prev) =>
        prev.map((s) => (s.shipmentId === shipmentId ? { ...s, status } : s))
      );
    },
    []
  );

  const toggleSelection = useCallback((shipmentId: string) => {
    setSelectedShipmentIds((prev) => {
      const next = new Set(prev);
      if (next.has(shipmentId)) {
        next.delete(shipmentId);
      } else {
        next.add(shipmentId);
      }
      return next;
    });
  }, []);

  const activeShipments = shipments.filter((s) => s.status !== "CANCELLED");

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/marketplaces/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold">{t("allegroShipments.title")}</h1>
            <p className="text-muted-foreground">
              {t("allegroShipments.description")}
            </p>
          </div>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                {t("allegroShipments.createShipment")}
              </Button>
            </DialogTrigger>
            <CreateShipmentDialog
              onCreated={(s) => {
                addShipment(s);
                setCreateDialogOpen(false);
              }}
            />
          </Dialog>
        </div>

        {/* Shipments table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              {t("allegroShipments.shipmentList")}
              <span className="text-sm font-normal text-muted-foreground">
                ({shipments.length})
              </span>
            </CardTitle>
            <CardDescription>
              {t("allegroShipments.shipmentListDescription")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {shipments.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-12 text-muted-foreground">
                <Truck className="h-12 w-12 opacity-40" />
                <p>{t("allegroShipments.noShipments")}</p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-10">
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded"
                        checked={
                          activeShipments.length > 0 &&
                          activeShipments.every((s) =>
                            selectedShipmentIds.has(s.shipmentId)
                          )
                        }
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedShipmentIds(
                              new Set(activeShipments.map((s) => s.shipmentId))
                            );
                          } else {
                            setSelectedShipmentIds(new Set());
                          }
                        }}
                      />
                    </TableHead>
                    <TableHead>{t("allegroShipments.shipmentId")}</TableHead>
                    <TableHead>{t("allegroShipments.deliveryService")}</TableHead>
                    <TableHead>{t("allegroShipments.receiver")}</TableHead>
                    <TableHead>{t("allegroShipments.city")}</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">{t("allegroShipments.actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {shipments.map((shipment) => (
                    <ShipmentRow
                      key={shipment.shipmentId}
                      shipment={shipment}
                      selected={selectedShipmentIds.has(shipment.shipmentId)}
                      onToggleSelect={() =>
                        toggleSelection(shipment.shipmentId)
                      }
                      onCancelled={() => removeShipment(shipment.shipmentId)}
                      onLabelDownloaded={() =>
                        updateShipmentStatus(
                          shipment.shipmentId,
                          "LABEL_DOWNLOADED"
                        )
                      }
                    />
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        {/* Bulk actions */}
        {shipments.length > 0 && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {/* Download labels for selected */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Download className="h-4 w-4" />
                  {t("allegroShipments.labels")}
                </CardTitle>
                <CardDescription>
                  {t("allegroShipments.labelsDescription")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <BulkLabelDownload
                  selectedIds={selectedShipmentIds}
                  onDownloaded={(id) =>
                    updateShipmentStatus(id, "LABEL_DOWNLOADED")
                  }
                />
              </CardContent>
            </Card>

            {/* Pickup scheduling */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Clock className="h-4 w-4" />
                  {t("allegroShipments.courierPickup")}
                </CardTitle>
                <CardDescription>
                  {t("allegroShipments.courierPickupDescription")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Dialog
                  open={pickupDialogOpen}
                  onOpenChange={setPickupDialogOpen}
                >
                  <DialogTrigger asChild>
                    <Button
                      variant="outline"
                      className="w-full"
                      disabled={selectedShipmentIds.size === 0}
                    >
                      <Truck className="mr-2 h-4 w-4" />
                      {t("allegroShipments.schedulePickup", { count: selectedShipmentIds.size })}
                    </Button>
                  </DialogTrigger>
                  <PickupDialog
                    selectedIds={Array.from(selectedShipmentIds)}
                    onScheduled={() => {
                      setPickupDialogOpen(false);
                      selectedShipmentIds.forEach((id) =>
                        updateShipmentStatus(id, "PICKUP_SCHEDULED")
                      );
                    }}
                  />
                </Dialog>
              </CardContent>
            </Card>

            {/* Protocol generation */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <FileText className="h-4 w-4" />
                  {t("allegroShipments.dispatchProtocol")}
                </CardTitle>
                <CardDescription>
                  {t("allegroShipments.dispatchProtocolDescription")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Dialog
                  open={protocolDialogOpen}
                  onOpenChange={setProtocolDialogOpen}
                >
                  <DialogTrigger asChild>
                    <Button
                      variant="outline"
                      className="w-full"
                      disabled={selectedShipmentIds.size === 0}
                    >
                      <FileText className="mr-2 h-4 w-4" />
                      {t("allegroShipments.generateProtocol", { count: selectedShipmentIds.size })}
                    </Button>
                  </DialogTrigger>
                  <ProtocolDialog
                    selectedIds={Array.from(selectedShipmentIds)}
                    onGenerated={() => setProtocolDialogOpen(false)}
                  />
                </Dialog>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminGuard>
  );
}

// ---- Create Shipment Dialog ----

function CreateShipmentDialog({
  onCreated,
}: {
  onCreated: (shipment: LocalShipment) => void;
}) {
  const t = useTranslations("marketplaces");
  const { data: servicesData, isLoading: servicesLoading } =
    useAllegroDeliveryServices();
  const createShipment = useCreateAllegroShipment();

  const [selectedServiceId, setSelectedServiceId] = useState("");

  // Receiver fields
  const [receiverName, setReceiverName] = useState("");
  const [receiverCompany, setReceiverCompany] = useState("");
  const [receiverStreet, setReceiverStreet] = useState("");
  const [receiverCity, setReceiverCity] = useState("");
  const [receiverZipCode, setReceiverZipCode] = useState("");
  const [receiverCountryCode, setReceiverCountryCode] = useState("PL");
  const [receiverPhone, setReceiverPhone] = useState("");
  const [receiverEmail, setReceiverEmail] = useState("");

  // Sender fields
  const [senderName, setSenderName] = useState("");
  const [senderCompany, setSenderCompany] = useState("");
  const [senderStreet, setSenderStreet] = useState("");
  const [senderCity, setSenderCity] = useState("");
  const [senderZipCode, setSenderZipCode] = useState("");
  const [senderCountryCode, setSenderCountryCode] = useState("PL");
  const [senderPhone, setSenderPhone] = useState("");
  const [senderEmail, setSenderEmail] = useState("");

  // Package fields
  const [pkgLength, setPkgLength] = useState("");
  const [pkgWidth, setPkgWidth] = useState("");
  const [pkgHeight, setPkgHeight] = useState("");
  const [pkgWeight, setPkgWeight] = useState("");

  const deliveryServices: AllegroDeliveryService[] =
    servicesData?.delivery_services ?? [];

  const selectedService = deliveryServices.find(
    (ds) => ds.id === selectedServiceId
  );

  const handleSubmit = () => {
    if (!selectedServiceId) {
      toast.error(t("allegroShipments.selectDeliveryService"));
      return;
    }
    if (!receiverStreet || !receiverCity || !receiverZipCode) {
      toast.error(t("allegroShipments.fillReceiverAddress"));
      return;
    }
    if (!senderStreet || !senderCity || !senderZipCode) {
      toast.error(t("allegroShipments.fillSenderAddress"));
      return;
    }

    const commandId = crypto.randomUUID();

    const cmd: AllegroCreateShipmentCommand = {
      commandId,
      input: {
        deliveryMethodId: selectedServiceId,
        sender: {
          name: senderName || undefined,
          company: senderCompany || undefined,
          street: senderStreet,
          city: senderCity,
          zipCode: senderZipCode,
          countryCode: senderCountryCode,
          phone: senderPhone || undefined,
          email: senderEmail || undefined,
        },
        receiver: {
          name: receiverName || undefined,
          company: receiverCompany || undefined,
          street: receiverStreet,
          city: receiverCity,
          zipCode: receiverZipCode,
          countryCode: receiverCountryCode,
          phone: receiverPhone || undefined,
          email: receiverEmail || undefined,
        },
        packages: [
          {
            ...(pkgLength
              ? {
                  length: {
                    value: parseFloat(pkgLength),
                    unit: "cm",
                  },
                }
              : {}),
            ...(pkgWidth
              ? {
                  width: {
                    value: parseFloat(pkgWidth),
                    unit: "cm",
                  },
                }
              : {}),
            ...(pkgHeight
              ? {
                  height: {
                    value: parseFloat(pkgHeight),
                    unit: "cm",
                  },
                }
              : {}),
            ...(pkgWeight
              ? {
                  weight: {
                    value: parseFloat(pkgWeight),
                    unit: "kg",
                  },
                }
              : {}),
          },
        ],
      },
    };

    createShipment.mutate(cmd, {
      onSuccess: (resp: AllegroCreateShipmentResponse) => {
        toast.success(t("allegroShipments.shipmentCreated"));
        onCreated({
          commandId: resp.commandId,
          shipmentId: resp.shipmentId,
          status: resp.status || "CREATED",
          deliveryServiceId: selectedServiceId,
          deliveryServiceName: selectedService?.name ?? selectedServiceId,
          receiverName: receiverName || receiverCompany || "---",
          receiverCity: receiverCity,
          createdAt: new Date().toISOString(),
        });
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("allegroShipments.createShipmentError")
        );
      },
    });
  };

  return (
    <DialogContent size="lg">
      <DialogHeader>
        <DialogTitle>{t("allegroShipments.createNewShipment")}</DialogTitle>
        <DialogDescription>
          {t("allegroShipments.createNewShipmentDescription")}
        </DialogDescription>
      </DialogHeader>

      <div className="max-h-[60vh] overflow-y-auto space-y-6 pr-2">
        {/* Delivery service selection */}
        <div className="space-y-2">
          <Label>{t("allegroShipments.deliveryService")}</Label>
          {servicesLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : (
            <Select
              value={selectedServiceId}
              onValueChange={setSelectedServiceId}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("allegroShipments.selectDeliveryServicePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {deliveryServices.map((ds) => (
                  <SelectItem key={ds.id} value={ds.id}>
                    {ds.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        <Separator />

        {/* Sender address */}
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">{t("allegroShipments.sender")}</h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.fullName")}</Label>
              <Input
                placeholder={t("allegroShipments.fullNamePlaceholder")}
                value={senderName}
                onChange={(e) => setSenderName(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.company")}</Label>
              <Input
                placeholder={t("allegroShipments.companyPlaceholder")}
                value={senderCompany}
                onChange={(e) => setSenderCompany(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("allegroShipments.street")} *</Label>
            <Input
              placeholder={t("allegroShipments.streetPlaceholderSender")}
              value={senderStreet}
              onChange={(e) => setSenderStreet(e.target.value)}
              required
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.city")} *</Label>
              <Input
                placeholder={t("allegroShipments.cityPlaceholderSender")}
                value={senderCity}
                onChange={(e) => setSenderCity(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.postalCode")} *</Label>
              <Input
                placeholder="00-001"
                value={senderZipCode}
                onChange={(e) => setSenderZipCode(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.country")}</Label>
              <Input
                value={senderCountryCode}
                onChange={(e) => setSenderCountryCode(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.phone")}</Label>
              <Input
                placeholder="+48 123 456 789"
                value={senderPhone}
                onChange={(e) => setSenderPhone(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Email</Label>
              <Input
                placeholder={t("allegroShipments.emailPlaceholderSender")}
                value={senderEmail}
                onChange={(e) => setSenderEmail(e.target.value)}
              />
            </div>
          </div>
        </div>

        <Separator />

        {/* Receiver address */}
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">{t("allegroShipments.receiver")}</h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.fullName")}</Label>
              <Input
                placeholder={t("allegroShipments.fullNamePlaceholderReceiver")}
                value={receiverName}
                onChange={(e) => setReceiverName(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.company")}</Label>
              <Input
                placeholder={t("allegroShipments.companyPlaceholder")}
                value={receiverCompany}
                onChange={(e) => setReceiverCompany(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("allegroShipments.street")} *</Label>
            <Input
              placeholder={t("allegroShipments.streetPlaceholderReceiver")}
              value={receiverStreet}
              onChange={(e) => setReceiverStreet(e.target.value)}
              required
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.city")} *</Label>
              <Input
                placeholder={t("allegroShipments.cityPlaceholderReceiver")}
                value={receiverCity}
                onChange={(e) => setReceiverCity(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.postalCode")} *</Label>
              <Input
                placeholder="30-001"
                value={receiverZipCode}
                onChange={(e) => setReceiverZipCode(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.country")}</Label>
              <Input
                value={receiverCountryCode}
                onChange={(e) => setReceiverCountryCode(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.phone")}</Label>
              <Input
                placeholder="+48 987 654 321"
                value={receiverPhone}
                onChange={(e) => setReceiverPhone(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Email</Label>
              <Input
                placeholder={t("allegroShipments.emailPlaceholderReceiver")}
                value={receiverEmail}
                onChange={(e) => setReceiverEmail(e.target.value)}
              />
            </div>
          </div>
        </div>

        <Separator />

        {/* Package dimensions */}
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">{t("allegroShipments.packageDimensions")}</h3>
          <div className="grid grid-cols-4 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.lengthCm")}</Label>
              <Input
                type="number"
                placeholder="30"
                min="0"
                step="0.1"
                value={pkgLength}
                onChange={(e) => setPkgLength(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.widthCm")}</Label>
              <Input
                type="number"
                placeholder="20"
                min="0"
                step="0.1"
                value={pkgWidth}
                onChange={(e) => setPkgWidth(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.heightCm")}</Label>
              <Input
                type="number"
                placeholder="15"
                min="0"
                step="0.1"
                value={pkgHeight}
                onChange={(e) => setPkgHeight(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">{t("allegroShipments.weightKg")}</Label>
              <Input
                type="number"
                placeholder="1.5"
                min="0"
                step="0.01"
                value={pkgWeight}
                onChange={(e) => setPkgWeight(e.target.value)}
              />
            </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button
          onClick={handleSubmit}
          disabled={createShipment.isPending}
        >
          {createShipment.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          <Plus className="mr-2 h-4 w-4" />
          {t("allegroShipments.createShipment")}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}

// ---- Shipment Row ----

function ShipmentRow({
  shipment,
  selected,
  onToggleSelect,
  onCancelled,
  onLabelDownloaded,
}: {
  shipment: LocalShipment;
  selected: boolean;
  onToggleSelect: () => void;
  onCancelled: () => void;
  onLabelDownloaded: () => void;
}) {
  const t = useTranslations("marketplaces");
  const cancelShipment = useCancelAllegroShipment();
  const [isDownloadingLabel, setIsDownloadingLabel] = useState(false);

  const isCancelled = shipment.status === "CANCELLED";

  const handleCancel = () => {
    if (
      !confirm(
        t("allegroShipments.confirmCancel", { id: shipment.shipmentId })
      )
    ) {
      return;
    }
    cancelShipment.mutate(shipment.shipmentId, {
      onSuccess: () => {
        toast.success(t("allegroShipments.shipmentCancelled"));
        onCancelled();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("allegroShipments.cancelShipmentError")
        );
      },
    });
  };

  const handleDownloadLabel = async () => {
    setIsDownloadingLabel(true);
    try {
      await downloadAllegroLabel(shipment.shipmentId);
      toast.success(t("allegroShipments.labelDownloaded"));
      onLabelDownloaded();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t("allegroShipments.downloadLabelError")
      );
    } finally {
      setIsDownloadingLabel(false);
    }
  };

  return (
    <TableRow className={isCancelled ? "opacity-50" : ""}>
      <TableCell>
        <input
          type="checkbox"
          className="h-4 w-4 rounded"
          checked={selected}
          onChange={onToggleSelect}
          disabled={isCancelled}
        />
      </TableCell>
      <TableCell>
        <p className="font-mono text-xs">{shipment.shipmentId}</p>
      </TableCell>
      <TableCell>
        <p className="text-sm">{shipment.deliveryServiceName}</p>
      </TableCell>
      <TableCell>
        <p className="text-sm">{shipment.receiverName}</p>
      </TableCell>
      <TableCell>
        <p className="text-sm">{shipment.receiverCity}</p>
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(shipment.status)}>
          {statusLabel(shipment.status, t)}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDownloadLabel}
            disabled={isCancelled || isDownloadingLabel}
            title={t("allegroShipments.downloadLabel")}
          >
            {isDownloadingLabel ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <Download className="h-3 w-3" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCancel}
            disabled={isCancelled || cancelShipment.isPending}
            title={t("allegroShipments.cancelShipment")}
          >
            {cancelShipment.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <Trash2 className="h-3 w-3 text-destructive" />
            )}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// ---- Bulk Label Download ----

function BulkLabelDownload({
  selectedIds,
  onDownloaded,
}: {
  selectedIds: Set<string>;
  onDownloaded: (id: string) => void;
}) {
  const t = useTranslations("marketplaces");
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownloadAll = async () => {
    if (selectedIds.size === 0) return;
    setIsDownloading(true);
    try {
      for (const id of selectedIds) {
        await downloadAllegroLabel(id);
        onDownloaded(id);
      }
      toast.success(t("allegroShipments.labelsDownloaded", { count: selectedIds.size }));
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t("allegroShipments.downloadLabelsError")
      );
    } finally {
      setIsDownloading(false);
    }
  };

  return (
    <Button
      variant="outline"
      className="w-full"
      disabled={selectedIds.size === 0 || isDownloading}
      onClick={handleDownloadAll}
    >
      {isDownloading ? (
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      ) : (
        <Download className="mr-2 h-4 w-4" />
      )}
      {t("allegroShipments.downloadLabels", { count: selectedIds.size })}
    </Button>
  );
}

// ---- Pickup Dialog ----

function PickupDialog({
  selectedIds,
  onScheduled,
}: {
  selectedIds: string[];
  onScheduled: () => void;
}) {
  const t = useTranslations("marketplaces");
  const getProposals = useAllegroPickupProposals();
  const schedulePickup = useScheduleAllegroPickup();

  const [deliveryMethodId, setDeliveryMethodId] = useState("");
  const [proposals, setProposals] = useState<AllegroPickupProposal[]>([]);
  const [selectedDate, setSelectedDate] = useState("");
  const [selectedWindow, setSelectedWindow] =
    useState<AllegroPickupTimeWindow | null>(null);

  const { data: servicesData } = useAllegroDeliveryServices();
  const deliveryServices: AllegroDeliveryService[] =
    servicesData?.delivery_services ?? [];

  const handleFetchProposals = () => {
    if (!deliveryMethodId) {
      toast.error(t("allegroShipments.selectDeliveryMethod"));
      return;
    }
    getProposals.mutate(
      { deliveryMethodId, shipmentIds: selectedIds },
      {
        onSuccess: (resp) => {
          setProposals(resp.proposals ?? []);
          if (!resp.proposals?.length) {
            toast.info(t("allegroShipments.noPickupSlots"));
          }
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("allegroShipments.fetchPickupProposalsError")
          );
        },
      }
    );
  };

  const handleSchedule = () => {
    if (!selectedDate || !selectedWindow) {
      toast.error(t("allegroShipments.selectDateAndTimeWindow"));
      return;
    }
    schedulePickup.mutate(
      {
        commandId: crypto.randomUUID(),
        pickupDate: selectedDate,
        timeWindow: selectedWindow,
        shipmentIds: selectedIds,
      },
      {
        onSuccess: () => {
          toast.success(t("allegroShipments.pickupScheduled"));
          onScheduled();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("allegroShipments.schedulePickupError")
          );
        },
      }
    );
  };

  const selectedProposal = proposals.find((p) => p.date === selectedDate);

  return (
    <DialogContent size="md">
      <DialogHeader>
        <DialogTitle>{t("allegroShipments.schedulePickupTitle")}</DialogTitle>
        <DialogDescription>
          {t("allegroShipments.schedulePickupDescription", { count: selectedIds.length })}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        {/* Delivery method selector */}
        <div className="space-y-2">
          <Label>{t("allegroShipments.deliveryMethod")}</Label>
          <Select value={deliveryMethodId} onValueChange={setDeliveryMethodId}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t("allegroShipments.selectDeliveryMethodPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {deliveryServices.map((ds) => (
                <SelectItem key={ds.id} value={ds.id}>
                  {ds.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <Button
          variant="outline"
          onClick={handleFetchProposals}
          disabled={!deliveryMethodId || getProposals.isPending}
          className="w-full"
        >
          {getProposals.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          {t("allegroShipments.checkAvailableSlots")}
        </Button>

        {/* Proposals */}
        {proposals.length > 0 && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>{t("allegroShipments.availableSlots")}</Label>
              <Select value={selectedDate} onValueChange={(v) => {
                setSelectedDate(v);
                setSelectedWindow(null);
              }}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("allegroShipments.selectDate")} />
                </SelectTrigger>
                <SelectContent>
                  {proposals.map((p) => (
                    <SelectItem key={p.date} value={p.date}>
                      {p.date}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedProposal && (
              <div className="space-y-2">
                <Label>{t("allegroShipments.timeWindow")}</Label>
                <div className="grid grid-cols-2 gap-2">
                  {selectedProposal.timeWindows.map((tw, idx) => (
                    <Button
                      key={idx}
                      variant={
                        selectedWindow?.from === tw.from &&
                        selectedWindow?.to === tw.to
                          ? "default"
                          : "outline"
                      }
                      size="sm"
                      onClick={() => setSelectedWindow(tw)}
                    >
                      {tw.from} - {tw.to}
                    </Button>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      <DialogFooter>
        <Button
          onClick={handleSchedule}
          disabled={
            !selectedDate || !selectedWindow || schedulePickup.isPending
          }
        >
          {schedulePickup.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          <Truck className="mr-2 h-4 w-4" />
          {t("allegroShipments.schedulePickupButton")}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}

// ---- Protocol Dialog ----

function ProtocolDialog({
  selectedIds,
  onGenerated,
}: {
  selectedIds: string[];
  onGenerated: () => void;
}) {
  const t = useTranslations("marketplaces");
  const [isGenerating, setIsGenerating] = useState(false);

  const handleGenerate = async () => {
    setIsGenerating(true);
    try {
      await downloadAllegroProtocol(selectedIds);
      toast.success(t("allegroShipments.protocolGenerated"));
      onGenerated();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t("allegroShipments.generateProtocolError")
      );
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <DialogContent size="sm">
      <DialogHeader>
        <DialogTitle>{t("allegroShipments.generateProtocolTitle")}</DialogTitle>
        <DialogDescription>
          {t("allegroShipments.generateProtocolDescription", { count: selectedIds.length })}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-2">
        <p className="text-sm text-muted-foreground">
          {t("allegroShipments.shipmentIds")}
        </p>
        <div className="max-h-32 overflow-y-auto rounded border p-2">
          {selectedIds.map((id) => (
            <p key={id} className="font-mono text-xs">
              {id}
            </p>
          ))}
        </div>
      </div>

      <DialogFooter>
        <Button
          onClick={handleGenerate}
          disabled={isGenerating}
        >
          {isGenerating ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <FileText className="mr-2 h-4 w-4" />
          )}
          {t("allegroShipments.generateAndDownload")}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}
