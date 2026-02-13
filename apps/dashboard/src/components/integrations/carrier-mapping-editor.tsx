"use client";

import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SHIPMENT_PROVIDERS, SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";

interface CarrierMappingEditorProps {
  value: Record<string, string>;
  onChange: (mapping: Record<string, string>) => void;
}

export function CarrierMappingEditor({ value, onChange }: CarrierMappingEditorProps) {
  const entries = Object.entries(value);

  const handleKeyChange = (oldKey: string, newKey: string) => {
    const updated: Record<string, string> = {};
    for (const [k, v] of Object.entries(value)) {
      updated[k === oldKey ? newKey : k] = v;
    }
    onChange(updated);
  };

  const handleValueChange = (key: string, newValue: string) => {
    onChange({ ...value, [key]: newValue });
  };

  const handleAdd = () => {
    onChange({ ...value, "": "inpost" });
  };

  const handleRemove = (key: string) => {
    const { [key]: _, ...rest } = value;
    onChange(rest);
  };

  return (
    <div className="space-y-2">
      {entries.length === 0 && (
        <p className="text-sm text-muted-foreground py-4 text-center">
          Brak mapowań. Dodaj mapowanie, aby automatycznie przypisywać kuriera do zamówień.
        </p>
      )}
      {entries.map(([key, val], index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            className="flex-1"
            placeholder="Nazwa dostawy z marketplace"
            value={key}
            onChange={(e) => handleKeyChange(key, e.target.value)}
          />
          <Select value={val} onValueChange={(v) => handleValueChange(key, v)}>
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SHIPMENT_PROVIDERS.filter(p => p !== "manual").map((p) => (
                <SelectItem key={p} value={p}>
                  {SHIPMENT_PROVIDER_LABELS[p] ?? p}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => handleRemove(key)}
          >
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
        <Plus className="h-4 w-4" />
        Dodaj mapowanie
      </Button>
    </div>
  );
}
