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
import { useTranslations } from "next-intl";

const ACTION_KEYS = [
  "order.created", "order.updated", "order.deleted", "order.status_changed",
  "product.created", "product.updated", "product.deleted",
  "user.created", "user.updated", "user.deleted",
  "shipment.created", "shipment.updated", "shipment.deleted", "shipment.status_changed",
  "integration.created", "integration.updated", "integration.deleted",
  "settings.updated",
  "warehouse.created", "warehouse.updated", "warehouse.deleted",
  "supplier.created", "supplier.updated", "supplier.deleted",
  "return.created", "return.updated", "return.status_changed",
  "role.created", "role.updated", "role.deleted",
  "automation_rule.created", "automation_rule.updated", "automation_rule.deleted",
  "exchange_rate.updated",
  "customer.created", "customer.updated", "customer.deleted",
  "invoice.created", "invoice.updated", "invoice.deleted",
  "variant.created", "variant.updated", "variant.deleted",
  "price_list.created", "price_list.updated", "price_list.deleted",
  "warehouse_document.created", "warehouse_document.updated",
  "stocktake.created", "stocktake.updated",
  "auth.login", "auth.logout",
] as const;

const ENTITY_TYPE_KEYS = [
  "order", "product", "user", "shipment", "integration", "settings",
  "warehouse", "supplier", "return", "role", "automation_rule",
  "exchange_rate", "customer", "invoice", "variant", "price_list",
  "warehouse_document", "stocktake",
] as const;

export default function AuditPage() {
  const t = useTranslations("audit");
  const tc = useTranslations("common");
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

  const getActionLabel = (action: string): string => {
    if (ACTION_KEYS.includes(action as (typeof ACTION_KEYS)[number])) {
      return t(`actions.${action}` as Parameters<typeof t>[0]);
    }
    return action;
  };

  const getEntityTypeLabel = (type: string): string => {
    if (ENTITY_TYPE_KEYS.includes(type as (typeof ENTITY_TYPE_KEYS)[number])) {
      return t(`entityTypes.${type}` as Parameters<typeof t>[0]);
    }
    return type;
  };

  const columns: ColumnDef<AuditLogEntry>[] = [
    {
      header: t("time"),
      accessorKey: "created_at",
      cell: (row) =>
        format(new Date(row.created_at), "dd.MM.yyyy HH:mm", { locale: pl }),
    },
    {
      header: t("user"),
      accessorKey: "user_name",
      cell: (row) => row.user_name || "System",
    },
    {
      header: t("action"),
      accessorKey: "action",
      cell: (row) => getActionLabel(row.action),
    },
    {
      header: t("type"),
      accessorKey: "entity_type",
      cell: (row) => (
        <Badge variant="secondary">
          {getEntityTypeLabel(row.entity_type)}
        </Badge>
      ),
    },
    {
      header: t("entityId"),
      accessorKey: "entity_id",
      cell: (row) => (
        <span className="font-mono text-xs">{shortId(row.entity_id)}</span>
      ),
    },
    {
      header: tc("details"),
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
        <h1 className="text-2xl font-bold">{t("auditLog")}</h1>
        <p className="text-muted-foreground mt-1">
          {t("subtitle")}
        </p>
      </div>

      <div className="flex items-center gap-4">
        <Select
          value={entityType || "__all__"}
          onValueChange={handleEntityTypeChange}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder={t("entityTypePlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t("entityTypeOptions.all")}</SelectItem>
            {ENTITY_TYPE_KEYS.map((key) => (
              <SelectItem key={key} value={key}>
                {t(`entityTypeOptions.${key}` as Parameters<typeof t>[0])}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          placeholder={t("filterByAction")}
          value={actionFilter}
          onChange={handleActionFilterChange}
          className="max-w-xs"
        />
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("retry")}
          </Button>
        </div>
      )}

      <div className="rounded-md border">
        <DataTable<AuditLogEntry>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyMessage={t("emptyLog")}
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
