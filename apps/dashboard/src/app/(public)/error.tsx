"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { isExpectedClientError } from "@/lib/expected-client-error";

export default function PublicError({
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
      <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        {te("somethingWentWrong")}
      </h2>
      <p className="text-slate-600 dark:text-slate-400">
        {te("unexpected")}
      </p>
      <button
        onClick={reset}
        className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-slate-200"
      >
        {tc("retry")}
      </button>
    </div>
  );
}
