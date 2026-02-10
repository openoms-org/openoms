"use client";

import { useState } from "react";
import Link from "next/link";
import { format } from "date-fns";
import { pl } from "date-fns/locale";
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

const EVENT_LABELS: Record<string, string> = {
  "order.created": "Zamówienie utworzone",
  "order.status_changed": "Status zamówienia zmieniony",
  "order.deleted": "Zamówienie usunięte",
  "product.created": "Produkt utworzony",
  "product.updated": "Produkt zaktualizowany",
  "product.deleted": "Produkt usunięty",
  "shipment.created": "Przesyłka utworzona",
  "shipment.updated": "Przesyłka zaktualizowana",
};

const EVENT_OPTIONS = [
  { value: "__all__", label: "Wszystkie zdarzenia" },
  { value: "order.created", label: "Zamówienie utworzone" },
  { value: "order.status_changed", label: "Status zmieniony" },
  { value: "order.deleted", label: "Zamówienie usunięte" },
  { value: "product.created", label: "Produkt utworzony" },
  { value: "product.updated", label: "Produkt zaktualizowany" },
  { value: "product.deleted", label: "Produkt usunięty" },
  { value: "shipment.created", label: "Przesyłka utworzona" },
  { value: "shipment.updated", label: "Przesyłka zaktualizowana" },
];

const STATUS_OPTIONS = [
  { value: "__all__", label: "Wszystkie statusy" },
  { value: "success", label: "Sukces" },
  { value: "failed", label: "Błąd" },
];

export default function WebhookDeliveriesPage() {
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
      header: "Czas",
      accessorKey: "created_at",
      cell: (row) =>
        format(new Date(row.created_at), "dd.MM.yyyy HH:mm:ss", { locale: pl }),
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
      header: "Zdarzenie",
      accessorKey: "event_type",
      cell: (row) => EVENT_LABELS[row.event_type] || row.event_type,
    },
    {
      header: "Status",
      accessorKey: "status",
      cell: (row) => (
        <Badge
          variant={row.status === "success" ? "default" : "destructive"}
        >
          {row.status === "success" ? "Sukces" : "Błąd"}
        </Badge>
      ),
    },
    {
      header: "Kod",
      accessorKey: "response_code",
      cell: (row) =>
        row.response_code ? String(row.response_code) : (
          <span className="text-muted-foreground">-</span>
        ),
    },
    {
      header: "Błąd",
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
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/settings/webhooks">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Powrót
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Log dostarczeń webhooków</h1>
          <p className="text-muted-foreground mt-1">
            Historia dostarczeń powiadomień do zewnętrznych systemów
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
          emptyMessage="Brak dostarczeń webhooków"
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
  );
}
