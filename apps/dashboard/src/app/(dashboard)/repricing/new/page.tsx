"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ArrowLeft, TrendingUp, Package, Clock, BarChart3 } from "lucide-react";
import { useCreateRepricingRule } from "@/hooks/use-repricing";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";

const STRATEGIES = [
  {
    value: "margin",
    label: "Marża",
    description: "Ustal ceny na podstawie kosztu + docelowej marży",
    icon: TrendingUp,
  },
  {
    value: "stock_based",
    label: "Stan magazynowy",
    description: "Dostosuj ceny na podstawie poziomu zapasów",
    icon: Package,
  },
  {
    value: "time_based",
    label: "Harmonogram",
    description: "Zaplanuj rabaty wg dnia/godziny",
    icon: Clock,
  },
  {
    value: "competitive",
    label: "Konkurencja",
    description: "Wkrótce - dopasuj ceny do konkurencji",
    icon: BarChart3,
    disabled: true,
  },
];

export default function NewRepricingRulePage() {
  const t = useTranslations("repricing");
  const router = useRouter();
  const createRule = useCreateRepricingRule();

  const [name, setName] = useState("");
  const [strategy, setStrategy] = useState("margin");
  const [scopeType, setScopeType] = useState("all");
  const [scopeValue, setScopeValue] = useState("");
  const [priority, setPriority] = useState(50);
  const [minPrice, setMinPrice] = useState("");
  const [maxPrice, setMaxPrice] = useState("");

  // Margin params
  const [targetMarginPct, setTargetMarginPct] = useState("25");
  const [minMarginPct, setMinMarginPct] = useState("15");
  const [costField, setCostField] = useState("supplier_cost");

  // Stock params
  const [lowStockThreshold, setLowStockThreshold] = useState("5");
  const [lowStockMarkupPct, setLowStockMarkupPct] = useState("10");
  const [highStockThreshold, setHighStockThreshold] = useState("100");
  const [highStockDiscountPct, setHighStockDiscountPct] = useState("15");

  // Time params
  const [timeDays, setTimeDays] = useState("mon,tue,wed,thu,fri");
  const [timeHours, setTimeHours] = useState("");
  const [timeDiscountPct, setTimeDiscountPct] = useState("10");

  const buildParams = () => {
    switch (strategy) {
      case "margin":
        return {
          target_margin_pct: parseFloat(targetMarginPct) || 25,
          min_margin_pct: parseFloat(minMarginPct) || 15,
          cost_field: costField || "supplier_cost",
        };
      case "stock_based":
        return {
          low_stock_threshold: parseInt(lowStockThreshold) || 5,
          low_stock_markup_pct: parseFloat(lowStockMarkupPct) || 10,
          high_stock_threshold: parseInt(highStockThreshold) || 100,
          high_stock_discount_pct: parseFloat(highStockDiscountPct) || 15,
        };
      case "time_based":
        return {
          schedule: [
            {
              days: timeDays,
              hours: timeHours || undefined,
              discount_pct: parseFloat(timeDiscountPct) || 10,
            },
          ],
        };
      default:
        return {};
    }
  };

  const buildScopeValue = () => {
    if (scopeType === "all") return undefined;
    if (scopeType === "product_ids") {
      return scopeValue
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
    }
    return scopeValue;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error(t("nazwaregułyjestwymagana"));
      return;
    }

    try {
      const result = await createRule.mutateAsync({
        name: name.trim(),
        strategy,
        priority,
        scope_type: scopeType,
        scope_value: buildScopeValue(),
        params: buildParams(),
        min_price: minPrice ? parseFloat(minPrice) : undefined,
        max_price: maxPrice ? parseFloat(maxPrice) : undefined,
        channels: ["internal"],
      });
      toast.success(t("regułarepricingzostałautworzona"));
      router.push(`/repricing/${result.id}`);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/repricing">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{t("nowaregułarepricing")}</h1>
          <p className="text-muted-foreground">
            {t("skonfigurujStrategieDynamicznegoUstalaniaCen")}
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Name */}
        <Card>
          <CardHeader>
            <CardTitle>Podstawowe</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t("nazwareguły")}</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("npMarzaNaElektronike")}
              />
            </div>

            <div className="space-y-2">
              <Label>Priorytet ({priority})</Label>
              <p className="text-xs text-muted-foreground">
                Wyższy priorytet = reguła oceniana jako pierwsza
              </p>
              <Slider
                value={[priority]}
                onValueChange={([v]) => setPriority(v)}
                min={0}
                max={100}
                step={1}
              />
            </div>
          </CardContent>
        </Card>

        {/* Strategy */}
        <Card>
          <CardHeader>
            <CardTitle>Strategia</CardTitle>
            <CardDescription>Wybierz algorytm ustalania cen</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {STRATEGIES.map((s) => (
                <button
                  key={s.value}
                  type="button"
                  disabled={s.disabled}
                  onClick={() => !s.disabled && setStrategy(s.value)}
                  className={cn(
                    "flex items-start gap-3 rounded-lg border p-4 text-left transition-colors",
                    strategy === s.value
                      ? "border-primary bg-primary/5"
                      : "border-border hover:border-primary/50",
                    s.disabled && "cursor-not-allowed opacity-50"
                  )}
                >
                  <s.icon className="mt-0.5 h-5 w-5 shrink-0" />
                  <div>
                    <p className="font-medium">
                      {s.label}
                      {s.disabled && (
                        <span className="ml-2 text-xs text-muted-foreground">
                          {t("wkrotce")}
                        </span>
                      )}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {s.description}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Strategy-specific params */}
        {strategy === "margin" && (
          <Card>
            <CardHeader>
              <CardTitle>{t("parametryMarzy")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="targetMargin">{t("docelowaMarza")}</Label>
                  <Input
                    id="targetMargin"
                    type="number"
                    step="0.1"
                    value={targetMarginPct}
                    onChange={(e) => setTargetMarginPct(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="minMargin">{t("minimalnaMarza")}</Label>
                  <Input
                    id="minMargin"
                    type="number"
                    step="0.1"
                    value={minMarginPct}
                    onChange={(e) => setMinMarginPct(e.target.value)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="costField">Pole kosztu w metadanych</Label>
                <Input
                  id="costField"
                  value={costField}
                  onChange={(e) => setCostField(e.target.value)}
                  placeholder="supplier_cost"
                />
                <p className="text-xs text-muted-foreground">
                  {t("nazwaPolaWMetadanychProduktuZawierajacegoKoszt")}
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {strategy === "stock_based" && (
          <Card>
            <CardHeader>
              <CardTitle>Parametry magazynowe</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>{t("progNiskiegoStanu")}</Label>
                  <Input
                    type="number"
                    value={lowStockThreshold}
                    onChange={(e) => setLowStockThreshold(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Narzut przy niskim stanie (%)</Label>
                  <Input
                    type="number"
                    step="0.1"
                    value={lowStockMarkupPct}
                    onChange={(e) => setLowStockMarkupPct(e.target.value)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>{t("progWysokiegoStanu")}</Label>
                  <Input
                    type="number"
                    value={highStockThreshold}
                    onChange={(e) => setHighStockThreshold(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Rabat przy wysokim stanie (%)</Label>
                  <Input
                    type="number"
                    step="0.1"
                    value={highStockDiscountPct}
                    onChange={(e) => setHighStockDiscountPct(e.target.value)}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {strategy === "time_based" && (
          <Card>
            <CardHeader>
              <CardTitle>Parametry harmonogramu</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Dni tygodnia (oddzielone przecinkami)</Label>
                <Input
                  value={timeDays}
                  onChange={(e) => setTimeDays(e.target.value)}
                  placeholder="mon,tue,wed,thu,fri"
                />
                <p className="text-xs text-muted-foreground">
                  {t("skrotyMonTueWedThuFriSat")}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Godziny (opcjonalne, np. 18-23)</Label>
                  <Input
                    value={timeHours}
                    onChange={(e) => setTimeHours(e.target.value)}
                    placeholder="18-23"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Rabat (%)</Label>
                  <Input
                    type="number"
                    step="0.1"
                    value={timeDiscountPct}
                    onChange={(e) => setTimeDiscountPct(e.target.value)}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Scope */}
        <Card>
          <CardHeader>
            <CardTitle>{t("zakresProduktow")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Typ zakresu</Label>
              <Select value={scopeType} onValueChange={setScopeType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Wszystkie produkty</SelectItem>
                  <SelectItem value="category">Kategoria</SelectItem>
                  <SelectItem value="tag">Tag</SelectItem>
                  <SelectItem value="product_ids">Wybrane produkty (ID)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {scopeType !== "all" && (
              <div className="space-y-2">
                <Label>
                  {scopeType === "category"
                    ? "Nazwa kategorii"
                    : scopeType === "tag"
                      ? "Tag"
                      : t("idProduktowOddzielonePrzecinkami")}
                </Label>
                <Input
                  value={scopeValue}
                  onChange={(e) => setScopeValue(e.target.value)}
                  placeholder={
                    scopeType === "product_ids"
                      ? "uuid1, uuid2, uuid3"
                      : "nazwa"
                  }
                />
              </div>
            )}
          </CardContent>
        </Card>

        {/* Price bounds */}
        <Card>
          <CardHeader>
            <CardTitle>Ograniczenia cenowe</CardTitle>
            <CardDescription>
              Opcjonalne minimalne i maksymalne ceny (zabezpieczenie)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Cena minimalna (PLN)</Label>
                <Input
                  type="number"
                  step="0.01"
                  value={minPrice}
                  onChange={(e) => setMinPrice(e.target.value)}
                  placeholder="Brak limitu"
                />
              </div>
              <div className="space-y-2">
                <Label>Cena maksymalna (PLN)</Label>
                <Input
                  type="number"
                  step="0.01"
                  value={maxPrice}
                  onChange={(e) => setMaxPrice(e.target.value)}
                  placeholder="Brak limitu"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Submit */}
        <div className="flex items-center gap-2">
          <Button type="submit" disabled={createRule.isPending}>
            {createRule.isPending ? "Tworzenie..." : t("utworzregułe")}
          </Button>
          <Button
            variant="outline"
            type="button"
            onClick={() => router.push("/repricing")}
          >
            Anuluj
          </Button>
        </div>
      </form>
    </div>
  );
}
