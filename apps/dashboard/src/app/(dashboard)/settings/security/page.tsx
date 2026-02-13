"use client";

import { useState, useEffect, useRef } from "react";
import { toast } from "sonner";
import { ShieldCheck, ShieldOff, Loader2, Copy, CheckCircle2 } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient, getErrorMessage } from "@/lib/api-client";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import QRCode from "qrcode";
import type { TwoFASetupResponse, TwoFAStatusResponse } from "@/types/api";

function QRCodeCanvas({ data, size }: { data: string; size: number }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    if (canvasRef.current && data) {
      QRCode.toCanvas(canvasRef.current, data, {
        width: size,
        margin: 1,
        color: { dark: "#000000", light: "#ffffff" },
      });
    }
  }, [data, size]);

  return <canvas ref={canvasRef} />;
}

export default function SecuritySettingsPage() {
  const queryClient = useQueryClient();
  const [showSetupDialog, setShowSetupDialog] = useState(false);
  const [showDisableDialog, setShowDisableDialog] = useState(false);
  const [setupData, setSetupData] = useState<TwoFASetupResponse | null>(null);
  const [verifyCode, setVerifyCode] = useState("");
  const [disablePassword, setDisablePassword] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [secretCopied, setSecretCopied] = useState(false);

  const { data: status, isLoading } = useQuery({
    queryKey: ["2fa-status"],
    queryFn: () => apiClient<TwoFAStatusResponse>("/v1/auth/2fa/status"),
  });

  const setupMutation = useMutation({
    mutationFn: () =>
      apiClient<TwoFASetupResponse>("/v1/auth/2fa/setup", { method: "POST" }),
    onSuccess: (data) => {
      setSetupData(data);
      setShowSetupDialog(true);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const verifyMutation = useMutation({
    mutationFn: (code: string) =>
      apiClient("/v1/auth/2fa/verify", {
        method: "POST",
        body: JSON.stringify({ code }),
      }),
    onSuccess: () => {
      toast.success("Uwierzytelnianie dwuskładnikowe zostało włączone");
      setShowSetupDialog(false);
      setSetupData(null);
      setVerifyCode("");
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
      setVerifyCode("");
    },
  });

  const disableMutation = useMutation({
    mutationFn: (data: { password: string; code: string }) =>
      apiClient("/v1/auth/2fa/disable", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      toast.success("Uwierzytelnianie dwuskładnikowe zostało wyłączone");
      setShowDisableDialog(false);
      setDisablePassword("");
      setDisableCode("");
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const copySecret = async () => {
    if (setupData?.secret) {
      await navigator.clipboard.writeText(setupData.secret);
      setSecretCopied(true);
      setTimeout(() => setSecretCopied(false), 2000);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Bezpieczeństwo</h1>
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Bezpieczeństwo</h1>
        <p className="text-muted-foreground">
          Zarządzaj ustawieniami bezpieczeństwa konta
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            Uwierzytelnianie dwuskładnikowe (2FA)
          </CardTitle>
          <CardDescription>
            Dodatkowa warstwa zabezpieczeń dla Twojego konta. Wymaga kodu z
            aplikacji uwierzytelniającej (np. Google Authenticator, Authy) przy
            każdym logowaniu.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">Status</p>
              {status?.enabled ? (
                <div className="flex items-center gap-2">
                  <Badge variant="default" className="bg-green-600">
                    2FA aktywne
                  </Badge>
                  {status.verified_at && (
                    <span className="text-xs text-muted-foreground">
                      Aktywowane:{" "}
                      {new Date(status.verified_at).toLocaleDateString("pl-PL")}
                    </span>
                  )}
                </div>
              ) : (
                <Badge variant="secondary">Wyłączone</Badge>
              )}
            </div>
            <div>
              {status?.enabled ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setShowDisableDialog(true)}
                >
                  <ShieldOff className="mr-2 h-4 w-4" />
                  Wyłącz 2FA
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={() => setupMutation.mutate()}
                  disabled={setupMutation.isPending}
                >
                  {setupMutation.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <ShieldCheck className="mr-2 h-4 w-4" />
                  )}
                  Włącz 2FA
                </Button>
              )}
            </div>
          </div>

          <Separator />

          <div className="rounded-lg bg-muted/50 p-4">
            <h4 className="text-sm font-medium mb-2">
              Jak działa uwierzytelnianie dwuskładnikowe?
            </h4>
            <ol className="text-sm text-muted-foreground space-y-1 list-decimal list-inside">
              <li>Zainstaluj aplikację uwierzytelniającą (Google Authenticator, Authy, itp.)</li>
              <li>Kliknij &quot;Włącz 2FA&quot; i zeskanuj kod QR w aplikacji</li>
              <li>Wpisz 6-cyfrowy kod z aplikacji, aby potwierdzić</li>
              <li>Przy każdym logowaniu będziesz proszony o kod z aplikacji</li>
            </ol>
          </div>
        </CardContent>
      </Card>

      {/* Setup Dialog */}
      <Dialog open={showSetupDialog} onOpenChange={setShowSetupDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Konfiguracja 2FA</DialogTitle>
            <DialogDescription>
              Zeskanuj kod QR w aplikacji uwierzytelniającej lub wpisz klucz ręcznie
            </DialogDescription>
          </DialogHeader>

          {setupData && (
            <div className="space-y-4">
              {/* QR Code rendered client-side (no external service) */}
              <div className="flex justify-center">
                <div className="rounded-lg border bg-white p-4">
                  <QRCodeCanvas data={setupData.qr_url} size={200} />
                </div>
              </div>

              {/* Manual secret */}
              <div className="space-y-2">
                <Label>Klucz ręczny</Label>
                <div className="flex gap-2">
                  <Input
                    readOnly
                    value={setupData.secret}
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={copySecret}
                  >
                    {secretCopied ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>

              <Separator />

              {/* Verification */}
              <div className="space-y-2">
                <Label htmlFor="verify-code">Kod weryfikacyjny</Label>
                <Input
                  id="verify-code"
                  type="text"
                  inputMode="numeric"
                  placeholder="000000"
                  maxLength={6}
                  value={verifyCode}
                  onChange={(e) =>
                    setVerifyCode(e.target.value.replace(/\D/g, "").slice(0, 6))
                  }
                  className="text-center text-lg tracking-widest font-mono"
                />
                <p className="text-xs text-muted-foreground">
                  Wpisz 6-cyfrowy kod z aplikacji uwierzytelniającej aby
                  potwierdzić konfigurację
                </p>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowSetupDialog(false);
                setSetupData(null);
                setVerifyCode("");
              }}
            >
              Anuluj
            </Button>
            <Button
              onClick={() => verifyMutation.mutate(verifyCode)}
              disabled={verifyCode.length !== 6 || verifyMutation.isPending}
            >
              {verifyMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              Potwierdź i włącz
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Disable Dialog */}
      <Dialog open={showDisableDialog} onOpenChange={setShowDisableDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Wyłącz uwierzytelnianie dwuskładnikowe</DialogTitle>
            <DialogDescription>
              Podaj hasło i aktualny kod 2FA, aby wyłączyć uwierzytelnianie
              dwuskładnikowe. Twoje konto będzie mniej bezpieczne.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="disable-password">Hasło</Label>
              <Input
                id="disable-password"
                type="password"
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="disable-code">Kod 2FA</Label>
              <Input
                id="disable-code"
                type="text"
                inputMode="numeric"
                placeholder="000000"
                maxLength={6}
                value={disableCode}
                onChange={(e) =>
                  setDisableCode(e.target.value.replace(/\D/g, "").slice(0, 6))
                }
                className="text-center text-lg tracking-widest font-mono"
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowDisableDialog(false);
                setDisablePassword("");
                setDisableCode("");
              }}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                disableMutation.mutate({
                  password: disablePassword,
                  code: disableCode,
                })
              }
              disabled={
                !disablePassword ||
                disableCode.length !== 6 ||
                disableMutation.isPending
              }
            >
              {disableMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              Wyłącz 2FA
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
