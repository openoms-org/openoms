"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";
import { useTranslations } from "next-intl";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useTranslations("errors");
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <html>
      <body>
        <div style={{ padding: "2rem", textAlign: "center" }}>
          <h2>{t("wystapiłNieoczekiwanyBład")}</h2>
          <p style={{ color: "#666", marginTop: "0.5rem" }}>
            {t("bładZostałAutomatycznieZgłoszonySprobujPonownie")}
          </p>
          <button
            onClick={reset}
            style={{
              marginTop: "1rem",
              padding: "0.5rem 1rem",
              cursor: "pointer",
            }}
          >
            {t("retry")}
          </button>
        </div>
      </body>
    </html>
  );
}
