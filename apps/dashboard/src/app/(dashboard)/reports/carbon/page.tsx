"use client";

import { useState, useCallback } from "react";
import { useTheme } from "next-themes";
import { format, subMonths } from "date-fns";
import { Leaf, Download, TrendingDown } from "lucide-react";
import {
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { useTranslations } from "next-intl";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useCarbonStats, useDownloadCarbonReport } from "@/hooks/use-carbon";
import { SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";

const CHART_COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
  "#22c55e",
  "#14b8a6",
  "#06b6d4",
];

export default function CarbonReportPage() {
  const t = useTranslations("reports");
  const [dateFrom, setDateFrom] = useState(
    format(subMonths(new Date(), 3), "yyyy-MM-dd")
  );
  const [dateTo, setDateTo] = useState(format(new Date(), "yyyy-MM-dd"));

  const { data: stats, isLoading } = useCarbonStats(dateFrom, dateTo);
  const downloadReport = useDownloadCarbonReport();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const axisColor = isDark ? "#a1a1aa" : "#71717a";
  const gridColor = isDark ? "#27272a" : "#e4e4e7";

  const handleDownload = useCallback(() => {
    downloadReport(dateFrom, dateTo);
  }, [downloadReport, dateFrom, dateTo]);

  // Calculate InPost savings tip
  const inpostStats = stats?.by_carrier.find(
    (c) => c.carrier === "inpost"
  );
  const otherCarriers = stats?.by_carrier.filter(
    (c) => c.carrier !== "inpost"
  );
  const avgOtherCarbon =
    otherCarriers && otherCarriers.length > 0
      ? otherCarriers.reduce((sum, c) => sum + c.avg_carbon_kg, 0) /
        otherCarriers.length
      : 0;
  const savingsPercent =
    inpostStats && avgOtherCarbon > 0
      ? Math.round(
          ((avgOtherCarbon - inpostStats.avg_carbon_kg) / avgOtherCarbon) * 100
        )
      : null;

  // Monthly trend in chronological order
  const monthlyData = stats?.monthly_trend
    ? [...stats.monthly_trend].reverse()
    : [];

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Leaf className="h-6 w-6 text-green-600" />
              {t("carbon.title")}
            </h1>
            <p className="text-muted-foreground mt-1">
              {t("carbon.subtitle")}
            </p>
          </div>
          <Button variant="outline" onClick={handleDownload}>
            <Download className="h-4 w-4" />
            {t("carbon.downloadCsv")}
          </Button>
        </div>

        {/* Date range filter */}
        <Card>
          <CardContent className="py-4">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <label htmlFor="date-from" className="text-sm font-medium">
                  {t("carbon.dateFrom")}
                </label>
                <Input
                  id="date-from"
                  type="date"
                  value={dateFrom}
                  onChange={(e) => setDateFrom(e.target.value)}
                  className="w-auto"
                />
              </div>
              <div className="flex items-center gap-2">
                <label htmlFor="date-to" className="text-sm font-medium">
                  {t("carbon.dateTo")}
                </label>
                <Input
                  id="date-to"
                  type="date"
                  value={dateTo}
                  onChange={(e) => setDateTo(e.target.value)}
                  className="w-auto"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Summary cards */}
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-4">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-28 w-full" />
            ))}
          </div>
        ) : stats ? (
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {t("carbon.totalCo2")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats.total_carbon_kg.toFixed(1)} kg
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("carbon.fromShipments", { count: stats.total_shipments })}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {t("carbon.avgPerShipment")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats.avg_carbon_per_shipment.toFixed(3)} kg
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("carbon.co2PerShipment")}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {t("carbon.totalDistance")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats.total_distance_km.toFixed(0)} km
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("carbon.estimatedDeliveryDistance")}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {t("carbon.shipmentCount")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{stats.total_shipments}</p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("carbon.withEmissionData")}
                </p>
              </CardContent>
            </Card>
          </div>
        ) : null}

        {/* Charts */}
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Monthly trend */}
          <Card>
            <CardHeader>
              <CardTitle>{t("carbon.monthlyTrend")}</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-[300px] w-full" />
              ) : !monthlyData || monthlyData.length === 0 ? (
                <div className="flex h-[300px] items-center justify-center text-muted-foreground">
                  {t("carbon.noData")}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={monthlyData}>
                    <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
                    <XAxis
                      dataKey="month"
                      fontSize={12}
                      tickLine={false}
                      axisLine={false}
                      tick={{ fill: axisColor }}
                    />
                    <YAxis
                      fontSize={12}
                      tickLine={false}
                      axisLine={false}
                      tick={{ fill: axisColor }}
                      tickFormatter={(v: number) => `${v.toFixed(1)} kg`}
                    />
                    <Tooltip
                      // @ts-expect-error Recharts formatter type mismatch
                      formatter={(value: number) => [
                        `${value.toFixed(3)} kg CO2`,
                        t("carbon.tooltipEmission"),
                      ]}
                      contentStyle={{
                        backgroundColor: isDark ? "#18181b" : "#ffffff",
                        borderColor: isDark ? "#27272a" : "#e4e4e7",
                        borderRadius: "0.5rem",
                        color: isDark ? "#fafafa" : "#09090b",
                      }}
                    />
                    <Bar
                      dataKey="total_carbon_kg"
                      fill="#22c55e"
                      radius={[4, 4, 0, 0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          {/* By carrier pie chart */}
          <Card>
            <CardHeader>
              <CardTitle>{t("carbon.co2ByCarrier")}</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-[300px] w-full" />
              ) : !stats?.by_carrier || stats.by_carrier.length === 0 ? (
                <div className="flex h-[300px] items-center justify-center text-muted-foreground">
                  {t("carbon.noData")}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={stats.by_carrier.map((c) => ({
                        name:
                          SHIPMENT_PROVIDER_LABELS[c.carrier] ??
                          c.carrier.toUpperCase(),
                        value: c.total_carbon_kg,
                        shipments: c.shipments,
                      }))}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={100}
                      paddingAngle={4}
                      dataKey="value"
                      nameKey="name"
                      // eslint-disable-next-line @typescript-eslint/no-explicit-any
                      label={((props: any) =>
                        `${props.name ?? ""} (${((props.percent ?? 0) * 100).toFixed(0)}%)`
                      ) as any}
                    >
                      {stats.by_carrier.map((_, index) => (
                        <Cell
                          key={`cell-${index}`}
                          fill={CHART_COLORS[index % CHART_COLORS.length]}
                        />
                      ))}
                    </Pie>
                    <Tooltip
                      // @ts-expect-error Recharts formatter type mismatch
                      formatter={(value: number) => [
                        `${value.toFixed(3)} kg CO2`,
                        t("carbon.tooltipEmission"),
                      ]}
                      contentStyle={{
                        backgroundColor: isDark ? "#18181b" : "#ffffff",
                        borderColor: isDark ? "#27272a" : "#e4e4e7",
                        borderRadius: "0.5rem",
                        color: isDark ? "#fafafa" : "#09090b",
                      }}
                    />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Carrier breakdown table */}
        {stats?.by_carrier && stats.by_carrier.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>{t("carbon.carrierDetails")}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-2 pr-4 font-medium">
                        {t("carbon.colCarrier")}
                      </th>
                      <th className="text-right py-2 px-4 font-medium">
                        {t("carbon.colShipments")}
                      </th>
                      <th className="text-right py-2 px-4 font-medium">
                        {t("carbon.colTotalCo2")}
                      </th>
                      <th className="text-right py-2 pl-4 font-medium">
                        {t("carbon.colAvgCo2")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {stats.by_carrier.map((c) => (
                      <tr key={c.carrier} className="border-b last:border-0">
                        <td className="py-2 pr-4 font-medium">
                          {SHIPMENT_PROVIDER_LABELS[c.carrier] ??
                            c.carrier.toUpperCase()}
                        </td>
                        <td className="text-right py-2 px-4">{c.shipments}</td>
                        <td className="text-right py-2 px-4">
                          {c.total_carbon_kg.toFixed(3)}
                        </td>
                        <td className="text-right py-2 pl-4">
                          {c.avg_carbon_kg.toFixed(3)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Eco tip */}
        {savingsPercent != null && savingsPercent > 0 && (
          <Card className="border-green-200 bg-green-50/50 dark:border-green-900 dark:bg-green-950/20">
            <CardContent className="py-4 flex items-start gap-3">
              <TrendingDown className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5 shrink-0" />
              <div>
                <p className="text-sm font-medium text-green-800 dark:text-green-300">
                  {t("carbon.ecoTipTitle")}
                </p>
                <p className="text-sm text-green-700 dark:text-green-400 mt-1">
                  {t("carbon.ecoTipText", { percent: savingsPercent })}
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Disclaimer */}
        <p className="text-xs text-muted-foreground text-center">
          {t("carbon.disclaimer")}
        </p>
      </div>
    </AdminGuard>
  );
}
