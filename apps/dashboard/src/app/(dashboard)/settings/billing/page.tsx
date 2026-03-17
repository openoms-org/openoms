"use client";

import { useState, useEffect } from "react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSubscription } from "@/hooks/use-billing";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  trialing: "secondary",
  active: "default",
  past_due: "destructive",
  canceled: "destructive",
  suspended: "destructive",
};

function computeDaysLeft(trialEnd: string): number {
  return Math.max(
    0,
    Math.ceil(
      (new Date(trialEnd).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
    )
  );
}

export default function BillingSettingsPage() {
  const t = useTranslations("subscription");
  const { data: subscription, isLoading } = useSubscription();
  const [daysLeft, setDaysLeft] = useState<number | null>(null);

  const STATUS_LABELS: Record<string, string> = {
    trialing: t("statusTrialing"),
    active: t("statusActive"),
    past_due: t("statusPastDue"),
    canceled: t("statusCanceled"),
    suspended: t("statusSuspended"),
  };

  const INTERVAL_LABELS: Record<string, string> = {
    month: t("intervalMonth"),
    year: t("intervalYear"),
  };

  useEffect(() => {
    if (subscription?.status === "trialing" && subscription.trial_end) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setDaysLeft(computeDaysLeft(subscription.trial_end));
    }
  }, [subscription?.status, subscription?.trial_end]);

  if (isLoading) {
    return (
      <AdminGuard>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </AdminGuard>
    );
  }

  if (!subscription) {
    return (
      <AdminGuard>
        <div className="mx-auto max-w-4xl space-y-6">
          <div>
            <h1 className="text-2xl font-bold">{t("title")}</h1>
            <p className="text-muted-foreground">{t("zarzadzajPlanemIPłatnosciami")}</p>
          </div>
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              {t("nieUdałoSieZaładowacDanychSubskrypcji")}
            </CardContent>
          </Card>
        </div>
      </AdminGuard>
    );
  }

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-muted-foreground">{t("zarzadzajPlanemIPłatnosciami")}</p>
        </div>

        {/* Plan & Status */}
        <Card>
          <CardHeader>
            <CardTitle>{t("twojPlan")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <span className="text-lg font-semibold capitalize">
                {subscription.plan}
              </span>
              <Badge variant={STATUS_VARIANTS[subscription.status] ?? "outline"}>
                {STATUS_LABELS[subscription.status] ?? subscription.status}
              </Badge>
              {subscription.billing_interval && (
                <Badge variant="outline">
                  {INTERVAL_LABELS[subscription.billing_interval] ??
                    subscription.billing_interval}
                </Badge>
              )}
            </div>

            {daysLeft !== null && (
              <p className="text-sm text-muted-foreground">
                {t("trialEndsIn")}{" "}
                <span className="font-medium text-foreground">{t("days", { count: daysLeft })}</span>
                {subscription.trial_end && (
                  <>
                    {" "}
                    ({new Date(subscription.trial_end).toLocaleDateString("pl-PL")})
                  </>
                )}
              </p>
            )}

            {subscription.status === "canceled" &&
              subscription.current_period_end && (
                <p className="text-sm text-muted-foreground">
                  {t("accessExpires")}{" "}
                  <span className="font-medium text-foreground">
                    {new Date(
                      subscription.current_period_end
                    ).toLocaleDateString("pl-PL")}
                  </span>
                </p>
              )}

            {subscription.status === "active" &&
              subscription.current_period_end && (
                <p className="text-sm text-muted-foreground">
                  {t("nextRenewal")}{" "}
                  <span className="font-medium text-foreground">
                    {new Date(
                      subscription.current_period_end
                    ).toLocaleDateString("pl-PL")}
                  </span>
                </p>
              )}
          </CardContent>
        </Card>

        {/* Plan Limits */}
        {subscription.limits && (
          <Card>
            <CardHeader>
              <CardTitle>{t("planLimits")}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                {subscription.limits.max_users > 0 && (
                  <div className="rounded-lg border p-4 text-center">
                    <p className="text-2xl font-bold">
                      {subscription.limits.max_users}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {t("maksUzytkownikow")}
                    </p>
                  </div>
                )}
                {subscription.limits.max_orders_monthly > 0 && (
                  <div className="rounded-lg border p-4 text-center">
                    <p className="text-2xl font-bold">
                      {subscription.limits.max_orders_monthly}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {t("zamowienMiesiac")}
                    </p>
                  </div>
                )}
                {subscription.limits.max_integrations > 0 && (
                  <div className="rounded-lg border p-4 text-center">
                    <p className="text-2xl font-bold">
                      {subscription.limits.max_integrations}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {t("maxIntegrations")}
                    </p>
                  </div>
                )}
                {!subscription.limits.max_users &&
                  !subscription.limits.max_orders_monthly &&
                  !subscription.limits.max_integrations && (
                    <p className="col-span-3 text-sm text-muted-foreground">
                      {t("brakLimitowWBiezacymPlanie")}
                    </p>
                  )}
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </AdminGuard>
  );
}
