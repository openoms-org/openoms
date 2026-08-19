"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, Pencil, Trash2, ExternalLink, FileDown, Tag, MapPin, Route, Truck, Leaf } from "lucide-react";
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
import { ShipmentLabelDownload } from "@/components/shipments/shipment-label-download";
import { DispatchOrderDialog } from "@/components/shipments/dispatch-order-dialog";
import { TrackingTimeline } from "@/components/shipments/tracking-timeline";
import {
  useShipment,
  useUpdateShipment,
  useDeleteShipment,
  useTransitionShipmentStatus,
  useShipmentTracking,
} from "@/hooks/use-shipments";
import { useOrder } from "@/hooks/use-orders";
import { SHIPMENT_STATUSES, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { formatDate, shortId } from "@/lib/utils";
import { useTranslations } from "next-intl";

const SENDING_METHOD_LABEL_KEYS: Record<string, string> = {
  dispatch_order: "sendingMethods.dispatchOrder",
  parcel_locker: "sendingMethods.parcelLocker",
  pop: "sendingMethods.pop",
  any_point: "sendingMethods.anyPoint",
  pok: "sendingMethods.pok",
  branch: "sendingMethods.branch",
};

export default function ShipmentDetailPage() {
  const t = useTranslations("shipments");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showLabelDialog, setShowLabelDialog] = useState(false);
  const [showDispatchDialog, setShowDispatchDialog] = useState(false);

  const { data: shipment, isLoading } = useShipment(params.id);
  const { data: order } = useOrder(shipment?.order_id ?? "");
  const updateShipment = useUpdateShipment(params.id);
  const deleteShipment = useDeleteShipment();
  const transitionStatus = useTransitionShipmentStatus(params.id);
  const hasTracking = !!shipment?.tracking_number;
  const { data: trackingEvents, isLoading: trackingLoading } = useShipmentTracking(params.id, hasTracking);

  const handleUpdate = (data: { tracking_number?: string; label_url?: string }) => {
    updateShipment.mutate(
      {
        tracking_number: data.tracking_number || undefined,
        label_url: data.label_url || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t("shipmentUpdated"));
          setIsEditing(false);
        },
        onError: (error) => {
          toast.error(error.message || t("updateFailed"));
        },
      }
    );
  };

  const handleDelete = () => {
    deleteShipment.mutate(params.id, {
      onSuccess: () => {
        toast.success(t("shipmentDeleted"));
        router.push("/shipments");
      },
      onError: (error) => {
        toast.error(error.message || t("deleteFailed"));
      },
    });
  };

  const handleStatusTransition = (status: string) => {
    transitionStatus.mutate(
      { status },
      {
        onSuccess: () => {
          toast.success(t("statusChanged"));
        },
        onError: (error) => {
          toast.error(error.message || t("statusChangeFailed"));
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
        <h1 className="text-2xl font-bold">{t("notFound")}</h1>
        <Button asChild variant="outline">
          <Link href="/shipments">{t("detail.backToList")}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/shipments">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">
              {t("shipmentTitle", { id: shortId(shipment.id) })}
            </h1>
            <p className="text-muted-foreground">
              {t("createdOn", { date: formatDate(shipment.created_at) })}
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
              {t("generateLabel")} {SHIPMENT_PROVIDER_LABELS[shipment.provider] ?? shipment.provider.toUpperCase()}
            </Button>
          )}
          {shipment.label_url && (
            <ShipmentLabelDownload shipmentId={shipment.id} variant="outline" size="sm">
              <FileDown className="h-4 w-4" />
              {t("downloadLabel")}
            </ShipmentLabelDownload>
          )}
          {shipment.provider === "inpost" &&
           shipment.status === "label_ready" &&
           !shipment.carrier_data?.dispatch_order_id && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowDispatchDialog(true)}
            >
              <Truck className="h-4 w-4" />
              {t("orderCourier")}
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            <Pencil className="h-4 w-4" />
            {isEditing ? t("cancelEdit") : t("edit")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="h-4 w-4" />
            {t("delete")}
       </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("editShipment")}</CardTitle>
          </CardHeader>
          <CardContent>
            <ShipmentForm
              shipment={shipment}
              onSubmit={handleUpdate}
              isPending={updateShipment.isPending}
            />
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>{t("shipmentDetails")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="font-mono text-sm">{shipment.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("columns.order")}</p>
                <Link
                  href={`/orders/${shipment.order_id}`}
                  className="inline-flex items-center gap-1 font-mono text-sm text-primary hover:underline"
                >
                  {shortId(shipment.order_id)}
                  <ExternalLink className="h-3 w-3" />
                </Link>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("provider")}</p>
                <p className="text-sm">{SHIPMENT_PROVIDER_LABELS[shipment.provider] ?? shipment.provider.toUpperCase()}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <StatusBadge
                  status={shipment.status}
                  statusMap={SHIPMENT_STATUSES}
                  translationPrefix="shipment"
                />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("packageNumber")}</p>
                <p className="text-sm font-medium">#{shipment.package_number}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("columns.trackingNumber")}</p>
                <p className="text-sm">
                  {shipment.tracking_number || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("labelUrl")}</p>
                {shipment.label_url ? (
                  <ShipmentLabelDownload
                    shipmentId={shipment.id}
                    variant="link"
                    className="h-auto p-0 text-sm"
                  >
                    {t("openLabel")}
                    <ExternalLink className="h-3 w-3" />
                  </ShipmentLabelDownload>
                ) : (
                  <p className="text-sm">-</p>
                )}
              </div>
              {shipment.provider === "inpost" &&
                typeof shipment.carrier_data?.target_point === "string" && (
                  <div>
                    <p className="text-sm text-muted-foreground">
                      {t("targetParcelLocker")}
                    </p>
                    <div className="inline-flex items-center gap-1.5 rounded-md border bg-muted/50 px-2.5 py-1 mt-1">
                      <MapPin className="h-3.5 w-3.5 text-primary" />
                      <span className="text-sm font-medium">
                        {shipment.carrier_data.target_point}
                      </span>
                    </div>
                  </div>
                )}
              {typeof shipment.carrier_data?.sending_method === "string" && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("sendingMethod")}</p>
                  <p className="text-sm">{SENDING_METHOD_LABEL_KEYS[shipment.carrier_data.sending_method] ? t(SENDING_METHOD_LABEL_KEYS[shipment.carrier_data.sending_method]) : shipment.carrier_data.sending_method}</p>
                </div>
              )}
              {shipment.carrier_data?.dispatch_order_id != null && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("dispatchOrder")}</p>
                  <p className="font-mono text-sm">#{String(shipment.carrier_data.dispatch_order_id)}</p>
                </div>
              )}
              {shipment.weight != null && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("weight")}</p>
                  <p className="text-sm">{shipment.weight} kg</p>
                </div>
              )}
              {(shipment.length != null || shipment.width != null || shipment.height != null) && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("dimensions")}</p>
                  <p className="text-sm">
                    {shipment.length ?? "-"} x {shipment.width ?? "-"} x {shipment.height ?? "-"}
                  </p>
                </div>
              )}
              {shipment.carbon_kg != null && (
                <div className="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900 dark:bg-green-950/30">
                  <p className="text-sm font-medium text-green-700 dark:text-green-400 flex items-center gap-1.5">
                    <Leaf className="h-4 w-4" />
                    {t("carbonFootprint")}
                  </p>
                  <p className="text-lg font-bold text-green-800 dark:text-green-300 mt-1">
                    {shipment.carbon_kg.toFixed(3)} kg CO2
                  </p>
                  {shipment.distance_km != null && (
                    <p className="text-xs text-green-600 dark:text-green-500 mt-0.5">
                      {t("distance")}: ~{shipment.distance_km.toFixed(0)} km
                      {shipment.carbon_method === "estimate" && ` (${t("estimated")})`}
                    </p>
                  )}
                </div>
              )}
              {shipment.notes && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("notesLabel")}</p>
                  <p className="text-sm">{shipment.notes}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">{t("createdDate")}</p>
                <p className="text-sm">{formatDate(shipment.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("lastUpdate")}
                </p>
                <p className="text-sm">{formatDate(shipment.updated_at)}</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("changeStatus")}</CardTitle>
              <CardDescription>
                {t("availableStatusTransitions")}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ShipmentStatusActions
                currentStatus={shipment.status}
                onTransition={handleStatusTransition}
                isPending={transitionStatus.isPending}
              />
            </CardContent>
          </Card>
        </div>
      )}

      {/* Tracking timeline — shows when shipment has a tracking number */}
      {hasTracking && !isEditing && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Route className="h-5 w-5 text-muted-foreground" />
                <CardTitle>{t("shipmentTracking")}</CardTitle>
              </div>
              {shipment.tracking_number && (
                <span className="text-xs font-mono text-muted-foreground">
                  {shipment.tracking_number}
                </span>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {trackingLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-3/4" />
                <Skeleton className="h-8 w-1/2" />
              </div>
            ) : (
              <TrackingTimeline events={trackingEvents ?? []} />
            )}
          </CardContent>
        </Card>
      )}

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("deleteShipment")}</DialogTitle>
            <DialogDescription>
              {t("deleteShipmentConfirm")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              {t("cancelAction")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteShipment.isPending}
            >
              {deleteShipment.isPending ? t("deleting") : t("delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <GenerateLabelDialog
        shipmentId={params.id}
        provider={shipment.provider}
        order={order}
        shipment={shipment}
        open={showLabelDialog}
        onOpenChange={setShowLabelDialog}
      />

      <DispatchOrderDialog
        shipmentIds={[params.id]}
        open={showDispatchDialog}
        onOpenChange={setShowDispatchDialog}
      />
    </div>
  );
}
