"use client";

import { useState } from "react";
import Link from "next/link";
import { formatDateTime } from "@/lib/utils";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWebhookDeliveries } from "@/hooks/use-webhooks";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ArrowLeft } from "lucide-react";
import type { WebhookDelivery } from "@/types/api";
import { useTranslations } from "next-intl";

// EVENT_LABELS, EVENT_OPTIONS, STATUS_OPTIONS moved inside component to use translations

export default function WebhookDeliveriesPage() {
  const t = useTranslations("settings");
  const tw = useTranslations("settings.webhooks");

  const EVENT_LABELS: Record<string, string> = {
    "order.created": tw("eventOrderCreated"),
    "order.status_changed": tw("eventOrderStatusChanged"),
    "order.deleted": tw("eventOrderDeleted"),
    "product.created": tw("eventProductCreated"),
    "product.updated": tw("eventProductUpdated"),
    "product.deleted": tw("eventProductDeleted"),
    "shipment.created": tw("eventShipmentCreated"),
    "shipment.updated": tw("eventShipmentUpdated"),
  };

  const EVENT_OPTIONS = [
    { value: "__all__", label: tw("allEvents") },
    { value: "order.created", label: tw("eventOrderCreated") },
    { value: "order.status_changed", label: tw("eventOrderStatusChanged") },
    { value: "order.deleted", label: tw("eventOrderDeleted") },
    { value: "product.created", label: tw("eventProductCreated") },
    { value: "product.updated", label: tw("eventProductUpdated") },
    { value: "product.deleted", label: tw("eventProductDeleted") },
    { value: "shipment.created", label: tw("eventShipmentCreated") },
    { value: "shipment.updated", label: tw("eventShipmentUpdated") },
  ];

  const STATUS_OPTIONS = [
    { value: "__all__", label: tw("allStatuses") },
    { value: "success", label: tw("success") },
    { value: "failed", label: tw("error") },
  ];

  const [eventType, setEventType] = useState<string>("");
  const [status, setStatus] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading } = useWebhookDeliveries({
    limit,
    offset,
    event_type: eventType || undefined,
    status: status || undefined,
  });

  const handleEventTypeChange = (value: string) => {
    setEventType(value === "__all__" ? "" : value);
    setOffset(0);
  };

  const handleStatusChange = (value: string) => {
    setStatus(value === "__all__" ? "" : value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  const columns: ColumnDef<WebhookDelivery>[] = [
    {
      header: tw("time"),
      accessorKey: "created_at",
      cell: (row) => formatDateTime(row.created_at),
    },
    {
      header: "URL",
      accessorKey: "url",
      cell: (row) => (
        <span className="max-w-[200px] truncate block" title={row.url}>
          {row.url}
        </span>
      ),
    },
    {
      header: tw("event"),
      accessorKey: "event_type",
      cell: (row) => EVENT_LABELS[row.event_type] || row.event_type,
    },
    {
      header: t("webhookDeliveries.columns.status"),
      accessorKey: "status",
      cell: (row) => (
        <Badge
          variant={row.status === "success" ? "default" : "destructive"}
        >
          {row.status === "success" ? tw("success") : tw("error")}
        </Badge>
      ),
    },
    {
      header: tw("code"),
      accessorKey: "response_code",
      cell: (row) =>
        row.response_code ? String(row.response_code) : (
          <span className="text-muted-foreground">-</span>
        ),
    },
    {
      header: tw("error"),
      accessorKey: "error",
      cell: (row) =>
        row.error ? (
          <span className="text-destructive text-xs max-w-[200px] truncate block" title={row.error}>
            {row.error}
          </span>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
    },
  ];

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/settings/webhooks">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("powrot")}
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">{t("logDostarczenWebhookow")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("historiaDostarczenPowiadomienDoZewnetrznychSystemo")}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <Select
          value={eventType || "__all__"}
          onValueChange={handleEventTypeChange}
        >
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="Typ zdarzenia" />
          </SelectTrigger>
          <SelectContent>
            {EVENT_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={status || "__all__"}
          onValueChange={handleStatusChange}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            {STATUS_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-md border">
        <DataTable<WebhookDelivery>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyMessage={t("brakDostarczenWebhookow")}
          rowId={(row) => row.id}
        />
      </div>

      {data && (
        <DataTablePagination
          total={data.total}
          limit={limit}
          offset={offset}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      )}
    </div>
    </AdminGuard>
  );
}
