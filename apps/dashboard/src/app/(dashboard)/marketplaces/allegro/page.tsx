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
  User,
  Star,
  Download,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useIntegrations,
  useCreateIntegration,
  useUpdateIntegration,
  useDeleteIntegration,
} from "@/hooks/use-integrations";
import { MarketplaceShipmentSettings } from "@/components/integrations/marketplace-shipment-settings";
import { AllegroTabNav } from "@/components/marketplaces/allegro-tab-nav";
import { useAllegroAccount } from "@/hooks/use-allegro";
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
import type { Integration } from "@/types/api";
import { useTranslations } from "next-intl";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function getRedirectURI() {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/marketplaces/allegro`;
  }
  return `${API_URL.replace(/:\d+$/, ":3000")}/marketplaces/allegro`;
}

export default function AllegroIntegrationPage() {
  const searchParams = useSearchParams();
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  if (code && state) {
    return <OAuthCallback code={code} state={state} />;
  }

  return <AllegroMainPage />;
}

function OAuthCallback({ code, state }: { code: string; state: string }) {
  const t = useTranslations("marketplaces");
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
          err instanceof Error ? err.message : t("autoryzacjaNiePowiodłaSie")
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
                {t("łaczenieZAllegro")}
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                {t("połaczonoZAllegroOknoZamknieSieAutomatycznie")}
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
                {t("closeWindow")}
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function AllegroMainPage() {
  const t = useTranslations("marketplaces");
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
            <Link href="/marketplaces">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{t("allegroIntegration")}</h1>
            <p className="text-muted-foreground">
              {t("połaczSwojeKontoAllegroAbySynchronizowacZamowienia")}
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
  const t = useTranslations("marketplaces");
  const createIntegration = useCreateIntegration();
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [isAuthorizing, setIsAuthorizing] = useState(false);

  const redirectURI = getRedirectURI();
  const devPortalURL = sandbox
    ? "https://apps.developer.allegro.pl.allegrosandbox.pl/"
    : "https://apps.developer.allegro.pl/";

  const openOAuthPopup = useCallback(async (onDone: () => void) => {
    setIsAuthorizing(true);
    try {
      const resp = await apiClient<{
        auth_url: string;
        state: string;
        redirect_uri: string;
      }>("/v1/integrations/allegro/auth-url");

      const popup = window.open(
        resp.auth_url,
        "allegro-oauth",
        "width=600,height=700,scrollbars=yes"
      );

      if (!popup) {
        toast.error(
          t("przegladarkaZablokowałaOknoPopupZezwolNaWyskakujac")
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
      toast.error(t("nieUdałoSiePobracAdresuAutoryzacjiAllegro"));
      setIsAuthorizing(false);
      onDone();
    }
  }, [t]);

  const handleSave = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error(t("clientIdIClientSecretSaWymagane"));
      return;
    }

    createIntegration.mutate(
      {
        provider: "allegro",
        label: sandbox ? "Allegro (Sandbox)" : "Allegro",
        credentials: {
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
          sandbox,
        },
      },
      {
        onSuccess: () => {
          toast.success(t("allegroSavedOpeningAuth"));
          // Automatically open OAuth popup after saving credentials
          openOAuthPopup(() => onCreated());
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("bładPodczasZapisywaniaDanych")
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
          <CardTitle>{t("krok1ZarejestrujAplikacjeWAllegro")}</CardTitle>
          <CardDescription>
            {t("przedPołaczeniemMusiszUtworzycAplikacjeWPanelu")}
            Allegro.
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
                {t("sandboxMode")}
              </Label>
            </div>
            {sandbox && (
              <p className="text-xs text-muted-foreground">
                {t("sandboxRequiresSeparateAccount")}
              </p>
            )}
          </div>

          <Separator />

          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>
              {t("goTo")}{" "}
              <a
                href={devPortalURL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-primary underline"
              >
                {sandbox
                  ? t("allegroSandboxDevCenter")
                  : t("allegroDevCenter")}
                <ExternalLink className="h-3 w-3" />
              </a>
            </li>
            <li>{t("kliknijQuotzarejestrujAplikacjequot")}</li>
            <li>
              W polu <strong>Redirect URI</strong>{t("wklejPonizszyAdres")}
            </li>
          </ol>

          <CopyableField label={t("redirectUriAllegroLabel")} value={redirectURI} />

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              {t("redirectUriMustBeExact")}
            </p>
          </div>

          <ol className="list-decimal list-inside space-y-2 text-sm" start={4}>
            <li>{t("allegroStep4CopyKeys")}</li>
          </ol>
        </CardContent>
      </Card>

      {/* Step 2: Enter credentials */}
      <Card>
        <CardHeader>
          <CardTitle>{t("krok2WprowadzDaneAplikacji")}</CardTitle>
          <CardDescription>
            {t("pasteAllegroKeys")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="client-id">Client ID</Label>
            <Input
              id="client-id"
              placeholder={t("pasteAllegroClientId")}
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
                placeholder={t("pasteAllegroClientSecret")}
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

          <Button
            onClick={handleSave}
            disabled={
              createIntegration.isPending ||
              !clientId.trim() ||
              !clientSecret.trim()
            }
          >
            {createIntegration.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <Save className="mr-2 h-4 w-4" />
            {t("zapiszIPrzejdzDoAutoryzacji")}
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
  const t = useTranslations("marketplaces");
  const updateIntegration = useUpdateIntegration(integration.id);
  const deleteIntegration = useDeleteIntegration();
  const [isReauthorizing, setIsReauthorizing] = useState(false);

  const handleDisconnect = () => {
    updateIntegration.mutate(
      { status: "inactive" },
      {
        onSuccess: () => {
          toast.success(t("integracjaAllegroZostałaDezaktywowana"));
          onRefetch();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("bładPodczasDezaktywacjiIntegracji")
          );
        },
      }
    );
  };

  const handleDelete = () => {
    if (
      !confirm(
        t("czyNaPewnoChceszUsunacIntegracjeAllegroTaOperacjaJ")
      )
    ) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success(t("integracjaAllegroZostałaUsunieta"));
        onRefetch();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("bładPodczasUsuwaniaIntegracji")
        );
      },
    });
  };

  const [debugInfo, setDebugInfo] = useState<{
    auth_url: string;
    redirect_uri: string;
  } | null>(null);

  const handleAuthorize = useCallback(() => {
    setIsReauthorizing(true);
    setDebugInfo(null);
    const doAuth = async () => {
      try {
        const resp = await apiClient<{
          auth_url: string;
          state: string;
          redirect_uri: string;
        }>("/v1/integrations/allegro/auth-url");

        setDebugInfo({
          auth_url: resp.auth_url,
          redirect_uri: resp.redirect_uri,
        });

        const popup = window.open(
          resp.auth_url,
          "allegro-oauth",
          "width=600,height=700,scrollbars=yes"
        );

        if (!popup) {
          toast.error(
            t("przegladarkaZablokowałaOknoPopupZezwolNaWyskakujac1")
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
        toast.error(t("nieUdałoSiePobracAdresuAutoryzacji"));
        setIsReauthorizing(false);
      }
    };
    doAuth();
  }, [onRefetch, t]);

  // Show OAuth prompt if status is not active OR if there's no last_sync_at (never authorized successfully)
  const needsOAuth = integration.status !== "active" || !integration.last_sync_at;

  return (
    <div className="space-y-6">
      {/* OAuth prompt if not yet authorized */}
      {needsOAuth && (
        <Card className="border-amber-200 dark:border-amber-800">
          <CardHeader>
            <CardTitle>{t("oauthAuthorization")}</CardTitle>
            <CardDescription>
              {t("daneAplikacjiZostałyZapisaneKliknijPonizejAby")}
              {t("dostepDoKontaAllegroOtworzySieOkno")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <CopyableField
                label={t("redirectUriMusiBycZarejestrowanyWAllegro")}
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
                {t("steps.integration.title")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Debug info - shows after auth attempt */}
      {debugInfo && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{t("oauthDiagnostics")}</CardTitle>
            <CardDescription>
              {t("jesliAllegroPokazujeBładSprawdzCzyPonizsze")}
              {t("sieZKonfiguracjaAplikacjiWDeveloperCenter")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CopyableField
              label={t("redirectUriWysłanyDoAllegro")}
              value={debugInfo.redirect_uri}
            />
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">
                {t("authUrlOpenedInPopup")}
              </Label>
              <code className="block rounded bg-muted px-3 py-2 text-xs font-mono break-all max-h-24 overflow-auto">
                {debugInfo.auth_url}
              </code>
            </div>
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <p className="text-xs text-amber-800 dark:text-amber-200">
                <strong>{t("czestePrzyczynyBłedu")}</strong>
              </p>
              <ul className="mt-1 list-disc list-inside text-xs text-amber-800 dark:text-amber-200 space-y-1">
                <li>
                  Client ID z <strong>produkcji</strong>{t("uzytyWTrybie")}{" "}
                  <strong>sandbox</strong> (lub odwrotnie) — konto i
                  {t("aplikacjaMuszaBycZTegoSamegoSrodowiska")}
                </li>
                <li>
                  Redirect URI niezarejestrowany w aplikacji Allegro — musi być{" "}
                  <strong>identyczny</strong> (bez trailing slash)
                </li>
                <li>
                  {t("aplikacjaNieMaWłaczonegoQuotbrowserAccessquotW")}
                  ustawieniach
                </li>
              </ul>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Sub-page tab navigation */}
      {integration.status === "active" && <AllegroTabNav />}

      {/* Account info */}
      {integration.status === "active" && <AllegroAccountCard />}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Status card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("statusPołaczenia")}</CardTitle>
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
                  {t("daneUwierzytelniajace")}
                </p>
                <p className="mt-1 font-medium">
                  {integration.has_credentials ? t("configured") : t("none")}
                </p>
              </div>
              {integration.label && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("label")}</p>
                  <p className="mt-1 font-medium">{integration.label}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("lastSync")}
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
                    {t("syncCursor")}
                  </p>
                  <p className="mt-1 font-mono text-xs truncate">
                    {integration.sync_cursor}
                  </p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">{t("integrationId")}</p>
                <p className="mt-1 font-mono text-xs">{integration.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("createdAt")}</p>
                <p className="mt-1 font-medium">
                  {formatDate(integration.created_at)}
                </p>
              </div>
            </div>

            {integration.status === "error" && integration.error_message && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
                <p className="text-sm font-medium text-destructive">
                  {t("bładIntegracji")}
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
            <CardTitle>{t("actions")}</CardTitle>
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
                {t("odswiezToken")}
              </Button>
            )}
            {integration.status === "active" && (
              <Button className="w-full" variant="outline" asChild>
                <Link href="/marketplaces/allegro/import">
                  <Download className="mr-2 h-4 w-4" />
                  {t("importOffers")}
                </Link>
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
              {t("deactivate")}
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
              {t("usunIntegracje")}
            </Button>
          </CardContent>
        </Card>

        {/* Credentials update card */}
        <CredentialsCard
          integrationId={integration.id}
          onUpdated={onRefetch}
        />
      </div>

      {/* Marketplace shipment settings */}
      <MarketplaceShipmentSettings
        provider="allegro"
        settings={
          (integration.settings ?? {}) as Record<string, unknown>
        }
        onSave={(newSettings) => {
          updateIntegration.mutate(
            { settings: newSettings },
            {
              onSuccess: () => {
                toast.success(t("ustawieniaPrzesyłekZostałyZapisane"));
                onRefetch();
              },
              onError: (error) => {
                toast.error(
                  error instanceof Error
                    ? error.message
                    : t("bładPodczasZapisywaniaUstawienPrzesyłek")
                );
              },
            }
          );
        }}
        isLoading={updateIntegration.isPending}
      />
    </div>
  );
}

function AllegroAccountCard() {
  const t = useTranslations("marketplaces");
  const { data, isLoading, isError } = useAllegroAccount();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            {t("sellerAccount")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (isError || !data) {
    return null;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <User className="h-5 w-5" />
          {t("sellerAccount")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <p className="text-sm text-muted-foreground">Login</p>
            <p className="mt-1 font-medium">{data.user.login}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Email</p>
            <p className="mt-1 font-medium">{data.user.email || "---"}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">{t("recommendations")}</p>
            <p className="mt-1 font-medium flex items-center gap-1">
              <Star className="h-4 w-4 text-yellow-500" />
              {data.quality.recommendPercentage
                ? `${data.quality.recommendPercentage}%`
                : "---"}
              {data.quality.recommendCount > 0 && (
                <span className="text-xs text-muted-foreground">
                  ({data.quality.recommendCount} {t("reviews")})
                </span>
              )}
            </p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">{t("accountId")}</p>
            <p className="mt-1 font-mono text-xs">{data.user.id}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function CredentialsCard({
  integrationId,
  onUpdated,
}: {
  integrationId: string;
  onUpdated: () => void;
}) {
  const t = useTranslations("marketplaces");
  const updateIntegration = useUpdateIntegration(integrationId);
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [sandbox, setSandbox] = useState(false);
  const [showSecret, setShowSecret] = useState(false);

  const handleUpdateCredentials = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error(t("clientIdIClientSecretSaWymagane"));
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
            t("daneZaktualizowaneKliknijPołaczZAllegroAbyPonownie")
          );
          setClientId("");
          setClientSecret("");
          onUpdated();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("bładPodczasAktualizacjiDanych")
          );
        },
      }
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("zmienDaneAplikacji")}</CardTitle>
        <CardDescription>
          {t("zaktualizujClientIdIClientSecretPo")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-client-id">Client ID</Label>
          <Input
            id="edit-client-id"
            placeholder={t("newClientId")}
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
              placeholder={t("newClientSecret")}
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
            {t("sandboxMode")}
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
          {t("updateCredentials")}
        </Button>
      </CardContent>
    </Card>
  );
}
