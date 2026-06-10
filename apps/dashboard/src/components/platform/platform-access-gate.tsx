"use client";

import { ShieldAlert } from "lucide-react";
import { useTranslations } from "next-intl";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { EmptyState } from "@/components/shared/empty-state";
import { usePlatformMe, hasPlatformAccess } from "@/hooks/use-platform-providers";

/**
 * Advisory frontend gate for the Provider Integration Studio. The backend
 * remains authoritative — this only avoids rendering empty/forbidden tables to
 * non-platform-admins by showing a clear access-required state instead.
 */
export function PlatformAccessGate({ children }: { children: React.ReactNode }) {
  const t = useTranslations("platform");
  const { data: me, isLoading } = usePlatformMe();

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!hasPlatformAccess(me)) {
    return (
      <EmptyState
        icon={ShieldAlert}
        title={t("accessRequired.title")}
        description={t("accessRequired.description")}
      />
    );
  }

  return <>{children}</>;
}
