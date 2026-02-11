"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useInvoice, useCancelInvoice } from "@/hooks/use-invoices";
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
} from "@/lib/constants";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { FileDown, XCircle } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function InvoiceDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [showCancelDialog, setShowCancelDialog] = useState(false);

  const { data: invoice, isLoading } = useInvoice(params.id);
  const cancelInvoice = useCancelInvoice();

  const handleCancel = async () => {
    try {
      await cancelInvoice.mutateAsync(params.id);
      toast.success("Faktura została anulowana");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDownloadPDF = () => {
    window.open(`${API_URL}/v1/invoices/${params.id}/pdf`, "_blank");
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
        <Button variant="outline" className="mt-4" onClick={() => router.push("/invoices")}>
          Wróć do listy
        </Button>
      </div>
    );
  }

  const canCancel = invoice.status !== "cancelled" && invoice.status !== "error";
  const hasPDF = invoice.external_id && invoice.status !== "draft" && invoice.status !== "error";

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">
              Faktura {invoice.external_number || `#${shortId(params.id)}`}
            </h1>
            <StatusBadge status={invoice.status} statusMap={INVOICE_STATUS_MAP} />
          </div>
          <p className="text-muted-foreground mt-1">
            Utworzona {formatDate(invoice.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {hasPDF && (
            <Button variant="outline" onClick={handleDownloadPDF}>
              <FileDown className="mr-2 h-4 w-4" />
              Pobierz PDF
            </Button>
          )}
          {canCancel && (
            <Button variant="destructive" onClick={() => setShowCancelDialog(true)}>
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
                    {INVOICE_TYPE_LABELS[invoice.invoice_type] || invoice.invoice_type}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Zamówienie</p>
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
                    {INVOICING_PROVIDER_LABELS[invoice.provider] || invoice.provider}
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
                  <p className="text-sm text-muted-foreground">Data wystawienia</p>
                  <p className="mt-1 text-sm">
                    {invoice.issue_date ? formatDate(invoice.issue_date) : "-"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Termin płatności</p>
                  <p className="mt-1 text-sm">
                    {invoice.due_date ? formatDate(invoice.due_date) : "-"}
                  </p>
                </div>
              </div>

              {invoice.error_message && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">Błąd</p>
                    <p className="mt-1 text-sm text-destructive">{invoice.error_message}</p>
                  </div>
                </>
              )}
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
                  <p className="text-sm text-muted-foreground">ID zewnętrzne</p>
                  <p className="mt-1 font-mono text-xs">{invoice.external_id}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">Utworzono</p>
                <p className="mt-1 text-sm">{formatDate(invoice.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Zaktualizowano</p>
                <p className="mt-1 text-sm">{formatDate(invoice.updated_at)}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={showCancelDialog}
        onOpenChange={setShowCancelDialog}
        title="Anulowanie faktury"
        description="Czy na pewno chcesz anulować tę fakturę? Ta operacja jest nieodwracalna."
        confirmLabel="Anuluj fakturę"
        variant="destructive"
        onConfirm={handleCancel}
        isLoading={cancelInvoice.isPending}
      />
    </div>
  );
}
