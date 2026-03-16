"use client";

import { useState, useMemo, useCallback, useEffect, useRef, Suspense } from "react";
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
import { useTranslations } from "next-intl";
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

const MARKETPLACE_IDS = [
  { id: "A1C3SOZRARQ6R3", key: "pl" },
  { id: "A1PA6795UKMFR9", key: "de" },
  { id: "A1F83G8C2ARO7P", key: "uk" },
  { id: "A13V1IB3VIYZZH", key: "fr" },
  { id: "APJ6JRA9NG5V4", key: "it" },
  { id: "A1RKKUPIHCS9HS", key: "es" },
  { id: "A21TJRUUN4KGV", key: "in" },
  { id: "ATVPDKIKX0DER", key: "us" },
];

function getRedirectURI() {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/marketplaces/amazon`;
  }
  return "";
}

export default function AmazonIntegrationPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[50vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      }
    >
      <AmazonIntegrationPageInner />
    </Suspense>
  );
}

function AmazonIntegrationPageInner() {
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
  const t = useTranslations("marketplaces");
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
          err instanceof Error ? err.message : t("amazon.authFailed")
        );
      });
  }, [code, state, sellingPartnerId, t]);

  return (
    <div className="flex min-h-[50vh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col items-center gap-4 pt-6">
          {status === "loading" && (
            <>
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t("amazon.connecting")}
              </p>
            </>
          )}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <p className="text-sm font-medium">
                {t("amazon.connectedSuccess")}
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
                {t("amazon.closeWindow")}
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function AmazonMainPage() {
  const t = useTranslations("marketplaces");
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
            <h1 className="text-2xl font-bold">{t("amazon.title")}</h1>
            <p className="text-muted-foreground">
              {t("amazon.subtitle")}
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
  const t = useTranslations("marketplaces");
  const createIntegration = useCreateIntegration();
  const [applicationId, setApplicationId] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [marketplaceId, setMarketplaceId] = useState(MARKETPLACE_IDS[0].id);
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
          toast.error(t("amazon.popupBlocked"));
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
        toast.error(t("amazon.authUrlError"));
        setIsAuthorizing(false);
        onDone();
      }
    },
    [t]
  );

  const handleSaveAndAuthorize = () => {
    if (
      !applicationId.trim() ||
      !clientId.trim() ||
      !clientSecret.trim() ||
      !marketplaceId
    ) {
      toast.error(t("amazon.setup.fieldsRequired"));
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
          toast.success(t("amazon.setup.savedOpeningAuth"));
          openOAuthPopup(() => onCreated());
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("amazon.setup.saveError")
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
      toast.error(t("amazon.setup.allFieldsRequired"));
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
      toast.success(t("amazon.setup.manualSetupSuccess"));
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("amazon.setup.manualSetupError")
      );
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("amazon.setup.step1Title")}</CardTitle>
          <CardDescription>
            {t("amazon.setup.step1Description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>
              {t("amazon.setup.step1GoTo")}{" "}
              <a
                href="https://sellercentral.amazon.pl/apps/manage"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-primary underline"
              >
                {t("amazon.setup.step1DevelopApps")}
                <ExternalLink className="h-3 w-3" />
              </a>
            </li>
            <li>{t("amazon.setup.step1CreateApp")}</li>
            <li>
              {t.rich("amazon.setup.step1RedirectUri", {
                field: (chunks) => <strong>{chunks}</strong>,
              })}
            </li>
          </ol>

          <CopyableField
            label={t("amazon.setup.redirectUriLabel")}
            value={redirectURI}
          />

          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              {t.rich("amazon.setup.redirectUriWarning", {
                exact: (chunks) => <strong>{chunks}</strong>,
              })}
            </p>
          </div>

          <ol className="list-decimal list-inside space-y-2 text-sm" start={4}>
            <li>
              {t.rich("amazon.setup.step1CopyKeys", {
                applicationId: (chunks) => <strong>{chunks}</strong>,
                clientId: (chunks) => <strong>{chunks}</strong>,
                clientSecret: (chunks) => <strong>{chunks}</strong>,
              })}
            </li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("amazon.setup.step2Title")}</CardTitle>
          <CardDescription>
            {t("amazon.setup.step2Description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="application-id">{t("amazon.setup.applicationIdLabel")}</Label>
            <Input
              id="application-id"
              placeholder={t("amazon.setup.applicationIdPlaceholder")}
              value={applicationId}
              onChange={(e) => setApplicationId(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              {t("amazon.setup.applicationIdHelp")}
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="client-id">{t("amazon.setup.clientIdLabel")}</Label>
            <Input
              id="client-id"
              placeholder={t("amazon.setup.clientIdPlaceholder")}
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="client-secret">{t("amazon.setup.clientSecretLabel")}</Label>
            <div className="relative">
              <Input
                id="client-secret"
                type={showSecret ? "text" : "password"}
                placeholder={t("amazon.setup.clientSecretPlaceholder")}
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
            <Label>{t("amazon.setup.marketplaceLabel")}</Label>
            <Select value={marketplaceId} onValueChange={setMarketplaceId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("amazon.setup.marketplacePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {MARKETPLACE_IDS.map((mp) => (
                  <SelectItem key={mp.id} value={mp.id}>
                    {t(`amazon.marketplaceLabels.${mp.key}`)}
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
              {t("amazon.setup.sandboxLabel")}
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
            {t("amazon.setup.saveAndConnect")}
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
              {t("amazon.setup.manualSetupToggle")}
            </Button>
          </div>

          {showManualSetup && (
            <div className="rounded-md border p-4 space-y-4">
              <p className="text-sm text-muted-foreground">
                {t("amazon.setup.manualSetupDescription")}
              </p>
              <div className="space-y-2">
                <Label htmlFor="refresh-token">{t("amazon.setup.refreshTokenLabel")}</Label>
                <Input
                  id="refresh-token"
                  type="password"
                  placeholder={t("amazon.setup.refreshTokenPlaceholder")}
                  value={refreshToken}
                  onChange={(e) => setRefreshToken(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  {t("amazon.setup.refreshTokenHelp")}
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
                {t("amazon.setup.manualSaveButton")}
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
  const t = useTranslations("marketplaces");
  const updateIntegration = useUpdateIntegration(integration.id);
  const deleteIntegration = useDeleteIntegration();
  const [isReauthorizing, setIsReauthorizing] = useState(false);

  const handleDisconnect = () => {
    updateIntegration.mutate(
      { status: "inactive" },
      {
        onSuccess: () => {
          toast.success(t("amazon.deactivated"));
          onRefetch();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : t("amazon.deactivateError")
          );
        },
      }
    );
  };

  const handleDelete = () => {
    if (!confirm(t("amazon.deleteConfirm"))) {
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        toast.success(t("amazon.deleted"));
        onRefetch();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : t("amazon.deleteError")
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
          toast.error(t("amazon.popupBlockedShort"));
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
        toast.error(t("amazon.authUrlError"));
        setIsReauthorizing(false);
      }
    };
    doAuth();
  }, [onRefetch, t]);

  const needsOAuth = integration.status !== "active";

  return (
    <div className="space-y-6">
      {needsOAuth && (
        <Card className="border-amber-200 dark:border-amber-800">
          <CardHeader>
            <CardTitle>{t("amazon.oauth.title")}</CardTitle>
            <CardDescription>
              {t("amazon.oauth.description")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <CopyableField
                label={t("amazon.oauth.redirectUriLabel")}
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
                {t("amazon.connectButton")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("amazon.status.connectionStatus")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">{t("amazon.status.statusLabel")}</p>
                <div className="mt-1">
                  <StatusBadge
                    status={integration.status}
                    statusMap={INTEGRATION_STATUSES}
                  />
                </div>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("amazon.status.credentials")}
                </p>
                <p className="mt-1 font-medium">
                  {integration.has_credentials ? t("amazon.status.configured") : t("amazon.status.notConfigured")}
                </p>
              </div>
              {integration.label && (
                <div>
                  <p className="text-sm text-muted-foreground">{t("amazon.status.label")}</p>
                  <p className="mt-1 font-medium">{integration.label}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("amazon.status.lastSync")}
                </p>
                <p className="mt-1 font-medium">
                  {integration.last_sync_at
                    ? formatDate(integration.last_sync_at)
                    : "---"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("amazon.status.integrationId")}</p>
                <p className="mt-1 font-mono text-xs">{integration.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("amazon.status.createdAt")}</p>
                <p className="mt-1 font-medium">
                  {formatDate(integration.created_at)}
                </p>
              </div>
            </div>

            {integration.status === "error" && integration.error_message && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
                <p className="text-sm font-medium text-destructive">
                  {t("amazon.integrationError")}
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
            <CardTitle>{t("amazon.actions.title")}</CardTitle>
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
                {t("amazon.refreshToken")}
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
              {t("amazon.deactivate")}
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
              {t("amazon.deleteIntegration")}
            </Button>
          </CardContent>
        </Card>

        <CredentialsCard
          onUpdated={onRefetch}
        />
      </div>
    </div>
  );
}

function CredentialsCard({ onUpdated }: { onUpdated: () => void }) {
  const t = useTranslations("marketplaces");
  const [applicationId, setApplicationId] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [marketplaceId, setMarketplaceId] = useState("");
  const [showSecret, setShowSecret] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const handleUpdateCredentials = async () => {
    if (!clientId.trim() || !clientSecret.trim()) {
      toast.error(t("amazon.credentials.clientIdRequired"));
      return;
    }

    setIsSaving(true);
    try {
      const body: Record<string, string> = {
        client_id: clientId.trim(),
        client_secret: clientSecret.trim(),
      };
      if (applicationId.trim()) {
        body.application_id = applicationId.trim();
      }
      if (marketplaceId) {
        body.marketplace_id = marketplaceId;
      }

      await apiClient("/v1/integrations/amazon/credentials", {
        method: "PUT",
        body: JSON.stringify(body),
      });

      toast.success(t("amazon.credentials.updated"));
      setApplicationId("");
      setClientId("");
      setClientSecret("");
      setMarketplaceId("");
      onUpdated();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t("amazon.credentials.updateError")
      );
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("amazon.credentials.title")}</CardTitle>
        <CardDescription>
          {t("amazon.credentials.description")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-application-id">{t("amazon.setup.applicationIdLabel")}</Label>
          <Input
            id="edit-application-id"
            placeholder={t("amazon.credentials.newApplicationId")}
            value={applicationId}
            onChange={(e) => setApplicationId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-client-id">{t("amazon.setup.clientIdLabel")}</Label>
          <Input
            id="edit-client-id"
            placeholder={t("amazon.credentials.newClientId")}
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit-client-secret">{t("amazon.setup.clientSecretLabel")}</Label>
          <div className="relative">
            <Input
              id="edit-client-secret"
              type={showSecret ? "text" : "password"}
              placeholder={t("amazon.credentials.newClientSecret")}
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
          <Label>{t("amazon.setup.marketplaceLabel")}</Label>
          <Select value={marketplaceId} onValueChange={setMarketplaceId}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t("amazon.credentials.marketplaceNoChange")} />
            </SelectTrigger>
            <SelectContent>
              {MARKETPLACE_IDS.map((mp) => (
                <SelectItem key={mp.id} value={mp.id}>
                  {t(`amazon.marketplaceLabels.${mp.key}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <Button
          onClick={handleUpdateCredentials}
          disabled={isSaving || !clientId.trim() || !clientSecret.trim()}
          variant="outline"
        >
          {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          <Save className="mr-2 h-4 w-4" />
          {t("amazon.credentials.updateButton")}
        </Button>
      </CardContent>
    </Card>
  );
}
