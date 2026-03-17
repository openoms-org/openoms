"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWebhookConfig, useUpdateWebhookConfig } from "@/hooks/use-webhooks";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Trash2, Plus, ExternalLink } from "lucide-react";
import type { WebhookEndpoint, WebhookConfig } from "@/types/api";
import { useTranslations } from "next-intl";

// WEBHOOK_EVENTS moved inside component to use translations

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
  const t = useTranslations("settings");
  const tw = useTranslations("settings.webhooks");

  const WEBHOOK_EVENTS: { value: string; label: string }[] = [
    { value: "order.created", label: tw("eventOrderCreated") },
    { value: "order.status_changed", label: tw("eventOrderStatusChanged") },
    { value: "order.deleted", label: tw("eventOrderDeleted") },
    { value: "product.created", label: tw("eventProductCreated") },
    { value: "product.updated", label: tw("eventProductUpdated") },
    { value: "product.deleted", label: tw("eventProductDeleted") },
    { value: "shipment.created", label: tw("eventShipmentCreated") },
    { value: "shipment.updated", label: tw("eventShipmentUpdated") },
  ];

  const { data: config, isLoading } = useWebhookConfig();
  const updateConfig = useUpdateWebhookConfig();

  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([]);

  useEffect(() => {
    if (config) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setEndpoints(config.endpoints.map((e) => ({ ...e })));
    }
  }, [config]);

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
        toast.error(t("nazwaEndpointuNieMozeBycPusta"));
        return;
      }
      if (!ep.url.trim()) {
        toast.error(t("urlEndpointuNieMozeBycPusty"));
        return;
      }
      if (ep.events.length === 0) {
        toast.error(tw("endpointMustHaveEvent", { name: ep.name }));
        return;
      }
    }

    const configToSave: WebhookConfig = { endpoints };

    try {
      await updateConfig.mutateAsync(configToSave);
      toast.success(t("konfiguracjaWebhookowZostałaZapisana"));
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t("bładPodczasZapisywania")
      );
    }
  };

  if (isLoading) {
    return <div className="p-6">{t("loading")}</div>;
  }

  return (
    <AdminGuard>
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("webhookiWychodzace")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("konfiguracjaEndpointowDoPowiadamianiaZewnetrznychS")}
          </p>
          <p className="text-sm text-muted-foreground">
            {t("webhookiWysyłajaPowiadomieniaHttpPostDoZewnetrznyc")}
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
                  placeholder={t("mojWebhook")}
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
    </AdminGuard>
  );
}
