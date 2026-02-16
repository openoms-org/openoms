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

function statusLabel(status: string) {
  switch (status) {
    case "CREATED":
    case "NEW":
      return "Utworzona";
    case "CANCELLED":
      return "Anulowana";
    case "LABEL_DOWNLOADED":
      return "Etykieta pobrana";
    case "PICKUP_SCHEDULED":
      return "Odbiór zaplanowany";
    default:
      return status;
  }
}

export default function AllegroShipmentsPage() {
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
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold">Wysylam z Allegro</h1>
            <p className="text-muted-foreground">
              Zarzadzaj przesylkami, etykietami i odbiorem kurierskim
            </p>
          </div>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Utworz przesylke
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
              Lista przesylek
              <span className="text-sm font-normal text-muted-foreground">
                ({shipments.length})
              </span>
            </CardTitle>
            <CardDescription>
              Przesylki utworzone w biezacej sesji. Zaznacz przesylki, aby pobrac
              etykiety lub wygenerowac protokol nadania.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {shipments.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-12 text-muted-foreground">
                <Truck className="h-12 w-12 opacity-40" />
                <p>Brak przesylek. Kliknij &quot;Utworz przesylke&quot; aby rozpoczac.</p>
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
                    <TableHead>ID przesylki</TableHead>
                    <TableHead>Usluga dostawy</TableHead>
                    <TableHead>Odbiorca</TableHead>
                    <TableHead>Miasto</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Akcje</TableHead>
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
                  Etykiety
                </CardTitle>
                <CardDescription>
                  Pobierz etykiety dla zaznaczonych przesylek
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
                  Odbior kurierski
                </CardTitle>
                <CardDescription>
                  Zamow odbior paczek przez kuriera
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
                      Zaplanuj odbior ({selectedShipmentIds.size})
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
                  Protokol nadania
                </CardTitle>
                <CardDescription>
                  Wygeneruj protokol dla zaznaczonych przesylek
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
                      Generuj protokol ({selectedShipmentIds.size})
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
      toast.error("Wybierz usluge dostawy");
      return;
    }
    if (!receiverStreet || !receiverCity || !receiverZipCode) {
      toast.error("Uzupelnij adres odbiorcy (ulica, miasto, kod pocztowy)");
      return;
    }
    if (!senderStreet || !senderCity || !senderZipCode) {
      toast.error("Uzupelnij adres nadawcy (ulica, miasto, kod pocztowy)");
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
        toast.success("Przesylka utworzona pomyslnie");
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
            : "Nie udalo sie utworzyc przesylki"
        );
      },
    });
  };

  return (
    <DialogContent size="lg">
      <DialogHeader>
        <DialogTitle>Utworz nowa przesylke</DialogTitle>
        <DialogDescription>
          Wypelnij dane przesylki, aby utworzyc ja przez Allegro.
        </DialogDescription>
      </DialogHeader>

      <div className="max-h-[60vh] overflow-y-auto space-y-6 pr-2">
        {/* Delivery service selection */}
        <div className="space-y-2">
          <Label>Usluga dostawy</Label>
          {servicesLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : (
            <Select
              value={selectedServiceId}
              onValueChange={setSelectedServiceId}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Wybierz usluge dostawy..." />
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
          <h3 className="font-semibold text-sm">Nadawca</h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Imie i nazwisko</Label>
              <Input
                placeholder="Jan Kowalski"
                value={senderName}
                onChange={(e) => setSenderName(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Firma</Label>
              <Input
                placeholder="Nazwa firmy (opcjonalnie)"
                value={senderCompany}
                onChange={(e) => setSenderCompany(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Ulica *</Label>
            <Input
              placeholder="ul. Przykladowa 1/2"
              value={senderStreet}
              onChange={(e) => setSenderStreet(e.target.value)}
              required
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Miasto *</Label>
              <Input
                placeholder="Warszawa"
                value={senderCity}
                onChange={(e) => setSenderCity(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Kod pocztowy *</Label>
              <Input
                placeholder="00-001"
                value={senderZipCode}
                onChange={(e) => setSenderZipCode(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Kraj</Label>
              <Input
                value={senderCountryCode}
                onChange={(e) => setSenderCountryCode(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Telefon</Label>
              <Input
                placeholder="+48 123 456 789"
                value={senderPhone}
                onChange={(e) => setSenderPhone(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Email</Label>
              <Input
                placeholder="nadawca@email.pl"
                value={senderEmail}
                onChange={(e) => setSenderEmail(e.target.value)}
              />
            </div>
          </div>
        </div>

        <Separator />

        {/* Receiver address */}
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">Odbiorca</h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Imie i nazwisko</Label>
              <Input
                placeholder="Anna Nowak"
                value={receiverName}
                onChange={(e) => setReceiverName(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Firma</Label>
              <Input
                placeholder="Nazwa firmy (opcjonalnie)"
                value={receiverCompany}
                onChange={(e) => setReceiverCompany(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Ulica *</Label>
            <Input
              placeholder="ul. Odbiorcza 5/10"
              value={receiverStreet}
              onChange={(e) => setReceiverStreet(e.target.value)}
              required
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Miasto *</Label>
              <Input
                placeholder="Krakow"
                value={receiverCity}
                onChange={(e) => setReceiverCity(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Kod pocztowy *</Label>
              <Input
                placeholder="30-001"
                value={receiverZipCode}
                onChange={(e) => setReceiverZipCode(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Kraj</Label>
              <Input
                value={receiverCountryCode}
                onChange={(e) => setReceiverCountryCode(e.target.value)}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Telefon</Label>
              <Input
                placeholder="+48 987 654 321"
                value={receiverPhone}
                onChange={(e) => setReceiverPhone(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Email</Label>
              <Input
                placeholder="odbiorca@email.pl"
                value={receiverEmail}
                onChange={(e) => setReceiverEmail(e.target.value)}
              />
            </div>
          </div>
        </div>

        <Separator />

        {/* Package dimensions */}
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">Wymiary paczki</h3>
          <div className="grid grid-cols-4 gap-3">
            <div className="space-y-1">
              <Label className="text-xs">Dlugosc (cm)</Label>
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
              <Label className="text-xs">Szerokosc (cm)</Label>
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
              <Label className="text-xs">Wysokosc (cm)</Label>
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
              <Label className="text-xs">Waga (kg)</Label>
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
          Utworz przesylke
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
  const cancelShipment = useCancelAllegroShipment();
  const [isDownloadingLabel, setIsDownloadingLabel] = useState(false);

  const isCancelled = shipment.status === "CANCELLED";

  const handleCancel = () => {
    if (
      !confirm(
        `Czy na pewno chcesz anulowac przesylke ${shipment.shipmentId}?`
      )
    ) {
      return;
    }
    cancelShipment.mutate(shipment.shipmentId, {
      onSuccess: () => {
        toast.success("Przesylka anulowana");
        onCancelled();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Nie udalo sie anulowac przesylki"
        );
      },
    });
  };

  const handleDownloadLabel = async () => {
    setIsDownloadingLabel(true);
    try {
      await downloadAllegroLabel(shipment.shipmentId);
      toast.success("Etykieta pobrana");
      onLabelDownloaded();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : "Nie udalo sie pobrac etykiety"
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
          {statusLabel(shipment.status)}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDownloadLabel}
            disabled={isCancelled || isDownloadingLabel}
            title="Pobierz etykiete"
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
            title="Anuluj przesylke"
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
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownloadAll = async () => {
    if (selectedIds.size === 0) return;
    setIsDownloading(true);
    try {
      for (const id of selectedIds) {
        await downloadAllegroLabel(id);
        onDownloaded(id);
      }
      toast.success(`Pobrano ${selectedIds.size} etykiet`);
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : "Nie udalo sie pobrac etykiet"
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
      Pobierz etykiety ({selectedIds.size})
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
      toast.error("Wybierz metode dostawy");
      return;
    }
    getProposals.mutate(
      { deliveryMethodId, shipmentIds: selectedIds },
      {
        onSuccess: (resp) => {
          setProposals(resp.proposals ?? []);
          if (!resp.proposals?.length) {
            toast.info("Brak dostepnych terminow odbioru");
          }
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie pobrac propozycji odbioru"
          );
        },
      }
    );
  };

  const handleSchedule = () => {
    if (!selectedDate || !selectedWindow) {
      toast.error("Wybierz date i okno czasowe odbioru");
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
          toast.success("Odbior kurierski zaplanowany");
          onScheduled();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie zaplanowac odbioru"
          );
        },
      }
    );
  };

  const selectedProposal = proposals.find((p) => p.date === selectedDate);

  return (
    <DialogContent size="md">
      <DialogHeader>
        <DialogTitle>Zaplanuj odbior kurierski</DialogTitle>
        <DialogDescription>
          Wybierz metode dostawy i termin odbioru dla {selectedIds.length}{" "}
          przesylek.
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        {/* Delivery method selector */}
        <div className="space-y-2">
          <Label>Metoda dostawy</Label>
          <Select value={deliveryMethodId} onValueChange={setDeliveryMethodId}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Wybierz metode dostawy..." />
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
          Sprawdz dostepne terminy
        </Button>

        {/* Proposals */}
        {proposals.length > 0 && (
          <>
            <Separator />
            <div className="space-y-2">
              <Label>Dostepne terminy</Label>
              <Select value={selectedDate} onValueChange={(v) => {
                setSelectedDate(v);
                setSelectedWindow(null);
              }}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Wybierz date..." />
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
                <Label>Okno czasowe</Label>
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
          Zaplanuj odbior
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
  const [isGenerating, setIsGenerating] = useState(false);

  const handleGenerate = async () => {
    setIsGenerating(true);
    try {
      await downloadAllegroProtocol(selectedIds);
      toast.success("Protokol nadania wygenerowany i pobrany");
      onGenerated();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : "Nie udalo sie wygenerowac protokolu"
      );
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <DialogContent size="sm">
      <DialogHeader>
        <DialogTitle>Generuj protokol nadania</DialogTitle>
        <DialogDescription>
          Protokol zostanie wygenerowany dla {selectedIds.length} zaznaczonych
          przesylek i pobrany jako plik PDF.
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-2">
        <p className="text-sm text-muted-foreground">
          ID przesylek:
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
          Generuj i pobierz
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}
