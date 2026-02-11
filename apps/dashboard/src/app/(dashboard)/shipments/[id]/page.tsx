"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, Pencil, Trash2, ExternalLink, FileDown, Tag, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { ShipmentForm } from "@/components/shipments/shipment-form";
import { ShipmentStatusActions } from "@/components/shipments/shipment-status-actions";
import { GenerateLabelDialog } from "@/components/shipments/generate-label-dialog";
import {
  useShipment,
  useUpdateShipment,
  useDeleteShipment,
  useTransitionShipmentStatus,
} from "@/hooks/use-shipments";
import { useOrder } from "@/hooks/use-orders";
import { SHIPMENT_STATUSES, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { formatDate, shortId } from "@/lib/utils";

export default function ShipmentDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showLabelDialog, setShowLabelDialog] = useState(false);

  const { data: shipment, isLoading } = useShipment(params.id);
  const { data: order } = useOrder(shipment?.order_id ?? "");
  const updateShipment = useUpdateShipment(params.id);
  const deleteShipment = useDeleteShipment();
  const transitionStatus = useTransitionShipmentStatus(params.id);

  const handleUpdate = (data: { tracking_number?: string; label_url?: string }) => {
    updateShipment.mutate(
      {
        tracking_number: data.tracking_number || undefined,
        label_url: data.label_url || undefined,
      },
      {
        onSuccess: () => {
          toast.success("Przesyłka została zaktualizowana");
          setIsEditing(false);
        },
        onError: (error) => {
          toast.error(error.message || "Nie udało się zaktualizować przesyłki");
        },
      }
    );
  };

  const handleDelete = () => {
    deleteShipment.mutate(params.id, {
      onSuccess: () => {
        toast.success("Przesyłka została usunięta");
        router.push("/shipments");
      },
      onError: (error) => {
        toast.error(error.message || "Nie udało się usunąć przesyłki");
      },
    });
  };

  const handleStatusTransition = (status: string) => {
    transitionStatus.mutate(
      { status },
      {
        onSuccess: () => {
          toast.success("Status przesyłki został zmieniony");
        },
        onError: (error) => {
          toast.error(error.message || "Nie udało się zmienić statusu");
        },
      }
    );
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!shipment) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Nie znaleziono przesyłki</h1>
        <Button asChild variant="outline">
          <Link href="/shipments">Wróć do listy</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/shipments">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">
              Przesyłka {shortId(shipment.id)}
            </h1>
            <p className="text-muted-foreground">
              Utworzona {formatDate(shipment.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {shipment.provider !== "manual" && shipment.status === "created" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowLabelDialog(true)}
            >
              <Tag className="h-4 w-4" />
              Generuj etykietę {SHIPMENT_PROVIDER_LABELS[shipment.provider] ?? shipment.provider.toUpperCase()}
            </Button>
          )}
          {shipment.label_url && (
            <Button variant="outline" size="sm" asChild>
              <a href={shipment.label_url} target="_blank" rel="noopener noreferrer">
                <FileDown className="h-4 w-4" />
                Pobierz etykietę
              </a>
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            <Pencil className="h-4 w-4" />
            {isEditing ? "Anuluj edycję" : "Edytuj"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="h-4 w-4" />
            Usuń
       </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>Edycja przesyłki</CardTitle>
          </CardHeader>
          <CardContent>
            <ShipmentForm
              shipment={shipment}
              onSubmit={handleUpdate}
              isLoading={updateShipment.isPending}
            />
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Szczegóły przesyłki</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="font-mono text-sm">{shipment.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Zamówienie</p>
                <Link
                  href={`/orders/${shipment.order_id}`}
                  className="inline-flex items-center gap-1 font-mono text-sm text-primary hover:underline"
                >
                  {shortId(shipment.order_id)}
                  <ExternalLink className="h-3 w-3" />
                </Link>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Dostawca</p>
                <p className="text-sm">{SHIPMENT_PROVIDER_LABELS[shipment.provider] ?? shipment.provider.toUpperCase()}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <StatusBadge
                  status={shipment.status}
                  statusMap={SHIPMENT_STATUSES}
                />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Numer śledzenia</p>
                <p className="text-sm">
                  {shipment.tracking_number || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">URL etykiety</p>
                {shipment.label_url ? (
                  <a
                    href={shipment.label_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                  >
                    Otwórz etykietę
                    <ExternalLink className="h-3 w-3" />
                  </a>
                ) : (
                  <p className="text-sm">-</p>
                )}
              </div>
              {shipment.provider === "inpost" &&
                typeof shipment.carrier_data?.target_point === "string" && (
                  <div>
                    <p className="text-sm text-muted-foreground">
                      Paczkomat docelowy
                    </p>
                    <div className="inline-flex items-center gap-1.5 rounded-md border bg-muted/50 px-2.5 py-1 mt-1">
                      <MapPin className="h-3.5 w-3.5 text-primary" />
                      <span className="text-sm font-medium">
                        {shipment.carrier_data.target_point}
                      </span>
                    </div>
                  </div>
                )}
              <div>
                <p className="text-sm text-muted-foreground">Data utworzenia</p>
                <p className="text-sm">{formatDate(shipment.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Ostatnia aktualizacja
                </p>
                <p className="text-sm">{formatDate(shipment.updated_at)}</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Zmiana statusu</CardTitle>
              <CardDescription>
                Dostępne przejścia statusu dla tej przesyłki
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ShipmentStatusActions
                currentStatus={shipment.status}
                onTransition={handleStatusTransition}
                isLoading={transitionStatus.isPending}
              />
            </CardContent>
          </Card>
        </div>
      )}

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Usuń przesyłkę</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz usunąć tę przesyłkę? Ta operacja jest
              nieodwracalna.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteShipment.isPending}
            >
              {deleteShipment.isPending ? "Usuwanie..." : "Usuń"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <GenerateLabelDialog
        shipmentId={params.id}
        provider={shipment.provider}
        order={order}
        open={showLabelDialog}
        onOpenChange={setShowLabelDialog}
      />
    </div>
  );
}
