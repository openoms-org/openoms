"use client";

import { useCallback } from "react";
import { useAuthStore } from "@/lib/auth";
import { useOnboarding } from "@/hooks/use-onboarding";
import { useOperationsDashboard } from "@/hooks/use-operations-dashboard";
import { FulfillmentControlTowerPanel } from "@/components/dashboard/fulfillment-control-tower-panel";
import { IntegrationHealthPanel } from "@/components/dashboard/integration-health-panel";
import { OperationalExceptions } from "@/components/dashboard/operational-exceptions";
import { OperationsActivity } from "@/components/dashboard/operations-activity";
import { OperationsSummaryStrip } from "@/components/dashboard/operations-summary-strip";
import { OrchestrationMap } from "@/components/dashboard/orchestration-map";
import { ParityReadinessIndicator } from "@/components/dashboard/parity-readiness-indicator";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { OnboardingWizard } from "@/components/onboarding/onboarding-wizard";
import Link from "next/link";
import {
  AlertTriangle,
  CircleCheck,
  Package,
  RefreshCw,
  Settings,
  ShoppingCart,
  X,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { useHydratedState } from "@/hooks/use-effect-synced-state";
import { isProcessBackedDashboard } from "@/lib/fulfillment-dashboard";

const QUICKSTART_DISMISSED_KEY = "openoms_quickstart_dismissed";

function QuickStartCard() {
  const t = useTranslations("common");
  const { allCompleted } = useOnboarding();
  const readDismissed = useCallback(
    () => localStorage.getItem(QUICKSTART_DISMISSED_KEY) === "true",
    [],
  );
  const [dismissed, setDismissed, hydrated] = useHydratedState(
    false,
    readDismissed,
  );

  if (!hydrated || !allCompleted || dismissed) return null;

  const handleDismiss = () => {
    localStorage.setItem(QUICKSTART_DISMISSED_KEY, "true");
    setDismissed(true);
  };

  return (
    <Card className="border-primary/20 bg-primary/5">
      <CardContent className="py-5">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="font-semibold">{t("quickStart.readyTitle")}</h3>
            <p className="text-sm text-muted-foreground mt-1">
              {t("quickStart.nextSteps")}
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
              {t("empty.addProduct")}
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
  // OPE-423c: build-time cutover flag. Default/unset → "heuristic", i.e. the
  // current production behavior (legacy heuristic section primary). Flip to
  // "process-backed" once the parity gate (OPE-423b) is met to make the
  // process-backed control tower primary. Both code paths are kept (dual-read),
  // so flipping is reversible. NOTE: the legacy hook below is always called
  // (React rules of hooks); in process-backed mode its data simply is not
  // rendered, keeping the legacy path warm and the switch reversible.
  const processBacked = isProcessBackedDashboard();
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
  const hasOperationalIssues =
    exceptions.length > 0 ||
    stages.some((stage) => stage.health !== "ok") ||
    integrationHealth.some((item) => item.health !== "ok");
  let HeaderStatusIcon = CircleCheck;
  let headerStatusClass =
    "inline-flex items-center gap-2 rounded-md border border-success/30 bg-success/10 px-3 py-2 font-medium text-success";
  let headerStatusKey = "operations.syncOk";

  // The header status banner derives from the heuristic model, so only drive it
  // off heuristic state when the heuristic view is primary. In process-backed
  // mode the control tower carries its own per-section status, so the header
  // stays neutral (syncOk) rather than reflecting heuristic loading/issues.
  if (!processBacked && isLoading) {
    HeaderStatusIcon = RefreshCw;
    headerStatusClass =
      "inline-flex items-center gap-2 rounded-md border border-info/30 bg-info/10 px-3 py-2 font-medium text-info";
    headerStatusKey = "operations.statusLoading";
  } else if (!processBacked && hasOperationalIssues) {
    HeaderStatusIcon = AlertTriangle;
    headerStatusClass =
      "inline-flex items-center gap-2 rounded-md border border-warning/40 bg-warning/15 px-3 py-2 font-medium text-warning";
    headerStatusKey = "operations.syncNeedsAttention";
  }

  return (
    <div className="-m-6 min-h-full bg-muted/30 p-4 sm:p-6 lg:p-7">
      <div className="mx-auto max-w-[1600px] space-y-5">
        <div className="flex flex-col gap-4 px-1 py-1 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-normal md:text-3xl">
              {t("operations.title")}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground md:text-base">
              {t("operations.subtitle")}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className={headerStatusClass}>
              <HeaderStatusIcon
                className={isLoading ? "h-4 w-4 animate-spin" : "h-4 w-4"}
                aria-hidden="true"
              />
              {t(headerStatusKey)}
            </span>
            {user?.name && (
              <span className="rounded-md border bg-background px-3 py-2 text-muted-foreground">
                {t("operations.operator", { name: user.name })}
              </span>
            )}
          </div>
        </div>

        {!processBacked && isError && (
          <div className="rounded-md border border-destructive bg-destructive/10 p-4">
            <p className="text-sm text-destructive">{tc("loadError")}</p>
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

        {processBacked ? (
          <>
            {/* OPE-423c: process-backed primary view. The control tower (OPE-419)
                is promoted to the primary operations section and the parity
                readiness indicator shows the cutover verdict. The legacy
                heuristic blocks are intentionally not rendered in this mode, but
                their code path is preserved (dual-read) so the flag is reversible. */}
            <FulfillmentControlTowerPanel />
            <ParityReadinessIndicator />
          </>
        ) : (
          <>
            <OperationsSummaryStrip
              stages={stages}
              exceptions={exceptions}
              integrationHealth={integrationHealth}
              isLoading={isLoading}
            />

            <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_400px]">
              <div className="space-y-5">
                <OrchestrationMap stages={stages} isLoading={isLoading} />
                <OperationsActivity items={activity} isLoading={isLoading} />
              </div>
              <OperationalExceptions
                exceptions={exceptions}
                isLoading={isLoading}
              />
            </div>

            <IntegrationHealthPanel
              items={integrationHealth}
              isLoading={isLoading}
            />

            {/* OPE-419 (additive): process-backed fulfillment control tower.
                Renders intentional empty/zero states while
                FULFILLMENT_PROCESS_ENABLED is off; self-guards on operator role.
                The heuristic dashboard above is unchanged. */}
            <FulfillmentControlTowerPanel />
          </>
        )}

        <div className="space-y-4">
          <OnboardingWizard />
          <QuickStartCard />
        </div>
      </div>
    </div>
  );
}
