"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Package, Eye, EyeOff, ShieldCheck } from "lucide-react";
import { getErrorMessage } from "@/lib/api-client";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";

const loginSchema = z.object({
  tenant_slug: z.string().min(1, "Slug organizacji jest wymagany"),
  email: z.string().email("Nieprawidłowy adres email"),
  password: z.string().min(1, "Hasło jest wymagane"),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const { login, verify2FALogin } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [requires2FA, setRequires2FA] = useState(false);
  const [tempToken, setTempToken] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [isVerifying2FA, setIsVerifying2FA] = useState(false);
  const codeInputRef = useRef<HTMLInputElement>(null);

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
      <div className="flex flex-col items-center">
        <div className="mb-6 flex flex-col items-center gap-2">
          <Package className="h-10 w-10 text-primary" />
          <span className="text-2xl font-bold tracking-tight">OpenOMS</span>
        </div>
        <Card className="w-full">
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
              <ShieldCheck className="h-6 w-6 text-primary" />
            </div>
            <CardTitle className="text-2xl">Weryfikacja 2FA</CardTitle>
            <CardDescription>
              Kod z aplikacji uwierzytelniającej
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="totp_code">Kod weryfikacyjny</Label>
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
                className="text-center text-2xl tracking-[0.5em] font-mono"
                disabled={isVerifying2FA}
              />
              <p className="text-xs text-muted-foreground text-center">
                Wpisz 6-cyfrowy kod z aplikacji uwierzytelniającej
              </p>
            </div>
          </CardContent>
          <CardFooter className="flex flex-col gap-4">
            <Button
              className="w-full"
              disabled={totpCode.length !== 6 || isVerifying2FA}
              onClick={() => handleTotpCodeChange(totpCode)}
            >
              {isVerifying2FA ? "Weryfikacja..." : "Zweryfikuj"}
            </Button>
            <button
              type="button"
              className="text-sm text-muted-foreground hover:text-foreground"
              onClick={() => {
                setRequires2FA(false);
                setTempToken("");
                setTotpCode("");
              }}
            >
              Wróć do logowania
            </button>
          </CardFooter>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center">
      <div className="mb-6 flex flex-col items-center gap-2">
        <Package className="h-10 w-10 text-primary" />
        <span className="text-2xl font-bold tracking-tight">OpenOMS</span>
      </div>
      <Card className="w-full">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Logowanie</CardTitle>
          <CardDescription>Zaloguj się do panelu OpenOMS</CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit(onSubmit)}>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tenant_slug">Organizacja</Label>
              <Input
                id="tenant_slug"
                placeholder="moja-firma"
                {...register("tenant_slug")}
              />
              {errors.tenant_slug && (
                <p className="text-sm text-destructive">{errors.tenant_slug.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="jan@example.com"
                {...register("email")}
              />
              {errors.email && (
                <p className="text-sm text-destructive">{errors.email.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Hasło</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  className="pr-10"
                  {...register("password")}
                />
                <button
                  type="button"
                  className="absolute right-0 top-0 flex h-9 w-9 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => setShowPassword((prev) => !prev)}
                  tabIndex={-1}
                  aria-label={showPassword ? "Ukryj hasło" : "Pokaż hasło"}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
              {errors.password && (
                <p className="text-sm text-destructive">{errors.password.message}</p>
              )}
            </div>
          </CardContent>
          <CardFooter className="flex flex-col gap-4">
            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? "Logowanie..." : "Zaloguj się"}
            </Button>
            <p className="text-sm text-muted-foreground">
              Nie masz konta?{" "}
              <Link href="/register" className="text-primary underline-offset-4 hover:underline">
                Zarejestruj się
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
