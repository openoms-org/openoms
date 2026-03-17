"use client";

import { useState, useCallback, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorBoundary } from "@/components/shared/error-boundary";
import { CommandPalette } from "@/components/shared/command-palette";
import { SidebarProvider } from "@/components/layout/sidebar-context";
import { TableDensityProvider } from "@/lib/table-density";
import { useKeyboardShortcuts } from "@/hooks/use-keyboard-shortcuts";
import { useServiceWorker } from "@/hooks/use-service-worker";
import { useSessionTimeout } from "@/hooks/use-session-timeout";
import { SubscriptionBanner } from "@/components/subscription-banner";
import { useOnboardingStatus } from "@/hooks/use-onboarding-wizard";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const isLoading = useAuthStore((s) => s.isLoading);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { data: onboardingStatus } = useOnboardingStatus();
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);

  // Redirect to login if auth restoration failed
  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  useEffect(() => {
    if (onboardingStatus && !onboardingStatus.completed && !onboardingStatus.dismissed) {
      router.replace("/onboarding");
    }
  }, [onboardingStatus, router]);

  const handleToggleCommandPalette = useCallback(() => {
    setCommandPaletteOpen((prev) => !prev);
  }, []);

  useKeyboardShortcuts({
    onToggleCommandPalette: handleToggleCommandPalette,
  });

  useServiceWorker();
  useSessionTimeout();

  // Show skeleton while loading auth, or while redirect is pending
  if (isLoading || !isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Skeleton className="h-8 w-32" />
      </div>
    );
  }

  return (
    <TableDensityProvider>
      <SidebarProvider>
        <div className="dashboard-base flex h-screen overflow-hidden">
          <Sidebar />
          <div className="flex flex-1 flex-col overflow-hidden">
            <SubscriptionBanner />
            <Header />
            <main className="flex-1 overflow-y-auto p-6">
              <ErrorBoundary>{children}</ErrorBoundary>
            </main>
          </div>
          <CommandPalette
            open={commandPaletteOpen}
            onOpenChange={setCommandPaletteOpen}
          />
        </div>
      </SidebarProvider>
    </TableDensityProvider>
  );
}
