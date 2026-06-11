"use client";

import { useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { FileText, Plus, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import {
  useOrderInvoices,
  useInvoicingSettings,
  useCreateInvoice,
  isActiveInvoice,
} from "@/hooks/use-invoices";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, shortId } from "@/lib/utils";
import type { Order } from "@/types/api";

/**
 * Manual invoice issuance for an order.
 *
 * IDEMPOTENCY: invoices are also auto-created by the backend on certain order
 * status changes (invoice_service.HandleOrderStatusChange). A manual "issue
 * invoice" button could therefore create a DUPLICATE. We guard the UI by
 * fetching the order's invoices and hiding the issue button whenever an active
 * (non-cancelled, non-error) invoice already exists — the exact predicate the
 * server uses to skip auto-creation (isActiveInvoice). The submit handler also
 * re-checks the guard to defend against a refetch race.
 */
export function OrderInvoicesSection({ order }: { order: Order }) {
  const t = useTranslations("orders.detail.invoiceSection");
  const tc = useTranslations("common");

  const { data: invoices, isLoading } = useOrderInvoices(order.id);
  const { data: invoicingSettings } = useInvoicingSettings();
  const createInvoice = useCreateInvoice();

  const hasActiveInvoice = (invoices ?? []).some(isActiveInvoice);
  const provider = (invoicingSettings?.provider ?? "").trim();
  const providerConfigured = provider.length > 0;

  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    customer_name: "",
    customer_email: "",
    nip: "",
    payment_method: "",
  });

  const openDialog = () => {
    setForm({
      customer_name: order.customer_name ?? "",
      customer_email: order.customer_email ?? "",
      nip: "",
      payment_method: order.payment_method ?? "",
    });
    setOpen(true);
  };

  const handleSubmit = () => {
    if (!providerConfigured) {
      toast.error(t("configureFirst"));
      return;
    }
    // Defensive re-check: button is hidden when an active invoice exists, but a
    // stale render could still mount it. Never issue a second active invoice.
    if (hasActiveInvoice) {
      toast.error(t("activeExists"));
      return;
    }
    createInvoice.mutate(
      {
        order_id: order.id,
        provider,
        customer_name: form.customer_name,
        customer_email: form.customer_email || undefined,
        nip: form.nip || undefined,
        payment_method: form.payment_method || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t("issued"));
          setOpen(false);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <FileText className="h-4 w-4 text-muted-foreground" />
          {t("title")}
        </CardTitle>
        {!hasActiveInvoice && (
          <Button
            size="sm"
            variant="outline"
            disabled={!providerConfigured}
            title={providerConfigured ? undefined : t("configureFirst")}
            onClick={openDialog}
          >
            <Plus className="mr-1 h-4 w-4" />
            {t("issue")}
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{tc("loading")}</p>
        ) : invoices && invoices.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("number")}</TableHead>
                <TableHead>{tc("status")}</TableHead>
                <TableHead>{t("issueDate")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {invoices.map((inv) => (
                <TableRow key={inv.id}>
                  <TableCell>
                    <Link
                      href={`/invoices/${inv.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {inv.external_number || shortId(inv.id)}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                      {inv.status}
                    </span>
                  </TableCell>
                  <TableCell>
                    {inv.issue_date ? formatDate(inv.issue_date) : "---"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <p className="text-sm text-muted-foreground">{t("empty")}</p>
        )}
        {hasActiveInvoice && (
          <p className="mt-3 text-xs text-muted-foreground">{t("activeExists")}</p>
        )}
      </CardContent>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dialogTitle")}</DialogTitle>
            <DialogDescription>{t("dialogDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="invoice-customer-name">{t("customerName")}</Label>
              <Input
                id="invoice-customer-name"
                value={form.customer_name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, customer_name: e.target.value }))
                }
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="invoice-customer-email">{t("customerEmail")}</Label>
              <Input
                id="invoice-customer-email"
                type="email"
                value={form.customer_email}
                onChange={(e) =>
                  setForm((f) => ({ ...f, customer_email: e.target.value }))
                }
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="invoice-nip">{t("nip")}</Label>
              <Input
                id="invoice-nip"
                value={form.nip}
                onChange={(e) => setForm((f) => ({ ...f, nip: e.target.value }))}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="invoice-payment-method">{t("paymentMethod")}</Label>
              <Input
                id="invoice-payment-method"
                value={form.payment_method}
                onChange={(e) =>
                  setForm((f) => ({ ...f, payment_method: e.target.value }))
                }
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              {tc("cancel")}
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={createInvoice.isPending || !form.customer_name.trim()}
            >
              {createInvoice.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              {t("submit")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
