"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ORDER_STATUSES } from "@/lib/constants";

interface OrderStatusChartProps {
  data?: Record<string, number>;
  isLoading: boolean;
}

export function OrderStatusChart({ data, isLoading }: OrderStatusChartProps) {
  const chartData = data
    ? Object.entries(data)
        .filter(([, count]) => count > 0)
        .map(([status, count]) => ({
          name: ORDER_STATUSES[status]?.label ?? status,
          count,
        }))
        .sort((a, b) => b.count - a.count)
    : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Zamówienia według statusu</CardTitle>
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
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" fontSize={12} tickLine={false} axisLine={false} />
              <YAxis fontSize={12} tickLine={false} axisLine={false} />
              <Tooltip />
              <Bar
                dataKey="count"
                fill="#8b5cf6"
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}
