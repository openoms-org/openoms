"use client";

import { useState } from "react";
import { format } from "date-fns";
import { pl } from "date-fns/locale";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSyncJobs } from "@/hooks/use-sync-jobs";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { shortId } from "@/lib/utils";
import type { SyncJob } from "@/types/api";

const STATUS_OPTIONS = [
  { value: "__all__", label: "Wszystkie" },
  { value: "running", label: "W trakcie" },
  { value: "completed", label: "Zakończone" },
  { value: "failed", label: "Nieudane" },
];

const JOB_TYPE_OPTIONS = [
  { value: "__all__", label: "Wszystkie" },
  { value: "orders", label: "Zamówienia" },
  { value: "products", label: "Produkty" },
  { value: "stock", label: "Stany magazynowe" },
  { value: "prices", label: "Ceny" },
];

function statusBadge(status: string) {
  switch (status) {
    case "running":
      return <Badge variant="info">W trakcie</Badge>;
    case "completed":
      return <Badge variant="success">Zakończone</Badge>;
    case "failed":
      return <Badge variant="destructive">Nieudane</Badge>;
    default:
      return <Badge variant="secondary">{status}</Badge>;
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return <span className="text-muted-foreground">&mdash;</span>;
  return format(new Date(dateStr), "dd.MM.yyyy HH:mm:ss", { locale: pl });
}

export default function SyncJobsPage() {
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [jobTypeFilter, setJobTypeFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [selectedJob, setSelectedJob] = useState<SyncJob | null>(null);

  const { data, isLoading, isError, refetch } = useSyncJobs({
    limit,
    offset,
    status: statusFilter || undefined,
    job_type: jobTypeFilter || undefined,
  });

  const handleStatusChange = (value: string) => {
    setStatusFilter(value === "__all__" ? "" : value);
    setOffset(0);
  };

  const handleJobTypeChange = (value: string) => {
    setJobTypeFilter(value === "__all__" ? "" : value);
    setOffset(0);
  };

  const handlePageSizeChange = (newLimit: number) => {
    setLimit(newLimit);
    setOffset(0);
  };

  const handlePageChange = (newOffset: number) => {
    setOffset(newOffset);
  };

  const columns: ColumnDef<SyncJob>[] = [
    {
      header: "Typ zadania",
      accessorKey: "job_type",
      cell: (row) => row.job_type,
    },
    {
      header: "Integracja",
      accessorKey: "integration_id",
      cell: (row) => (
        <span className="font-mono text-xs">{shortId(row.integration_id)}</span>
      ),
    },
    {
      header: "Status",
      accessorKey: "status",
      cell: (row) => statusBadge(row.status),
    },
    {
      header: "Rozpoczęto",
      accessorKey: "started_at",
      cell: (row) => formatDate(row.started_at),
    },
    {
      header: "Zakończono",
      accessorKey: "finished_at",
      cell: (row) => formatDate(row.finished_at),
    },
    {
      header: "Przetworzone",
      accessorKey: "items_processed",
      cell: (row) => row.items_processed,
    },
    {
      header: "Błędy",
      accessorKey: "items_failed",
      cell: (row) =>
        row.items_failed > 0 ? (
          <span className="text-destructive font-medium">{row.items_failed}</span>
        ) : (
          row.items_failed
        ),
    },
    {
      header: "Komunikat",
      accessorKey: "error_message",
      cell: (row) =>
        row.error_message ? (
          <span className="text-destructive text-xs truncate max-w-[200px] block">
            {row.error_message}
          </span>
        ) : (
          <span className="text-muted-foreground">&mdash;</span>
        ),
    },
  ];

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Historia synchronizacji</h1>
        <p className="text-muted-foreground mt-1">
          Przegląd wszystkich zadań synchronizacji z zewnętrznymi integracjami
        </p>
      </div>

      <div className="flex items-center gap-4">
        <Select
          value={statusFilter || "__all__"}
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

        <Select
          value={jobTypeFilter || "__all__"}
          onValueChange={handleJobTypeChange}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Typ zadania" />
          </SelectTrigger>
          <SelectContent>
            {JOB_TYPE_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
        <DataTable<SyncJob>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyMessage="Brak zadań synchronizacji"
          rowId={(row) => row.id}
          onRowClick={(row) => setSelectedJob(row)}
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

      <Dialog open={!!selectedJob} onOpenChange={(open) => !open && setSelectedJob(null)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Szczegóły synchronizacji</DialogTitle>
            <DialogDescription>
              ID: {selectedJob?.id}
            </DialogDescription>
          </DialogHeader>
          {selectedJob && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="font-medium text-muted-foreground">Typ zadania:</span>
                  <p>{selectedJob.job_type}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Status:</span>
                  <p className="mt-1">{statusBadge(selectedJob.status)}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Integracja:</span>
                  <p className="font-mono text-xs">{selectedJob.integration_id}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Utworzono:</span>
                  <p>{formatDate(selectedJob.created_at)}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Rozpoczęto:</span>
                  <p>{formatDate(selectedJob.started_at)}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Zakończono:</span>
                  <p>{formatDate(selectedJob.finished_at)}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Przetworzone:</span>
                  <p>{selectedJob.items_processed}</p>
                </div>
                <div>
                  <span className="font-medium text-muted-foreground">Błędy:</span>
                  <p>{selectedJob.items_failed}</p>
                </div>
              </div>
              {selectedJob.error_message && (
                <div>
                  <span className="font-medium text-muted-foreground text-sm">Komunikat błędu:</span>
                  <p className="text-destructive text-sm mt-1">{selectedJob.error_message}</p>
                </div>
              )}
              <div>
                <span className="font-medium text-muted-foreground text-sm">Metadane (JSON):</span>
                <pre className="mt-1 rounded-md bg-muted p-3 text-xs overflow-auto max-h-64">
                  {JSON.stringify(selectedJob.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
    </AdminGuard>
  );
}
