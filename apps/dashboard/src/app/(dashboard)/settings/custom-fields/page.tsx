"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCustomFields } from "@/hooks/use-custom-fields";
import { useUpdateCustomFields } from "@/hooks/use-settings";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Trash2, Plus } from "lucide-react";
import type { CustomFieldDef, CustomFieldsConfig } from "@/types/api";

const TYPE_OPTIONS: { value: CustomFieldDef["type"]; label: string }[] = [
  { value: "text", label: "Tekst" },
  { value: "number", label: "Liczba" },
  { value: "select", label: "Lista wyboru" },
  { value: "date", label: "Data" },
  { value: "checkbox", label: "Tak/Nie" },
];

export default function CustomFieldsPage() {
  const { data: config, isLoading } = useCustomFields();
  const updateCustomFields = useUpdateCustomFields();

  const [fields, setFields] = useState<CustomFieldDef[]>([]);

  useEffect(() => {
    if (config) {
      setFields([...config.fields]);
    }
  }, [config]);

  const handleAddField = () => {
    const newPosition = fields.length + 1;
    setFields([
      ...fields,
      {
        key: "",
        label: "",
        type: "text",
        required: false,
        position: newPosition,
      },
    ]);
  };

  const handleRemoveField = (index: number) => {
    setFields(fields.filter((_, i) => i !== index));
  };

  const handleFieldChange = (
    index: number,
    field: keyof CustomFieldDef,
    value: string | boolean | string[]
  ) => {
    const newFields = [...fields];
    newFields[index] = { ...newFields[index], [field]: value };
    setFields(newFields);
  };

  const handleSave = async () => {
    for (const f of fields) {
      if (!f.key || !f.label) {
        toast.error("Wszystkie pola muszą mieć klucz i etykietę");
        return;
      }
    }

    const keys = fields.map((f) => f.key);
    if (new Set(keys).size !== keys.length) {
      toast.error("Klucze pól muszą być unikalne");
      return;
    }

    const configToSave: CustomFieldsConfig = {
      fields: fields.map((f, i) => ({ ...f, position: i + 1 })),
    };

    try {
      await updateCustomFields.mutateAsync(configToSave);
      toast.success("Pola dodatkowe zostały zapisane");
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
    <AdminGuard>
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Pola dodatkowe</h1>
        <p className="text-muted-foreground mt-1">
          Zdefiniuj dodatkowe pola dla zamówień.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Pola</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {fields.map((field, index) => (
            <div key={index} className="space-y-3 rounded-md border p-4">
              <div className="flex items-start justify-between">
                <span className="text-sm text-muted-foreground">
                  {index + 1}.
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleRemoveField(index)}
                >
                  <Trash2 className="h-4 w-4 text-muted-foreground" />
                </Button>
              </div>
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
                <div className="space-y-1">
                  <Label>Klucz</Label>
                  <Input
                    placeholder="np. numer_faktury"
                    value={field.key}
                    onChange={(e) =>
                      handleFieldChange(
                        index,
                        "key",
                        e.target.value
                          .toLowerCase()
                          .replace(/[^a-z0-9_]/g, "")
                      )
                    }
                  />
                </div>
                <div className="space-y-1">
                  <Label>Etykieta</Label>
                  <Input
                    placeholder="np. Numer faktury"
                    value={field.label}
                    onChange={(e) =>
                      handleFieldChange(index, "label", e.target.value)
                    }
                  />
                </div>
                <div className="space-y-1">
                  <Label>Typ</Label>
                  <Select
                    value={field.type}
                    onValueChange={(v) => handleFieldChange(index, "type", v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TYPE_OPTIONS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-end gap-2 pb-1">
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={field.required}
                      onChange={(e) =>
                        handleFieldChange(index, "required", e.target.checked)
                      }
                      className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
                    />
                    Wymagane
                  </label>
                </div>
              </div>
              {field.type === "select" && (
                <div className="space-y-1">
                  <Label>Opcje (po przecinku)</Label>
                  <Input
                    placeholder="opcja1, opcja2, opcja3"
                    value={(field.options || []).join(", ")}
                    onChange={(e) =>
                      handleFieldChange(
                        index,
                        "options",
                        e.target.value
                          .split(",")
                          .map((s) => s.trim())
                          .filter(Boolean)
                      )
                    }
                  />
                </div>
              )}
            </div>
          ))}
          <Button variant="outline" size="sm" onClick={handleAddField}>
            <Plus className="mr-2 h-4 w-4" />
            Dodaj pole
          </Button>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateCustomFields.isPending}>
          {updateCustomFields.isPending ? "Zapisywanie..." : "Zapisz"}
        </Button>
      </div>
    </div>
    </AdminGuard>
  );
}
