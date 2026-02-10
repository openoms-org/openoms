"use client";

import { useState, useCallback, useRef } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { apiClient } from "@/lib/api-client";

interface AllegroConnectProps {
  onConnected: () => void;
}

export function AllegroConnect({ onConnected }: AllegroConnectProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const cleanup = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const handleConnect = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const { auth_url } = await apiClient<{ auth_url: string }>(
        "/v1/integrations/allegro/auth-url"
      );

      const popup = window.open(
        auth_url,
        "allegro-oauth",
        "width=600,height=700,scrollbars=yes"
      );

      if (!popup) {
        setError("Przeglądarka zablokowała okno popup. Zezwól na wyskakujące okna i spróbuj ponownie.");
        setIsLoading(false);
        return;
      }

      pollRef.current = setInterval(() => {
        if (popup.closed) {
          cleanup();
          setIsLoading(false);
          onConnected();
        }
      }, 500);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Nie udało się pobrać adresu autoryzacji"
      );
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <Button onClick={handleConnect} disabled={isLoading}>
        {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        Połącz z Allegro
      </Button>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
