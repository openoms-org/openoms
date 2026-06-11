"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  useReconciliationSummary,
  useSettlements,
  useImportCSV,
  useCreateSettlement,
} from "@/hooks/use-reconciliation";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { Button } from "@/components/ui/button";
import { QueryError } from "@/components/shared/query-error";
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
import { useTranslations } from "next-intl";

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  matched: "default",
  partial_match: "outline",
  unmatched: "destructive",
};

function formatCurrency(amount: number, currency: string = "PLN"): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount);
}

const EMPTY_MANUAL = {
  settlement_id: "",
  settlement_date: "",
  total_amount: "",
  fee_amount: "",
  net_amount: "",
  currency: "PLN",
  notes: "",
};

export default function ReconciliationPage() {
  const t = useTranslations("reconciliation");
  const tf = useTranslations("feHooks");
  const router = useRouter();
  const [providerFilter, setProviderFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [dateFrom, setDateFrom] = useState<string>("");
  const [dateTo, setDateTo] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  // Import dialog state
  const [importOpen, setImportOpen] = useState(false);
  const [importMode, setImportMode] = useState<"csv" | "manual">("csv");
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importProvider, setImportProvider] = useState<string>("payu");
  const [manualForm, setManualForm] = useState(EMPTY_MANUAL);

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
  const createSettlement = useCreateSettlement();

  const handleCreateManual = async () => {
    const total = Number(manualForm.total_amount);
    if (!manualForm.settlement_date || !Number.isFinite(total) || total <= 0) {
      toast.error(tf("settlementInvalid"));
      return;
    }
    const fee = Number(manualForm.fee_amount) || 0;
    const net = manualForm.net_amount.trim()
      ? Number(manualForm.net_amount)
      : total - fee;
    try {
      const created = await createSettlement.mutateAsync({
        provider: importProvider,
        settlement_id: manualForm.settlement_id || undefined,
        settlement_date: manualForm.settlement_date,
        total_amount: total,
        fee_amount: fee,
        net_amount: net,
        currency: manualForm.currency || undefined,
        notes: manualForm.notes || undefined,
      });
      toast.success(tf("settlementCreated"));
      setImportOpen(false);
      setManualForm(EMPTY_MANUAL);
      refetch();
      router.push(`/reconciliation/${created.id}`);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleImport = async () => {
    if (!importFile) return;
    try {
      const result = await importCSV.mutateAsync({
        file: importFile,
        provider: importProvider,
      });
      toast.success(
        t("importSuccess", { provider: result.provider, date: result.settlement_date })
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
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("subtitle")}
          </p>
        </div>
        <Dialog open={importOpen} onOpenChange={setImportOpen}>
          <DialogTrigger asChild>
            <Button>
              <Upload className="mr-2 h-4 w-4" />
              {t("importSettlement")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("importSettlementCsv")}</DialogTitle>
              <DialogDescription>
                {t("importDialogDescription")}
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="inline-flex rounded-md border p-0.5">
                <Button
                  type="button"
                  size="sm"
                  variant={importMode === "csv" ? "default" : "ghost"}
                  className="flex-1"
                  onClick={() => setImportMode("csv")}
                >
                  {tf("settlementModeCsv")}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant={importMode === "manual" ? "default" : "ghost"}
                  className="flex-1"
                  onClick={() => setImportMode("manual")}
                >
                  {tf("settlementModeManual")}
                </Button>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="import-provider">{t("paymentProvider")}</Label>
                <Select
                  value={importProvider}
                  onValueChange={setImportProvider}
                >
                  <SelectTrigger id="import-provider">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(["payu", "przelewy24", "tpay", "stripe", "manual"] as const).map((key) => (
                      <SelectItem key={key} value={key}>
                        {t(`providerLabels.${key}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {importMode === "csv" ? (
                <div className="grid gap-2">
                  <Label htmlFor="import-file">{t("csvFile")}</Label>
                  <Input
                    id="import-file"
                    type="file"
                    accept=".csv"
                    onChange={(e) =>
                      setImportFile(e.target.files?.[0] || null)
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("csvFileHint")}
                  </p>
                </div>
              ) : (
                <div className="grid gap-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div className="grid gap-1.5">
                      <Label htmlFor="manual-date">{tf("settlementDate")}</Label>
                      <Input
                        id="manual-date"
                        type="date"
                        value={manualForm.settlement_date}
                        onChange={(e) =>
                          setManualForm((f) => ({ ...f, settlement_date: e.target.value }))
                        }
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="manual-id">{tf("settlementId")}</Label>
                      <Input
                        id="manual-id"
                        value={manualForm.settlement_id}
                        onChange={(e) =>
                          setManualForm((f) => ({ ...f, settlement_id: e.target.value }))
                        }
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-3">
                    <div className="grid gap-1.5">
                      <Label htmlFor="manual-total">{tf("settlementTotal")}</Label>
                      <Input
                        id="manual-total"
                        type="number"
                        step="0.01"
                        value={manualForm.total_amount}
                        onChange={(e) =>
                          setManualForm((f) => ({ ...f, total_amount: e.target.value }))
                        }
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="manual-fee">{tf("settlementFee")}</Label>
                      <Input
                        id="manual-fee"
                        type="number"
                        step="0.01"
                        value={manualForm.fee_amount}
                        onChange={(e) =>
                          setManualForm((f) => ({ ...f, fee_amount: e.target.value }))
                        }
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="manual-net">{tf("settlementNet")}</Label>
                      <Input
                        id="manual-net"
                        type="number"
                        step="0.01"
                        value={manualForm.net_amount}
                        onChange={(e) =>
                          setManualForm((f) => ({ ...f, net_amount: e.target.value }))
                        }
                      />
                    </div>
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="manual-currency">{tf("settlementCurrency")}</Label>
                    <Input
                      id="manual-currency"
                      value={manualForm.currency}
                      onChange={(e) =>
                        setManualForm((f) => ({ ...f, currency: e.target.value }))
                      }
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="manual-notes">{tf("settlementNotes")}</Label>
                    <Input
                      id="manual-notes"
                      value={manualForm.notes}
                      onChange={(e) =>
                        setManualForm((f) => ({ ...f, notes: e.target.value }))
                      }
                    />
                  </div>
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setImportOpen(false)}
              >
                {t("cancel")}
              </Button>
              {importMode === "csv" ? (
                <Button
                  onClick={handleImport}
                  disabled={!importFile || importCSV.isPending}
                >
                  {importCSV.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {t("import")}
                </Button>
              ) : (
                <Button
                  onClick={handleCreateManual}
                  disabled={createSettlement.isPending}
                >
                  {createSettlement.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {tf("settlementCreate")}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{t("matched")}</CardTitle>
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
              <CardTitle className="text-sm font-medium">{t("unmatched")}</CardTitle>
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
              <CardTitle className="text-sm font-medium">{t("discrepancies")}</CardTitle>
              <AlertTriangle className="h-4 w-4 text-yellow-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{summary.discrepancy_count}</div>
              <p className="text-xs text-muted-foreground">
                {t("amountDifferences")}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{t("fees")}</CardTitle>
              <CreditCard className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatCurrency(summary.total_fees)}
              </div>
              <p className="text-xs text-muted-foreground">
                {t("totalGatewayFees")}
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
              <SelectValue placeholder={t("columns.provider")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("allProviders")}</SelectItem>
              {(["payu", "przelewy24", "tpay", "stripe", "manual"] as const).map((key) => (
                <SelectItem key={key} value={key}>
                  {t(`providerLabels.${key}`)}
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
              <SelectItem value="all">{t("allStatuses")}</SelectItem>
              {(["pending", "matched", "partial_match", "unmatched"] as const).map((key) => (
                <SelectItem key={key} value={key}>
                  {t(`statusLabels.${key}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2">
          <Label className="text-sm whitespace-nowrap">{t("dateFrom")}</Label>
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
          <Label className="text-sm whitespace-nowrap">{t("dateTo")}</Label>
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
        <QueryError onRetry={() => refetch()} messageOverride={t("loadError")} />
      )}

      {/* Settlements table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("columns.provider")}</TableHead>
              <TableHead>{t("columns.settlementDate")}</TableHead>
              <TableHead>{t("columns.settlementId")}</TableHead>
              <TableHead className="text-right">{t("columns.grossAmount")}</TableHead>
              <TableHead className="text-right">{t("columns.fees")}</TableHead>
              <TableHead className="text-right">{t("columns.netAmount")}</TableHead>
              <TableHead>{t("columns.currency")}</TableHead>
              <TableHead>{t("columns.status")}</TableHead>
              <TableHead>{t("columns.importedAt")}</TableHead>
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
                  {t("emptyState")}
                </TableCell>
              </TableRow>
            )}
            {settlements?.items?.map((settlement: PaymentSettlement) => {
              const statusVariant = STATUS_VARIANTS[settlement.status] || "secondary";
              const statusLabel = t(`statusLabels.${settlement.status}`, { defaultValue: settlement.status });
              return (
                <TableRow
                  key={settlement.id}
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() =>
                    router.push(`/reconciliation/${settlement.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    {t(`providerLabels.${settlement.provider}`, { defaultValue: settlement.provider })}
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
                    <Badge variant={statusVariant}>{statusLabel}</Badge>
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
