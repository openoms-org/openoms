"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useInvoices } from "@/hooks/use-invoices";
import { useBulkSendToKSeF } from "@/hooks/use-ksef";
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
import {
  INVOICE_STATUS_MAP,
  INVOICING_PROVIDER_LABELS,
  KSEF_STATUS_MAP,
} from "@/lib/constants";
import { getErrorMessage } from "@/lib/api-client";
import { Send, Loader2 } from "lucide-react";
import type { Invoice } from "@/types/api";

export default function InvoicesPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [ksefFilter, setKsefFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useInvoices({
    status: statusFilter || undefined,
    provider: providerFilter || undefined,
    limit,
    offset,
  });

  const bulkSend = useBulkSendToKSeF();

  const handleStatusChange = (value: string) => {
    setStatusFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handleProviderChange = (value: string) => {
    setProviderFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handleKSeFFilterChange = (value: string) => {
    setKsefFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  // Filter by KSeF status client-side (since it's a new column)
  const filteredItems = (data?.items || []).filter((inv) => {
    if (!ksefFilter) return true;
    return inv.ksef_status === ksefFilter;
  });

  // Get IDs of invoices eligible for KSeF sending
  const eligibleForKSeF = filteredItems.filter(
    (inv) =>
      inv.ksef_status === "not_sent" &&
      inv.status !== "cancelled" &&
      inv.status !== "error" &&
      inv.status !== "draft"
  );

  const handleBulkSendToKSeF = async () => {
    if (eligibleForKSeF.length === 0) return;
    try {
      const result = await bulkSend.mutateAsync(
        eligibleForKSeF.map((inv) => inv.id)
      );
      toast.success(
        `Wysłano ${result.sent} z ${result.total} faktur do KSeF`
      );
      if (result.errors && result.errors.length > 0) {
        toast.error(`Błędy: ${result.errors.join(", ")}`);
      }
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
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
        {eligibleForKSeF.length > 0 && (
          <Button
            variant="outline"
            onClick={handleBulkSendToKSeF}
            disabled={bulkSend.isPending}
          >
            {bulkSend.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Send className="mr-2 h-4 w-4" />
            )}
            Wyślij do KSeF ({eligibleForKSeF.length})
          </Button>
        )}
      </div>

      <div className="flex items-center gap-4 flex-wrap">
        <div className="w-[200px]">
          <Select
            value={statusFilter || "all"}
            onValueChange={handleStatusChange}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie statusy</SelectItem>
              {Object.entries(INVOICE_STATUS_MAP).map(([key, { label }]) => (
                <SelectItem key={key} value={key}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-[200px]">
          <Select
            value={providerFilter || "all"}
            onValueChange={handleProviderChange}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Dostawca" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszyscy dostawcy</SelectItem>
              {Object.entries(INVOICING_PROVIDER_LABELS).map(([key, label]) => (
                <SelectItem key={key} value={key}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-[200px]">
          <Select
            value={ksefFilter || "all"}
            onValueChange={handleKSeFFilterChange}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status KSeF" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie (KSeF)</SelectItem>
              {Object.entries(KSEF_STATUS_MAP).map(([key, { label }]) => (
                <SelectItem key={key} value={key}>
                  {label}
                </SelectItem>
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
          data={filteredItems}
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
