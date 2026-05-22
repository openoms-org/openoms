"use client";

import Link from "next/link";
import { useAuthStore } from "@/lib/auth";
import { useSubscription } from "@/hooks/use-billing";
import { AlertTriangle, Clock, XCircle } from "lucide-react";
import { useTranslations } from "next-intl";
import { isRouteAccessible } from "@/lib/readiness";

function computeDaysLeft(trialEnd: string): number {
  return Math.max(
    0,
    Math.ceil(
      (new Date(trialEnd).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
    )
  );
}

function isPaymentAttentionStatus(status: string | undefined): boolean {
  return status === "past_due" || status === "unpaid" || status === "incomplete";
}

function isInactiveStatus(status: string | undefined): boolean {
  return status === "canceled" || status === "paused" || status === "incomplete_expired";
}

export function SubscriptionBanner() {
  const t = useTranslations("shared");
  const tenant = useAuthStore((s) => s.tenant);
  const { data: subscription } = useSubscription();
  const daysLeft =
    subscription?.status === "trialing" && subscription.trial_end
      ? computeDaysLeft(subscription.trial_end)
      : null;

  if (!tenant) return null;

  const status = subscription?.status;
  const canManageBilling = isRouteAccessible("/settings/billing");

  if (status === "suspended") {
    return (
      <div className="bg-destructive text-destructive-foreground px-4 py-3 text-center text-sm font-medium">
        <XCircle className="mr-2 inline h-4 w-4" />
        {t("suspended")}
      </div>
    );
  }

  if (isPaymentAttentionStatus(status)) {
    return (
      <div className="bg-orange-500 text-white px-4 py-3 text-center text-sm font-medium">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        {t("pastDue")}
      </div>
    );
  }

  if (status === "trialing" && subscription?.trial_end && daysLeft !== null) {
    return (
      <div className="bg-blue-500 text-white px-4 py-3 text-center text-sm font-medium">
        <Clock className="mr-2 inline h-4 w-4" />
        {t("trialDaysLeft", { days: daysLeft })}{" "}
        {canManageBilling ? (
          <Link href="/settings/billing" className="underline hover:no-underline">
            {t("zarzadzajSubskrypcja")}
          </Link>
        ) : (
          <span className="font-medium">{t("subscriptionManagedBySupport")}</span>
        )}
      </div>
    );
  }

  if (isInactiveStatus(status) && subscription?.current_period_end) {
    const expiresAt = new Date(subscription.current_period_end).toLocaleDateString("pl-PL");
    return (
      <div className="bg-orange-500 text-white px-4 py-3 text-center text-sm font-medium">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        {t("subscriptionCancelled", { date: expiresAt })}{" "}
        {canManageBilling ? (
          <Link href="/settings/billing" className="underline hover:no-underline">
            {t("odnowSubskrypcje")}
          </Link>
        ) : (
          <span className="font-medium">{t("renewViaSupport")}</span>
        )}
      </div>
    );
  }

  return null;
}
