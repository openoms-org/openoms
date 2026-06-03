"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useIntegrations } from "@/hooks/use-integrations";
import { useCreateListingSyncConfig } from "@/hooks/use-listing-sync";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { getErrorMessage } from "@/lib/api-client";

const PROVIDER_LABELS: Record<string, string> = {
  allegro: "Allegro",
  amazon: "Amazon",
  ebay: "eBay",
  kaufland: "Kaufland",
  olx: "OLX",
  woocommerce: "WooCommerce",
  erli: "Erli",
  mirakl: "Mirakl/Empik",
  shoper: "Shoper",
  prestashop: "PrestaShop",
  shopify: "Shopify",
};

export default function NewListingSyncPage() {
  const router = useRouter();
  const t = useTranslations("listings");
  const { data: integrations, isLoading: loadingIntegrations } = useIntegrations();
  const createConfig = useCreateListingSyncConfig();

  const [integrationId, setIntegrationId] = useState("");
  const [syncDirection, setSyncDirection] = useState<"push" | "pull" | "bidirectional">("bidirectional");
  const [autoSync, setAutoSync] = useState(false);
  const [syncInterval, setSyncInterval] = useState(30);
  const [priceRule, setPriceRule] = useState<"same" | "markup_pct" | "markup_fixed" | "custom">("same");
  const [priceModifier, setPriceModifier] = useState(0);
  const [stockBuffer, setStockBuffer] = useState(0);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!integrationId) {
      toast.error(t("sync.integrationRequired"));
      return;
    }

    try {
      const config = await createConfig.mutateAsync({
        integration_id: integrationId,
        sync_direction: syncDirection,
        auto_sync: autoSync,
        sync_interval_minutes: syncInterval,
        price_rule: priceRule,
        price_modifier: priceModifier,
        stock_buffer: stockBuffer,
      });
      toast.success(t("sync.configCreated"));
      router.push(`/listing-sync/${config.id}`);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const activeIntegrations = (integrations || []).filter(
    (i) => i.status === "active"
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/listing-sync">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Nowa konfiguracja synchronizacji</h1>
          <p className="text-muted-foreground">
            Skonfiguruj synchronizacje ofert z platforma marketplace
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="max-w-2xl space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Integracja</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="integration">Platforma marketplace</Label>
              {loadingIntegrations ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Ladowanie integracji...
                </div>
              ) : (
                <Select value={integrationId} onValueChange={setIntegrationId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Wybierz integracje" />
                  </SelectTrigger>
                  <SelectContent>
                    {activeIntegrations.map((integration) => (
                      <SelectItem key={integration.id} value={integration.id}>
                        {PROVIDER_LABELS[integration.provider] || integration.provider}
                        {integration.label ? ` (${integration.label})` : ""}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {activeIntegrations.length === 0 && !loadingIntegrations && (
                <p className="text-sm text-muted-foreground">
                  Brak aktywnych integracji.{" "}
                  <Link href="/integrations" className="text-primary underline">
                    Dodaj integracje
                  </Link>
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="direction">Kierunek synchronizacji</Label>
              <Select
                value={syncDirection}
                onValueChange={(v) =>
                  setSyncDirection(v as "push" | "pull" | "bidirectional")
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="bidirectional">Dwukierunkowa</SelectItem>
                  <SelectItem value="push">Wyslij do marketplace</SelectItem>
                  <SelectItem value="pull">Pobierz z marketplace</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Reguly cenowe</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Regula ceny</Label>
              <Select
                value={priceRule}
                onValueChange={(v) =>
                  setPriceRule(v as "same" | "markup_pct" | "markup_fixed" | "custom")
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="same">Bez zmian (taka sama cena)</SelectItem>
                  <SelectItem value="markup_pct">Narzut procentowy</SelectItem>
                  <SelectItem value="markup_fixed">Narzut stalowy (PLN)</SelectItem>
                  <SelectItem value="custom">Niestandardowa</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {(priceRule === "markup_pct" || priceRule === "markup_fixed") && (
              <div className="space-y-2">
                <Label htmlFor="priceModifier">
                  {priceRule === "markup_pct"
                    ? "Narzut procentowy (%)"
                    : "Narzut stalowy (PLN)"}
                </Label>
                <Input
                  id="priceModifier"
                  type="number"
                  step={priceRule === "markup_pct" ? "0.1" : "0.01"}
                  value={priceModifier}
                  onChange={(e) => setPriceModifier(parseFloat(e.target.value) || 0)}
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="stockBuffer">Bufor magazynowy (szt.)</Label>
              <Input
                id="stockBuffer"
                type="number"
                min={0}
                value={stockBuffer}
                onChange={(e) => setStockBuffer(parseInt(e.target.value) || 0)}
              />
              <p className="text-xs text-muted-foreground">
                Ilosc sztuk odejmowana od dostepnego stanu magazynowego
                (zapas bezpieczenstwa)
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Automatyzacja</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Automatyczna synchronizacja</Label>
                <p className="text-sm text-muted-foreground">
                  Automatycznie synchronizuj oferty w ustalonym interwale
                </p>
              </div>
              <Switch checked={autoSync} onCheckedChange={setAutoSync} />
            </div>

            {autoSync && (
              <div className="space-y-2">
                <Label htmlFor="syncInterval">Interwal synchronizacji (minuty)</Label>
                <Input
                  id="syncInterval"
                  type="number"
                  min={5}
                  max={1440}
                  value={syncInterval}
                  onChange={(e) => setSyncInterval(parseInt(e.target.value) || 30)}
                />
              </div>
            )}
          </CardContent>
        </Card>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={createConfig.isPending || !integrationId}>
            {createConfig.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Utworz konfiguracje
          </Button>
          <Link href="/listing-sync">
            <Button type="button" variant="outline">
              Anuluj
            </Button>
          </Link>
        </div>
      </form>
    </div>
  );
}
