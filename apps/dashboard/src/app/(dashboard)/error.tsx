"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
      <h2 className="text-xl font-semibold">Cos poszlo nie tak</h2>
      <p className="text-muted-foreground">Wystapil nieoczekiwany blad.</p>
      <Button onClick={reset}>Sprobuj ponownie</Button>
    </div>
  );
}
