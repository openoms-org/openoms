"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
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
import type { PrintTemplatesConfig } from "@/types/api";

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
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: templates, isLoading } = usePrintTemplates();
  const updateTemplates = useUpdatePrintTemplates();

  const [form, setForm] = useState<PrintTemplatesConfig>(DEFAULT_CONFIG);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.replace("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (templates) {
      setForm({
        ...DEFAULT_CONFIG,
        ...templates,
      });
    }
  }, [templates]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  const handleSave = async () => {
    try {
      await updateTemplates.mutateAsync(form);
      toast.success("Szablony wydruku zapisane");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udalo sie zapisac szablonow";
      toast.error(message);
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Szablony wydruku</h1>
          <p className="text-muted-foreground mt-1">
            Dostosuj szablony HTML dla listów przewozowych, podsumowań zamówień i
            formularzy zwrotów. Pozostaw puste, aby używać domyślnych szablonów.
          </p>
        </div>
        <Button onClick={handleSave} disabled={updateTemplates.isPending}>
          {updateTemplates.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          Zapisz
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Zmienne szablonów</CardTitle>
          <CardDescription>
            Szablony używają składni Go html/template. Dostępne zmienne zależą od
            typu szablonu. Funkcja {`{{inc $i}}`} zwiększa indeks o 1.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3 text-sm">
            <div>
              <p className="font-medium mb-1">List przewozowy</p>
              <code className="block text-xs text-muted-foreground whitespace-pre-wrap">
                {`.CompanyName .CompanyAddress .CompanyNIP
.OrderID .OrderDate .Source
.CustomerName .ShippingAddress
.Items (Name, SKU, Quantity, Price, Total)
.TotalAmount .Currency .Notes`}
              </code>
            </div>
            <div>
              <p className="font-medium mb-1">Podsumowanie zamówienia</p>
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
              <p className="font-medium mb-1">Formularz zwrotu</p>
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
          <TabsTrigger value="packing_slip">List przewozowy</TabsTrigger>
          <TabsTrigger value="order_summary">Podsumowanie zamówienia</TabsTrigger>
          <TabsTrigger value="return_slip">Formularz zwrotu</TabsTrigger>
        </TabsList>

        <TabsContent value="packing_slip" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>List przewozowy</CardTitle>
                  <CardDescription>
                    Szablon HTML do drukowania listów przewozowych.
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="packing_slip_html">Szablon HTML</Label>
                <Textarea
                  id="packing_slip_html"
                  value={form.packing_slip_html}
                  onChange={(e) =>
                    setForm({ ...form, packing_slip_html: e.target.value })
                  }
                  placeholder="Pozostaw puste, aby używać domyślnego szablonu..."
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
                  <CardTitle>Podsumowanie zamówienia</CardTitle>
                  <CardDescription>
                    Szablon HTML do drukowania podsumowań zamówień.
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="order_summary_html">Szablon HTML</Label>
                <Textarea
                  id="order_summary_html"
                  value={form.order_summary_html}
                  onChange={(e) =>
                    setForm({ ...form, order_summary_html: e.target.value })
                  }
                  placeholder="Pozostaw puste, aby używać domyślnego szablonu..."
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
                  <CardTitle>Formularz zwrotu</CardTitle>
                  <CardDescription>
                    Szablon HTML do drukowania formularzy zwrotów.
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Label htmlFor="return_slip_html">Szablon HTML</Label>
                <Textarea
                  id="return_slip_html"
                  value={form.return_slip_html}
                  onChange={(e) =>
                    setForm({ ...form, return_slip_html: e.target.value })
                  }
                  placeholder="Pozostaw puste, aby używać domyślnego szablonu..."
                  rows={16}
                  className="font-mono text-xs"
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
