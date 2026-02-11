"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useProductCategories, useUpdateProductCategories } from "@/hooks/use-product-categories";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Trash2, Plus } from "lucide-react";
import type { CategoryDef } from "@/types/api";

export default function ProductCategoriesPage() {
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: config, isLoading } = useProductCategories();
  const updateCategories = useUpdateProductCategories();

  const [categories, setCategories] = useState<CategoryDef[]>([]);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.replace("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (config) {
      setCategories([...config.categories]);
    }
  }, [config]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  const handleAddCategory = () => {
    const newPosition = categories.length + 1;
    setCategories([...categories, { key: "", label: "", color: "#6b7280", position: newPosition }]);
  };

  const handleRemoveCategory = (index: number) => {
    setCategories(categories.filter((_, i) => i !== index));
  };

  const handleCategoryChange = (index: number, field: keyof CategoryDef, value: string | number) => {
    const newCategories = [...categories];
    newCategories[index] = { ...newCategories[index], [field]: value };
    setCategories(newCategories);
  };

  const handleSave = async () => {
    for (const c of categories) {
      if (!c.key || !c.label) {
        toast.error("Wszystkie kategorie muszą mieć klucz i etykietę");
        return;
      }
    }

    const keys = categories.map((c) => c.key);
    if (new Set(keys).size !== keys.length) {
      toast.error("Klucze kategorii muszą być unikalne");
      return;
    }

    try {
      await updateCategories.mutateAsync({
        categories: categories.map((c, i) => ({ ...c, position: i + 1 })),
      });
      toast.success("Kategorie produktów zostały zapisane");
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
      <div>
        <h1 className="text-2xl font-bold">Kategorie produktów</h1>
        <p className="text-muted-foreground mt-1">
          Zdefiniuj kategorie dla produktów
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Kategorie</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {categories.map((category, index) => (
            <div key={index} className="flex items-center gap-3">
              <span className="text-sm text-muted-foreground w-6">{index + 1}.</span>
              <Input
                placeholder="Klucz (np. electronics)"
                value={category.key}
                onChange={(e) =>
                  handleCategoryChange(
                    index,
                    "key",
                    e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, "")
                  )
                }
                className="w-40"
              />
              <Input
                placeholder="Etykieta (np. Elektronika)"
                value={category.label}
                onChange={(e) => handleCategoryChange(index, "label", e.target.value)}
                className="w-48"
              />
              <input
                type="color"
                value={category.color}
                onChange={(e) => handleCategoryChange(index, "color", e.target.value)}
                className="h-9 w-12 cursor-pointer rounded border border-input p-1"
                title="Kolor kategorii"
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleRemoveCategory(index)}
              >
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>
          ))}
          <Button variant="outline" size="sm" onClick={handleAddCategory}>
            <Plus className="mr-2 h-4 w-4" />
            Dodaj kategorię
          </Button>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateCategories.isPending}>
          {updateCategories.isPending ? "Zapisywanie..." : "Zapisz"}
        </Button>
      </div>
    </div>
  );
}
