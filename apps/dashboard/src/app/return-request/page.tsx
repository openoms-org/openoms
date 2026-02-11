"use client";

import { Suspense, useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { RotateCcw, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { PublicReturnRequest, PublicReturnResponse } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function PublicReturnForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [step, setStep] = useState<"form" | "success">("form");
  const [orderId, setOrderId] = useState("");
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState("");
  const [notes, setNotes] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [returnToken, setReturnToken] = useState<string | null>(null);

  useEffect(() => {
    const orderIdParam = searchParams.get("order_id");
    if (orderIdParam) {
      setOrderId(orderIdParam);
    }
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const body: PublicReturnRequest = {
        order_id: orderId.trim(),
        email: email.trim(),
        reason: reason.trim(),
        notes: notes.trim() || undefined,
      };

      const res = await fetch(`${API_URL}/v1/public/returns`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: "Wystąpił błąd" }));
        setError(data.error || "Wystąpił błąd podczas składania zwrotu");
        return;
      }

      const data: PublicReturnResponse = await res.json();
      setReturnToken(data.return_token);
      setStep("success");
    } catch {
      setError("Nie udało się połączyć z serwerem. Spróbuj ponownie.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      {step === "form" && (
        <Card>
          <CardHeader>
            <CardTitle>Zgłoś zwrot</CardTitle>
            <CardDescription>
              Wypełnij formularz, aby zgłosić zwrot zamówienia.
              Podaj numer zamówienia i adres email, który został
              użyty przy składaniu zamówienia.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="order-id">Numer zamówienia (ID)</Label>
                <Input
                  id="order-id"
                  value={orderId}
                  onChange={(e) => setOrderId(e.target.value)}
                  placeholder="np. a1b2c3d4-e5f6-..."
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="email">Adres email</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="twoj@email.pl"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="reason">Powód zwrotu</Label>
                <Textarea
                  id="reason"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Opisz powód zwrotu..."
                  rows={3}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="notes">Dodatkowe uwagi (opcjonalnie)</Label>
                <Textarea
                  id="notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Dodatkowe informacje..."
                  rows={2}
                />
              </div>

              {error && (
                <div className="rounded-md border border-destructive bg-destructive/10 p-3">
                  <p className="text-sm text-destructive">{error}</p>
                </div>
              )}

              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting || !orderId || !email || !reason}
              >
                {isSubmitting ? "Wysyłanie..." : "Zgłoś zwrot"}
                {!isSubmitting && <ArrowRight className="ml-2 h-4 w-4" />}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {step === "success" && returnToken && (
        <Card>
          <CardHeader>
            <CardTitle>Zwrot został zgłoszony</CardTitle>
            <CardDescription>
              Twoje zgłoszenie zwrotu zostało przyjęte. Możesz śledzić
              jego status pod poniższym linkiem.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-md bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-4">
              <p className="text-sm text-green-800 dark:text-green-200">
                Twoje zgłoszenie zostało zarejestrowane. Otrzymasz informacje o zmianach statusu.
              </p>
            </div>

            <Button
              className="w-full"
              onClick={() => router.push(`/return-request/${returnToken}`)}
            >
              Sprawdź status zwrotu
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </CardContent>
        </Card>
      )}
    </>
  );
}

export default function PublicReturnPage() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-2">
            <RotateCcw className="h-8 w-8 text-primary" />
            <span className="text-2xl font-bold">OpenOMS</span>
          </div>
          <p className="text-muted-foreground">Formularz zwrotu towaru</p>
        </div>

        <Suspense
          fallback={
            <Card>
              <CardContent className="py-12">
                <div className="flex items-center justify-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
                </div>
                <p className="text-center mt-4 text-muted-foreground">
                  Ładowanie...
                </p>
              </CardContent>
            </Card>
          }
        >
          <PublicReturnForm />
        </Suspense>
      </div>
    </div>
  );
}
