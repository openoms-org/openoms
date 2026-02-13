"use client";

import { useState } from "react";
import Link from "next/link";
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

const registerSchema = z.object({
  tenant_name: z.string().min(1, "Nazwa organizacji jest wymagana"),
  tenant_slug: z
    .string()
    .min(1, "Slug organizacji jest wymagany")
    .regex(/^[a-z0-9-]+$/, "Slug może zawierać tylko małe litery, cyfry i myślniki"),
  name: z.string().min(1, "Imię i nazwisko jest wymagane"),
  email: z.string().email("Nieprawidłowy adres email"),
  password: z.string().min(8, "Hasło musi mieć minimum 8 znaków"),
});

type RegisterForm = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const { register: registerUser } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
  });

  const onSubmit = async (data: RegisterForm) => {
    setIsSubmitting(true);
    try {
      await registerUser(data);
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
        <CardDescription>Utwórz nową organizację w OpenOMS</CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CardContent className="space-y-4">
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
          <Button type="submit" className="w-full" disabled={isSubmitting}>
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
