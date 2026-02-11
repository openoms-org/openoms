"use client";

import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { INTEGRATION_PROVIDERS, INTEGRATION_PROVIDER_LABELS } from "@/lib/constants";

const CREDENTIAL_PLACEHOLDERS: Record<string, string> = {
  woocommerce: `{
  "store_url": "https://twoj-sklep.pl",
  "consumer_key": "ck_...",
  "consumer_secret": "cs_..."
}`,
  allegro: `{
  "client_id": "...",
  "client_secret": "..."
}`,
};

const CREDENTIAL_HINTS: Record<string, string> = {
  woocommerce:
    "Klucze API WooCommerce znajdziesz w: WooCommerce > Ustawienia > Zaawansowane > REST API",
};

const integrationSchema = z.object({
  provider: z.string().min(1, "Dostawca jest wymagany"),
  credentials: z.string().min(1, "Dane uwierzytelniające są wymagane").refine(
    (val) => {
      try {
        JSON.parse(val);
        return true;
      } catch {
        return false;
      }
    },
    { message: "Nieprawidłowy format JSON" }
  ),
  settings: z.string().optional().refine(
    (val) => {
      if (!val || val.trim() === "") return true;
      try {
        JSON.parse(val);
        return true;
      } catch {
        return false;
      }
    },
    { message: "Nieprawidłowy format JSON" }
  ),
});

type IntegrationFormValues = z.infer<typeof integrationSchema>;

interface IntegrationFormProps {
  onSubmit: (data: {
    provider: string;
    credentials: Record<string, unknown>;
    settings?: Record<string, unknown>;
  }) => void;
  isLoading?: boolean;
}

export function IntegrationForm({ onSubmit, isLoading = false }: IntegrationFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<IntegrationFormValues>({
    resolver: zodResolver(integrationSchema),
    defaultValues: {
      provider: "",
      credentials: "",
      settings: "",
    },
  });

  const selectedProvider = watch("provider");

  const handleFormSubmit = (data: IntegrationFormValues) => {
    const parsed: {
      provider: string;
      credentials: Record<string, unknown>;
      settings?: Record<string, unknown>;
    } = {
      provider: data.provider,
      credentials: JSON.parse(data.credentials),
    };
    if (data.settings && data.settings.trim() !== "") {
      parsed.settings = JSON.parse(data.settings);
    }
    onSubmit(parsed);
  };

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor="provider">Dostawca</Label>
        <Select
          value={selectedProvider}
          onValueChange={(value) => setValue("provider", value, { shouldValidate: true })}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz dostawcę" />
          </SelectTrigger>
          <SelectContent>
            {INTEGRATION_PROVIDERS.map((provider) => (
              <SelectItem key={provider} value={provider}>
                {INTEGRATION_PROVIDER_LABELS[provider] ?? provider.charAt(0).toUpperCase() + provider.slice(1)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.provider && (
          <p className="text-sm text-destructive">{errors.provider.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="credentials">Dane uwierzytelniające (JSON)</Label>
        <Textarea
          id="credentials"
          placeholder={
            CREDENTIAL_PLACEHOLDERS[selectedProvider] ??
            '{"api_key": "...", "secret": "..."}'
          }
          className="min-h-32 font-mono text-sm"
          {...register("credentials")}
        />
        {CREDENTIAL_HINTS[selectedProvider] && (
          <p className="text-sm text-muted-foreground">
            {CREDENTIAL_HINTS[selectedProvider]}
          </p>
        )}
        {errors.credentials && (
          <p className="text-sm text-destructive">{errors.credentials.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="settings">Ustawienia (JSON, opcjonalne)</Label>
        <Textarea
          id="settings"
          placeholder='{"webhook_url": "...", "sync_interval": 3600}'
          className="min-h-24 font-mono text-sm"
          {...register("settings")}
        />
        {errors.settings && (
          <p className="text-sm text-destructive">{errors.settings.message}</p>
        )}
      </div>

      <div className="flex justify-end gap-3">
        <Button type="submit" disabled={isLoading}>
          {isLoading ? "Tworzenie..." : "Utwórz integrację"}
        </Button>
      </div>
    </form>
  );
}
