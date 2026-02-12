"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useInvoice, useCancelInvoice } from "@/hooks/use-invoices";
import { useSendToKSeF, useCheckKSeFStatus } from "@/hooks/use-ksef";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  INVOICE_STATUS_MAP,
  INVOICE_TYPE_LABELS,
  INVOICING_PROVIDER_LABELS,
  KSEF_STATUS_MAP,
} from "@/lib/constants";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage, apiFetch } from "@/lib/api-client";
import {
  FileDown,
  XCircle,
  Send,
  RefreshCw,
  Download,
  Loader2,
} from "lucide-react";

export default function InvoiceDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  const { data: invoice, isLoading, refetch } = useInvoice(params.id);
  const cancelInvoice = useCancelInvoice();
  const sendToKSeF = useSendToKSeF();
  const checkKSeFStatus = useCheckKSeFStatus();

  const handleCancel = async () => {
    try {
      await cancelInvoice.mutateAsync(params.id);
      toast.success("Faktura zostala anulowana");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDownloadPDF = async () => {
    const res = await apiFetch(`/v1/invoices/${params.id}/pdf`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank");
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  };

  const handleSendToKSeF = async () => {
    try {
      await sendToKSeF.mutateAsync(params.id);
      toast.success("Faktura wyslana do KSeF");
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleCheckKSeFStatus = async () => {
    try {
      await checkKSeFStatus.mutateAsync(params.id);
      toast.success("Status KSeF zaktualizowany");
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDownloadUPO = async () => {
    try {
      const res = await apiFetch(`/v1/invoices/${params.id}/ksef/upo`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `upo-${params.id}.xml`;
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!invoice) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">Nie znaleziono faktury</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => router.push("/invoices")}
        >
          Wroc do listy
        </Button>
      </div>
    );
  }

  const canCancel =
    invoice.status !== "cancelled" && invoice.status !== "error";
  const hasPDF =
    invoice.external_id &&
    invoice.status !== "draft" &&
    invoice.status !== "error";
  const canSendToKSeF =
    invoice.ksef_status === "not_sent" &&
    invoice.status !== "cancelled" &&
    invoice.status !== "error" &&
    invoice.status !== "draft";
  const canCheckKSeFStatus = invoice.ksef_status === "pending";
  const canDownloadUPO = invoice.ksef_status === "accepted";

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">
              Faktura {invoice.external_number || `#${shortId(params.id)}`}
            </h1>
            <StatusBadge
              status={invoice.status}
              statusMap={INVOICE_STATUS_MAP}
            />
          </div>
          <p className="text-muted-foreground mt-1">
            Utworzona {formatDate(invoice.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {canSendToKSeF && (
            <Button
              variant="outline"
              onClick={handleSendToKSeF}
              disabled={sendToKSeF.isPending}
            >
              {sendToKSeF.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Send className="mr-2 h-4 w-4" />
              )}
              Wyslij do KSeF
            </Button>
          )}
          {hasPDF && (
            <Button variant="outline" onClick={handleDownloadPDF}>
              <FileDown className="mr-2 h-4 w-4" />
              Pobierz PDF
            </Button>
          )}
          {canCancel && (
            <Button
              variant="destructive"
              onClick={() => setShowCancelDialog(true)}
            >
              <XCircle className="mr-2 h-4 w-4" />
              Anuluj
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Dane faktury</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Numer</p>
                  <p className="mt-1 font-medium font-mono">
                    {invoice.external_number || "-"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Typ</p>
                  <p className="mt-1">
                    {INVOICE_TYPE_LABELS[invoice.invoice_type] ||
                      invoice.invoice_type}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Zamowienie</p>
                  <Link
                    href={`/orders/${invoice.order_id}`}
                    className="mt-1 font-mono text-sm text-primary hover:underline"
                  >
                    {shortId(invoice.order_id)}
                  </Link>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Dostawca</p>
                  <p className="mt-1">
                    {INVOICING_PROVIDER_LABELS[invoice.provider] ||
                      invoice.provider}
                  </p>
                </div>
              </div>

              <Separator />

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Kwota netto</p>
                  <p className="mt-1 font-medium">
                    {formatCurrency(invoice.total_net, invoice.currency)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Kwota brutto</p>
                  <p className="mt-1 font-medium">
                    {formatCurrency(invoice.total_gross, invoice.currency)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    Data wystawienia
                  </p>
                  <p className="mt-1 text-sm">
                    {invoice.issue_date ? formatDate(invoice.issue_date) : "-"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    Termin platnosci
                  </p>
                  <p className="mt-1 text-sm">
                    {invoice.due_date ? formatDate(invoice.due_date) : "-"}
                  </p>
                </div>
              </div>

              {invoice.error_message && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">Blad</p>
                    <p className="mt-1 text-sm text-destructive">
                      {invoice.error_message}
                    </p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* KSeF Section */}
          <Card>
            <CardHeader>
              <CardTitle>KSeF - Krajowy System e-Faktur</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Status KSeF</p>
                  <div className="mt-1">
                    <StatusBadge
                      status={invoice.ksef_status}
                      statusMap={KSEF_STATUS_MAP}
                    />
                  </div>
                </div>
                {invoice.ksef_number && (
                  <div>
                    <p className="text-sm text-muted-foreground">Numer KSeF</p>
                    <p className="mt-1 font-mono text-sm font-medium">
                      {invoice.ksef_number}
                    </p>
                  </div>
                )}
                {invoice.ksef_sent_at && (
                  <div>
                    <p className="text-sm text-muted-foreground">
                      Data wyslania
                    </p>
                    <p className="mt-1 text-sm">
                      {formatDate(invoice.ksef_sent_at)}
                    </p>
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2 pt-2">
                {canSendToKSeF && (
                  <Button
                    size="sm"
                    onClick={handleSendToKSeF}
                    disabled={sendToKSeF.isPending}
                  >
                    {sendToKSeF.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Send className="mr-2 h-4 w-4" />
                    )}
                    Wyslij do KSeF
                  </Button>
                )}
                {canCheckKSeFStatus && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleCheckKSeFStatus}
                    disabled={checkKSeFStatus.isPending}
                  >
                    {checkKSeFStatus.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="mr-2 h-4 w-4" />
                    )}
                    Sprawdz status
                  </Button>
                )}
                {canDownloadUPO && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleDownloadUPO}
                  >
                    <Download className="mr-2 h-4 w-4" />
                    Pobierz UPO
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Informacje</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="mt-1 font-mono text-xs">{invoice.id}</p>
              </div>
              {invoice.external_id && (
                <div>
                  <p className="text-sm text-muted-foreground">ID zewnetrzne</p>
                  <p className="mt-1 font-mono text-xs">
                    {invoice.external_id}
                  </p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">Utworzono</p>
                <p className="mt-1 text-sm">
                  {formatDate(invoice.created_at)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Zaktualizowano</p>
                <p className="mt-1 text-sm">
                  {formatDate(invoice.updated_at)}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={showCancelDialog}
        onOpenChange={setShowCancelDialog}
        title="Anulowanie faktury"
        description="Czy na pewno chcesz anulowac te fakture? Ta operacja jest nieodwracalna."
        confirmLabel="Anuluj fakture"
        variant="destructive"
        onConfirm={handleCancel}
        isLoading={cancelInvoice.isPending}
      />
    </div>
  );
}
