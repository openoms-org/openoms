"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useWebhookConfig, useUpdateWebhookConfig } from "@/hooks/use-webhooks";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Trash2, Plus, ExternalLink } from "lucide-react";
import type { WebhookEndpoint, WebhookConfig } from "@/types/api";

const WEBHOOK_EVENTS: { value: string; label: string }[] = [
  { value: "order.created", label: "Zamówienie utworzone" },
  { value: "order.status_changed", label: "Status zamówienia zmieniony" },
  { value: "order.deleted", label: "Zamówienie usunięte" },
  { value: "product.created", label: "Produkt utworzony" },
  { value: "product.updated", label: "Produkt zaktualizowany" },
  { value: "product.deleted", label: "Produkt usunięty" },
  { value: "shipment.created", label: "Przesyłka utworzona" },
  { value: "shipment.updated", label: "Przesyłka zaktualizowana" },
];

function createEmptyEndpoint(): WebhookEndpoint {
  return {
    id: crypto.randomUUID(),
    name: "",
    url: "",
    secret: "",
    events: [],
    active: true,
  };
}

export default function WebhooksPage() {
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: config, isLoading } = useWebhookConfig();
  const updateConfig = useUpdateWebhookConfig();

  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([]);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.replace("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (config) {
      setEndpoints(config.endpoints.map((e) => ({ ...e })));
    }
  }, [config]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  const handleAddEndpoint = () => {
    setEndpoints([...endpoints, createEmptyEndpoint()]);
  };

  const handleRemoveEndpoint = (index: number) => {
    setEndpoints(endpoints.filter((_, i) => i !== index));
  };

  const handleEndpointChange = (
    index: number,
    field: keyof WebhookEndpoint,
    value: string | boolean | string[]
  ) => {
    const updated = [...endpoints];
    updated[index] = { ...updated[index], [field]: value };
    setEndpoints(updated);
  };

  const handleEventToggle = (index: number, event: string) => {
    const endpoint = endpoints[index];
    const events = endpoint.events.includes(event)
      ? endpoint.events.filter((e) => e !== event)
      : [...endpoint.events, event];
    handleEndpointChange(index, "events", events);
  };

  const handleSave = async () => {
    for (const ep of endpoints) {
      if (!ep.name.trim()) {
        toast.error("Nazwa endpointu nie może być pusta");
        return;
      }
      if (!ep.url.trim()) {
        toast.error("URL endpointu nie może być pusty");
        return;
      }
      if (ep.events.length === 0) {
        toast.error(`Endpoint "${ep.name}" musi mieć co najmniej jedno zdarzenie`);
        return;
      }
    }

    const configToSave: WebhookConfig = { endpoints };

    try {
      await updateConfig.mutateAsync(configToSave);
      toast.success("Konfiguracja webhooków została zapisana");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Błąd podczas zapisywania"
      );
    }
  };

  if (isLoading) {
    return <div className="p-6">Ładowanie...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Webhooki wychodzące</h1>
          <p className="text-muted-foreground mt-1">
            Konfiguracja endpointów do powiadamiania zewnętrznych systemów
          </p>
        </div>
        <Link href="/settings/webhooks/deliveries">
          <Button variant="outline" size="sm">
            <ExternalLink className="mr-2 h-4 w-4" />
            Zobacz log dostaw
          </Button>
        </Link>
      </div>

      {endpoints.map((endpoint, index) => (
        <Card key={endpoint.id}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <CardTitle className="text-base">
              Endpoint {index + 1}
            </CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleRemoveEndpoint(index)}
            >
              <Trash2 className="h-4 w-4 text-muted-foreground" />
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Nazwa</Label>
                <Input
                  placeholder="Mój webhook"
                  value={endpoint.name}
                  onChange={(e) =>
                    handleEndpointChange(index, "name", e.target.value)
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>URL</Label>
                <Input
                  placeholder="https://example.com/webhook"
                  value={endpoint.url}
                  onChange={(e) =>
                    handleEndpointChange(index, "url", e.target.value)
                  }
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label>Secret</Label>
              <Input
                type="password"
                placeholder="Klucz tajny do podpisywania"
                value={endpoint.secret}
                onChange={(e) =>
                  handleEndpointChange(index, "secret", e.target.value)
                }
                className="font-mono"
              />
            </div>

            <div className="flex items-center gap-3">
              <Switch
                checked={endpoint.active}
                onCheckedChange={(checked) =>
                  handleEndpointChange(index, "active", checked)
                }
              />
              <Label>Aktywny</Label>
            </div>

            <div className="space-y-2">
              <Label>Zdarzenia</Label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {WEBHOOK_EVENTS.map((event) => {
                  const isSelected = endpoint.events.includes(event.value);
                  return (
                    <label
                      key={event.value}
                      className="flex items-center gap-2 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => handleEventToggle(index, event.value)}
                        className="cursor-pointer"
                      />
                      <span className="text-sm">{event.label}</span>
                      <span className="text-xs text-muted-foreground">
                        ({event.value})
                      </span>
                    </label>
                  );
                })}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}

      <div className="flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={handleAddEndpoint}>
          <Plus className="mr-2 h-4 w-4" />
          Dodaj endpoint
        </Button>
      </div>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateConfig.isPending}>
          {updateConfig.isPending ? "Zapisywanie..." : "Zapisz"}
        </Button>
      </div>
    </div>
  );
}
