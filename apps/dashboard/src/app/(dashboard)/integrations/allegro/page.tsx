"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Loader2, RefreshCw, Unplug } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useIntegrations, useUpdateIntegration } from "@/hooks/use-integrations";
import { AllegroConnect } from "@/components/integrations/allegro-connect";
import { StatusBadge } from "@/components/shared/status-badge";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Integration } from "@/types/api";

export default function AllegroIntegrationPage() {
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: integrations, isLoading, refetch } = useIntegrations();

  const allegro = useMemo(
    () => integrations?.find((i) => i.provider === "allegro") ?? null,
    [integrations]
  );

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/integrations">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Integracja Allegro</h1>
          <p className="text-muted-foreground">
            Połącz swoje konto Allegro, aby synchronizować zamówienia i produkty
          </p>
        </div>
      </div>

      {allegro ? (
        <ConnectedState integration={allegro} onRefetch={refetch} />
      ) : (
        <NotConnectedState onConnected={refetch} />
      )}
    </div>
  );
}

function NotConnectedState({ onConnected }: { onConnected: () => void }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Połącz konto Allegro</CardTitle>
        <CardDescription>
          Połączenie z Allegro umożliwi automatyczną synchronizację zamówień,
          produktów i stanów magazynowych. Zostaniesz przekierowany do strony
          logowania Allegro w celu autoryzacji dostępu.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <AllegroConnect onConnected={onConnected} />
      </CardContent>
    </Card>
  );
}

function ConnectedState({
  integration,
  onRefetch,
}: {
  integration: Integration;
  onRefetch: () => void;
}) {
  const updateIntegration = useUpdateIntegration(integration.id);
  const [isReauthorizing, setIsReauthorizing] = useState(false);

  const handleDisconnect = () => {
    updateIntegration.mutate(
      { status: "inactive" },
      {
        onSuccess: () => {
          toast.success("Integracja Allegro została dezaktywowana");
          onRefetch();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Błąd podczas dezaktywacji integracji"
          );
        },
      }
    );
  };

  const handleReauthorize = useCallback(() => {
    setIsReauthorizing(true);
    // Reuse the connect flow
    const doReauth = async () => {
      try {
        const { auth_url } = await (
          await import("@/lib/api-client")
        ).apiClient<{ auth_url: string }>("/v1/integrations/allegro/auth-url");

        const popup = window.open(
          auth_url,
          "allegro-oauth",
          "width=600,height=700,scrollbars=yes"
        );

        if (!popup) {
          toast.error(
            "Przeglądarka zablokowała okno popup. Zezwól na wyskakujące okna."
          );
          setIsReauthorizing(false);
          return;
        }

        const poll = setInterval(() => {
          if (popup.closed) {
            clearInterval(poll);
            setIsReauthorizing(false);
            onRefetch();
          }
        }, 500);
      } catch {
        toast.error("Nie udało się pobrać adresu autoryzacji");
        setIsReauthorizing(false);
      }
    };
    doReauth();
  }, [onRefetch]);

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Status połączenia</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <div className="mt-1">
                <StatusBadge
                  status={integration.status}
                  statusMap={INTEGRATION_STATUSES}
                />
              </div>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">
                Dane uwierzytelniające
              </p>
              <p className="mt-1 font-medium">
                {integration.has_credentials ? "Skonfigurowane" : "Brak"}
              </p>
            </div>
            {integration.label && (
              <div>
                <p className="text-sm text-muted-foreground">Etykieta</p>
                <p className="mt-1 font-medium">{integration.label}</p>
              </div>
            )}
            <div>
              <p className="text-sm text-muted-foreground">
                Ostatnia synchronizacja
              </p>
              <p className="mt-1 font-medium">
                {integration.last_sync_at
                  ? formatDate(integration.last_sync_at)
                  : "---"}
              </p>
            </div>
            {integration.sync_cursor && (
              <div>
                <p className="text-sm text-muted-foreground">Kursor synchronizacji</p>
                <p className="mt-1 font-mono text-xs truncate">
                  {integration.sync_cursor}
                </p>
              </div>
            )}
            <div>
              <p className="text-sm text-muted-foreground">ID integracji</p>
              <p className="mt-1 font-mono text-xs">{integration.id}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Data utworzenia</p>
              <p className="mt-1 font-medium">
                {formatDate(integration.created_at)}
              </p>
            </div>
          </div>

          {integration.status === "error" && integration.error_message && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
              <p className="text-sm font-medium text-destructive">
                Błąd integracji
              </p>
              <p className="mt-1 text-sm text-destructive/80">
                {integration.error_message}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Akcje</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button
            className="w-full"
            variant="outline"
            onClick={handleReauthorize}
            disabled={isReauthorizing}
          >
            {isReauthorizing ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-4 w-4" />
            )}
            Odśwież token
          </Button>
          <Button
            className="w-full"
            variant="destructive"
            onClick={handleDisconnect}
            disabled={updateIntegration.isPending}
          >
            {updateIntegration.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Unplug className="mr-2 h-4 w-4" />
            )}
            Rozłącz
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
