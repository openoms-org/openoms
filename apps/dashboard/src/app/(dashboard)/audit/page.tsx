"use client";

import { useState } from "react";
import { format } from "date-fns";
import { pl } from "date-fns/locale";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useAuditLog } from "@/hooks/use-audit";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { shortId } from "@/lib/utils";
import type { AuditLogEntry } from "@/types/api";

const ACTION_LABELS: Record<string, string> = {
  "order.created": "Utworzono zamówienie",
  "order.updated": "Zaktualizowano zamówienie",
  "order.deleted": "Usunięto zamówienie",
  "order.status_changed": "Zmieniono status",
  "product.created": "Utworzono produkt",
  "product.updated": "Zaktualizowano produkt",
  "product.deleted": "Usunięto produkt",
  "user.created": "Utworzono użytkownika",
  "user.updated": "Zaktualizowano użytkownika",
  "user.deleted": "Usunięto użytkownika",
  "shipment.created": "Utworzono przesyłkę",
  "shipment.updated": "Zaktualizowano przesyłkę",
  "auth.login": "Logowanie",
  "auth.logout": "Wylogowanie",
};

const ENTITY_TYPE_LABELS: Record<string, string> = {
  order: "zamówienie",
  product: "produkt",
  user: "użytkownik",
  shipment: "przesyłka",
  integration: "integracja",
};

const ENTITY_TYPE_OPTIONS = [
  { value: "__all__", label: "Wszystkie" },
  { value: "order", label: "Zamówienia" },
  { value: "product", label: "Produkty" },
  { value: "user", label: "Użytkownicy" },
  { value: "shipment", label: "Przesyłki" },
  { value: "integration", label: "Integracje" },
];

export default function AuditPage() {
  const [entityType, setEntityType] = useState<string>("");
  const [actionFilter, setActionFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useAuditLog({
    limit,
    offset,
    entity_type: entityType || undefined,
    action: actionFilter || undefined,
  });

  const handleEntityTypeChange = (value: string) => {
    setEntityType(value === "__all__" ? "" : value);
    setOffset(0);
  };

  const handleActionFilterChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setActionFilter(e.target.value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  const columns: ColumnDef<AuditLogEntry>[] = [
    {
      header: "Czas",
      accessorKey: "created_at",
      cell: (row) =>
        format(new Date(row.created_at), "dd.MM.yyyy HH:mm", { locale: pl }),
    },
    {
      header: "Użytkownik",
      accessorKey: "user_name",
      cell: (row) => row.user_name || "System",
    },
    {
      header: "Akcja",
      accessorKey: "action",
      cell: (row) => ACTION_LABELS[row.action] || row.action,
    },
    {
      header: "Typ",
      accessorKey: "entity_type",
      cell: (row) => (
        <Badge variant="secondary">
          {ENTITY_TYPE_LABELS[row.entity_type] || row.entity_type}
        </Badge>
      ),
    },
    {
      header: "ID encji",
      accessorKey: "entity_id",
      cell: (row) => (
        <span className="font-mono text-xs">{shortId(row.entity_id)}</span>
      ),
    },
    {
      header: "Szczegóły",
      accessorKey: "changes",
      cell: (row) => {
        if (!row.changes || Object.keys(row.changes).length === 0) {
          return <span className="text-muted-foreground">—</span>;
        }
        return (
          <div className="max-w-[300px] space-y-0.5">
            {Object.entries(row.changes).map(([k, v]) => (
              <div key={k} className="flex items-baseline gap-1 text-xs">
                <span className="font-medium text-muted-foreground shrink-0">{k}:</span>
                <span className="truncate text-foreground">{String(v)}</span>
              </div>
            ))}
          </div>
        );
      },
    },
  ];

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dziennik aktywności</h1>
        <p className="text-muted-foreground mt-1">
          Historia zmian i działań w systemie
        </p>
      </div>

      <div className="flex items-center gap-4">
        <Select
          value={entityType || "__all__"}
          onValueChange={handleEntityTypeChange}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Typ encji" />
          </SelectTrigger>
          <SelectContent>
            {ENTITY_TYPE_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          placeholder="Filtruj po akcji..."
          value={actionFilter}
          onChange={handleActionFilterChange}
          className="max-w-xs"
        />
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      <div className="rounded-md border">
        <DataTable<AuditLogEntry>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyMessage="Brak wpisów w dzienniku"
          rowId={(row) => String(row.id)}
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
