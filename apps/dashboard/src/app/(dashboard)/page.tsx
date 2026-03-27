"use client";

import { useEffect, useState } from "react";
import { useDashboardStats } from "@/hooks/use-dashboard-stats";
import { useAuthStore } from "@/lib/auth";
import { useOnboarding } from "@/hooks/use-onboarding";
import { StatCards } from "@/components/dashboard/stat-cards";
import { RevenueChart } from "@/components/dashboard/revenue-chart";
import { OrderStatusChart } from "@/components/dashboard/order-status-chart";
import { OrderSourceChart } from "@/components/dashboard/order-source-chart";
import { RecentOrdersTable } from "@/components/dashboard/recent-orders-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { OnboardingWizard } from "@/components/onboarding/onboarding-wizard";
import Link from "next/link";
import { ShoppingCart, Package, Settings, X } from "lucide-react";
import { useTranslations } from "next-intl";

const QUICKSTART_DISMISSED_KEY = "openoms_quickstart_dismissed";

function QuickStartCard() {
  const t = useTranslations("common");
  const { allCompleted } = useOnboarding();
  const [dismissed, setDismissed] = useState<boolean | null>(null);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- SSR hydration guard: read localStorage after mount to avoid mismatch
    setDismissed(localStorage.getItem(QUICKSTART_DISMISSED_KEY) === "true");
  }, []);

  if (dismissed === null || !allCompleted || dismissed) return null;

  const handleDismiss = () => {
    localStorage.setItem(QUICKSTART_DISMISSED_KEY, "true");
    setDismissed(true);
  };

  return (
    <Card className="border-primary/20 bg-primary/5">
      <CardContent className="py-5">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="font-semibold">Twoje konto jest gotowe!</h3>
            <p className="text-sm text-muted-foreground mt-1">
              {t("otoCoMozeszZrobicDalej")}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleDismiss}
            className="shrink-0"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
        <div className="mt-4 flex flex-wrap gap-3">
          <Button variant="outline" size="sm" asChild>
            <Link href="/orders/new">
              <ShoppingCart className="mr-2 h-4 w-4" />
              {t("empty.addOrder")}
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href="/products/new">
              <Package className="mr-2 h-4 w-4" />
              Dodaj produkt
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href="/integrations">
              <Settings className="mr-2 h-4 w-4" />
              {t("empty.connectAllegro")}
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  const t = useTranslations("common");
  const { data: stats, isLoading, isError, refetch } = useDashboardStats();
  const user = useAuthStore((s) => s.user);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("mainPanel")}</h1>
        {user?.name && (
          <p className="text-muted-foreground mt-1">Witaj, {user.name}!</p>
        )}
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("retry")}
          </Button>
        </div>
      )}

      <OnboardingWizard />
      <QuickStartCard />

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
