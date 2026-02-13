"use client";

import { Package, MapPin, Clock, CheckCircle2, Truck, AlertCircle } from "lucide-react";
import type { TrackingEvent } from "@/types/api";

const STATUS_CONFIG: Record<string, { icon: typeof Package; color: string; label: string }> = {
  created: { icon: Package, color: "text-blue-500", label: "Utworzona" },
  confirmed: { icon: CheckCircle2, color: "text-blue-500", label: "Potwierdzona" },
  offers_prepared: { icon: Package, color: "text-blue-500", label: "Oferty przygotowane" },
  offer_selected: { icon: Package, color: "text-blue-500", label: "Oferta wybrana" },
  dispatched_by_sender: { icon: Truck, color: "text-orange-500", label: "Nadana" },
  collected_from_sender: { icon: Truck, color: "text-orange-500", label: "Odebrana od nadawcy" },
  taken_by_courier: { icon: Truck, color: "text-orange-500", label: "Pobrana przez kuriera" },
  adopted_at_source_branch: { icon: MapPin, color: "text-orange-500", label: "W oddziale nadawczym" },
  sent_from_source_branch: { icon: Truck, color: "text-orange-500", label: "Wysłana z oddziału" },
  adopted_at_sorting_center: { icon: MapPin, color: "text-orange-500", label: "W sortowni" },
  sent_from_sorting_center: { icon: Truck, color: "text-orange-500", label: "Wysłana z sortowni" },
  adopted_at_target_branch: { icon: MapPin, color: "text-yellow-500", label: "W oddziale docelowym" },
  out_for_delivery: { icon: Truck, color: "text-yellow-500", label: "W doręczeniu" },
  ready_to_pickup: { icon: CheckCircle2, color: "text-green-500", label: "Gotowa do odbioru" },
  delivered: { icon: CheckCircle2, color: "text-green-600", label: "Doręczona" },
  pickup_reminder_sent: { icon: Clock, color: "text-yellow-500", label: "Przypomnienie o odbiorze" },
  undelivered: { icon: AlertCircle, color: "text-red-500", label: "Niedoręczona" },
  returned_to_sender: { icon: AlertCircle, color: "text-red-500", label: "Zwrócona do nadawcy" },
  canceled: { icon: AlertCircle, color: "text-red-500", label: "Anulowana" },
};

function getStatusConfig(status: string) {
  return STATUS_CONFIG[status] ?? {
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
  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Package className="h-8 w-8 text-muted-foreground/50 mb-2" />
        <p className="text-sm text-muted-foreground">Brak danych śledzenia. Informacje pojawią się po nadaniu przesyłki.</p>
      </div>
    );
  }

  return (
    <div className="relative space-y-0">
      {events.map((event, index) => {
        const config = getStatusConfig(event.status);
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
