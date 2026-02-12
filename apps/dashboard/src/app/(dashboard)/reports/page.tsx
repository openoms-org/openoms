"use client";

import { useTheme } from "next-themes";
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts";
import { format, subDays, eachDayOfInterval } from "date-fns";
import { pl } from "date-fns/locale";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatCurrency } from "@/lib/utils";
import {
  useTopProducts,
  useRevenueBySource,
  useOrderTrends,
  usePaymentMethodStats,
} from "@/hooks/use-reports";

const CHART_COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
];

function RevenueBySourceChart() {
  const { data, isLoading } = useRevenueBySource(30);
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const axisColor = isDark ? "#a1a1aa" : "#71717a";
  const gridColor = isDark ? "#27272a" : "#e4e4e7";

  const chartData = data?.map((d) => ({
    ...d,
    label: d.source || "Nieznane",
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle>Przychód wg źródła (ostatnie 30 dni)</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : !chartData || chartData.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            Brak danych
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
              <XAxis
                dataKey="label"
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
                tickFormatter={(value: number) => formatCurrency(value, "PLN")}
              />
              <Tooltip
                // @ts-expect-error Recharts formatter type mismatch
                formatter={(value: number | string) => [
                  formatCurrency(Number(value), "PLN"),
                  "Przychód",
                ]}
                contentStyle={{
                  backgroundColor: isDark ? "#18181b" : "#ffffff",
                  borderColor: isDark ? "#27272a" : "#e4e4e7",
                  borderRadius: "0.5rem",
                  color: isDark ? "#fafafa" : "#09090b",
                }}
              />
              <Bar dataKey="revenue" radius={[4, 4, 0, 0]}>
                {chartData.map((_, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={CHART_COLORS[index % CHART_COLORS.length]}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}

function fillTrendGaps(data: { date: string; count: number; avg_value: number }[] | undefined, days: number) {
  if (!data) return undefined;
  const dataMap = new Map(data.map((d) => [d.date, d]));
  const today = new Date();
  const allDays = eachDayOfInterval({ start: subDays(today, days - 1), end: today });
  return allDays.map((day) => {
    const key = format(day, "yyyy-MM-dd");
    const entry = dataMap.get(key);
    return {
      date: key,
      count: entry?.count ?? 0,
      avg_value: entry?.avg_value ?? 0,
      label: format(day, "dd MMM", { locale: pl }),
    };
  });
}

function DailyRevenueTrendChart() {
  const { data, isLoading } = useOrderTrends(30);
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const axisColor = isDark ? "#a1a1aa" : "#71717a";
  const gridColor = isDark ? "#27272a" : "#e4e4e7";

  const chartData = fillTrendGaps(data, 30);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Trend zamówień (ostatnie 30 dni)</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : !chartData || chartData.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            Brak danych
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
              <XAxis
                dataKey="label"
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
                tickFormatter={(value: number) => formatCurrency(value, "PLN")}
              />
              <Tooltip
                // @ts-expect-error Recharts formatter type mismatch
                formatter={(value: number | string, name: string) => {
                  const v = Number(value);
                  if (name === "avg_value")
                    return [formatCurrency(v, "PLN"), "Średnia wartość"];
                  return [v, "Liczba"];
                }}
                contentStyle={{
                  backgroundColor: isDark ? "#18181b" : "#ffffff",
                  borderColor: isDark ? "#27272a" : "#e4e4e7",
                  borderRadius: "0.5rem",
                  color: isDark ? "#fafafa" : "#09090b",
                }}
              />
              <Legend
                formatter={(value: string) => {
                  if (value === "avg_value") return "Średnia wartość";
                  return "Liczba zamówień";
                }}
              />
              <Line
                type="monotone"
                dataKey="avg_value"
                stroke={CHART_COLORS[0]}
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}

function TopProductsTable() {
  const { data, isLoading } = useTopProducts(10);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Top produkty</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : !data || data.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            Brak danych
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>SKU</TableHead>
                <TableHead className="text-right">Ilość</TableHead>
                <TableHead className="text-right">Przychód</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((product, index) => (
                <TableRow key={index}>
                  <TableCell className="font-medium">{product.name}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {product.sku || "-"}
                  </TableCell>
                  <TableCell className="text-right">
                    {product.total_quantity}
                  </TableCell>
                  <TableCell className="text-right">
                    {formatCurrency(product.total_revenue, "PLN")}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function OrderTrendsChart() {
  const { data, isLoading } = useOrderTrends(30);
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const axisColor = isDark ? "#a1a1aa" : "#71717a";
  const gridColor = isDark ? "#27272a" : "#e4e4e7";

  const chartData = fillTrendGaps(data, 30);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Trend zamówień</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : !chartData || chartData.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            Brak danych
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
              <XAxis
                dataKey="label"
                fontSize={12}
                tickLine={false}
                axisLine={false}
                tick={{ fill: axisColor }}
              />
              <YAxis
                yAxisId="left"
                fontSize={12}
                tickLine={false}
                axisLine={false}
                tick={{ fill: axisColor }}
              />
              <YAxis
                yAxisId="right"
                orientation="right"
                fontSize={12}
                tickLine={false}
                axisLine={false}
                tick={{ fill: axisColor }}
                tickFormatter={(value: number) => formatCurrency(value, "PLN")}
              />
              <Tooltip
                // @ts-expect-error Recharts formatter type mismatch
                formatter={(value: number | string, name: string) => {
                  const v = Number(value);
                  if (name === "avg_value")
                    return [formatCurrency(v, "PLN"), "Średnia wartość"];
                  return [v, "Liczba zamówień"];
                }}
                contentStyle={{
                  backgroundColor: isDark ? "#18181b" : "#ffffff",
                  borderColor: isDark ? "#27272a" : "#e4e4e7",
                  borderRadius: "0.5rem",
                  color: isDark ? "#fafafa" : "#09090b",
                }}
              />
              <Legend
                formatter={(value: string) => {
                  if (value === "avg_value") return "Średnia wartość";
                  return "Liczba zamówień";
                }}
              />
              <Bar
                yAxisId="left"
                dataKey="count"
                fill={CHART_COLORS[0]}
                radius={[4, 4, 0, 0]}
              />
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="avg_value"
                stroke={CHART_COLORS[1]}
                strokeWidth={2}
                dot={false}
              />
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}

function PaymentMethodChart() {
  const { data, isLoading } = usePaymentMethodStats();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

  const chartData = data
    ? Object.entries(data)
        .filter(([, count]) => count > 0)
        .map(([method, count]) => ({
          name: (method === "unknown" || !method) ? "Nieznana" : method,
          value: count,
        }))
        .sort((a, b) => b.value - a.value)
    : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Metody płatności</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : chartData.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            Brak danych
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={chartData}
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
                {chartData.map((_, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={CHART_COLORS[index % CHART_COLORS.length]}
                  />
                ))}
              </Pie>
              <Tooltip
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
  );
}

export default function ReportsPage() {
  return (
    <AdminGuard>
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Raporty</h1>
        <p className="text-muted-foreground mt-1">
          Szczegółowe statystyki i analizy sprzedaży
        </p>
      </div>

      <Tabs defaultValue="revenue">
        <TabsList>
          <TabsTrigger value="revenue">Przychody</TabsTrigger>
          <TabsTrigger value="products">Produkty</TabsTrigger>
          <TabsTrigger value="trends">Trendy</TabsTrigger>
          <TabsTrigger value="payments">Płatności</TabsTrigger>
        </TabsList>

        <TabsContent value="revenue" className="space-y-6">
          <RevenueBySourceChart />
          <DailyRevenueTrendChart />
        </TabsContent>

        <TabsContent value="products" className="space-y-6">
          <TopProductsTable />
        </TabsContent>

        <TabsContent value="trends" className="space-y-6">
          <OrderTrendsChart />
        </TabsContent>

        <TabsContent value="payments" className="space-y-6">
          <PaymentMethodChart />
        </TabsContent>
      </Tabs>
    </div>
    </AdminGuard>
  );
}
