"use client";

import { useDashboardStats } from "@/hooks/use-dashboard-stats";
import { useAuthStore } from "@/lib/auth";
import { StatCards } from "@/components/dashboard/stat-cards";
import { RevenueChart } from "@/components/dashboard/revenue-chart";
import { OrderStatusChart } from "@/components/dashboard/order-status-chart";
import { OrderSourceChart } from "@/components/dashboard/order-source-chart";
import { RecentOrdersTable } from "@/components/dashboard/recent-orders-table";
import { Button } from "@/components/ui/button";

export default function DashboardPage() {
  const { data: stats, isLoading, isError, refetch } = useDashboardStats();
  const user = useAuthStore((s) => s.user);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Panel główny</h1>
        {user?.name && (
          <p className="text-muted-foreground mt-1">Witaj, {user.name}!</p>
        )}
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

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
