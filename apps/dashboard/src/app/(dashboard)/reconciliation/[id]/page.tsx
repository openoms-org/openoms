"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  useSettlement,
  useAutoMatch,
  useManualMatch,
  useUnmatchTransaction,
} from "@/hooks/use-reconciliation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { getErrorMessage } from "@/lib/api-client";
import {
  ArrowLeft,
  Loader2,
  Wand2,
  Link2,
  Unlink,
  ExternalLink,
} from "lucide-react";
import type { PaymentTransaction } from "@/types/api";
import { useTranslations } from "next-intl";

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  matched: "default",
  partial_match: "outline",
  unmatched: "destructive",
};

const MATCH_STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  matched: "default",
  unmatched: "destructive",
  discrepancy: "outline",
  manual_match: "secondary",
};

function formatCurrency(amount: number, currency: string = "PLN"): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount);
}

export default function SettlementDetailPage() {
  const t = useTranslations("reconciliation");
  const params = useParams();
  const router = useRouter();
  const settlementId = params.id as string;

  const { data: settlement, isLoading, isError, refetch } = useSettlement(settlementId);
  const autoMatch = useAutoMatch();
  const manualMatch = useManualMatch();
  const unmatch = useUnmatchTransaction();

  // Manual match dialog state
  const [matchDialogOpen, setMatchDialogOpen] = useState(false);
  const [matchTransactionId, setMatchTransactionId] = useState<string>("");
  const [matchOrderId, setMatchOrderId] = useState<string>("");
  const [matchNotes, setMatchNotes] = useState<string>("");

  const handleAutoMatch = async () => {
    try {
      const result = await autoMatch.mutateAsync(settlementId);
      toast.success(
        t("detail.autoMatchResult", { matched: result.matched, unmatched: result.unmatched, discrepancy: result.discrepancy })
      );
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleManualMatch = async () => {
    if (!matchOrderId) return;
    try {
      await manualMatch.mutateAsync({
        transactionId: matchTransactionId,
        data: {
          order_id: matchOrderId,
          notes: matchNotes || undefined,
        },
      });
      toast.success(t("detail.manualMatchSuccess"));
      setMatchDialogOpen(false);
      setMatchOrderId("");
      setMatchNotes("");
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleUnmatch = async (transactionId: string) => {
    try {
      await unmatch.mutateAsync(transactionId);
      toast.success(t("detail.unlinkSuccess"));
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const openMatchDialog = (transactionId: string) => {
    setMatchTransactionId(transactionId);
    setMatchOrderId("");
    setMatchNotes("");
    setMatchDialogOpen(true);
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  if (isError || !settlement) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => router.push("/reconciliation")}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          {t("detail.backToReconciliation")}
        </Button>
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("detail.loadError")}
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
      </div>
    );
  }

  const statusVariant = STATUS_VARIANTS[settlement.status] || "secondary";
  const statusLabel = t(`statusLabels.${settlement.status}`, { defaultValue: settlement.status });

  const matchedCount = settlement.transactions.filter(
    (t) => t.match_status === "matched" || t.match_status === "manual_match"
  ).length;
  const unmatchedCount = settlement.transactions.filter(
    (t) => t.match_status === "unmatched"
  ).length;
  const discrepancyCount = settlement.transactions.filter(
    (t) => t.match_status === "discrepancy"
  ).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => router.push("/reconciliation")}
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("detail.back")}
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">
                {t("detail.settlementTitle", { provider: t(`providerLabels.${settlement.provider}`, { defaultValue: settlement.provider }) })}
              </h1>
              <Badge variant={statusVariant}>{statusLabel}</Badge>
            </div>
            <p className="text-muted-foreground mt-1">
              {settlement.settlement_date}
              {settlement.settlement_id && (
                <span className="ml-2 font-mono text-xs">
                  ID: {settlement.settlement_id}
                </span>
              )}
            </p>
          </div>
        </div>
        <Button
          onClick={handleAutoMatch}
          disabled={autoMatch.isPending}
        >
          {autoMatch.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Wand2 className="mr-2 h-4 w-4" />
          )}
          {t("detail.autoMatch")}
        </Button>
      </div>

      {/* Settlement summary cards */}
      <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-6">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("detail.grossAmount")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold">
              {formatCurrency(settlement.total_amount, settlement.currency)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("detail.fees")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-muted-foreground">
              {formatCurrency(settlement.fee_amount, settlement.currency)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("detail.netAmount")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold">
              {formatCurrency(settlement.net_amount, settlement.currency)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-green-600">
              {t("matched")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-green-600">{matchedCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-red-600">
              {t("unmatched")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-red-600">{unmatchedCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-yellow-600">
              {t("discrepancies")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-yellow-600">{discrepancyCount}</div>
          </CardContent>
        </Card>
      </div>

      {settlement.notes && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">{t("detail.notes")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">{settlement.notes}</p>
          </CardContent>
        </Card>
      )}

      {/* Transactions table */}
      <div>
        <h2 className="text-lg font-semibold mb-4">
          {t("detail.transactions", { count: settlement.transactions.length })}
        </h2>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("detail.transactionColumns.externalId")}</TableHead>
                <TableHead>{t("detail.transactionColumns.type")}</TableHead>
                <TableHead>{t("detail.transactionColumns.date")}</TableHead>
                <TableHead className="text-right">{t("detail.transactionColumns.amount")}</TableHead>
                <TableHead className="text-right">{t("detail.transactionColumns.fee")}</TableHead>
                <TableHead className="text-right">{t("detail.transactionColumns.net")}</TableHead>
                <TableHead>{t("detail.transactionColumns.status")}</TableHead>
                <TableHead>{t("detail.transactionColumns.order")}</TableHead>
                <TableHead>{t("detail.transactionColumns.notes")}</TableHead>
                <TableHead className="text-right">{t("detail.transactionColumns.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {settlement.transactions.length === 0 && (
                <TableRow>
                  <TableCell colSpan={10} className="text-center py-8 text-muted-foreground">
                    {t("detail.emptyTransactions")}
                  </TableCell>
                </TableRow>
              )}
              {settlement.transactions.map((txn: PaymentTransaction) => {
                const matchVariant = MATCH_STATUS_VARIANTS[txn.match_status] || "secondary";
                const matchLabel = t(`matchStatusLabels.${txn.match_status}`, { defaultValue: txn.match_status });
                const isUnmatched = txn.match_status === "unmatched";
                const isDiscrepancy = txn.match_status === "discrepancy";
                const isMatched = txn.match_status === "matched" || txn.match_status === "manual_match";

                let rowClass = "";
                if (isUnmatched) rowClass = "bg-yellow-50 dark:bg-yellow-950/20";
                if (isDiscrepancy) rowClass = "bg-red-50 dark:bg-red-950/20";

                return (
                  <TableRow key={txn.id} className={rowClass}>
                    <TableCell className="font-mono text-xs">
                      {txn.external_transaction_id || "-"}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-xs">
                        {t(`txTypeLabels.${txn.transaction_type}`, { defaultValue: txn.transaction_type })}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {new Date(txn.transaction_date).toLocaleDateString("pl-PL")}
                    </TableCell>
                    <TableCell className="text-right font-medium">
                      {formatCurrency(txn.amount, txn.currency)}
                    </TableCell>
                    <TableCell className="text-right text-muted-foreground">
                      {formatCurrency(txn.fee, txn.currency)}
                    </TableCell>
                    <TableCell className="text-right">
                      {formatCurrency(txn.net_amount, txn.currency)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={matchVariant}>{matchLabel}</Badge>
                    </TableCell>
                    <TableCell>
                      {txn.order_id ? (
                        <Button
                          variant="link"
                          size="sm"
                          className="p-0 h-auto text-xs font-mono"
                          onClick={(e) => {
                            e.stopPropagation();
                            router.push(`/orders/${txn.order_id}`);
                          }}
                        >
                          <ExternalLink className="mr-1 h-3 w-3" />
                          {txn.order_id.slice(0, 8)}...
                        </Button>
                      ) : (
                        <span className="text-muted-foreground text-xs">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[200px] truncate">
                      {txn.match_notes || "-"}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        {isUnmatched && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openMatchDialog(txn.id)}
                          >
                            <Link2 className="mr-1 h-3 w-3" />
                            {t("detail.link")}
                          </Button>
                        )}
                        {isMatched && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleUnmatch(txn.id)}
                            disabled={unmatch.isPending}
                          >
                            <Unlink className="mr-1 h-3 w-3" />
                            {t("detail.unlink")}
                          </Button>
                        )}
                        {isDiscrepancy && (
                          <>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openMatchDialog(txn.id)}
                            >
                              <Link2 className="mr-1 h-3 w-3" />
                              {t("detail.reassign")}
                            </Button>
                            {txn.order_id && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleUnmatch(txn.id)}
                                disabled={unmatch.isPending}
                              >
                                <Unlink className="mr-1 h-3 w-3" />
                              </Button>
                            )}
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* Manual match dialog */}
      <Dialog open={matchDialogOpen} onOpenChange={setMatchDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("detail.manualMatchTitle")}</DialogTitle>
            <DialogDescription>
              {t("detail.manualMatchDescription")}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="order-id">{t("detail.orderId")}</Label>
              <Input
                id="order-id"
                value={matchOrderId}
                onChange={(e) => setMatchOrderId(e.target.value)}
                placeholder={t("detail.orderIdPlaceholder")}
                className="font-mono text-sm"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="match-notes">{t("detail.notesOptional")}</Label>
              <Input
                id="match-notes"
                value={matchNotes}
                onChange={(e) => setMatchNotes(e.target.value)}
                placeholder={t("detail.notesPlaceholder")}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setMatchDialogOpen(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              onClick={handleManualMatch}
              disabled={!matchOrderId || manualMatch.isPending}
            >
              {manualMatch.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              {t("detail.link")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
