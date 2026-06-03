"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSetupPrestaShop } from "@/hooks/use-store-integrations";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function PrestaShopSetupPage() {
  const [shopUrl, setShopUrl] = useState("");
  const [apiKey, setApiKey] = useState("");

  const setupMutation = useSetupPrestaShop();
  const t = useTranslations("marketplaces");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!shopUrl || !apiKey) {
      toast.error(t("setup.allFieldsRequired"));
      return;
    }

    setupMutation.mutate(
      {
        shop_url: shopUrl,
        api_key: apiKey,
      },
      {
        onSuccess: () => {
          toast.success(t("setup.prestaShopConfigured"));
        },
        onError: (err) => {
          toast.error(getErrorMessage(err));
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/marketplaces">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Integracja PrestaShop</h1>
            <p className="text-muted-foreground">
              Polacz swoj sklep PrestaShop, aby synchronizowac zamowienia i produkty
            </p>
          </div>
        </div>

        <DevelopmentBanner />

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Dane uwierzytelniajace PrestaShop Web Service</CardTitle>
            <CardDescription>
              Podaj adres sklepu i klucz API. Klucz API znajdziesz w panelu
              administracyjnym PrestaShop w sekcji Zaawansowane &rarr; Uslugi sieciowe.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="shop_url">Adres sklepu</Label>
                <Input
                  id="shop_url"
                  value={shopUrl}
                  onChange={(e) => setShopUrl(e.target.value)}
                  placeholder="https://mojsklep.pl"
                />
                <p className="text-xs text-muted-foreground">
                  Pelny adres URL Twojego sklepu PrestaShop (bez /api)
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="api_key">Klucz API</Label>
                <Input
                  id="api_key"
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder="Klucz API uslugi sieciowej"
                />
              </div>
              <div className="flex gap-2 pt-4">
                <Button type="submit" disabled={setupMutation.isPending}>
                  {setupMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Zapisz i zweryfikuj
                </Button>
                <Button type="button" variant="outline" asChild>
                  <Link href="/marketplaces">Anuluj</Link>
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Jak uzyskac klucz API?</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-2">
            <ol className="list-decimal list-inside space-y-1">
              <li>Zaloguj sie do panelu administracyjnego PrestaShop</li>
              <li>Przejdz do Zaawansowane &rarr; Uslugi sieciowe (Web Service)</li>
              <li>Wlacz uslugi sieciowe jesli sa wylaczone</li>
              <li>Kliknij &quot;Dodaj nowy klucz&quot;</li>
              <li>Ustaw uprawnienia: orders, order_details, products, stock_availables, customers, addresses (GET + PUT)</li>
              <li>Skopiuj wygenerowany klucz do formularza powyzej</li>
            </ol>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
