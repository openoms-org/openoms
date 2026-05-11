"use client";

import { useEffect, useState } from "react";
import { useAuthStore } from "@/lib/auth";
import { useOnboarding } from "@/hooks/use-onboarding";
import { useOperationsDashboard } from "@/hooks/use-operations-dashboard";
import { IntegrationHealthPanel } from "@/components/dashboard/integration-health-panel";
import { OperationalExceptions } from "@/components/dashboard/operational-exceptions";
import { OperationsActivity } from "@/components/dashboard/operations-activity";
import { OrchestrationMap } from "@/components/dashboard/orchestration-map";
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
            <Link href="/marketplaces/new">
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
  const t = useTranslations("dashboard");
  const tc = useTranslations("common");
  const {
    stages,
    exceptions,
    integrationHealth,
    activity,
    isLoading,
    isError,
    refetch,
  } = useOperationsDashboard();
  const user = useAuthStore((s) => s.user);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("operations.title")}</h1>
          <p className="text-muted-foreground mt-1">{t("operations.subtitle")}</p>
        </div>
        {user?.name && (
          <p className="text-sm text-muted-foreground">Operator: {user.name}</p>
        )}
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {tc("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {tc("retry")}
          </Button>
        </div>
      )}

      <OnboardingWizard />
      <QuickStartCard />

      <OrchestrationMap stages={stages} isLoading={isLoading} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <OperationalExceptions exceptions={exceptions} isLoading={isLoading} />
        <IntegrationHealthPanel items={integrationHealth} isLoading={isLoading} />
      </div>

      <OperationsActivity items={activity} isLoading={isLoading} />
    </div>
  );
}
