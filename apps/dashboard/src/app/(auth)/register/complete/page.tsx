"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { API_URL, getErrorMessage } from "@/lib/api-client";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { CheckoutSessionStatus } from "@/types/api";

const registerSchema = z.object({
  tenant_name: z.string().min(1, "Nazwa organizacji jest wymagana"),
  tenant_slug: z
    .string()
    .min(1, "Slug organizacji jest wymagany")
    .regex(/^[a-z0-9-]+$/, "Slug może zawierać tylko małe litery, cyfry i myślniki"),
  name: z.string().min(1, "Imię i nazwisko jest wymagane"),
  password: z.string()
    .min(8, "Hasło musi mieć minimum 8 znaków")
    .regex(/[A-Z]/, "Hasło musi zawierać wielką literę")
    .regex(/[0-9]/, "Hasło musi zawierać cyfrę"),
});

type CompleteFormValues = z.infer<typeof registerSchema>;

function CompleteRegistrationForm() {
  const { register: registerUser } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session_id") || "";

  const [session, setSession] = useState<CheckoutSessionStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [retryCount, setRetryCount] = useState(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CompleteFormValues>({
    resolver: zodResolver(registerSchema),
  });

  useEffect(() => {
    if (!sessionId) {
      setError("Brak identyfikatora sesji płatności");
      return;
    }

    let cancelled = false;
    const controller = new AbortController();

    const poll = async () => {
      const maxAttempts = 15;
      const intervalMs = 2000;

      for (let attempt = 0; attempt < maxAttempts; attempt++) {
        if (cancelled) return;
        try {
          const res = await fetch(`${API_URL}/v1/billing/checkout/${sessionId}`, {
            signal: controller.signal,
            credentials: "include",
          });
          if (!res.ok) {
            if (res.status === 404) {
              setError("Sesja płatności nie została znaleziona");
              return;
            }
            throw new Error("Błąd serwera");
          }
          const data: CheckoutSessionStatus = await res.json();

          if (data.status === "registered") {
            setError("Ta sesja płatności została już wykorzystana");
            return;
          }

          if (data.status === "completed") {
            setSession(data);
            return;
          }

          // Still pending — wait and retry
          if (attempt < maxAttempts - 1) {
            await new Promise((r) => setTimeout(r, intervalMs));
          }
        } catch (err) {
          if (cancelled || (err instanceof DOMException && err.name === "AbortError")) return;
          if (attempt === maxAttempts - 1) {
            setError("Nie udało się zweryfikować płatności. Spróbuj ponownie za chwilę.");
            return;
          }
          await new Promise((r) => setTimeout(r, intervalMs));
        }
      }

      if (!cancelled) {
        setError("Płatność nie została jeszcze potwierdzona. Spróbuj odświeżyć stronę za chwilę.");
      }
    };

    poll();

    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [sessionId, retryCount]);

  if (!sessionId) {
    return (
      <Card className="max-w-md mx-auto">
        <CardHeader className="text-center">
          <CardTitle>Błąd</CardTitle>
        </CardHeader>
        <CardContent className="text-center text-muted-foreground">
          <p>Brak identyfikatora sesji. Wróć na stronę wyboru planu.</p>
        </CardContent>
        <CardFooter className="justify-center">
          <Link href="/register">
            <Button variant="outline">Wybierz plan</Button>
          </Link>
        </CardFooter>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="max-w-md mx-auto">
        <CardHeader className="text-center">
          <CardTitle>Błąd</CardTitle>
        </CardHeader>
        <CardContent className="text-center">
          <p className="text-destructive">{error}</p>
        </CardContent>
        <CardFooter className="flex flex-col gap-2 items-center">
          <Button variant="outline" onClick={() => { setError(null); setRetryCount((c) => c + 1); }}>
            Spróbuj ponownie
          </Button>
          <Link href="/register" className="text-sm text-primary underline-offset-4 hover:underline">
            Wybierz inny plan
          </Link>
        </CardFooter>
      </Card>
    );
  }

  if (!session) {
    return (
      <Card className="max-w-md mx-auto">
        <CardHeader className="text-center">
          <CardTitle>Weryfikacja płatności...</CardTitle>
          <CardDescription>Sprawdzamy status Twojej płatności</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </CardContent>
      </Card>
    );
  }

  const onSubmit = async (data: CompleteFormValues) => {
    setIsSubmitting(true);
    try {
      await registerUser({
        ...data,
        email: session.email,
        checkout_session_id: sessionId,
      });
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Card className="max-w-md mx-auto">
      <CardHeader className="text-center">
        <CardTitle className="text-2xl">Dokończ rejestrację</CardTitle>
        <CardDescription>
          Plan: <span className="font-medium text-foreground">{session.plan}</span>
        </CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Email</Label>
            <Input value={session.email} disabled />
          </div>
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
            {isSubmitting ? "Rejestracja..." : "Utwórz konto"}
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

export default function CompleteRegistrationPage() {
  return (
    <div className="max-w-md mx-auto">
      <Suspense>
        <CompleteRegistrationForm />
      </Suspense>
    </div>
  );
}
