"use client";

import { Package, MapPin, Clock, CheckCircle2, Truck, AlertCircle } from "lucide-react";
import type { TrackingEvent } from "@/types/api";
import { useTranslations } from "next-intl";

const STATUS_ICON_CONFIG: Record<string, { icon: typeof Package; color: string; labelKey: string }> = {
  created: { icon: Package, color: "text-blue-500", labelKey: "tracking.created" },
  confirmed: { icon: CheckCircle2, color: "text-blue-500", labelKey: "tracking.confirmed" },
  offers_prepared: { icon: Package, color: "text-blue-500", labelKey: "tracking.offersPrepared" },
  offer_selected: { icon: Package, color: "text-blue-500", labelKey: "tracking.offerSelected" },
  dispatched_by_sender: { icon: Truck, color: "text-orange-500", labelKey: "tracking.dispatchedBySender" },
  collected_from_sender: { icon: Truck, color: "text-orange-500", labelKey: "tracking.collectedFromSender" },
  taken_by_courier: { icon: Truck, color: "text-orange-500", labelKey: "tracking.takenByCourier" },
  adopted_at_source_branch: { icon: MapPin, color: "text-orange-500", labelKey: "tracking.adoptedAtSourceBranch" },
  sent_from_source_branch: { icon: Truck, color: "text-orange-500", labelKey: "tracking.sentFromSourceBranch" },
  adopted_at_sorting_center: { icon: MapPin, color: "text-orange-500", labelKey: "tracking.adoptedAtSortingCenter" },
  sent_from_sorting_center: { icon: Truck, color: "text-orange-500", labelKey: "tracking.sentFromSortingCenter" },
  adopted_at_target_branch: { icon: MapPin, color: "text-yellow-500", labelKey: "tracking.adoptedAtTargetBranch" },
  out_for_delivery: { icon: Truck, color: "text-yellow-500", labelKey: "tracking.outForDelivery" },
  ready_to_pickup: { icon: CheckCircle2, color: "text-green-500", labelKey: "tracking.readyToPickup" },
  delivered: { icon: CheckCircle2, color: "text-green-600", labelKey: "tracking.delivered" },
  pickup_reminder_sent: { icon: Clock, color: "text-yellow-500", labelKey: "tracking.pickupReminderSent" },
  undelivered: { icon: AlertCircle, color: "text-red-500", labelKey: "tracking.undelivered" },
  returned_to_sender: { icon: AlertCircle, color: "text-red-500", labelKey: "tracking.returnedToSender" },
  canceled: { icon: AlertCircle, color: "text-red-500", labelKey: "tracking.canceled" },
};

function getStatusConfig(status: string, t: (key: string) => string) {
  const config = STATUS_ICON_CONFIG[status];
  if (config) {
    return { icon: config.icon, color: config.color, label: t(config.labelKey) };
  }
  return {
    icon: Package,
    color: "text-muted-foreground",
    label: status.replace(/_/g, " "),
  };
}

function formatTimestamp(ts: string) {
  try {
    const date = new Date(ts);
    return date.toLocaleString("pl-PL", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return ts;
  }
}

interface TrackingTimelineProps {
  events: TrackingEvent[];
}

export function TrackingTimeline({ events }: TrackingTimelineProps) {
  const t = useTranslations("shipments");
  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Package className="h-8 w-8 text-muted-foreground/50 mb-2" />
        <p className="text-sm text-muted-foreground">{t("brakDanychSledzeniaInformacjePojawiaSiePo")}</p>
      </div>
    );
  }

  return (
    <div className="relative space-y-0">
      {events.map((event, index) => {
        const config = getStatusConfig(event.status, t);
        const Icon = config.icon;
        const isFirst = index === 0;

        return (
          <div key={`${event.status}-${event.timestamp}-${index}`} className="relative flex gap-4 pb-6 last:pb-0">
            {/* Vertical line */}
            {index < events.length - 1 && (
              <div className="absolute left-[15px] top-[30px] bottom-0 w-px bg-border" />
            )}

            {/* Icon */}
            <div className={`relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 ${
              isFirst ? "border-primary bg-primary/10" : "border-border bg-background"
            }`}>
              <Icon className={`h-4 w-4 ${isFirst ? "text-primary" : config.color}`} />
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0 pt-0.5">
              <p className={`text-sm font-medium ${isFirst ? "text-foreground" : "text-muted-foreground"}`}>
                {config.label}
              </p>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-0.5">
                <span className="text-xs text-muted-foreground">
                  {formatTimestamp(event.timestamp)}
                </span>
                {event.location && (
                  <span className="text-xs text-muted-foreground">
                    {event.location}
                  </span>
                )}
              </div>
              {event.details && event.details !== event.status && (
                <p className="text-xs text-muted-foreground mt-0.5">{event.details}</p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
