"use client";

import { Component, type ReactNode } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { ApiClientError } from "@/lib/api-client";

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

function ErrorDisplay({
  error,
  onReset,
}: {
  error: Error | null;
  onReset: () => void;
}) {
  const t = useTranslations("shared.errorBoundary");

  let title: string;
  let message: string;
  let showLogin = false;

  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        title = t("sessionExpired");
        message = t("sessionExpiredMessage");
        showLogin = true;
        break;
      case 429:
        title = t("tooManyRequests");
        message = t("tooManyRequestsMessage");
        break;
      case 500:
        title = t("serverError");
        message = t("serverErrorMessage");
        break;
      default:
        title = t("genericError");
        message = t("genericErrorMessage");
    }
  } else {
    title = t("genericError");
    message = t("genericErrorMessage");
  }

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20">
      <h2 className="text-xl font-semibold">{title}</h2>
      <p className="text-muted-foreground text-sm">{message}</p>
      <div className="flex items-center gap-2">
        <Button variant="outline" onClick={onReset}>
          {t("tryAgain")}
        </Button>
        {showLogin && (
          <Button asChild>
            <Link href="/login">{t("logIn")}</Link>
          </Button>
        )}
      </div>
    </div>
  );
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    if (process.env.NODE_ENV === "development") {
      console.error("ErrorBoundary caught:", error, info.componentStack);
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <ErrorDisplay
          error={this.state.error}
          onReset={() => this.setState({ hasError: false, error: null })}
        />
      );
    }

    return this.props.children;
  }
}
