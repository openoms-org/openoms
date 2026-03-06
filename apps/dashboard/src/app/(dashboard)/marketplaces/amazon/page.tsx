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
  Copy,
  Check,
  ExternalLink,
  Wrench,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Integration } from "@/types/api";

const MARKETPLACES = [
  { id: "A1C3SOZRARQ6R3", label: "amazon.pl (Polska)" },
  { id: "A1PA6795UKMFR9", label: "amazon.de (Niemcy)" },
  { id: "A1F83G8C2ARO7P", label: "amazon.co.uk (Wielka Brytania)" },
  { id: "A13V1IB3VIYZZH", label: "amazon.fr (Francja)" },
  { id: "APJ6JRA9NG5V4", label: "amazon.it (Wlochy)" },
  { id: "A1RKKUPIHCS9HS", label: "amazon.es (Hiszpania)" },
  { id: "A21TJRUUN4KGV", label: "amazon.in (Indie)" },
  { id: "ATVPDKIKX0DER", label: "amazon.com (USA)" },
];

function getRedirectURI() {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/marketplaces/amazon`;
  }
  return "";
}

function getMarketplaceLabel(id: string) {
  return MARKETPLACES.find((m) => m.id === id)?.label ?? id;
}

export default function AmazonIntegrationPage() {
  const searchParams = useSearchParams();
  const code = searchParams.get("spapi_oauth_code");
  const state = searchParams.get("state");
  const sellingPartnerId = searchParams.get("selling_partner_id");

  if (code && state) {
    return (
      <OAuthCallback
        code={code}
        state={state}
        sellingPartnerId={sellingPartnerId ?? ""}
      />
    );
  }

  return <AmazonMainPage />;
}

function OAuthCallback({
  code,
  state,
  sellingPartnerId,
}: {
  code: string;
  state: string;
  sellingPartnerId: string;
}) {
  const [status, setStatus] = useState<"loading" | "success" | "error">(
    "loading"
  );
  const [errorMsg, setErrorMsg] = useState("");
  const didRun = useRef(false);

  useEffect(() => {
    if (didRun.current) return;
    didRun.current = true;

    apiClient("/v1/integrations/amazon/callback", {
      method: "POST",
      body: JSON.stringify({
        spapi_oauth_code: code,
        state,
        selling_partner_id: sellingPartnerId,
      }),
    })
      .then(() => {
        setStatus("success");
        setTimeout(() => window.close(), 1500);
      })
      .catch((err) => {
        setStatus("error");
        setErrorMsg(
          err instanceof Error ? err.message : "Autoryzacja nie powiodla sie"
        );
      });
  }, [code, state, sellingPartnerId]);

  return (
    <div className="flex min-h-[50vh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col items-center gap-4 pt-6">
          {status === "loading" && (
            <>
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                Laczenie z Amazon...
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                Polaczono z Amazon! Okno zamknie sie automatycznie.
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

function AmazonMainPage() {
  const { data: integrations, isLoading, refetch } = useIntegrations();

  const amazon = useMemo(
    () => integrations?.find((i) => i.provider === "amazon") ?? null,
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
            <h1 className="text-2xl font-bold">Konfiguracja Amazon SP-API</h1>
            <p className="text-muted-foreground">
              Polacz swoje konto Amazon Seller, aby synchronizowac zamowienia
            </p>
          </div>
        </div>

        {amazon ? (
          <ConnectedState integration={amazon} onRefetch={refetch} />
        ) : (
          <SetupState onCreated={refetch} />
        )}
      </div>
    </AdminGuard>
  );
}

function CopyableField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <div className="flex items-center gap-2">
        <code className="flex-1 rounded bg-muted px-3 py-2 text-sm font-mono break-all">
          {value}
        </code>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={handleCopy}
          className="shrink-0"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-600" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}

function SetupState({ onCreated }: { onCreated: () => void }) {
  const createIntegration = useCreateIntegration();
  const [applicationId, setApplicationId] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [marketplaceId, setMarketplaceId] = useState(MARKETPLACES[0].id);
  const [showSecret, setShowSecret] = useState(false);
  const [isAuthorizing, setIsAuthorizing] = useState(false);
  const [showManualSetup, setShowManualSetup] = useState(false);
  const [refreshToken, setRefreshToken] = useState("");
  const [sandbox, setSandbox] = useState(false);

  const redirectURI = getRedirectURI();

  const openOAuthPopup = useCallback(
    async (onDone: () => void) => {
      setIsAuthorizing(true);
      try {
        const resp = await apiClient<{
          auth_url: string;
          state: string;
          redirect_uri: string;
        }>("/v1/integrations/amazon/auth-url");

        const popup = window.open(
          resp.auth_url,
          "amazon-oauth",
          "width=600,height=700,scrollbars=yes"
        );

        if (!popup) {
          toast.error(
            "Przegladarka zablokowala okno popup. Zezwol na wyskakujace okna i sprobuj ponownie."
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
        toast.error("Nie udalo sie pobrac adresu autoryzacji Amazon");
        setIsAuthorizing(false);
        onDone();
      }
    },
    []
  );

  const handleSaveAndAuthorize = () => {
    if (
      !applicationId.trim() ||
      !clientId.trim() ||
      !clientSecret.trim() ||
      !marketplaceId
    ) {
      toast.error(
        "Application ID, Client ID, Client Secret i Marketplace sa wymagane"
      );
      return;
    }

    createIntegration.mutate(
      {
        provider: "amazon",
        label: "Amazon",
        credentials: {
          application_id: applicationId.trim(),
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          marketplace_id: marketplaceId,
          sandbox,
        },
      },
      {
        onSuccess: () => {
          toast.success("Dane Amazon zapisane. Otwieranie autoryzacji...");
          openOAuthPopup(() => onCreated());
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Blad podczas zapisywania danych"
          );
        },
      }
    );
  };

  const handleManualSetup = async () => {
    if (
      !clientId.trim() ||
      !clientSecret.trim() ||
      !refreshToken.trim() ||
      !marketplaceId
    ) {
      toast.error("Wszystkie pola sa wymagane");
      return;
    }

    try {
      await apiClient("/v1/integrations/amazon/setup", {
        method: "POST",
        body: JSON.stringify({
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          refresh_token: refreshToken.trim(),
          marketplace_id: marketplaceId,
          sandbox,
        }),
      });
      toast.success("Integracja Amazon zostala skonfigurowana");
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Nie udalo sie skonfigurowac integracji"
      );
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Krok 1: Przygotuj aplikacje Amazon SP-API</CardTitle>
          <CardDescription>
            Przed polaczeniem potrzebujesz aplikacji zarejestrowanej w Amazon
            Developer Console.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>
              Przejdz do{" "}
              <a
                href="https://sellercentral.amazon.pl/apps/manage"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-primary underline"
              >
                Amazon Seller Central &gt; Develop Apps
                <ExternalLink className="h-3 w-3" />
              </a>
            </li>
            <li>Utworz nowa aplikacje lub uzyj istniejacej</li>
            <li>
              W ustawieniach aplikacji, w polu{" "}
              <strong>OAuth Redirect URI</strong> wklej ponizszy adres:
            </li>
          </ol>

          <CopyableField
            label="Adres przekierowania (Redirect URI)"
            value={redirectURI}
          />

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              Redirect URI musi byc <strong>dokladnie taki sam</strong> jak
              powyzej. Roznica w nawet jednym znaku spowoduje blad autoryzacji.
            </p>
          </div>

          <ol className="list-decimal list-inside space-y-2 text-sm" start={4}>
            <li>
              Skopiuj <strong>Application ID</strong>,{" "}
              <strong>LWA Client ID</strong> i{" "}
              <strong>LWA Client Secret</strong>
            </li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Krok 2: Wprowadz dane aplikacji</CardTitle>
          <CardDescription>
            Wklej dane z Amazon Developer Console, a nastepnie przejdz do
            autoryzacji OAuth.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="application-id">Application ID</Label>
            <Input
              id="application-id"
              placeholder="amzn1.sellerapps.app.xxx"
              value={applicationId}
              onChange={(e) => setApplicationId(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              ID aplikacji z Amazon Developer Console (wymagane do OAuth)
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="client-id">Client ID (LWA)</Label>
            <Input
              id="client-id"
              placeholder="amzn1.application-oa2-client.xxx"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="client-secret">Client Secret (LWA)</Label>
            <div className="relative">
              <Input
                id="client-secret"
                type={showSecret ? "text" : "password"}
                placeholder="Klucz tajny aplikacji"
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
          <div className="space-y-2">
            <Label>Marketplace</Label>
            <Select value={marketplaceId} onValueChange={setMarketplaceId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Wybierz marketplace" />
              </SelectTrigger>
              <SelectContent>
                {MARKETPLACES.map((mp) => (
                  <SelectItem key={mp.id} value={mp.id}>
                    {mp.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="sandbox"
              checked={sandbox}
              onChange={(e) => setSandbox(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            <Label htmlFor="sandbox" className="text-sm font-normal">
              Tryb sandbox (testowy)
            </Label>
          </div>

          <Button
            onClick={handleSaveAndAuthorize}
            disabled={
              createIntegration.isPending ||
              isAuthorizing ||
              !applicationId.trim() ||
              !clientId.trim() ||
              !clientSecret.trim()
            }
          >
            {(createIntegration.isPending || isAuthorizing) && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <Save className="mr-2 h-4 w-4" />
            Zapisz i polacz z Amazon
          </Button>

          <div className="border-t pt-4">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowManualSetup(!showManualSetup)}
              className="text-muted-foreground"
            >
              <Wrench className="mr-2 h-4 w-4" />
              Konfiguracja reczna (dla prywatnych aplikacji)
            </Button>
          </div>

          {showManualSetup && (
            <div className="rounded-md border p-4 space-y-4">
              <p className="text-sm text-muted-foreground">
                Jesli masz prywatna aplikacje (self-authorized), mozesz wkleic
                Refresh Token bezposrednio zamiast korzystac z autoryzacji OAuth.
              </p>
              <div className="space-y-2">
                <Label htmlFor="refresh-token">Refresh Token</Label>
                <Input
                  id="refresh-token"
                  type="password"
                  placeholder="Atzr|..."
                  value={refreshToken}
                  onChange={(e) => setRefreshToken(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Token uzyskany po autoryzacji aplikacji w Amazon Seller Central
                </p>
              </div>
              <Button
                onClick={handleManualSetup}
                variant="outline"
                disabled={
                  !clientId.trim() ||
                  !clientSecret.trim() ||
                  !refreshToken.trim()
                }
              >
                Zapisz i zweryfikuj (konfiguracja reczna)
              </Button>
            </div>
          )}
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

  const handleDisconnect = () => {
    updateIntegration.mutate(
      { status: "inactive" },
      {
        onSuccess: () => {
          toast.success("Integracja Amazon zostala dezaktywowana");
          onRefetch();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Blad podczas dezaktywacji integracji"
          );
        },
      }
    );
  };

  const handleDelete = () => {
    if (
      !confirm(
        "Czy na pewno chcesz usunac integracje Amazon? Ta operacja jest nieodwracalna."
      )
    ) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success("Integracja Amazon zostala usunieta");
        onRefetch();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Blad podczas usuwania integracji"
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
        }>("/v1/integrations/amazon/auth-url");

        const popup = window.open(
          resp.auth_url,
          "amazon-oauth",
          "width=600,height=700,scrollbars=yes"
        );

        if (!popup) {
          toast.error(
            "Przegladarka zablokowala okno popup. Zezwol na wyskakujace okna."
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
        toast.error("Nie udalo sie pobrac adresu autoryzacji");
        setIsReauthorizing(false);
      }
    };
    doAuth();
  }, [onRefetch]);

  const needsOAuth =
    integration.status !== "active" || !integration.last_sync_at;

  return (
    <div className="space-y-6">
      {needsOAuth && (
        <Card className="border-amber-200 dark:border-amber-800">
          <CardHeader>
            <CardTitle>Autoryzacja OAuth</CardTitle>
            <CardDescription>
              Dane aplikacji zostaly zapisane. Kliknij ponizej, aby autoryzowac
              dostep do konta Amazon. Otworzy sie okno popup z logowaniem Seller
              Central.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <CopyableField
                label="Redirect URI (musi byc zarejestrowany w Amazon)"
                value={getRedirectURI()}
              />
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
                Polacz z Amazon
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Status polaczenia</CardTitle>
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
                  Dane uwierzytelniajace
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
                  Blad integracji
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
                Odswiez token
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
              Usun integracje
            </Button>
          </CardContent>
        </Card>

        <CredentialsCard
          integrationId={integration.id}
          onUpdated={onRefetch}
        />
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
  const [applicationId, setApplicationId] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [marketplaceId, setMarketplaceId] = useState("");
  const [showSecret, setShowSecret] = useState(false);

  const handleUpdateCredentials = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error("Client ID i Client Secret sa wymagane");
      return;
    }

    const credentials: Record<string, string> = {
      client_id: clientId.trim(),
      client_secret: clientSecret.trim(),
    };
    if (applicationId.trim()) {
      credentials.application_id = applicationId.trim();
    }
    if (marketplaceId) {
      credentials.marketplace_id = marketplaceId;
    }

    updateIntegration.mutate(
      { credentials },
      {
        onSuccess: () => {
          toast.success(
            "Dane zaktualizowane. Kliknij 'Polacz z Amazon' aby ponownie autoryzowac."
          );
          setApplicationId("");
          setClientId("");
          setClientSecret("");
          setMarketplaceId("");
          onUpdated();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Blad podczas aktualizacji danych"
          );
        },
      }
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Zmien dane aplikacji</CardTitle>
        <CardDescription>
          Zaktualizuj dane aplikacji Amazon. Po zmianie konieczna bedzie ponowna
          autoryzacja OAuth.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-application-id">Application ID</Label>
          <Input
            id="edit-application-id"
            placeholder="Nowy Application ID"
            value={applicationId}
            onChange={(e) => setApplicationId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-client-id">Client ID (LWA)</Label>
          <Input
            id="edit-client-id"
            placeholder="Nowy Client ID"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-client-secret">Client Secret (LWA)</Label>
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
        <div className="space-y-2">
          <Label>Marketplace</Label>
          <Select value={marketplaceId} onValueChange={setMarketplaceId}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Bez zmian" />
            </SelectTrigger>
            <SelectContent>
              {MARKETPLACES.map((mp) => (
                <SelectItem key={mp.id} value={mp.id}>
                  {mp.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
