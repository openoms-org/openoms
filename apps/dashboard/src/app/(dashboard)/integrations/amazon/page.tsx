"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
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

const MARKETPLACES = [
  { id: "A1C3SOZRARQ6R3", label: "amazon.pl (Polska)" },
  { id: "A1PA6795UKMFR9", label: "amazon.de (Niemcy)" },
  { id: "A1F83G8C2ARO7P", label: "amazon.co.uk (Wielka Brytania)" },
  { id: "A13V1IB3VIYZZH", label: "amazon.fr (Francja)" },
  { id: "APJ6JRA9NG5V4", label: "amazon.it (Włochy)" },
  { id: "A1RKKUPIHCS9HS", label: "amazon.es (Hiszpania)" },
  { id: "A21TJRUUN4KGV", label: "amazon.in (Indie)" },
  { id: "ATVPDKIKX0DER", label: "amazon.com (USA)" },
];

export default function AmazonSetupPage() {
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [refreshToken, setRefreshToken] = useState("");
  const [marketplaceId, setMarketplaceId] = useState(MARKETPLACES[0].id);
  const [sandbox, setSandbox] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!clientId || !clientSecret || !refreshToken || !marketplaceId) {
      toast.error("Wszystkie pola są wymagane");
      return;
    }

    setIsSubmitting(true);
    try {
      await apiClient("/v1/integrations/amazon/setup", {
        method: "POST",
        body: JSON.stringify({
          client_id: clientId,
          client_secret: clientSecret,
          refresh_token: refreshToken,
          marketplace_id: marketplaceId,
          sandbox,
        }),
      });
      toast.success("Integracja Amazon została skonfigurowana");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się skonfigurować integracji";
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Integracja Amazon</h1>
            <p className="text-muted-foreground">
              Połącz swoje konto Amazon Seller, aby synchronizować zamówienia
            </p>
          </div>
        </div>

        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>Dane uwierzytelniające Amazon SP-API</CardTitle>
            <CardDescription>
              Podaj dane z aplikacji Amazon Seller Partner API. Dane znajdziesz
              w panelu Amazon Seller Central w sekcji Developer.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="client_id">Client ID</Label>
                <Input
                  id="client_id"
                  value={clientId}
                  onChange={(e) => setClientId(e.target.value)}
                  placeholder="amzn1.application-oa2-client..."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="client_secret">Client Secret</Label>
                <Input
                  id="client_secret"
                  type="password"
                  value={clientSecret}
                  onChange={(e) => setClientSecret(e.target.value)}
                  placeholder="Klucz tajny aplikacji"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="refresh_token">Refresh Token</Label>
                <Input
                  id="refresh_token"
                  type="password"
                  value={refreshToken}
                  onChange={(e) => setRefreshToken(e.target.value)}
                  placeholder="Atzr|..."
                />
                <p className="text-xs text-muted-foreground">
                  Token uzyskany po autoryzacji aplikacji w Amazon Seller Central
                </p>
              </div>
              <div className="space-y-2">
                <Label>Marketplace</Label>
                <Select value={marketplaceId} onValueChange={setMarketplaceId}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Wybierz marketplace" />
                  </SelectTrigger>
                  <SelectContent>
                    {MARKETPLACES.map((mp) => (
                      <SelectItem key={mp.id} value={mp.id}>
                        {mp.label}
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
                  Tryb sandbox (testowy)
                </Label>
              </div>
              <div className="flex gap-2 pt-4">
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Zapisz i zweryfikuj
                </Button>
                <Button type="button" variant="outline" asChild>
                  <Link href="/integrations">Anuluj</Link>
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
