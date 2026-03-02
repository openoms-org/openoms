"use client";

import Link from "next/link";
import { useAuthStore } from "@/lib/auth";
import { useSubscription } from "@/hooks/use-billing";
import { AlertTriangle, Clock, XCircle } from "lucide-react";

export function SubscriptionBanner() {
  const tenant = useAuthStore((s) => s.tenant);
  const { data: subscription } = useSubscription();

  if (!tenant) return null;

  // Suspended — hard block (from tenant.plan or subscription status)
  const status = subscription?.status ?? tenant.plan;

  if (status === "suspended") {
    return (
      <div className="bg-destructive text-destructive-foreground px-4 py-3 text-center text-sm font-medium">
        <XCircle className="mr-2 inline h-4 w-4" />
        Subskrypcja została zawieszona. Odnów płatność aby kontynuować korzystanie z systemu.
      </div>
    );
  }

  if (status === "past_due") {
    return (
      <div className="bg-orange-500 text-white px-4 py-3 text-center text-sm font-medium">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        Płatność zaległa. Tworzenie nowych zasobów zostało zablokowane do momentu uregulowania należności.
      </div>
    );
  }

  if (status === "trialing" && subscription?.trial_end) {
    const daysLeft = Math.max(
      0,
      Math.ceil(
        (new Date(subscription.trial_end).getTime() - Date.now()) /
          (1000 * 60 * 60 * 24)
      )
    );
    return (
      <div className="bg-blue-500 text-white px-4 py-3 text-center text-sm font-medium">
        <Clock className="mr-2 inline h-4 w-4" />
        Okres próbny — pozostało {daysLeft} dni.{" "}
        <Link href="/settings/billing" className="underline hover:no-underline">
          Zarządzaj subskrypcją →
        </Link>
      </div>
    );
  }

  if (status === "canceled" && subscription?.current_period_end) {
    const expiresAt = new Date(subscription.current_period_end).toLocaleDateString("pl-PL");
    return (
      <div className="bg-orange-500 text-white px-4 py-3 text-center text-sm font-medium">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        Subskrypcja anulowana. Dostęp wygasa {expiresAt}.{" "}
        <Link href="/settings/billing" className="underline hover:no-underline">
          Odnów subskrypcję →
        </Link>
      </div>
    );
  }

  return null;
}
