"use client";

import { useDashboardStats } from "@/hooks/use-dashboard-stats";
import { StatCards } from "@/components/dashboard/stat-cards";
import { RevenueChart } from "@/components/dashboard/revenue-chart";
import { OrderStatusChart } from "@/components/dashboard/order-status-chart";
import { OrderSourceChart } from "@/components/dashboard/order-source-chart";
import { RecentOrdersTable } from "@/components/dashboard/recent-orders-table";

export default function DashboardPage() {
  const { data: stats, isLoading } = useDashboardStats();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Panel główny</h1>
        <p className="text-muted-foreground mt-1">Przegląd zamówień w systemie</p>
      </div>

      <StatCards orderCounts={stats?.order_counts} isLoading={isLoading} />

      <RevenueChart
        data={stats?.revenue.daily}
        currency={stats?.revenue.currency}
        isLoading={isLoading}
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <OrderStatusChart data={stats?.order_counts.by_status} isLoading={isLoading} />
        <OrderSourceChart data={stats?.order_counts.by_source} isLoading={isLoading} />
      </div>

      <RecentOrdersTable orders={stats?.recent_orders} isLoading={isLoading} />
    </div>
  );
}
