"use client";

import { useAuthStore } from "@/lib/auth";
import { AlertTriangle, XCircle } from "lucide-react";

export function SubscriptionBanner() {
  const tenant = useAuthStore((s) => s.tenant);

  if (!tenant) return null;

  if (tenant.plan === "suspended") {
    return (
      <div className="bg-destructive text-destructive-foreground px-4 py-3 text-center text-sm font-medium">
        <XCircle className="mr-2 inline h-4 w-4" />
        Subskrypcja została zawieszona. Odnów płatność aby kontynuować korzystanie z systemu.
      </div>
    );
  }

  if (tenant.plan === "past_due") {
    return (
      <div className="bg-orange-500 text-white px-4 py-3 text-center text-sm font-medium">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        Płatność zaległa. Tworzenie nowych zasobów zostało zablokowane do momentu uregulowania należności.
      </div>
    );
  }

  return null;
}
