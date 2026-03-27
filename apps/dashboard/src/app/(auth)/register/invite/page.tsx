"use client";

import { Suspense, useState, useMemo } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/api-client";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { useTranslations } from "next-intl";

function createRegisterSchema(t: (key: string) => string) {
  return z.object({
    tenant_name: z.string().min(1, t("validation.orgNameRequired")),
    tenant_slug: z
      .string()
      .min(1, t("validation.orgSlugRequired"))
      .regex(/^[a-z0-9-]+$/, t("validation.slugFormat")),
    name: z.string().min(1, t("validation.fullNameRequired")),
    email: z.string().email(t("validation.emailInvalid")),
    password: z.string()
      .min(8, t("validation.passwordMinLength"))
      .regex(/[A-Z]/, t("validation.passwordUppercase"))
      .regex(/[0-9]/, t("validation.passwordDigit")),
  });
}

type RegisterFormValues = z.infer<ReturnType<typeof createRegisterSchema>>;

function InviteRegisterForm() {
  const t = useTranslations("auth");
  const { register: registerUser } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const inviteToken = searchParams.get("token") || "";
  const licenseToken = searchParams.get("license_token") || "";
  const hasToken = inviteToken || licenseToken;
  const [isSubmitting, setIsSubmitting] = useState(false);

  const registerSchema = useMemo(() => createRegisterSchema(t), [t]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
  });

  if (!hasToken) {
    return (
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">{t("register.title")}</CardTitle>
          <CardDescription>{t("completeInviteRegistration")}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
            <p className="text-sm text-destructive">
              {t("missingTokenUseInviteLink")}
            </p>
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-4">
          <Link href="/register" className="text-sm text-primary underline-offset-4 hover:underline">
            {t("choosePlan")}
          </Link>
          <p className="text-sm text-muted-foreground">
            {t("alreadyHaveAccount")}{" "}
            <Link href="/login" className="text-primary underline-offset-4 hover:underline">
              {t("login.submit")}
            </Link>
          </p>
        </CardFooter>
      </Card>
    );
  }

  const onSubmit = async (data: RegisterFormValues) => {
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
        <CardTitle className="text-2xl">{t("register.title")}</CardTitle>
        <CardDescription>{t("completeInviteRegistration")}</CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="tenant_name">{t("form.orgName")} <span className="text-destructive">*</span></Label>
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
            <Label htmlFor="tenant_slug">{t("form.orgSlug")} <span className="text-destructive">*</span></Label>
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
            <Label htmlFor="name">{t("form.fullName")}<span className="text-destructive">*</span></Label>
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
            <Label htmlFor="password">{t("login.password")}<span className="text-destructive">*</span></Label>
            <Input
              id="password"
              type="password"
              placeholder={t("minimum8Characters")}
              aria-invalid={!!errors.password}
              {...register("password")}
            />
            {errors.password && (
              <p className="text-destructive text-xs mt-1">{errors.password.message}</p>
            )}
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-4">
          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? t("registering") : t("login.register")}
          </Button>
          <p className="text-sm text-muted-foreground">
            {t("alreadyHaveAccount")}{" "}
            <Link href="/login" className="text-primary underline-offset-4 hover:underline">
              {t("login.submit")}
            </Link>
          </p>
        </CardFooter>
      </form>
    </Card>
  );
}

export default function InviteRegisterPage() {
  return (
    <Suspense>
      <InviteRegisterForm />
    </Suspense>
  );
}
