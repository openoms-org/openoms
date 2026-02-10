"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { useOrderStatuses, COLOR_PRESETS } from "@/hooks/use-order-statuses";
import { useUpdateOrderStatuses } from "@/hooks/use-settings";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Trash2, Plus } from "lucide-react";
import type { StatusDef, OrderStatusConfig } from "@/types/api";

const COLOR_OPTIONS = Object.entries(COLOR_PRESETS).map(([key, classes]) => ({
  key,
  classes,
}));

export default function OrderStatusesPage() {
  const { data: config, isLoading } = useOrderStatuses();
  const updateStatuses = useUpdateOrderStatuses();

  const [statuses, setStatuses] = useState<StatusDef[]>([]);
  const [transitions, setTransitions] = useState<Record<string, string[]>>({});

  useEffect(() => {
    if (config) {
      setStatuses([...config.statuses]);
      setTransitions({ ...config.transitions });
    }
  }, [config]);

  const handleAddStatus = () => {
    const newPosition = statuses.length + 1;
    setStatuses([...statuses, { key: "", label: "", color: "gray", position: newPosition }]);
  };

  const handleRemoveStatus = (index: number) => {
    const removed = statuses[index];
    const newStatuses = statuses.filter((_, i) => i !== index);
    setStatuses(newStatuses);
    // Remove from transitions
    const newTransitions = { ...transitions };
    delete newTransitions[removed.key];
    for (const [from, targets] of Object.entries(newTransitions)) {
      newTransitions[from] = targets.filter((t) => t !== removed.key);
    }
    setTransitions(newTransitions);
  };

  const handleStatusChange = (index: number, field: keyof StatusDef, value: string | number) => {
    const newStatuses = [...statuses];
    newStatuses[index] = { ...newStatuses[index], [field]: value };
    setStatuses(newStatuses);
  };

  const handleTransitionToggle = (from: string, to: string) => {
    const newTransitions = { ...transitions };
    const targets = newTransitions[from] || [];
    if (targets.includes(to)) {
      newTransitions[from] = targets.filter((t) => t !== to);
    } else {
      newTransitions[from] = [...targets, to];
    }
    setTransitions(newTransitions);
  };

  const handleSave = async () => {
    // Validate
    for (const s of statuses) {
      if (!s.key || !s.label) {
        toast.error("Wszystkie statusy musza miec klucz i etykiete");
        return;
      }
    }

    const keys = statuses.map((s) => s.key);
    if (new Set(keys).size !== keys.length) {
      toast.error("Klucze statusow musza byc unikalne");
      return;
    }

    const configToSave: OrderStatusConfig = {
      statuses: statuses.map((s, i) => ({ ...s, position: i + 1 })),
      transitions,
    };

    try {
      await updateStatuses.mutateAsync(configToSave);
      toast.success("Statusy zamowien zostaly zapisane");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas zapisywania"
      );
    }
  };

  if (isLoading) {
    return <div className="p-6">Ladowanie...</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Statusy zamowien</h1>
        <p className="text-muted-foreground mt-1">
          Zdefiniuj statusy i przejscia dla zamowien
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Statusy</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {statuses.map((status, index) => (
            <div key={index} className="flex items-center gap-3">
              <span className="text-sm text-muted-foreground w-6">{index + 1}.</span>
              <Input
                placeholder="Klucz (np. new)"
                value={status.key}
                onChange={(e) => handleStatusChange(index, "key", e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ""))}
                className="w-40"
              />
              <Input
                placeholder="Etykieta (np. Nowe)"
                value={status.label}
                onChange={(e) => handleStatusChange(index, "label", e.target.value)}
                className="w-48"
              />
              <Select
                value={status.color}
                onValueChange={(v) => handleStatusChange(index, "color", v)}
              >
                <SelectTrigger className="w-36">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {COLOR_OPTIONS.map((opt) => (
                    <SelectItem key={opt.key} value={opt.key}>
                      <div className="flex items-center gap-2">
                        <span className={`inline-block h-3 w-3 rounded-full ${opt.classes.split(" ")[0]}`} />
                        {opt.key}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleRemoveStatus(index)}
              >
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          ))}
          <Button variant="outline" size="sm" onClick={handleAddStatus}>
            <Plus className="mr-2 h-4 w-4" />
            Dodaj status
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Przejscia</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {statuses.map((from) => (
              <div key={from.key} className="space-y-1">
                <p className="text-sm font-medium">{from.label || from.key}</p>
                <div className="flex flex-wrap gap-2">
                  {statuses
                    .filter((s) => s.key !== from.key)
                    .map((to) => {
                      const isAllowed = (transitions[from.key] || []).includes(to.key);
                      return (
                        <button
                          key={to.key}
                          type="button"
                          onClick={() => handleTransitionToggle(from.key, to.key)}
                          className={`rounded-full px-3 py-1 text-xs font-medium border transition-colors ${
                            isAllowed
                              ? "bg-primary text-primary-foreground border-primary"
                              : "bg-background text-muted-foreground border-border hover:border-primary/50"
                          }`}
                        >
                          {to.label || to.key}
                        </button>
                      );
                    })}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateStatuses.isPending}>
          {updateStatuses.isPending ? "Zapisywanie..." : "Zapisz zmiany"}
        </Button>
      </div>
    </div>
  );
}
