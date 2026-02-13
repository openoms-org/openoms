"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  RotateCcw,
  XCircle,
  CreditCard,
  Package,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroReturns,
  useRejectAllegroReturn,
  useCreateAllegroRefund,
  useAllegroRefunds,
} from "@/hooks/use-allegro";
import type {
  AllegroCustomerReturn,
  AllegroCreateRefundRequest,
} from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { EmptyState } from "@/components/shared/empty-state";
import { formatDate } from "@/lib/utils";

const RETURN_STATUS_MAP: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  EXCHANGE: { label: "Wymiana", variant: "secondary" },
  REFUND: { label: "Zwrot", variant: "default" },
  REFUND_AND_RETURN: { label: "Zwrot + odesłanie", variant: "default" },
  WAITING: { label: "Oczekujacy", variant: "outline" },
  ACCEPTED: { label: "Zaakceptowany", variant: "default" },
  REJECTED: { label: "Odrzucony", variant: "destructive" },
  CANCELLED: { label: "Anulowany", variant: "secondary" },
};

export default function AllegroReturnsPage() {
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [limit] = useState(25);
  const [offset, setOffset] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [rejectDialogReturn, setRejectDialogReturn] = useState<AllegroCustomerReturn | null>(null);
  const [refundDialogReturn, setRefundDialogReturn] = useState<AllegroCustomerReturn | null>(null);

  const { data, isLoading, isError, refetch } = useAllegroReturns({
    limit,
    offset,
    status: statusFilter || undefined,
  });

  const returns = data?.customerReturns ?? [];

  return (
    <AdminGuard>
      <div className="space-y-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Zwroty Allegro</h1>
            <p className="text-muted-foreground">
              Zarzadzaj zwrotami od kupujacych na Allegro
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="w-[200px]">
            <Select
              value={statusFilter || "all"}
              onValueChange={(v) => {
                setStatusFilter(v === "all" ? "" : v);
                setOffset(0);
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Wszystkie</SelectItem>
                <SelectItem value="EXCHANGE">Wymiana</SelectItem>
                <SelectItem value="REFUND">Zwrot</SelectItem>
                <SelectItem value="WAITING">Oczekujacy</SelectItem>
                <SelectItem value="ACCEPTED">Zaakceptowany</SelectItem>
                <SelectItem value="REJECTED">Odrzucony</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            Odswiez
          </Button>
        </div>

        {isError && (
          <Card className="border-destructive">
            <CardContent className="pt-4">
              <p className="text-sm text-destructive">
                Blad podczas ladowania zwrotow. Sprawdz polaczenie z Allegro.
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-2"
                onClick={() => refetch()}
              >
                Sprobuj ponownie
              </Button>
            </CardContent>
          </Card>
        )}

        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full" />
            ))}
          </div>
        )}

        {!isLoading && returns.length === 0 && (
          <EmptyState
            icon={RotateCcw}
            title="Brak zwrotow"
            description="Nie znaleziono zwrotow do wyswietlenia na Allegro."
          />
        )}

        {!isLoading && returns.length > 0 && (
          <div className="space-y-2">
            {returns.map((ret) => (
              <ReturnCard
                key={ret.id}
                ret={ret}
                isExpanded={expandedId === ret.id}
                onToggle={() =>
                  setExpandedId(expandedId === ret.id ? null : ret.id)
                }
                onReject={() => setRejectDialogReturn(ret)}
                onRefund={() => setRefundDialogReturn(ret)}
              />
            ))}
          </div>
        )}

        {/* Pagination */}
        {data && data.count > limit && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Wyswietlanie {offset + 1}-{Math.min(offset + limit, data.count)} z{" "}
              {data.count}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
              >
                Poprzednia
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={offset + limit >= data.count}
                onClick={() => setOffset(offset + limit)}
              >
                Nastepna
              </Button>
            </div>
          </div>
        )}

        {/* Reject dialog */}
        {rejectDialogReturn && (
          <RejectDialog
            ret={rejectDialogReturn}
            onClose={() => setRejectDialogReturn(null)}
          />
        )}

        {/* Refund dialog */}
        {refundDialogReturn && (
          <RefundDialog
            ret={refundDialogReturn}
            onClose={() => setRefundDialogReturn(null)}
          />
        )}
      </div>
    </AdminGuard>
  );
}

function ReturnCard({
  ret,
  isExpanded,
  onToggle,
  onReject,
  onRefund,
}: {
  ret: AllegroCustomerReturn;
  isExpanded: boolean;
  onToggle: () => void;
  onReject: () => void;
  onRefund: () => void;
}) {
  const statusInfo = RETURN_STATUS_MAP[ret.status] ?? {
    label: ret.status,
    variant: "outline" as const,
  };

  return (
    <Card>
      <CardContent className="pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div>
              <div className="flex items-center gap-2">
                <span className="font-mono text-sm font-medium">
                  {ret.referenceNumber || ret.id.slice(0, 12)}
                </span>
                <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
              </div>
              <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                <span>Kupujacy: {ret.buyer.login}</span>
                <span>{formatDate(ret.createdAt)}</span>
                {ret.parcelSentByBuyer && (
                  <Badge variant="outline" className="text-[10px]">
                    <Package className="mr-1 h-3 w-3" />
                    Paczka wyslana
                  </Badge>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {ret.refund && (
              <span className="text-sm font-medium">
                {ret.refund.amount} {ret.refund.currency}
              </span>
            )}
            <Button variant="ghost" size="icon" onClick={onToggle}>
              {isExpanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>

        {isExpanded && (
          <>
            <Separator className="my-3" />
            <div className="space-y-3">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">
                  Pozycje
                </p>
                <div className="space-y-1">
                  {ret.items.map((item, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between text-sm"
                    >
                      <span className="truncate max-w-[60%]">{item.name}</span>
                      <div className="flex items-center gap-3">
                        <span className="text-muted-foreground">
                          Szt.: {item.quantity}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">
                          ID: {item.offerId.slice(0, 10)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span>Email: {ret.buyer.email}</span>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={onReject}
                >
                  <XCircle className="mr-2 h-4 w-4" />
                  Odrzuc zwrot
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onRefund}
                >
                  <CreditCard className="mr-2 h-4 w-4" />
                  Zwroc pieniadze
                </Button>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function RejectDialog({
  ret,
  onClose,
}: {
  ret: AllegroCustomerReturn;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const rejectMutation = useRejectAllegroReturn(ret.id);

  const handleReject = () => {
    if (!reason.trim()) {
      toast.error("Podaj powod odrzucenia");
      return;
    }

    rejectMutation.mutate(reason.trim(), {
      onSuccess: () => {
        toast.success("Zwrot zostal odrzucony");
        onClose();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Nie udalo sie odrzucic zwrotu"
        );
      },
    });
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Odrzuc zwrot</DialogTitle>
          <DialogDescription>
            Zwrot {ret.referenceNumber || ret.id.slice(0, 12)} od{" "}
            {ret.buyer.login}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div>
            <Label htmlFor="reject-reason">Powod odrzucenia</Label>
            <Textarea
              id="reject-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Opisz powod odrzucenia zwrotu..."
              className="mt-1"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            variant="destructive"
            onClick={handleReject}
            disabled={!reason.trim() || rejectMutation.isPending}
          >
            {rejectMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Odrzuc
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RefundDialog({
  ret,
  onClose,
}: {
  ret: AllegroCustomerReturn;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const [paymentId, setPaymentId] = useState("");
  const createRefund = useCreateAllegroRefund();

  const handleRefund = () => {
    if (!reason.trim()) {
      toast.error("Podaj powod zwrotu pieniedzy");
      return;
    }
    if (!paymentId.trim()) {
      toast.error("Podaj ID platnosci");
      return;
    }

    const lineItems = ret.items.map((item) => ({
      offerId: item.offerId,
      quantity: item.quantity,
      amount: {
        amount: ret.refund?.amount ?? "0",
        currency: ret.refund?.currency ?? "PLN",
      },
    }));

    const request: AllegroCreateRefundRequest = {
      payment: { id: paymentId.trim() },
      reason: reason.trim(),
      lineItems,
    };

    createRefund.mutate(request, {
      onSuccess: () => {
        toast.success("Zwrot pieniedzy zostal utworzony");
        onClose();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Nie udalo sie utworzyc zwrotu pieniedzy"
        );
      },
    });
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Zwrot pieniedzy</DialogTitle>
          <DialogDescription>
            Utworz zwrot pieniedzy dla zwrotu{" "}
            {ret.referenceNumber || ret.id.slice(0, 12)}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div>
            <Label htmlFor="payment-id">ID platnosci</Label>
            <Input
              id="payment-id"
              value={paymentId}
              onChange={(e) => setPaymentId(e.target.value)}
              placeholder="ID platnosci z zamowienia Allegro"
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="refund-reason">Powod zwrotu</Label>
            <Textarea
              id="refund-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Opisz powod zwrotu pieniedzy..."
              className="mt-1"
            />
          </div>
          {ret.refund && (
            <div className="rounded-md border p-3 bg-muted/50">
              <p className="text-sm">
                Kwota zwrotu:{" "}
                <span className="font-medium">
                  {ret.refund.amount} {ret.refund.currency}
                </span>
              </p>
            </div>
          )}
          <div>
            <p className="text-xs font-medium text-muted-foreground mb-1">
              Pozycje do zwrotu
            </p>
            <div className="space-y-1">
              {ret.items.map((item, idx) => (
                <div key={idx} className="text-sm flex justify-between">
                  <span className="truncate max-w-[70%]">{item.name}</span>
                  <span className="text-muted-foreground">
                    x{item.quantity}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            onClick={handleRefund}
            disabled={
              !reason.trim() || !paymentId.trim() || createRefund.isPending
            }
          >
            {createRefund.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <CreditCard className="mr-2 h-4 w-4" />
            Utworz zwrot
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
