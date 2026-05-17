"use client";

import { Package, PackagePlus, Truck, PackageCheck } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { OrderCounts } from "@/types/api";
import { useTranslations } from "next-intl";

interface StatCardsProps {
  orderCounts?: OrderCounts;
  isLoading: boolean;
}

interface StatCardProps {
  title: string;
  value?: number;
  icon: React.ReactNode;
  isLoading: boolean;
}

function StatCard({ title, value, icon, isLoading }: StatCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium text-muted-foreground flex items-center justify-between">
          {title}
          {icon}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-8 w-20" />
        ) : (
          <p className="text-3xl font-bold">{value ?? 0}</p>
        )}
      </CardContent>
    </Card>
  );
}

export function StatCards({ orderCounts, isLoading }: StatCardsProps) {
  const t = useTranslations("dashboard");
  return (
    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        title={t("stats.totalOrders")}
        value={orderCounts?.total}
        icon={<Package className="h-4 w-4 text-muted-foreground" />}
        isLoading={isLoading}
      />
      <StatCard
        title={t("stats.new")}
        value={orderCounts?.by_status.new}
        icon={<PackagePlus className="h-4 w-4 text-muted-foreground" />}
        isLoading={isLoading}
      />
      <StatCard
        title={t("stats.inTransit")}
        value={orderCounts?.by_status.in_transit}
        icon={<Truck className="h-4 w-4 text-muted-foreground" />}
        isLoading={isLoading}
      />
      <StatCard
        title={t("stats.delivered")}
        value={orderCounts?.by_status.delivered}
        icon={<PackageCheck className="h-4 w-4 text-muted-foreground" />}
        isLoading={isLoading}
      />
    </div>
  );
}
