"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { Loader2, Save } from "lucide-react";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import type { PrintTemplatesConfig } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_CONFIG: PrintTemplatesConfig = {
  packing_slip_html: "",
  order_summary_html: "",
  return_slip_html: "",
};

function usePrintTemplates() {
  return useQuery({
    queryKey: ["settings", "print-templates"],
    queryFn: () =>
      apiClient<PrintTemplatesConfig>("/v1/settings/print-templates"),
  });
}

function useUpdatePrintTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: PrintTemplatesConfig) =>
      apiClient<PrintTemplatesConfig>("/v1/settings/print-templates", {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "print-templates"] });
    },
  });
}

export default function PrintTemplatesPage() {
  const t = useTranslations("settings");
  const tp = useTranslations("settings.printTemplates");
  const { data: templates, isLoading } = usePrintTemplates();
  const updateTemplates = useUpdatePrintTemplates();

  const [form, setForm] = useState<PrintTemplatesConfig>(DEFAULT_CONFIG);

  useEffect(() => {
    if (templates) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setForm({
        ...DEFAULT_CONFIG,
        ...templates,
      });
    }
  }, [templates]);

  const handleSave = async () => {
    try {
      await updateTemplates.mutateAsync(form);
      toast.success(tp("saved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("templatesSaveError");
      toast.error(message);
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <AdminGuard>
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{tp("title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("dostosujSzablonyHtmlDlaListowPrzewozowychPodsumowa")}
            {t("formularzyZwrotowPozostawPusteAbyUzywacDomyslnych")}
          </p>
        </div>
        <Button onClick={handleSave} disabled={updateTemplates.isPending}>
          {updateTemplates.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          {tp("saveButton")}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("zmienneSzablonow")}</CardTitle>
          <CardDescription>
            {t("templatesUseGoHtmlTemplateSyntax")}
            {tp("templateVarInfo")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3 text-sm">
            <div>
              <p className="font-medium mb-1">{tp("packingSlip")}</p>
              <code className="block text-xs text-muted-foreground whitespace-pre-wrap">
                {`.CompanyName .CompanyAddress .CompanyNIP
.OrderID .OrderDate .Source
.CustomerName .ShippingAddress
.Items (Name, SKU, Quantity, Price, Total)
.TotalAmount .Currency .Notes`}
              </code>
            </div>
            <div>
              <p className="font-medium mb-1">{t("podsumowanieZamowienia")}</p>
              <code className="block text-xs text-muted-foreground whitespace-pre-wrap">
                {`.CompanyName .CompanyAddress .CompanyNIP
.OrderID .OrderDate .Source .Status
.CustomerName .CustomerEmail .CustomerPhone
.ShippingAddress .BillingAddress
.Items (Name, SKU, Quantity, Price, Total)
.TotalAmount .Currency
.PaymentStatus .PaymentMethod .Notes`}
              </code>
            </div>
            <div>
              <p className="font-medium mb-1">{tp("returnForm")}</p>
              <code className="block text-xs text-muted-foreground whitespace-pre-wrap">
                {`.CompanyName .CompanyAddress .CompanyNIP
.ReturnID .OrderID .ReturnDate .Status
.Reason
.Items (Name, SKU, Quantity)
.RefundAmount .Notes`}
              </code>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="packing_slip">
        <TabsList>
          <TabsTrigger value="packing_slip">{tp("packingSlip")}</TabsTrigger>
          <TabsTrigger value="order_summary">{t("podsumowanieZamowienia")}</TabsTrigger>
          <TabsTrigger value="return_slip">{tp("returnForm")}</TabsTrigger>
        </TabsList>

        <TabsContent value="packing_slip" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>{tp("packingSlip")}</CardTitle>
                  <CardDescription>
                    {t("szablonHtmlDoDrukowaniaListowPrzewozowych")}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="packing_slip_html">{tp("htmlTemplate")}</Label>
                <Textarea
                  id="packing_slip_html"
                  value={form.packing_slip_html}
                  onChange={(e) =>
                    setForm({ ...form, packing_slip_html: e.target.value })
                  }
                  placeholder={t("pozostawPusteAbyUzywacDomyslnegoSzablonu")}
                  rows={16}
                  className="font-mono text-xs"
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="order_summary" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>{t("podsumowanieZamowienia")}</CardTitle>
                  <CardDescription>
                    {t("szablonHtmlDoDrukowaniaPodsumowanZamowien")}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="order_summary_html">{tp("htmlTemplate")}</Label>
                <Textarea
                  id="order_summary_html"
                  value={form.order_summary_html}
                  onChange={(e) =>
                    setForm({ ...form, order_summary_html: e.target.value })
                  }
                  placeholder={t("pozostawPusteAbyUzywacDomyslnegoSzablonu")}
                  rows={16}
                  className="font-mono text-xs"
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="return_slip" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>{tp("returnForm")}</CardTitle>
                  <CardDescription>
                    {t("szablonHtmlDoDrukowaniaFormularzyZwrotow")}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="return_slip_html">{tp("htmlTemplate")}</Label>
                <Textarea
                  id="return_slip_html"
                  value={form.return_slip_html}
                  onChange={(e) =>
                    setForm({ ...form, return_slip_html: e.target.value })
                  }
                  placeholder={t("pozostawPusteAbyUzywacDomyslnegoSzablonu")}
                  rows={16}
                  className="font-mono text-xs"
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
    </AdminGuard>
  );
}
