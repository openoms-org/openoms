"use client";

import { useState, useCallback } from "react";
import { useAuthStore } from "@/lib/auth";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorBoundary } from "@/components/shared/error-boundary";
import { CommandPalette } from "@/components/shared/command-palette";
import { useKeyboardShortcuts } from "@/hooks/use-keyboard-shortcuts";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const isLoading = useAuthStore((s) => s.isLoading);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);

  const handleToggleCommandPalette = useCallback(() => {
    setCommandPaletteOpen((prev) => !prev);
  }, []);

  useKeyboardShortcuts({
    onToggleCommandPalette: handleToggleCommandPalette,
  });

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Skeleton className="h-8 w-32" />
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
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
  );
}
