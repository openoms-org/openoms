"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth";
import { Skeleton } from "@/components/ui/skeleton";

export function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, isLoading } = useAuthStore();
  const isAdmin = user?.role === "admin" || user?.role === "owner";

  useEffect(() => {
    if (!isLoading && !isAdmin) {
      router.replace("/");
    }
  }, [isLoading, isAdmin, router]);

  if (isLoading || !isAdmin) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-64" />
      </div>
    );
  }

  return <>{children}</>;
}
