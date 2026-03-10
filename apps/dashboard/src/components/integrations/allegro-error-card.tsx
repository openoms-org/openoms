"use client";

import Link from "next/link";
import { AlertTriangle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";

interface AllegroErrorCardProps {
  error: Error | null;
  onRetry?: () => void;
}

export function AllegroErrorCard({ error, onRetry }: AllegroErrorCardProps) {
  const t = useTranslations("integrations");
  if (!error) return null;

  const msg = error.message || "";
  const isTokenError = msg.includes("502") || msg.includes("422") || msg.includes("Token") || msg.includes("401");

  return (
    <Card className="border-destructive">
      <CardContent className="flex items-start gap-3 pt-6">
        <AlertTriangle className="h-5 w-5 shrink-0 text-destructive mt-0.5" />
        <div className="space-y-2">
          <p className="text-sm text-destructive">
            {isTokenError
              ? t("tokenAllegroWygasłLubJestNieprawidłowyPołaczPonown")
              : t("nieUdałoSiePobracDanychZAllegroSprawdzPołaczenieZK")}
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" asChild>
              <Link href="/marketplaces/allegro">Ustawienia Allegro</Link>
            </Button>
            {onRetry && (
              <Button variant="ghost" size="sm" onClick={onRetry}>
                {t("retry")}
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
