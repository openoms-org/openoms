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
  ExternalLink,
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
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Integration } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const MARKETPLACE_OPTIONS = [
  { value: "EBAY_PL", label: "eBay Polska (EBAY_PL)" },
  { value: "EBAY_DE", label: "eBay Niemcy (EBAY_DE)" },
  { value: "EBAY_US", label: "eBay USA (EBAY_US)" },
  { value: "EBAY_GB", label: "eBay Wielka Brytania (EBAY_GB)" },
] as const;

const CURRENCY_OPTIONS = [
  { value: "PLN", label: "PLN — Polski złoty" },
  { value: "EUR", label: "EUR — Euro" },
  { value: "USD", label: "USD — Dolar amerykański" },
  { value: "GBP", label: "GBP — Funt brytyjski" },
] as const;

function getRedirectURI() {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/marketplaces/ebay`;
  }
  return `${API_URL.replace(/:\d+$/, ":3000")}/marketplaces/ebay`;
}

export default function EbayIntegrationPage() {
  const searchParams = useSearchParams();
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  if (code && state) {
    return <OAuthCallback code={code} state={state} />;
  }

  return <EbayMainPage />;
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

    apiClient("/v1/integrations/ebay/callback", {
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
                Łączenie z eBay...
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                Połączono z eBay! Okno zamknie się automatycznie.
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

function EbayMainPage() {
  const { data: integrations, isLoading, refetch } = useIntegrations();

  const ebay = useMemo(
    () => integrations?.find((i) => i.provider === "ebay") ?? null,
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
            <Link href="/marketplaces">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Integracja eBay</h1>
            <p className="text-muted-foreground">
              Połącz swoje konto eBay, aby synchronizować zamówienia i produkty
            </p>
          </div>
        </div>

        {ebay ? (
          <ConnectedState integration={ebay} onRefetch={refetch} />
        ) : (
          <SetupState onCreated={refetch} />
        )}
      </div>
    </AdminGuard>
  );
}

function SetupState({ onCreated }: { onCreated: () => void }) {
  const createIntegration = useCreateIntegration();
  const [appId, setAppId] = useState("");
  const [certId, setCertId] = useState("");
  const [devId, setDevId] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [marketplaceId, setMarketplaceId] = useState("EBAY_PL");
  const [currency, setCurrency] = useState("PLN");
  const [showAppId, setShowAppId] = useState(false);
  const [showCertId, setShowCertId] = useState(false);
  const [showDevId, setShowDevId] = useState(false);
  const [isAuthorizing, setIsAuthorizing] = useState(false);

  const redirectURI = getRedirectURI();
  const devPortalURL = sandbox
    ? "https://developer.ebay.com/my/keys?env=sandbox"
    : "https://developer.ebay.com/my/keys?env=production";

  const openOAuthPopup = useCallback(async (onDone: () => void) => {
    setIsAuthorizing(true);
    try {
      const resp = await apiClient<{
        auth_url: string;
        state: string;
        redirect_uri: string;
      }>("/v1/integrations/ebay/auth-url");

      const popup = window.open(
        resp.auth_url,
        "ebay-oauth",
        "width=600,height=700,scrollbars=yes"
      );

      if (!popup) {
        toast.error(
          "Przeglądarka zablokowała okno popup. Zezwól na wyskakujące okna i spróbuj ponownie."
        );
        setIsAuthorizing(false);
        onDone();
        return;
      }

      const poll = setInterval(() => {
        if (popup.closed) {
          clearInterval(poll);
          setIsAuthorizing(false);
          onDone();
        }
      }, 500);
    } catch {
      toast.error("Nie udało się pobrać adresu autoryzacji eBay");
      setIsAuthorizing(false);
      onDone();
    }
  }, []);

  const handleSave = () => {
    if (!appId.trim() || !certId.trim() || !devId.trim()) {
      toast.error("App ID, Cert ID i Dev ID są wymagane");
      return;
    }

    createIntegration.mutate(
      {
        provider: "ebay",
        label: sandbox ? "eBay (Sandbox)" : "eBay",
        credentials: {
          app_id: appId.trim(),
          cert_id: certId.trim(),
          dev_id: devId.trim(),
          sandbox,
        },
        settings: {
          marketplace_id: marketplaceId,
          currency,
        },
      },
      {
        onSuccess: () => {
          toast.success("Dane eBay zapisane. Otwieranie autoryzacji...");
          openOAuthPopup(() => onCreated());
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
    <div className="space-y-6">
      {/* Step 1: Prerequisites */}
      <Card>
        <CardHeader>
          <CardTitle>Krok 1: Zarejestruj aplikację w eBay Developer Program</CardTitle>
          <CardDescription>
            Przed połączeniem musisz utworzyć aplikację w panelu deweloperskim
            eBay.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <Switch
                id="setup-sandbox"
                checked={sandbox}
                onCheckedChange={setSandbox}
              />
              <Label htmlFor="setup-sandbox" className="cursor-pointer">
                Tryb sandbox (testowy)
              </Label>
            </div>
            {sandbox && (
              <p className="text-xs text-muted-foreground">
                Sandbox wymaga osobnych kluczy API z panelu eBay Developer
              </p>
            )}
          </div>

          <Separator />

          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>
              Przejdź do{" "}
              <a
                href={devPortalURL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-primary underline"
              >
                eBay Developer Program
                <ExternalLink className="h-3 w-3" />
              </a>
            </li>
            <li>Skopiuj App ID (Client ID), Cert ID (Client Secret) i Dev ID</li>
            <li>
              W sekcji <strong>User Tokens</strong> dodaj Redirect URI (RuName):
            </li>
          </ol>

          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">
              Redirect URI (do wklejenia w eBay)
            </Label>
            <code className="block rounded bg-muted px-3 py-2 text-sm font-mono break-all">
              {redirectURI}
            </code>
          </div>

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              Redirect URI musi być <strong>dokładnie taki sam</strong> jak
              powyżej. Różnica w nawet jednym znaku spowoduje błąd autoryzacji.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Step 2: Enter credentials + settings */}
      <Card>
        <CardHeader>
          <CardTitle>Krok 2: Wprowadź dane aplikacji</CardTitle>
          <CardDescription>
            Wklej klucze API z panelu deweloperskiego eBay.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="app-id">App ID (Client ID)</Label>
            <div className="relative">
              <Input
                id="app-id"
                type={showAppId ? "text" : "password"}
                placeholder="Wklej App ID aplikacji eBay"
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-full px-3"
                onClick={() => setShowAppId(!showAppId)}
              >
                {showAppId ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="cert-id">Cert ID (Client Secret)</Label>
            <div className="relative">
              <Input
                id="cert-id"
                type={showCertId ? "text" : "password"}
                placeholder="Wklej Cert ID aplikacji eBay"
                value={certId}
                onChange={(e) => setCertId(e.target.value)}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-full px-3"
                onClick={() => setShowCertId(!showCertId)}
              >
                {showCertId ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dev-id">Dev ID</Label>
            <div className="relative">
              <Input
                id="dev-id"
                type={showDevId ? "text" : "password"}
                placeholder="Wklej Dev ID aplikacji eBay"
                value={devId}
                onChange={(e) => setDevId(e.target.value)}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0 top-0 h-full px-3"
                onClick={() => setShowDevId(!showDevId)}
              >
                {showDevId ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <Separator />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="marketplace-id">Marketplace</Label>
              <Select value={marketplaceId} onValueChange={setMarketplaceId}>
                <SelectTrigger id="marketplace-id" className="w-full">
                  <SelectValue placeholder="Wybierz marketplace" />
                </SelectTrigger>
                <SelectContent>
                  {MARKETPLACE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="currency">Waluta</Label>
              <Select value={currency} onValueChange={setCurrency}>
                <SelectTrigger id="currency" className="w-full">
                  <SelectValue placeholder="Wybierz walutę" />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCY_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <Button
            onClick={handleSave}
            disabled={
              createIntegration.isPending ||
              !appId.trim() ||
              !certId.trim() ||
              !devId.trim()
            }
          >
            {createIntegration.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <Save className="mr-2 h-4 w-4" />
            Zapisz i przejdź do autoryzacji
          </Button>
        </CardContent>
      </Card>
    </div>
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

  const settings = (integration.settings ?? {}) as Record<string, unknown>;
  const [marketplaceId, setMarketplaceId] = useState(
    (settings.marketplace_id as string) || "EBAY_PL"
  );
  const [currency, setCurrency] = useState(
    (settings.currency as string) || "PLN"
  );

  const handleDisconnect = () => {
    updateIntegration.mutate(
      { status: "inactive" },
      {
        onSuccess: () => {
          toast.success("Integracja eBay została dezaktywowana");
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
    if (
      !confirm(
        "Czy na pewno chcesz usunąć integrację eBay? Ta operacja jest nieodwracalna."
      )
    ) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success("Integracja eBay została usunięta");
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

  const handleAuthorize = useCallback(() => {
    setIsReauthorizing(true);
    const doAuth = async () => {
      try {
        const resp = await apiClient<{
          auth_url: string;
          state: string;
          redirect_uri: string;
        }>("/v1/integrations/ebay/auth-url");

        const popup = window.open(
          resp.auth_url,
          "ebay-oauth",
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
    doAuth();
  }, [onRefetch]);

  const handleSaveSettings = () => {
    updateIntegration.mutate(
      {
        settings: {
          ...settings,
          marketplace_id: marketplaceId,
          currency,
        },
      },
      {
        onSuccess: () => {
          toast.success("Ustawienia zostały zapisane");
          onRefetch();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Błąd podczas zapisywania ustawień"
          );
        },
      }
    );
  };

  const needsOAuth =
    integration.status !== "active" || !integration.last_sync_at;

  return (
    <div className="space-y-6">
      {/* OAuth prompt if not yet authorized */}
      {needsOAuth && (
        <Card className="border-amber-200 dark:border-amber-800">
          <CardHeader>
            <CardTitle>Autoryzacja OAuth</CardTitle>
            <CardDescription>
              Dane aplikacji zostały zapisane. Kliknij poniżej, aby autoryzować
              dostęp do konta eBay. Otworzy się okno popup z logowaniem eBay.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleAuthorize}
              disabled={isReauthorizing}
              className="w-full"
            >
              {isReauthorizing ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <ExternalLink className="mr-2 h-4 w-4" />
              )}
              Połącz z eBay
            </Button>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Status card */}
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

        {/* Actions card */}
        <Card>
          <CardHeader>
            <CardTitle>Akcje</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {integration.status === "active" && (
              <Button
                className="w-full"
                variant="outline"
                onClick={handleAuthorize}
                disabled={isReauthorizing}
              >
                {isReauthorizing ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="mr-2 h-4 w-4" />
                )}
                Odśwież token
              </Button>
            )}
            <Button
              className="w-full"
              variant="outline"
              onClick={handleDisconnect}
              disabled={
                updateIntegration.isPending ||
                integration.status === "inactive"
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

        {/* Credentials update card */}
        <CredentialsCard
          integrationId={integration.id}
          onUpdated={onRefetch}
        />

        {/* Marketplace & currency settings */}
        <Card>
          <CardHeader>
            <CardTitle>Ustawienia marketplace</CardTitle>
            <CardDescription>
              Wybierz rynek eBay i walutę do synchronizacji.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="settings-marketplace-id">Marketplace</Label>
              <Select value={marketplaceId} onValueChange={setMarketplaceId}>
                <SelectTrigger id="settings-marketplace-id" className="w-full">
                  <SelectValue placeholder="Wybierz marketplace" />
                </SelectTrigger>
                <SelectContent>
                  {MARKETPLACE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="settings-currency">Waluta</Label>
              <Select value={currency} onValueChange={setCurrency}>
                <SelectTrigger id="settings-currency" className="w-full">
                  <SelectValue placeholder="Wybierz walutę" />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCY_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              onClick={handleSaveSettings}
              disabled={updateIntegration.isPending}
              variant="outline"
            >
              {updateIntegration.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              <Save className="mr-2 h-4 w-4" />
              Zapisz ustawienia
            </Button>
          </CardContent>
        </Card>
      </div>
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
  const [appId, setAppId] = useState("");
  const [certId, setCertId] = useState("");
  const [devId, setDevId] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [showAppId, setShowAppId] = useState(false);
  const [showCertId, setShowCertId] = useState(false);
  const [showDevId, setShowDevId] = useState(false);

  const handleUpdateCredentials = () => {
    if (!appId.trim() || !certId.trim() || !devId.trim()) {
      toast.error("App ID, Cert ID i Dev ID są wymagane");
      return;
    }

    updateIntegration.mutate(
      {
        credentials: {
          app_id: appId.trim(),
          cert_id: certId.trim(),
          dev_id: devId.trim(),
          sandbox,
        },
      },
      {
        onSuccess: () => {
          toast.success(
            "Dane zaktualizowane. Kliknij 'Połącz z eBay' aby ponownie autoryzować."
          );
          setAppId("");
          setCertId("");
          setDevId("");
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
          Zaktualizuj App ID, Cert ID i Dev ID. Po zmianie konieczna będzie
          ponowna autoryzacja OAuth.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-app-id">App ID (Client ID)</Label>
          <div className="relative">
            <Input
              id="edit-app-id"
              type={showAppId ? "text" : "password"}
              placeholder="Nowy App ID"
              value={appId}
              onChange={(e) => setAppId(e.target.value)}
              className="pr-10"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3"
              onClick={() => setShowAppId(!showAppId)}
            >
              {showAppId ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-cert-id">Cert ID (Client Secret)</Label>
          <div className="relative">
            <Input
              id="edit-cert-id"
              type={showCertId ? "text" : "password"}
              placeholder="Nowy Cert ID"
              value={certId}
              onChange={(e) => setCertId(e.target.value)}
              className="pr-10"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3"
              onClick={() => setShowCertId(!showCertId)}
            >
              {showCertId ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-dev-id">Dev ID</Label>
          <div className="relative">
            <Input
              id="edit-dev-id"
              type={showDevId ? "text" : "password"}
              placeholder="Nowy Dev ID"
              value={devId}
              onChange={(e) => setDevId(e.target.value)}
              className="pr-10"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-full px-3"
              onClick={() => setShowDevId(!showDevId)}
            >
              {showDevId ? (
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
            !appId.trim() ||
            !certId.trim() ||
            !devId.trim()
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
