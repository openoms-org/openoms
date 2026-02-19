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

const MATCH_STATUS_LABELS: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  matched: { label: "Uzgodnione", variant: "default" },
  unmatched: { label: "Nieuzgodnione", variant: "destructive" },
  discrepancy: { label: "Rozbieznosc", variant: "outline" },
  manual_match: { label: "Reczne", variant: "secondary" },
};

const TX_TYPE_LABELS: Record<string, string> = {
  payment: "Platnosc",
  refund: "Zwrot",
  chargeback: "Chargeback",
  fee: "Prowizja",
};

function formatCurrency(amount: number, currency: string = "PLN"): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount);
}

export default function SettlementDetailPage() {
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
        `Automatyczne uzgadnianie: ${result.matched} uzgodnionych, ${result.unmatched} nieuzgodnionych, ${result.discrepancy} rozbieznosci`
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
      toast.success("Transakcja zostala recznie powiazana z zamowieniem");
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
      toast.success("Powiazanie zostalo usuniete");
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
          Powrot do rozliczen
        </Button>
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Nie udalo sie zaladowac rozliczenia.
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
      </div>
    );
  }

  const statusInfo = STATUS_LABELS[settlement.status] || {
    label: settlement.status,
    variant: "secondary" as const,
  };

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
            Powrot
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">
                Rozliczenie {PROVIDER_LABELS[settlement.provider] || settlement.provider}
              </h1>
              <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
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
          Automatyczne uzgadnianie
        </Button>
      </div>

      {/* Settlement summary cards */}
      <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-6">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Kwota brutto
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
              Prowizje
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
              Kwota netto
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
              Uzgodnione
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-green-600">{matchedCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-red-600">
              Nieuzgodnione
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold text-red-600">{unmatchedCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-yellow-600">
              Rozbieznosci
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
            <CardTitle className="text-sm font-medium">Notatki</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">{settlement.notes}</p>
          </CardContent>
        </Card>
      )}

      {/* Transactions table */}
      <div>
        <h2 className="text-lg font-semibold mb-4">
          Transakcje ({settlement.transactions.length})
        </h2>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID zewnetrzne</TableHead>
                <TableHead>Typ</TableHead>
                <TableHead>Data</TableHead>
                <TableHead className="text-right">Kwota</TableHead>
                <TableHead className="text-right">Prowizja</TableHead>
                <TableHead className="text-right">Netto</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Zamowienie</TableHead>
                <TableHead>Notatki</TableHead>
                <TableHead className="text-right">Akcje</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {settlement.transactions.length === 0 && (
                <TableRow>
                  <TableCell colSpan={10} className="text-center py-8 text-muted-foreground">
                    Brak transakcji w tym rozliczeniu.
                  </TableCell>
                </TableRow>
              )}
              {settlement.transactions.map((txn: PaymentTransaction) => {
                const matchInfo = MATCH_STATUS_LABELS[txn.match_status] || {
                  label: txn.match_status,
                  variant: "secondary" as const,
                };
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
                        {TX_TYPE_LABELS[txn.transaction_type] || txn.transaction_type}
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
                      <Badge variant={matchInfo.variant}>{matchInfo.label}</Badge>
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
                            Powiaz
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
                            Odepnij
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
                              Przepnij
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
            <DialogTitle>Reczne powiazanie transakcji</DialogTitle>
            <DialogDescription>
              Wpisz ID zamowienia, z ktorym chcesz powiazac te transakcje.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="order-id">ID zamowienia (UUID)</Label>
              <Input
                id="order-id"
                value={matchOrderId}
                onChange={(e) => setMatchOrderId(e.target.value)}
                placeholder="np. a1b2c3d4-e5f6-..."
                className="font-mono text-sm"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="match-notes">Notatki (opcjonalnie)</Label>
              <Input
                id="match-notes"
                value={matchNotes}
                onChange={(e) => setMatchNotes(e.target.value)}
                placeholder="Powod recznego powiazania..."
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setMatchDialogOpen(false)}
            >
              Anuluj
            </Button>
            <Button
              onClick={handleManualMatch}
              disabled={!matchOrderId || manualMatch.isPending}
            >
              {manualMatch.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Powiaz
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
