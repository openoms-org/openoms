"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { isExpectedClientError } from "@/lib/expected-client-error";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    if (!isExpectedClientError(error)) {
      Sentry.captureException(error);
    }
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
      <h2 className="text-xl font-semibold">Cos poszlo nie tak</h2>
      <p className="text-muted-foreground">Wystapil nieoczekiwany blad.</p>
      <Button onClick={reset}>Sprobuj ponownie</Button>
    </div>
  );
}
