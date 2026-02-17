"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
import { DevelopmentBanner } from "@/components/shared/development-banner";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSetupShopify } from "@/hooks/use-store-integrations";
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

export default function ShopifySetupPage() {
  const [shopDomain, setShopDomain] = useState("");
  const [accessToken, setAccessToken] = useState("");

  const setupMutation = useSetupShopify();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!shopDomain || !accessToken) {
      toast.error("Wszystkie pola sa wymagane");
      return;
    }

    setupMutation.mutate(
      {
        shop_domain: shopDomain,
        access_token: accessToken,
      },
      {
        onSuccess: () => {
          toast.success("Integracja Shopify zostala skonfigurowana");
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
            <Link href="/integrations">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Integracja Shopify</h1>
            <p className="text-muted-foreground">
              Polacz swoj sklep Shopify, aby synchronizowac zamowienia i produkty
            </p>
          </div>
        </div>

        <DevelopmentBanner />

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Dane uwierzytelniajace Shopify Admin API</CardTitle>
            <CardDescription>
              Podaj domene sklepu i token dostepu. Token uzyskasz tworzac
              prywatna aplikacje w panelu Shopify Admin &rarr; Ustawienia &rarr; Aplikacje.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="shop_domain">Domena sklepu</Label>
                <Input
                  id="shop_domain"
                  value={shopDomain}
                  onChange={(e) => setShopDomain(e.target.value)}
                  placeholder="mojsklep.myshopify.com"
                />
                <p className="text-xs text-muted-foreground">
                  Domena Twojego sklepu Shopify (np. mojsklep.myshopify.com lub mojsklep)
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="access_token">Access Token</Label>
                <Input
                  id="access_token"
                  type="password"
                  value={accessToken}
                  onChange={(e) => setAccessToken(e.target.value)}
                  placeholder="shpat_..."
                />
                <p className="text-xs text-muted-foreground">
                  Admin API access token z prywatnej aplikacji Shopify
                </p>
              </div>
              <div className="flex gap-2 pt-4">
                <Button type="submit" disabled={setupMutation.isPending}>
                  {setupMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Zapisz i zweryfikuj
                </Button>
                <Button type="button" variant="outline" asChild>
                  <Link href="/integrations">Anuluj</Link>
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Jak uzyskac token dostepu?</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-2">
            <ol className="list-decimal list-inside space-y-1">
              <li>Zaloguj sie do panelu Shopify Admin</li>
              <li>Przejdz do Ustawienia &rarr; Aplikacje i kanaly sprzedazy</li>
              <li>Kliknij &quot;Tworzenie aplikacji&quot; (lub &quot;Develop apps&quot;)</li>
              <li>Utworz nowa aplikacje i skonfiguruj uprawnienia Admin API:
                <br />read_orders, write_orders, read_products, write_products, read_inventory, write_inventory
              </li>
              <li>Zainstaluj aplikacje i skopiuj Admin API access token</li>
            </ol>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
