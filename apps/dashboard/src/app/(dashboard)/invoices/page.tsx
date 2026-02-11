"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useInvoices } from "@/hooks/use-invoices";
import { InvoiceTable } from "@/components/invoices/invoice-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { INVOICE_STATUS_MAP, INVOICING_PROVIDER_LABELS } from "@/lib/constants";
import type { Invoice } from "@/types/api";

export default function InvoicesPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useInvoices({
    status: statusFilter || undefined,
    provider: providerFilter || undefined,
    limit,
    offset,
  });

  const handleStatusChange = (value: string) => {
    setStatusFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handleProviderChange = (value: string) => {
    setProviderFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Faktury</h1>
          <p className="text-muted-foreground mt-1">
            Lista wystawionych faktur
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <div className="w-[200px]">
          <Select value={statusFilter || "all"} onValueChange={handleStatusChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie statusy</SelectItem>
              {Object.entries(INVOICE_STATUS_MAP).map(([key, { label }]) => (
                <SelectItem key={key} value={key}>{label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-[200px]">
          <Select value={providerFilter || "all"} onValueChange={handleProviderChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Dostawca" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszyscy dostawcy</SelectItem>
              {Object.entries(INVOICING_PROVIDER_LABELS).map(([key, label]) => (
                <SelectItem key={key} value={key}>{label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
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
        <InvoiceTable
          data={data?.items || []}
          isLoading={isLoading}
          onRowClick={(row: Invoice) => router.push(`/invoices/${row.id}`)}
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
