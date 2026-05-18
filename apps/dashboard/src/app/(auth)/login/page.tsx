"use client";

import { type ReactNode, useState, useRef, useEffect, useMemo } from "react";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Check, Eye, EyeOff, ShieldCheck } from "lucide-react";
import { getErrorMessage } from "@/lib/api-client";
import { useAuth } from "@/hooks/use-auth";
import { usePublicConfig } from "@/hooks/use-public-config";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useTranslations } from "next-intl";

const logoSrc = "/logos/openoms-community-logo-v2-slack-512.png";

const inputClassName = "h-11 rounded-[8px] border-[#cfdbe2] bg-white px-3 text-[#111820] shadow-[0_1px_2px_rgba(17,24,32,0.04)] placeholder:text-[#8a98a3] focus-visible:border-[#05b7c7] focus-visible:ring-[#05b7c7]/25";

function BrandLockup({ compact = false }: { compact?: boolean }) {
  return (
    <div className={compact ? "flex items-center justify-center gap-3" : "flex items-center gap-3"}>
      <img
        src={logoSrc}
        alt=""
        className={compact ? "h-11 w-11 rounded-[12px]" : "h-9 w-9 rounded-[10px]"}
        width={compact ? 44 : 36}
        height={compact ? 44 : 36}
      />
      <span className={compact ? "text-xl font-semibold tracking-normal text-[#111820]" : "text-[17px] font-semibold tracking-normal text-[#111820]"}>
        OpenOMS
      </span>
    </div>
  );
}

function ConstellationBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 overflow-hidden">
      <div className="auth-orbit auth-orbit-slow absolute left-1/2 top-[47%] h-[520px] w-[520px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#05b7c7]/20" />
      <div className="auth-orbit auth-orbit-medium absolute left-1/2 top-[47%] h-[720px] w-[720px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#111820]/10" />
      <div className="auth-orbit auth-orbit-wide absolute left-1/2 top-[47%] h-[920px] w-[920px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#f5a623]/20" />
      <svg className="absolute inset-0 h-full w-full" viewBox="0 0 1200 820" preserveAspectRatio="none">
        <path className="auth-route" d="M170 510 C360 360, 520 610, 710 410 S980 315, 1080 220" />
        <path className="auth-route auth-route-secondary" d="M210 250 C420 180, 520 300, 650 245 S860 165, 1030 330" />
      </svg>
      <span className="auth-point left-[21%] top-[31%]" />
      <span className="auth-point left-[74%] top-[28%]" />
      <span className="auth-point left-[79%] top-[67%]" />
      <span className="auth-point left-[24%] top-[69%]" />
    </div>
  );
}

function AuthConstellationShell({ children }: { children: ReactNode }) {
  return (
    <main
      className="fixed inset-0 isolate overflow-y-auto px-5 py-8 text-[#18232d] sm:px-6 sm:py-10"
      style={{
        background: "radial-gradient(circle at 50% 35%, rgba(32, 214, 208, 0.16), transparent 34%), linear-gradient(135deg, #f8fbfc 0%, #eef4f6 46%, #f6fafb 100%)",
      }}
    >
      <ConstellationBackground />
      <div className="absolute left-5 top-5 z-10 sm:left-7 sm:top-7">
        <BrandLockup />
      </div>
      <div className="relative z-10 flex min-h-[calc(100dvh-4rem)] items-center justify-center py-12 sm:min-h-[calc(100dvh-5rem)]">
        {children}
      </div>
      <style>{`
        .auth-route {
          fill: none;
          stroke: rgba(5, 183, 199, 0.28);
          stroke-width: 1.4;
          stroke-dasharray: 8 16;
          animation: auth-dash 18s linear infinite;
        }
        .auth-route-secondary {
          stroke: rgba(245, 166, 35, 0.2);
          animation-duration: 24s;
        }
        .auth-orbit {
          transform-origin: center;
          animation: auth-spin 34s linear infinite;
        }
        .auth-orbit-medium {
          animation-duration: 44s;
          animation-direction: reverse;
        }
        .auth-orbit-wide {
          animation-duration: 58s;
        }
        .auth-point {
          position: absolute;
          width: 10px;
          height: 10px;
          border-radius: 9999px;
          background: #05b7c7;
          box-shadow: 0 0 0 8px rgba(5, 183, 199, 0.1);
          animation: auth-pulse 3.8s ease-in-out infinite;
        }
        @keyframes auth-dash {
          to { stroke-dashoffset: -160; }
        }
        @keyframes auth-spin {
          to { transform: translate(-50%, -50%) rotate(360deg); }
        }
        @keyframes auth-pulse {
          0%, 100% { opacity: 0.58; transform: scale(1); }
          50% { opacity: 1; transform: scale(1.18); }
        }
        @media (prefers-reduced-motion: reduce) {
          .auth-route,
          .auth-orbit,
          .auth-point {
            animation: none !important;
          }
        }
        @media (max-width: 640px) {
          .auth-route {
            opacity: 0.42;
          }
          .auth-point {
            opacity: 0.5;
          }
        }
      `}</style>
    </main>
  );
}

function AuthCard({ children }: { children: ReactNode }) {
  return (
    <section className="w-full max-w-[430px] rounded-[14px] border border-[#d9e2e8]/85 bg-white/90 px-6 py-8 shadow-[0_24px_70px_rgba(17,24,32,0.13)] backdrop-blur-xl sm:px-8 sm:py-9">
      <div className="mb-7 flex flex-col items-center gap-4 text-center">
        <BrandLockup compact />
      </div>
      {children}
    </section>
  );
}

function SecurityNote({ label }: { label: string }) {
  return (
    <div className="mt-1 flex items-start gap-2 rounded-[10px] border border-[#d9e2e8]/70 bg-[#f8fbfc]/80 px-3 py-3 text-xs leading-5 text-[#63717d]">
      <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[#05b7c7]/12 text-[#007d89]">
        <Check className="h-3.5 w-3.5" />
      </span>
      <span>{label}</span>
    </div>
  );
}

function FieldError({ message }: { message?: string }) {
  if (!message) {
    return null;
  }

  return <p className="mt-1 text-xs text-destructive">{message}</p>;
}

function createLoginSchema(t: (key: string) => string) {
  return z.object({
    tenant_slug: z.string().min(1, t("validation.orgRequired")),
    email: z.string().email(t("validation.emailInvalid")),
    password: z.string().min(1, t("validation.passwordRequired")),
  });
}

type LoginForm = z.infer<ReturnType<typeof createLoginSchema>>;

export default function LoginPage() {
  const t = useTranslations("auth");
  const { login, verify2FALogin } = useAuth();
  const config = usePublicConfig();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [requires2FA, setRequires2FA] = useState(false);
  const [tempToken, setTempToken] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [isVerifying2FA, setIsVerifying2FA] = useState(false);
  const codeInputRef = useRef<HTMLInputElement>(null);

  const loginSchema = useMemo(() => createLoginSchema(t), [t]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  });

  useEffect(() => {
    if (requires2FA && codeInputRef.current) {
      codeInputRef.current.focus();
    }
  }, [requires2FA]);

  const onSubmit = async (data: LoginForm) => {
    setIsSubmitting(true);
    try {
      const result = await login(data);
      if (result.requires2FA && result.tempToken) {
        setRequires2FA(true);
        setTempToken(result.tempToken);
      }
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTotpCodeChange = async (value: string) => {
    // Only allow digits
    const digits = value.replace(/\D/g, "").slice(0, 6);
    setTotpCode(digits);

    // Auto-submit when 6 digits are entered
    if (digits.length === 6) {
      setIsVerifying2FA(true);
      try {
        await verify2FALogin(tempToken, digits);
      } catch (error) {
        toast.error(getErrorMessage(error));
        setTotpCode("");
      } finally {
        setIsVerifying2FA(false);
      }
    }
  };

  if (requires2FA) {
    return (
      <AuthConstellationShell>
        <AuthCard>
          <div className="mb-7 text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-[#05b7c7]/25 bg-[#05b7c7]/10 text-[#007d89]">
              <ShieldCheck className="h-6 w-6" />
            </div>
            <h1 className="text-[27px] font-semibold leading-tight tracking-normal text-[#111820]">{t("twoFa.title")}</h1>
            <p className="mt-2 text-sm leading-6 text-[#63717d]">
              {t("twoFa.subtitle")}
            </p>
          </div>
          <div className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="totp_code" className="text-sm font-medium text-[#18232d]">{t("twoFa.codeLabel")}</Label>
              <Input
                ref={codeInputRef}
                id="totp_code"
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                placeholder="000000"
                maxLength={6}
                value={totpCode}
                onChange={(e) => handleTotpCodeChange(e.target.value)}
                className={`${inputClassName} h-12 text-center font-mono text-2xl`}
                disabled={isVerifying2FA}
              />
              <p className="text-center text-xs leading-5 text-[#63717d]">
                {t("twoFa.codeHint")}
              </p>
            </div>
            <Button
              className="h-11 w-full rounded-[8px] bg-[#111820] text-white shadow-[0_12px_24px_rgba(17,24,32,0.18)] hover:bg-[#18232d] focus-visible:ring-[#05b7c7]/30"
              disabled={totpCode.length !== 6 || isVerifying2FA}
              onClick={() => handleTotpCodeChange(totpCode)}
            >
              {isVerifying2FA ? t("twoFa.verifying") : t("twoFa.verify")}
            </Button>
            <button
              type="button"
              className="w-full text-center text-sm text-[#63717d] transition-colors hover:text-[#111820] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#05b7c7]/35"
              onClick={() => {
                setRequires2FA(false);
                setTempToken("");
                setTotpCode("");
              }}
            >
              {t("twoFa.backToLogin")}
            </button>
            <SecurityNote label={t("login.securityNote")} />
          </div>
        </AuthCard>
      </AuthConstellationShell>
    );
  }

  return (
    <AuthConstellationShell>
      <AuthCard>
        <div className="mb-7 text-center">
          <h1 className="text-[27px] font-semibold leading-tight tracking-normal text-[#111820]">{t("login.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-[#63717d]">{t("login.subtitle")}</p>
        </div>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tenant_slug" className="text-sm font-medium text-[#18232d]">
                {t("login.organization")} <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="tenant_slug"
                autoComplete="organization"
                placeholder={t("login.organizationPlaceholder")}
                aria-invalid={!!errors.tenant_slug}
                className={inputClassName}
                {...register("tenant_slug")}
              />
              <FieldError message={errors.tenant_slug?.message} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email" className="text-sm font-medium text-[#18232d]">
                {t("login.email")} <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="jan@example.com"
                aria-invalid={!!errors.email}
                className={inputClassName}
                {...register("email")}
              />
              <FieldError message={errors.email?.message} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password" className="text-sm font-medium text-[#18232d]">
                {t("login.password")} <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  className={`${inputClassName} pr-11`}
                  aria-invalid={!!errors.password}
                  {...register("password")}
                />
                <button
                  type="button"
                  className="absolute right-1 top-1 flex h-9 w-9 items-center justify-center rounded-[8px] text-[#63717d] transition-colors hover:bg-[#f1f6f7] hover:text-[#111820] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#05b7c7]/35"
                  onClick={() => setShowPassword((prev) => !prev)}
                  tabIndex={-1}
                  aria-label={showPassword ? t("login.hidePassword") : t("login.showPassword")}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
              <FieldError message={errors.password?.message} />
            </div>
          </div>
          <div className="space-y-4">
            <Button
              type="submit"
              className="h-11 w-full rounded-[8px] bg-[#111820] text-white shadow-[0_12px_24px_rgba(17,24,32,0.18)] hover:bg-[#18232d] focus-visible:ring-[#05b7c7]/30"
              disabled={isSubmitting}
            >
              {isSubmitting ? t("login.submitting") : t("login.submit")}
            </Button>
            {(config.registration_mode === "open" || (config.registration_mode === "invite" && config.license_enabled)) && (
              <p className="text-center text-sm text-[#63717d]">
                {t("login.noAccount")}{" "}
                <Link href="/register" className="font-medium text-[#007d89] underline-offset-4 hover:underline">
                  {t("login.register")}
                </Link>
              </p>
            )}
            <SecurityNote label={t("login.securityNote")} />
          </div>
        </form>
      </AuthCard>
    </AuthConstellationShell>
  );
}
