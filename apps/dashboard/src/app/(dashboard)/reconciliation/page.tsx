"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  useReconciliationSummary,
  useSettlements,
  useImportCSV,
} from "@/hooks/use-reconciliation";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getErrorMessage } from "@/lib/api-client";
import { Upload, Loader2, CreditCard, CheckCircle, AlertTriangle, XCircle } from "lucide-react";
import type { PaymentSettlement } from "@/types/api";

const PROVIDER_LABELS: Record<string, string> = {
  payu: "PayU",
  przelewy24: "Przelewy24",
  tpay: "Tpay",
  stripe: "Stripe",
  manual: "Reczne",
};

const STATUS_LABELS: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  pending: { label: "Oczekujace", variant: "secondary" },
  matched: { label: "Uzgodnione", variant: "default" },
  partial_match: { label: "Czesciowe", variant: "outline" },
  unmatched: { label: "Nieuzgodnione", variant: "destructive" },
};

function formatCurrency(amount: number, currency: string = "PLN"): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount);
}

export default function ReconciliationPage() {
  const router = useRouter();
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [dateFrom, setDateFrom] = useState<string>("");
  const [dateTo, setDateTo] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  // Import dialog state
  const [importOpen, setImportOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importProvider, setImportProvider] = useState<string>("payu");

  const { data: summary } = useReconciliationSummary(
    dateFrom || undefined,
    dateTo || undefined
  );

  const { data: settlements, isLoading, isError, refetch } = useSettlements({
    limit,
    offset,
    provider: providerFilter || undefined,
    status: statusFilter || undefined,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  });

  const importCSV = useImportCSV();

  const handleImport = async () => {
    if (!importFile) return;
    try {
      const result = await importCSV.mutateAsync({
        file: importFile,
        provider: importProvider,
      });
      toast.success(
        `Zaimportowano rozliczenie: ${result.provider} z ${result.settlement_date}`
      );
      setImportOpen(false);
      setImportFile(null);
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleProviderChange = (value: string) => {
    setProviderFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handleStatusChange = (value: string) => {
    setStatusFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Rozliczenia platnosci</h1>
          <p className="text-muted-foreground mt-1">
            Uzgadnianie transakcji z bramek platniczych z zamowieniami
          </p>
        </div>
        <Dialog open={importOpen} onOpenChange={setImportOpen}>
          <DialogTrigger asChild>
            <Button>
              <Upload className="mr-2 h-4 w-4" />
              Importuj rozliczenie
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Importuj rozliczenie CSV</DialogTitle>
              <DialogDescription>
                Wybierz dostawce platnosci i plik CSV z rozliczeniem.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="import-provider">Dostawca platnosci</Label>
                <Select
                  value={importProvider}
                  onValueChange={setImportProvider}
                >
                  <SelectTrigger id="import-provider">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(PROVIDER_LABELS).map(([key, label]) => (
                      <SelectItem key={key} value={key}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="import-file">Plik CSV</Label>
                <Input
                  id="import-file"
                  type="file"
                  accept=".csv"
                  onChange={(e) =>
                    setImportFile(e.target.files?.[0] || null)
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Maks. 10 MB. PayU/Przelewy24: separator &quot;;&quot;
                </p>
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setImportOpen(false)}
              >
                Anuluj
              </Button>
              <Button
                onClick={handleImport}
                disabled={!importFile || importCSV.isPending}
              >
                {importCSV.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Importuj
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Uzgodnione</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{summary.matched_count + summary.manual_match_count}</div>
              <p className="text-xs text-muted-foreground">
                {formatCurrency(summary.matched_amount)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Nieuzgodnione</CardTitle>
              <XCircle className="h-4 w-4 text-red-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{summary.unmatched_count}</div>
              <p className="text-xs text-muted-foreground">
                {formatCurrency(summary.unmatched_amount)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Rozbieznosci</CardTitle>
              <AlertTriangle className="h-4 w-4 text-yellow-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{summary.discrepancy_count}</div>
              <p className="text-xs text-muted-foreground">
                Roznice kwotowe
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Prowizje</CardTitle>
              <CreditCard className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatCurrency(summary.total_fees)}
              </div>
              <p className="text-xs text-muted-foreground">
                Suma prowizji bramek
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Filters */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="w-[180px]">
          <Select
            value={providerFilter || "all"}
            onValueChange={handleProviderChange}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Dostawca" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszyscy dostawcy</SelectItem>
              {Object.entries(PROVIDER_LABELS).map(([key, label]) => (
                <SelectItem key={key} value={key}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-[180px]">
          <Select
            value={statusFilter || "all"}
            onValueChange={handleStatusChange}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie statusy</SelectItem>
              {Object.entries(STATUS_LABELS).map(([key, { label }]) => (
                <SelectItem key={key} value={key}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2">
          <Label className="text-sm whitespace-nowrap">Od:</Label>
          <Input
            type="date"
            value={dateFrom}
            onChange={(e) => {
              setDateFrom(e.target.value);
              setOffset(0);
            }}
            className="w-[160px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <Label className="text-sm whitespace-nowrap">Do:</Label>
          <Input
            type="date"
            value={dateTo}
            onChange={(e) => {
              setDateTo(e.target.value);
              setOffset(0);
            }}
            className="w-[160px]"
          />
        </div>
      </div>

      {/* Error state */}
      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {/* Settlements table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Dostawca</TableHead>
              <TableHead>Data rozliczenia</TableHead>
              <TableHead>ID rozliczenia</TableHead>
              <TableHead className="text-right">Kwota brutto</TableHead>
              <TableHead className="text-right">Prowizje</TableHead>
              <TableHead className="text-right">Kwota netto</TableHead>
              <TableHead>Waluta</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Zaimportowano</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <TableRow>
                <TableCell colSpan={9} className="text-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin mx-auto" />
                </TableCell>
              </TableRow>
            )}
            {!isLoading && (!settlements?.items || settlements.items.length === 0) && (
              <TableRow>
                <TableCell colSpan={9} className="text-center py-8 text-muted-foreground">
                  Brak rozliczen. Zaimportuj pierwsze rozliczenie CSV.
                </TableCell>
              </TableRow>
            )}
            {settlements?.items?.map((settlement: PaymentSettlement) => {
              const statusInfo = STATUS_LABELS[settlement.status] || {
                label: settlement.status,
                variant: "secondary" as const,
              };
              return (
                <TableRow
                  key={settlement.id}
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() =>
                    router.push(`/reconciliation/${settlement.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    {PROVIDER_LABELS[settlement.provider] || settlement.provider}
                  </TableCell>
                  <TableCell>{settlement.settlement_date}</TableCell>
                  <TableCell className="text-muted-foreground text-xs font-mono">
                    {settlement.settlement_id || "-"}
                  </TableCell>
                  <TableCell className="text-right">
                    {formatCurrency(settlement.total_amount, settlement.currency)}
                  </TableCell>
                  <TableCell className="text-right text-muted-foreground">
                    {formatCurrency(settlement.fee_amount, settlement.currency)}
                  </TableCell>
                  <TableCell className="text-right font-medium">
                    {formatCurrency(settlement.net_amount, settlement.currency)}
                  </TableCell>
                  <TableCell>{settlement.currency}</TableCell>
                  <TableCell>
                    <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(settlement.imported_at).toLocaleDateString("pl-PL")}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {settlements && (
        <DataTablePagination
          total={settlements.total}
          limit={limit}
          offset={offset}
          onPageChange={setOffset}
          onPageSizeChange={(newLimit) => {
            setLimit(newLimit);
            setOffset(0);
          }}
        />
      )}
    </div>
  );
}
