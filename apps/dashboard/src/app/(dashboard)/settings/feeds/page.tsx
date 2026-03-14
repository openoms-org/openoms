"use client";

import { useState, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
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
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  useFeedSettings,
  useUpdateFeedSettings,
  useRegenerateFeedToken,
} from "@/hooks/use-feed-settings";
import { useProductCategories } from "@/hooks/use-product-categories";
import { useAuthStore } from "@/lib/auth";
import { Loader2, Save, Copy, RefreshCw, ExternalLink } from "lucide-react";
import type { ProductFeedConfig } from "@/types/api";
import { useTranslations } from "next-intl";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const DEFAULT_FEED_CONFIG: ProductFeedConfig = {
  ceneo_enabled: false,
  ceneo_feed_token: "",
  google_enabled: false,
  google_feed_token: "",
  default_currency: "PLN",
  default_shipping_cost: "12.99",
  excluded_categories: [],
  exclude_out_of_stock: false,
};

function buildFeedURL(
  feedType: "ceneo" | "google",
  tenantId: string,
  token: string
): string {
  return `${API_URL}/v1/feeds/${feedType}/${tenantId}/${token}`;
}

export default function FeedSettingsPage() {
  const t = useTranslations("settings");
  const { data: feedSettings, isLoading } = useFeedSettings();
  const updateFeedSettings = useUpdateFeedSettings();
  const regenerateToken = useRegenerateFeedToken();
  const { data: categoriesConfig } = useProductCategories();
  const tenant = useAuthStore((s) => s.tenant);

  const [form, setForm] = useState<ProductFeedConfig>(DEFAULT_FEED_CONFIG);

  useEffect(() => {
    if (feedSettings) {
      setForm({
        ...DEFAULT_FEED_CONFIG,
        ...feedSettings,
        excluded_categories: feedSettings.excluded_categories || [],
      });
    }
  }, [feedSettings]);

  const handleSave = async () => {
    try {
      await updateFeedSettings.mutateAsync(form);
      toast.success("Ustawienia zapisane");
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Nie udalo sie zapisac ustawien";
      toast.error(message);
    }
  };

  const handleRegenerateToken = async (feedType: "ceneo" | "google") => {
    const label = feedType === "ceneo" ? "Ceneo" : "Google Shopping";
    try {
      const result = await regenerateToken.mutateAsync(feedType);
      setForm((prev) => ({
        ...prev,
        ceneo_feed_token: result.ceneo_feed_token,
        google_feed_token: result.google_feed_token,
      }));
      toast.success(`Token ${label} wygenerowany ponownie`);
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : `Nie udalo sie wygenerowac tokena ${label}`;
      toast.error(message);
    }
  };

  const handleCopyURL = useCallback((url: string) => {
    navigator.clipboard.writeText(url).then(
      () => toast.success("URL skopiowany do schowka"),
      () => toast.error("Nie udalo sie skopiowac URL")
    );
  }, []);

  const handleCategoryToggle = (categoryKey: string) => {
    setForm((prev) => {
      const excluded = prev.excluded_categories || [];
      if (excluded.includes(categoryKey)) {
        return {
          ...prev,
          excluded_categories: excluded.filter((c) => c !== categoryKey),
        };
      }
      return {
        ...prev,
        excluded_categories: [...excluded, categoryKey],
      };
    });
  };

  const tenantId = tenant?.id || "";

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const categories = categoriesConfig?.categories || [];

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Feedy produktowe</h1>
          <p className="text-muted-foreground">
            {t("generujPlikiXmlDlaPorownywarekCenI")}
          </p>
        </div>

        {/* Ceneo Feed */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  Ceneo
                  {form.ceneo_enabled && (
                    <Badge variant="default" className="text-xs">
                      Aktywny
                    </Badge>
                  )}
                </CardTitle>
                <CardDescription>
                  {t("feedXmlDlaCeneoplPolskiejPorownywarkiCen")}
                </CardDescription>
              </div>
              <Switch
                checked={form.ceneo_enabled}
                onCheckedChange={(checked) =>
                  setForm({ ...form, ceneo_enabled: checked })
                }
              />
            </div>
          </CardHeader>
          {form.ceneo_enabled && form.ceneo_feed_token && tenantId && (
            <CardContent className="space-y-3">
              <div className="space-y-2">
                <Label className="text-xs text-muted-foreground">
                  URL feeda Ceneo
                </Label>
                <FeedURLField
                  url={buildFeedURL("ceneo", tenantId, form.ceneo_feed_token)}
                  onCopy={() =>
                    handleCopyURL(
                      buildFeedURL("ceneo", tenantId, form.ceneo_feed_token)
                    )
                  }
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleRegenerateToken("ceneo")}
                disabled={regenerateToken.isPending}
              >
                {regenerateToken.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
                Wygeneruj nowy token
              </Button>
            </CardContent>
          )}
          {form.ceneo_enabled && !form.ceneo_feed_token && (
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Zapisz ustawienia, aby wygenerowac token i uzyskac URL feeda.
              </p>
            </CardContent>
          )}
        </Card>

        {/* Google Shopping Feed */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  Google Shopping
                  {form.google_enabled && (
                    <Badge variant="default" className="text-xs">
                      Aktywny
                    </Badge>
                  )}
                </CardTitle>
                <CardDescription>
                  Feed RSS/XML dla Google Merchant Center
                </CardDescription>
              </div>
              <Switch
                checked={form.google_enabled}
                onCheckedChange={(checked) =>
                  setForm({ ...form, google_enabled: checked })
                }
              />
            </div>
          </CardHeader>
          {form.google_enabled && form.google_feed_token && tenantId && (
            <CardContent className="space-y-3">
              <div className="space-y-2">
                <Label className="text-xs text-muted-foreground">
                  URL feeda Google Shopping
                </Label>
                <FeedURLField
                  url={buildFeedURL("google", tenantId, form.google_feed_token)}
                  onCopy={() =>
                    handleCopyURL(
                      buildFeedURL("google", tenantId, form.google_feed_token)
                    )
                  }
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleRegenerateToken("google")}
                disabled={regenerateToken.isPending}
              >
                {regenerateToken.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
                Wygeneruj nowy token
              </Button>
            </CardContent>
          )}
          {form.google_enabled && !form.google_feed_token && (
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Zapisz ustawienia, aby wygenerowac token i uzyskac URL feeda.
              </p>
            </CardContent>
          )}
        </Card>

        {/* General Feed Settings */}
        <Card>
          <CardHeader>
            <CardTitle>{t("ustawieniaOgolne")}</CardTitle>
            <CardDescription>
              {t("parametryWspolneDlaWszystkichFeedow")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Domyslna waluta</Label>
                <Input
                  value={form.default_currency}
                  onChange={(e) =>
                    setForm({ ...form, default_currency: e.target.value })
                  }
                  placeholder="PLN"
                />
              </div>
              <div className="space-y-2">
                <Label>Domyslny koszt wysylki</Label>
                <Input
                  value={form.default_shipping_cost}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      default_shipping_cost: e.target.value,
                    })
                  }
                  placeholder="12.99"
                />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-md border p-4">
              <div>
                <p className="font-medium">Ukryj produkty bez stanu</p>
                <p className="text-sm text-muted-foreground">
                  {t("nieUwzgledniajProduktowZZerowymStanemMagazynowym")}
                </p>
              </div>
              <Switch
                checked={form.exclude_out_of_stock}
                onCheckedChange={(checked) =>
                  setForm({ ...form, exclude_out_of_stock: checked })
                }
              />
            </div>
          </CardContent>
        </Card>

        {/* Category Exclusion */}
        {categories.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Wykluczone kategorie</CardTitle>
              <CardDescription>
                Produkty z zaznaczonych kategorii nie pojawia sie w feedach
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {categories.map((cat) => (
                  <div key={cat.key} className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      id={`exclude-cat-${cat.key}`}
                      checked={(form.excluded_categories || []).includes(
                        cat.key
                      )}
                      onChange={() => handleCategoryToggle(cat.key)}
                      className="h-4 w-4 rounded border-border"
                    />
                    <label
                      htmlFor={`exclude-cat-${cat.key}`}
                      className="flex items-center gap-2 text-sm"
                    >
                      {cat.color && (
                        <span
                          className="inline-block h-3 w-3 rounded-full"
                          style={{ backgroundColor: cat.color }}
                        />
                      )}
                      {cat.label}
                    </label>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        <div className="flex justify-end">
          <Button
            onClick={handleSave}
            disabled={updateFeedSettings.isPending}
          >
            {updateFeedSettings.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Zapisz ustawienia
          </Button>
        </div>
      </div>
    </AdminGuard>
  );
}

function FeedURLField({
  url,
  onCopy,
}: {
  url: string;
  onCopy: () => void;
}) {
  return (
    <div className="flex gap-2">
      <Input
        readOnly
        value={url}
        className="font-mono text-xs"
        onClick={(e) => (e.target as HTMLInputElement).select()}
      />
      <Button variant="outline" size="icon" onClick={onCopy} title="Kopiuj">
        <Copy className="h-4 w-4" />
      </Button>
      <Button
        variant="outline"
        size="icon"
        onClick={() => window.open(url, "_blank")}
        title="Otworz"
      >
        <ExternalLink className="h-4 w-4" />
      </Button>
    </div>
  );
}
