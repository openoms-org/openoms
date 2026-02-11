"use client";

import Link from "next/link";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { StatusBadge } from "@/components/shared/status-badge";
import { INVOICE_STATUS_MAP, INVOICE_TYPE_LABELS, INVOICING_PROVIDER_LABELS } from "@/lib/constants";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import type { Invoice } from "@/types/api";
import { EmptyState } from "@/components/shared/empty-state";
import { FileText } from "lucide-react";

interface InvoiceTableProps {
  data: Invoice[];
  isLoading: boolean;
  onRowClick?: (row: Invoice) => void;
}

export const invoiceColumns: ColumnDef<Invoice>[] = [
  {
    header: "Numer",
    accessorKey: "external_number",
    cell: (row) => (
      <span className="font-mono text-sm">
        {row.external_number || shortId(row.id)}
      </span>
    ),
  },
  {
    header: "Zamówienie",
    accessorKey: "order_id",
    cell: (row) => (
      <Link
        href={`/orders/${row.order_id}`}
        className="font-mono text-xs text-primary hover:underline"
        onClick={(e) => e.stopPropagation()}
      >
        {shortId(row.order_id)}
      </Link>
    ),
  },
  {
    header: "Status",
    accessorKey: "status",
    cell: (row) => <StatusBadge status={row.status} statusMap={INVOICE_STATUS_MAP} />,
  },
  {
    header: "Typ",
    accessorKey: "invoice_type",
    cell: (row) => (
      <span className="text-sm">
        {INVOICE_TYPE_LABELS[row.invoice_type] || row.invoice_type}
      </span>
    ),
  },
  {
    header: "Kwota brutto",
    accessorKey: "total_gross",
    cell: (row) => formatCurrency(row.total_gross, row.currency),
  },
  {
    header: "Data wystawienia",
    accessorKey: "issue_date",
    cell: (row) => (row.issue_date ? formatDate(row.issue_date) : "-"),
  },
  {
    header: "Dostawca",
    accessorKey: "provider",
    cell: (row) => (
      <span className="text-sm">
        {INVOICING_PROVIDER_LABELS[row.provider] || row.provider}
      </span>
    ),
  },
];

export function InvoiceTable({ data, isLoading, onRowClick }: InvoiceTableProps) {
  return (
    <DataTable<Invoice>
      columns={invoiceColumns}
      data={data}
      isLoading={isLoading}
      emptyState={
        <EmptyState
          icon={FileText}
          title="Brak faktur"
          description="Nie znaleziono faktur do wyświetlenia."
        />
      }
      onRowClick={onRowClick}
    />
  );
}
