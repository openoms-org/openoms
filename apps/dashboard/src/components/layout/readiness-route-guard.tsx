"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LockKeyhole } from "lucide-react";
import { isRouteAccessible } from "@/lib/readiness";
import { Button } from "@/components/ui/button";

export function ReadinessRouteGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  if (isRouteAccessible(pathname)) {
    return <>{children}</>;
  }

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="max-w-md rounded-lg border bg-card p-6 text-center shadow-sm">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <LockKeyhole className="h-6 w-6 text-muted-foreground" />
        </div>
        <h1 className="text-xl font-semibold">Funkcja nie jest jeszcze dostępna</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Ten moduł jest ukryty w trybie produkcyjnym, dopóki nie przejdzie checklisty gotowości.
        </p>
        <Button asChild className="mt-6">
          <Link href="/">Wróć do pulpitu</Link>
        </Button>
      </div>
    </div>
  );
}
