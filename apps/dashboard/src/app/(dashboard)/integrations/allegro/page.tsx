"use client";

import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  RefreshCw,
  Unplug,
  CheckCircle2,
  XCircle,
  Save,
  Eye,
  EyeOff,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useIntegrations,
  useCreateIntegration,
  useUpdateIntegration,
  useDeleteIntegration,
} from "@/hooks/use-integrations";
import { StatusBadge } from "@/components/shared/status-badge";
import { INTEGRATION_STATUSES } from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { apiClient } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
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
  const searchParams = useSearchParams();
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  // Handle OAuth callback (runs in popup after Allegro redirect)
  if (code && state) {
    return <OAuthCallback code={code} state={state} />;
  }

  return <AllegroMainPage />;
}

function OAuthCallback({ code, state }: { code: string; state: string }) {
  const [status, setStatus] = useState<"loading" | "success" | "error">(
    "loading"
  );
  const [errorMsg, setErrorMsg] = useState("");
  const didRun = useRef(false);

  useEffect(() => {
    if (didRun.current) return;
    didRun.current = true;

    apiClient("/v1/integrations/allegro/callback", {
      method: "POST",
      body: JSON.stringify({ code, state }),
    })
      .then(() => {
        setStatus("success");
        setTimeout(() => window.close(), 1500);
      })
      .catch((err) => {
        setStatus("error");
        setErrorMsg(
          err instanceof Error ? err.message : "Autoryzacja nie powiodła się"
        );
      });
  }, [code, state]);

  return (
    <div className="flex min-h-[50vh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col items-center gap-4 pt-6">
          {status === "loading" && (
            <>
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                Łączenie z Allegro...
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                Połączono z Allegro! Okno zamknie się automatycznie.
              </p>
            </>
          )}
          {status === "error" && (
            <>
              <XCircle className="h-8 w-8 text-destructive" />
              <p className="text-sm text-destructive">{errorMsg}</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => window.close()}
              >
                Zamknij okno
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function AllegroMainPage() {
  const { data: integrations, isLoading, refetch } = useIntegrations();

  const allegro = useMemo(
    () => integrations?.find((i) => i.provider === "allegro") ?? null,
    [integrations]
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <AdminGuard>
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
              Połącz swoje konto Allegro, aby synchronizować zamówienia i
              produkty
            </p>
          </div>
        </div>

        {allegro ? (
          <ConnectedState integration={allegro} onRefetch={refetch} />
        ) : (
          <SetupState onCreated={refetch} />
        )}
      </div>
    </AdminGuard>
  );
}

function SetupState({ onCreated }: { onCreated: () => void }) {
  const createIntegration = useCreateIntegration();
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [showSecret, setShowSecret] = useState(false);

  const handleSave = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error("Client ID i Client Secret są wymagane");
      return;
    }

    createIntegration.mutate(
      {
        provider: "allegro",
        label: "Allegro",
        credentials: {
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          sandbox,
        },
      },
      {
        onSuccess: () => {
          toast.success("Dane Allegro zapisane. Możesz teraz połączyć konto.");
          onCreated();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Błąd podczas zapisywania danych"
          );
        },
      }
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Konfiguracja Allegro</CardTitle>
        <CardDescription>
          Wprowadź dane aplikacji z{" "}
          <a
            href="https://apps.developer.allegro.pl/"
            target="_blank"
            rel="noopener noreferrer"
            className="underline"
          >
            Allegro Developer Center
          </a>
          . Po zapisaniu będziesz mógł połączyć konto OAuth.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="client-id">Client ID</Label>
          <Input
            id="client-id"
            placeholder="Wklej Client ID aplikacji Allegro"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="client-secret">Client Secret</Label>
          <div className="relative">
            <Input
              id="client-secret"
              type={showSecret ? "text" : "password"}
              placeholder="Wklej Client Secret aplikacji Allegro"
              value={clientSecret}
              onChange={(e) => setClientSecret(e.target.value)}
              className="pr-10"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3"
              onClick={() => setShowSecret(!showSecret)}
            >
              {showSecret ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Switch
            id="sandbox"
            checked={sandbox}
            onCheckedChange={setSandbox}
          />
          <Label htmlFor="sandbox" className="cursor-pointer">
            Tryb sandbox (testowy)
          </Label>
        </div>
        <p className="text-xs text-muted-foreground">
          Włącz sandbox, aby testować integrację bez wpływu na produkcyjne konto
          Allegro.
        </p>

        <Button
          onClick={handleSave}
          disabled={
            createIntegration.isPending || !clientId.trim() || !clientSecret.trim()
          }
        >
          {createIntegration.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          <Save className="mr-2 h-4 w-4" />
          Zapisz i kontynuuj
        </Button>
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
  const deleteIntegration = useDeleteIntegration();
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

  const handleDelete = () => {
    if (!confirm("Czy na pewno chcesz usunąć integrację Allegro? Ta operacja jest nieodwracalna.")) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success("Integracja Allegro została usunięta");
        onRefetch();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Błąd podczas usuwania integracji"
        );
      },
    });
  };

  const handleReauthorize = useCallback(() => {
    setIsReauthorizing(true);
    const doReauth = async () => {
      try {
        const { auth_url } = await apiClient<{ auth_url: string }>(
          "/v1/integrations/allegro/auth-url"
        );

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
                <p className="text-sm text-muted-foreground">
                  Kursor synchronizacji
                </p>
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
            {integration.status === "active"
              ? "Odśwież token"
              : "Połącz z Allegro"}
          </Button>
          <Button
            className="w-full"
            variant="outline"
            onClick={handleDisconnect}
            disabled={
              updateIntegration.isPending || integration.status === "inactive"
            }
          >
            {updateIntegration.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Unplug className="mr-2 h-4 w-4" />
            )}
            Dezaktywuj
          </Button>
          <Button
            className="w-full"
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteIntegration.isPending}
          >
            {deleteIntegration.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Trash2 className="mr-2 h-4 w-4" />
            )}
            Usuń integrację
          </Button>
        </CardContent>
      </Card>

      <CredentialsCard
        integrationId={integration.id}
        onUpdated={onRefetch}
      />
    </div>
  );
}

function CredentialsCard({
  integrationId,
  onUpdated,
}: {
  integrationId: string;
  onUpdated: () => void;
}) {
  const updateIntegration = useUpdateIntegration(integrationId);
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [showSecret, setShowSecret] = useState(false);

  const handleUpdateCredentials = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error("Client ID i Client Secret są wymagane");
      return;
    }

    updateIntegration.mutate(
      {
        credentials: {
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          sandbox,
        },
      },
      {
        onSuccess: () => {
          toast.success(
            "Dane zaktualizowane. Kliknij 'Połącz z Allegro' aby ponownie autoryzować."
          );
          setClientId("");
          setClientSecret("");
          onUpdated();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Błąd podczas aktualizacji danych"
          );
        },
      }
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Zmień dane aplikacji</CardTitle>
        <CardDescription>
          Zaktualizuj Client ID i Client Secret. Po zmianie konieczna będzie
          ponowna autoryzacja OAuth.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-client-id">Client ID</Label>
          <Input
            id="edit-client-id"
            placeholder="Nowy Client ID"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-client-secret">Client Secret</Label>
          <div className="relative">
            <Input
              id="edit-client-secret"
              type={showSecret ? "text" : "password"}
              placeholder="Nowy Client Secret"
              value={clientSecret}
              onChange={(e) => setClientSecret(e.target.value)}
              className="pr-10"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3"
              onClick={() => setShowSecret(!showSecret)}
            >
              {showSecret ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Switch
            id="edit-sandbox"
            checked={sandbox}
            onCheckedChange={setSandbox}
          />
          <Label htmlFor="edit-sandbox" className="cursor-pointer">
            Tryb sandbox (testowy)
          </Label>
        </div>

        <Button
          onClick={handleUpdateCredentials}
          disabled={
            updateIntegration.isPending ||
            !clientId.trim() ||
            !clientSecret.trim()
          }
          variant="outline"
        >
          {updateIntegration.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          <Save className="mr-2 h-4 w-4" />
          Zaktualizuj dane
        </Button>
      </CardContent>
    </Card>
  );
}
