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
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Integration } from "@/types/api";
import { useTranslations } from "next-intl";

function getRedirectURI() {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/marketplaces/olx`;
  }
  return "";
}

export default function OlxIntegrationPage() {
  const searchParams = useSearchParams();
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  if (code && state) {
    return <OAuthCallback code={code} state={state} />;
  }

  return <OlxMainPage />;
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

    apiClient("/v1/integrations/olx/callback", {
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
                {t("łaczenieZOlx")}
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                {t("połaczonoZOlxOknoZamknieSieAutomatycznie")}
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

function OlxMainPage() {
  const t = useTranslations("marketplaces");
  const { data: integrations, isLoading, refetch } = useIntegrations();

  const olx = useMemo(
    () => integrations?.find((i) => i.provider === "olx") ?? null,
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
            <h1 className="text-2xl font-bold">Integracja OLX</h1>
            <p className="text-muted-foreground">
              {t("połaczSwojeKontoOlxAbySynchronizowacOgłoszenia")}
              transakcje
            </p>
          </div>
        </div>

        {olx ? (
          <ConnectedState integration={olx} onRefetch={refetch} />
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
  const [showSecret, setShowSecret] = useState(false);
  const [isAuthorizing, setIsAuthorizing] = useState(false);

  const redirectURI = getRedirectURI();

  const openOAuthPopup = useCallback(async (onDone: () => void) => {
    setIsAuthorizing(true);
    try {
      const resp = await apiClient<{
        auth_url: string;
        state: string;
        redirect_uri: string;
      }>("/v1/integrations/olx/auth-url");

      const popup = window.open(
        resp.auth_url,
        "olx-oauth",
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
      toast.error(t("nieUdałoSiePobracAdresuAutoryzacjiOlx"));
      setIsAuthorizing(false);
      onDone();
    }
  }, []);

  const handleSave = () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error(t("clientIdIClientSecretSaWymagane"));
      return;
    }

    createIntegration.mutate(
      {
        provider: "olx",
        label: "OLX",
        credentials: {
          client_id: clientId.trim(),
          client_secret: clientSecret.trim(),
        },
      },
      {
        onSuccess: () => {
          toast.success("Dane OLX zapisane. Otwieranie autoryzacji...");
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
      <Card>
        <CardHeader>
          <CardTitle>{t("krok1ZarejestrujAplikacjeWOlx")}</CardTitle>
          <CardDescription>
            {t("przedPołaczeniemMusiszUtworzycAplikacjeWPanelu")}
            OLX.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>
              Przejdź do{" "}
              <a
                href="https://developer.olx.pl/"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-primary underline"
              >
                OLX Developer Portal
                <ExternalLink className="h-3 w-3" />
              </a>
            </li>
            <li>{t("zarejestrujNowaAplikacje")}</li>
            <li>
              W polu <strong>Adres przekierowania (Redirect URI)</strong> wklej
              {t("ponizszyAdres")}
            </li>
          </ol>

          <CopyableField
            label="Redirect URI (do wklejenia w ustawieniach aplikacji OLX)"
            value={redirectURI}
          />

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              Redirect URI musi być <strong>{t("dokładnieTakiSam")}</strong> jak
              {t("powyzejRoznicaWNawetJednymZnakuSpowoduje")}
            </p>
          </div>

          <ol className="list-decimal list-inside space-y-2 text-sm" start={4}>
            <li>Po rejestracji skopiuj Client ID i Client Secret</li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("krok2WprowadzDaneAplikacji")}</CardTitle>
          <CardDescription>
            Wklej Client ID i Client Secret z panelu deweloperskiego OLX.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="client-id">Client ID</Label>
            <Input
              id="client-id"
              placeholder="Wklej Client ID aplikacji OLX"
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
                placeholder="Wklej Client Secret aplikacji OLX"
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
              isAuthorizing ||
              !clientId.trim() ||
              !clientSecret.trim()
            }
          >
            {(createIntegration.isPending || isAuthorizing) && (
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
          toast.success(t("integracjaOlxZostałaDezaktywowana"));
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
        t("czyNaPewnoChceszUsunacIntegracjeOlxTaOperacjaJestN")
      )
    ) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success(t("integracjaOlxZostałaUsunieta"));
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

  const handleAuthorize = useCallback(() => {
    setIsReauthorizing(true);
    const doAuth = async () => {
      try {
        const resp = await apiClient<{
          auth_url: string;
          state: string;
          redirect_uri: string;
        }>("/v1/integrations/olx/auth-url");

        const popup = window.open(
          resp.auth_url,
          "olx-oauth",
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
              {t("daneAplikacjiZostałyZapisaneKliknijPonizejAby")}
              {t("dostepDoKontaOlxOtworzySieOkno")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <CopyableField
                label={t("redirectUriMusiBycZarejestrowanyWOlx")}
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
                {t("połaczZOlx")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
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
                  {t("bładIntegracji")}
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
                {t("odswiezToken")}
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
              {t("usunIntegracje")}
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
  const t = useTranslations("marketplaces");
  const updateIntegration = useUpdateIntegration(integrationId);
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
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
        },
      },
      {
        onSuccess: () => {
          toast.success(
            t("daneZaktualizowaneKliknijPołaczZOlxAbyPonownieAuto")
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
