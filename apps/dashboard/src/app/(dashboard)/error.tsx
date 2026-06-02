"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { isExpectedClientError } from "@/lib/expected-client-error";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const te = useTranslations("errors");
  const tc = useTranslations("common");

  useEffect(() => {
    if (!isExpectedClientError(error)) {
      Sentry.captureException(error);
    }
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
      <h2 className="text-xl font-semibold">{te("somethingWentWrong")}</h2>
      <p className="text-muted-foreground">{te("unexpected")}</p>
      <Button onClick={reset}>{tc("retry")}</Button>
    </div>
  );
}
