"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  TrendingUp,
  TrendingDown,
  Minus,
  AlertTriangle,
  ShoppingCart,
  Package,
  Clock,
  Settings2,
  ChevronRight,
  Info,
} from "lucide-react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Tooltip as UITooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { formatCurrency } from "@/lib/utils";
import {
  useReorderRecommendations,
  useProductVelocity,
  useForecastConfig,
  useUpdateForecastConfig,
} from "@/hooks/use-forecast";
import type { ForecastConfig, ReorderRecommendation } from "@/types/api";

const CHART_COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
];

const ABC_COLORS: Record<string, string> = {
  A: "#22c55e",
  B: "#eab308",
  C: "#6b7280",
};

function urgencyBadge(urgency: string) {
  switch (urgency) {
    case "critical":
      return <Badge variant="destructive">Krytyczny</Badge>;
    case "soon":
      return (
        <Badge className="bg-yellow-500/15 text-yellow-700 dark:text-yellow-400 border-yellow-500/30">
          Wkrotce
        </Badge>
      );
    case "planned":
      return (
        <Badge className="bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/30">
          Planowany
        </Badge>
      );
    default:
      return <Badge variant="secondary">{urgency}</Badge>;
  }
}

function SummaryCards({
  recommendations,
}: {
  recommendations: ReorderRecommendation[] | undefined;
}) {
  if (!recommendations) return null;

  const critical = recommendations.filter((r) => r.urgency === "critical").length;
  const soon = recommendations.filter((r) => r.urgency === "soon").length;
  const planned = recommendations.filter((r) => r.urgency === "planned").length;

  const avgDaysOfStock =
    recommendations.length > 0
      ? recommendations.reduce((s, r) => s + r.days_until_stockout, 0) /
        recommendations.length
      : 0;

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-red-500/10 p-2.5">
              <AlertTriangle className="h-5 w-5 text-red-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Krytyczne</p>
              <p className="text-2xl font-bold">{critical}</p>
            </div>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-yellow-500/10 p-2.5">
              <Clock className="h-5 w-5 text-yellow-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Wkrotce</p>
              <p className="text-2xl font-bold">{soon}</p>
            </div>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-2.5">
              <ShoppingCart className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Planowane</p>
              <p className="text-2xl font-bold">{planned}</p>
            </div>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-muted p-2.5">
              <Package className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">
                Srednia dni do wyczerpania
              </p>
              <p className="text-2xl font-bold">{avgDaysOfStock.toFixed(1)}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ReorderTable() {
  const router = useRouter();
  const { data: recommendations, isLoading } = useReorderRecommendations();

  if (isLoading) return <Skeleton className="h-[400px] w-full" />;
  if (!recommendations || recommendations.length === 0) {
    return (
      <Card>
        <CardContent className="flex h-[200px] items-center justify-center text-muted-foreground">
          Brak rekomendacji dostawy - wszystkie produkty maja wystarczajacy stan
          magazynowy.
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Rekomendacje dostawy
          <TooltipProvider>
            <UITooltip>
              <TooltipTrigger>
                <Info className="h-4 w-4 text-muted-foreground" />
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">
                Rekomendacje oparte na prognozowanym popycie, aktualnym stanie
                magazynowym i zdefiniowanym czasie realizacji dostawy.
              </TooltipContent>
            </UITooltip>
          </TooltipProvider>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Produkt</TableHead>
              <TableHead>SKU</TableHead>
              <TableHead className="text-right">Stan</TableHead>
              <TableHead className="text-right">Prognoza</TableHead>
              <TableHead className="text-right">Dni do wyczerpania</TableHead>
              <TableHead className="text-right">Zalecana ilosc</TableHead>
              <TableHead>Dostawca</TableHead>
              <TableHead className="text-right">Szac. koszt</TableHead>
              <TableHead>Pilnosc</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {recommendations.map((rec) => (
              <TableRow key={rec.product_id}>
                <TableCell className="font-medium max-w-[200px] truncate">
                  <button
                    className="text-left hover:underline"
                    onClick={() => router.push(`/reports/forecast/${rec.product_id}`)}
                  >
                    {rec.product_name}
                  </button>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {rec.sku || "-"}
                </TableCell>
                <TableCell className="text-right">{rec.current_stock}</TableCell>
                <TableCell className="text-right">
                  {rec.forecasted_demand.toFixed(0)}
                </TableCell>
                <TableCell className="text-right">
                  {rec.days_until_stockout.toFixed(1)}
                </TableCell>
                <TableCell className="text-right font-medium">
                  {rec.recommended_qty}
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {rec.supplier_name || "-"}
                </TableCell>
                <TableCell className="text-right">
                  {rec.estimated_cost > 0
                    ? formatCurrency(rec.estimated_cost, "PLN")
                    : "-"}
                </TableCell>
                <TableCell>{urgencyBadge(rec.urgency)}</TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      const params = new URLSearchParams();
                      if (rec.supplier_name) {
                        params.set("supplier_name", rec.supplier_name);
                      }
                      if (rec.supplier_id) {
                        params.set("supplier_id", rec.supplier_id);
                      }
                      router.push(`/purchase-orders/new?${params.toString()}`);
                    }}
                  >
                    Zamow
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function VelocityChart() {
  const { data: velocity, isLoading } = useProductVelocity();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const axisColor = isDark ? "#a1a1aa" : "#71717a";
  const gridColor = isDark ? "#27272a" : "#e4e4e7";

  if (isLoading) return <Skeleton className="h-[400px] w-full" />;
  if (!velocity || velocity.length === 0) {
    return (
      <Card>
        <CardContent className="flex h-[200px] items-center justify-center text-muted-foreground">
          Brak danych o predkosci sprzedazy
        </CardContent>
      </Card>
    );
  }

  const top20 = velocity.slice(0, 20);
  const chartData = top20.map((v) => ({
    name: v.product_name.length > 20 ? v.product_name.slice(0, 20) + "..." : v.product_name,
    revenue: v.total_revenue,
    abcClass: v.abc_class,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Predkosc sprzedazy (top 20)
          <div className="flex gap-2 ml-auto text-sm font-normal">
            <span className="flex items-center gap-1">
              <span className="inline-block h-3 w-3 rounded-sm" style={{ backgroundColor: ABC_COLORS.A }} />
              Klasa A (80% przychodu)
            </span>
            <span className="flex items-center gap-1">
              <span className="inline-block h-3 w-3 rounded-sm" style={{ backgroundColor: ABC_COLORS.B }} />
              Klasa B (15%)
            </span>
            <span className="flex items-center gap-1">
              <span className="inline-block h-3 w-3 rounded-sm" style={{ backgroundColor: ABC_COLORS.C }} />
              Klasa C (5%)
            </span>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={400}>
          <BarChart data={chartData} layout="vertical" margin={{ left: 120 }}>
            <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
            <XAxis
              type="number"
              fontSize={12}
              tickLine={false}
              axisLine={false}
              tick={{ fill: axisColor }}
              tickFormatter={(value: number) => formatCurrency(value, "PLN")}
            />
            <YAxis
              type="category"
              dataKey="name"
              fontSize={11}
              tickLine={false}
              axisLine={false}
              tick={{ fill: axisColor }}
              width={120}
            />
            <Tooltip
              // @ts-expect-error Recharts formatter type mismatch
              formatter={(value: number) => [formatCurrency(value, "PLN"), "Przychod"]}
              contentStyle={{
                backgroundColor: isDark ? "#18181b" : "#ffffff",
                borderColor: isDark ? "#27272a" : "#e4e4e7",
                borderRadius: "0.5rem",
                color: isDark ? "#fafafa" : "#09090b",
              }}
            />
            <Bar dataKey="revenue" radius={[0, 4, 4, 0]}>
              {chartData.map((entry, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={ABC_COLORS[entry.abcClass] || ABC_COLORS.C}
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

function ConfigDialog() {
  const { data: config, isLoading } = useForecastConfig();
  const updateConfig = useUpdateForecastConfig();
  const [localConfig, setLocalConfig] = useState<ForecastConfig | null>(null);
  const [open, setOpen] = useState(false);

  const handleOpen = (isOpen: boolean) => {
    setOpen(isOpen);
    if (isOpen && config) {
      setLocalConfig({ ...config });
    }
  };

  const handleSave = () => {
    if (!localConfig) return;
    updateConfig.mutate(localConfig, {
      onSuccess: () => setOpen(false),
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Settings2 className="h-4 w-4 mr-2" />
          Konfiguracja
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Konfiguracja prognozy</DialogTitle>
        </DialogHeader>
        {isLoading || !localConfig ? (
          <Skeleton className="h-[200px] w-full" />
        ) : (
          <div className="space-y-4">
            <div>
              <Label htmlFor="lead_time">
                Domyslny czas realizacji dostawy (dni)
              </Label>
              <Input
                id="lead_time"
                type="number"
                min={1}
                max={365}
                value={localConfig.default_lead_time_days}
                onChange={(e) =>
                  setLocalConfig({
                    ...localConfig,
                    default_lead_time_days: parseInt(e.target.value) || 14,
                  })
                }
              />
            </div>
            <div>
              <Label htmlFor="safety_stock">
                Zapas bezpieczenstwa (dni)
              </Label>
              <Input
                id="safety_stock"
                type="number"
                min={0}
                max={365}
                value={localConfig.safety_stock_days}
                onChange={(e) =>
                  setLocalConfig({
                    ...localConfig,
                    safety_stock_days: parseInt(e.target.value) || 7,
                  })
                }
              />
            </div>
            <div>
              <Label htmlFor="forecast_horizon">
                Horyzont prognozy (dni)
              </Label>
              <Input
                id="forecast_horizon"
                type="number"
                min={1}
                max={365}
                value={localConfig.forecast_days_ahead}
                onChange={(e) =>
                  setLocalConfig({
                    ...localConfig,
                    forecast_days_ahead: parseInt(e.target.value) || 30,
                  })
                }
              />
            </div>
            <Button
              onClick={handleSave}
              disabled={updateConfig.isPending}
              className="w-full"
            >
              {updateConfig.isPending ? "Zapisywanie..." : "Zapisz"}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default function ForecastPage() {
  const { data: recommendations } = useReorderRecommendations();

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Prognoza popytu</h1>
            <p className="text-muted-foreground mt-1">
              Analiza predkosc sprzedazy, prognoza popytu i rekomendacje
              uzupelnienia magazynu
            </p>
          </div>
          <ConfigDialog />
        </div>

        <SummaryCards recommendations={recommendations} />

        <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 p-4 text-sm text-muted-foreground">
          <div className="flex items-start gap-2">
            <Info className="h-4 w-4 mt-0.5 text-blue-500 shrink-0" />
            <div>
              Prognozy oparte na analizie statystycznej historii zamowien z
              ostatnich 90 dni. Uwzgledniaja trend sprzedazy i sezonowosc
              tygodniowa. Dokladnosc rosnie wraz z iloscia danych historycznych.
            </div>
          </div>
        </div>

        <ReorderTable />

        <VelocityChart />
      </div>
    </AdminGuard>
  );
}
