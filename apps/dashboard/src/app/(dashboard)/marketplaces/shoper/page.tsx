"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSetupShoper } from "@/hooks/use-store-integrations";
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

export default function ShoperSetupPage() {
  const [shopUrl, setShopUrl] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");

  const setupMutation = useSetupShoper();
  const t = useTranslations("marketplaces");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!shopUrl || !clientId || !clientSecret) {
      toast.error(t("setup.allFieldsRequired"));
      return;
    }

    setupMutation.mutate(
      {
        shop_url: shopUrl,
        client_id: clientId,
        client_secret: clientSecret,
      },
      {
        onSuccess: () => {
          toast.success(t("setup.shoperConfigured"));
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
            <h1 className="text-2xl font-bold">Integracja Shoper</h1>
            <p className="text-muted-foreground">
              Polacz swoj sklep Shoper, aby synchronizowac zamowienia i produkty
            </p>
          </div>
        </div>

        <DevelopmentBanner />

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Dane uwierzytelniajace Shoper WebAPI</CardTitle>
            <CardDescription>
              Podaj dane dostepu do WebAPI Shoper. Znajdziesz je w panelu
              administracyjnym sklepu w zakladce Integracje &rarr; WebAPI.
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
                  placeholder="https://mojsklep.shoper.pl"
                />
                <p className="text-xs text-muted-foreground">
                  Pelny adres URL Twojego sklepu Shoper
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="client_id">Client ID (Login WebAPI)</Label>
                <Input
                  id="client_id"
                  value={clientId}
                  onChange={(e) => setClientId(e.target.value)}
                  placeholder="Login WebAPI ze Shoper"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="client_secret">Client Secret (Haslo WebAPI)</Label>
                <Input
                  id="client_secret"
                  type="password"
                  value={clientSecret}
                  onChange={(e) => setClientSecret(e.target.value)}
                  placeholder="Haslo WebAPI ze Shoper"
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
            <CardTitle>Jak uzyskac dane WebAPI?</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-2">
            <ol className="list-decimal list-inside space-y-1">
              <li>Zaloguj sie do panelu administracyjnego Shoper</li>
              <li>Przejdz do Integracje &rarr; WebAPI</li>
              <li>Kliknij &quot;Dodaj konto WebAPI&quot;</li>
              <li>Ustaw uprawnienia: Zamowienia (odczyt), Produkty (odczyt/zapis)</li>
              <li>Skopiuj Login i Haslo do formularza powyzej</li>
            </ol>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
