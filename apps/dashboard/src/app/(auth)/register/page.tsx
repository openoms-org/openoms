"use client";

import { Suspense, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/api-client";
import { useAuth } from "@/hooks/use-auth";
import { usePublicConfig } from "@/hooks/use-public-config";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";

const registerSchema = z.object({
  tenant_name: z.string().min(1, "Nazwa organizacji jest wymagana"),
  tenant_slug: z
    .string()
    .min(1, "Slug organizacji jest wymagany")
    .regex(/^[a-z0-9-]+$/, "Slug może zawierać tylko małe litery, cyfry i myślniki"),
  name: z.string().min(1, "Imię i nazwisko jest wymagane"),
  email: z.string().email("Nieprawidłowy adres email"),
  password: z.string()
    .min(8, "Hasło musi mieć minimum 8 znaków")
    .regex(/[A-Z]/, "Hasło musi zawierać wielką literę")
    .regex(/[0-9]/, "Hasło musi zawierać cyfrę"),
});

type RegisterForm = z.infer<typeof registerSchema>;

function RegisterForm() {
  const { register: registerUser } = useAuth();
  const { registration_mode } = usePublicConfig();
  const router = useRouter();
  const searchParams = useSearchParams();
  const inviteToken = searchParams.get("token") || "";
  const licenseToken = searchParams.get("license_token") || "";
  const hasToken = inviteToken || licenseToken;
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
  });

  // Redirect to login if registration is disabled
  if (registration_mode === "disabled") {
    router.replace("/login");
    return null;
  }

  const onSubmit = async (data: RegisterForm) => {
    setIsSubmitting(true);
    try {
      await registerUser({
        ...data,
        ...(inviteToken ? { invite_token: inviteToken } : {}),
        ...(licenseToken ? { license_token: licenseToken } : {}),
      });
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Card>
      <CardHeader className="text-center">
        <CardTitle className="text-2xl">Rejestracja</CardTitle>
        <CardDescription>
          {registration_mode === "invite"
            ? "Dokończ rejestrację z zaproszenia"
            : "Utwórz nową organizację w OpenOMS"}
        </CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CardContent className="space-y-4">
          {registration_mode === "invite" && !hasToken && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
              <p className="text-sm text-destructive">
                Brak tokenu. Użyj linku otrzymanego w zaproszeniu lub po zakupie subskrypcji.
              </p>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="tenant_name">Nazwa organizacji <span className="text-destructive">*</span></Label>
            <Input
              id="tenant_name"
              placeholder="Moja Firma Sp. z o.o."
              aria-invalid={!!errors.tenant_name}
              {...register("tenant_name")}
            />
            {errors.tenant_name && (
              <p className="text-destructive text-xs mt-1">{errors.tenant_name.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="tenant_slug">Slug organizacji <span className="text-destructive">*</span></Label>
            <Input
              id="tenant_slug"
              placeholder="moja-firma"
              aria-invalid={!!errors.tenant_slug}
              {...register("tenant_slug")}
            />
            {errors.tenant_slug && (
              <p className="text-destructive text-xs mt-1">{errors.tenant_slug.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="name">Imię i nazwisko <span className="text-destructive">*</span></Label>
            <Input
              id="name"
              placeholder="Jan Kowalski"
              aria-invalid={!!errors.name}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-destructive text-xs mt-1">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">Email <span className="text-destructive">*</span></Label>
            <Input
              id="email"
              type="email"
              placeholder="jan@example.com"
              aria-invalid={!!errors.email}
              {...register("email")}
            />
            {errors.email && (
              <p className="text-destructive text-xs mt-1">{errors.email.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Hasło <span className="text-destructive">*</span></Label>
            <Input
              id="password"
              type="password"
              placeholder="Minimum 8 znaków"
              aria-invalid={!!errors.password}
              {...register("password")}
            />
            {errors.password && (
              <p className="text-destructive text-xs mt-1">{errors.password.message}</p>
            )}
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || (registration_mode === "invite" && !hasToken)}
          >
            {isSubmitting ? "Rejestracja..." : "Zarejestruj się"}
          </Button>
          <p className="text-sm text-muted-foreground">
            Masz już konto?{" "}
            <Link href="/login" className="text-primary underline-offset-4 hover:underline">
              Zaloguj się
            </Link>
          </p>
        </CardFooter>
      </form>
    </Card>
  );
}

export default function RegisterPage() {
  return (
    <Suspense>
      <RegisterForm />
    </Suspense>
  );
}
