"use client";

import { Component, type ReactNode } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ApiClientError } from "@/lib/api-client";

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

function getErrorDisplay(error: Error | null): {
  title: string;
  message: string;
  showLogin: boolean;
} {
  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        return {
          title: "Sesja wygasła",
          message: "Sesja wygasła. Zaloguj się ponownie.",
          showLogin: true,
        };
      case 429:
        return {
          title: "Zbyt wiele żądań",
          message: "Zbyt wiele żądań. Poczekaj chwilę i spróbuj ponownie.",
          showLogin: false,
        };
      case 500:
        return {
          title: "Błąd serwera",
          message: "Błąd serwera. Spróbuj ponownie później.",
          showLogin: false,
        };
    }
  }
  return {
    title: "Coś poszło nie tak",
    message: "Wystąpił nieoczekiwany błąd. Spróbuj odświeżyć stronę.",
    showLogin: false,
  };
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
      const { title, message, showLogin } = getErrorDisplay(this.state.error);
      return (
        <div className="flex flex-col items-center justify-center gap-4 py-20">
          <h2 className="text-xl font-semibold">{title}</h2>
          <p className="text-muted-foreground text-sm">{message}</p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              Spróbuj ponownie
            </Button>
            {showLogin && (
              <Button asChild>
                <Link href="/login">Zaloguj się</Link>
              </Button>
            )}
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
