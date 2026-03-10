"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useCategoryTree,
  useCreateCategory,
  useDeleteCategory,
} from "@/hooks/use-categories";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Trash2,
  Plus,
  ChevronRight,
  ChevronDown,
  Pencil,
  X,
  Check,
  FolderPlus,
} from "lucide-react";
import type { ProductCategory, UpdateCategoryRequest } from "@/types/api";
import { useTranslations } from "next-intl";

interface EditState {
  id: string;
  name: string;
  color: string;
}

function CategoryRow({
  category,
  expanded,
  onToggle,
  onEdit,
  onDelete,
  onAddChild,
  editState,
  onSaveEdit,
  onCancelEdit,
  onEditChange,
}: {
  category: ProductCategory;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  onEdit: (cat: ProductCategory) => void;
  onDelete: (id: string) => void;
  onAddChild: (parentId: string) => void;
  editState: EditState | null;
  onSaveEdit: () => void;
  onCancelEdit: () => void;
  onEditChange: (field: "name" | "color", value: string) => void;
}) {
  const t = useTranslations("settings");
  const hasChildren = category.children && category.children.length > 0;
  const isExpanded = expanded.has(category.id);
  const isEditing = editState?.id === category.id;

  return (
    <>
      <div
        className="flex items-center gap-2 py-1.5 px-2 rounded-md hover:bg-muted/50 group"
        style={{ paddingLeft: `${category.depth * 24 + 8}px` }}
      >
        <button
          onClick={() => hasChildren && onToggle(category.id)}
          className="h-5 w-5 flex items-center justify-center shrink-0"
        >
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )
          ) : (
            <span className="h-4 w-4" />
          )}
        </button>

        {isEditing ? (
          <>
            <input
              type="color"
              value={editState.color}
              onChange={(e) => onEditChange("color", e.target.value)}
              className="h-6 w-8 cursor-pointer rounded border border-input p-0.5"
            />
            <Input
              value={editState.name}
              onChange={(e) => onEditChange("name", e.target.value)}
              className="h-7 w-48 text-sm"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") onSaveEdit();
                if (e.key === "Escape") onCancelEdit();
              }}
            />
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onSaveEdit}>
              <Check className="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onCancelEdit}>
              <X className="h-3.5 w-3.5" />
            </Button>
          </>
        ) : (
          <>
            <span
              className="inline-block h-3 w-3 rounded-full shrink-0"
              style={{ backgroundColor: category.color }}
            />
            <span className="text-sm font-medium flex-1">{category.name}</span>
            <span className="text-xs text-muted-foreground">{category.slug}</span>
            <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
              {category.depth < 5 && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0"
                  onClick={() => onAddChild(category.id)}
                  title={t("dodajPodkategorie")}
                >
                  <FolderPlus className="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={() => onEdit(category)}
                title="Edytuj"
              >
                <Pencil className="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0"
                onClick={() => onDelete(category.id)}
                title={t("delete")}
              >
                <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
            </div>
          </>
        )}
      </div>

      {hasChildren && isExpanded &&
        category.children!.map((child) => (
          <CategoryRow
            key={child.id}
            category={child}
            expanded={expanded}
            onToggle={onToggle}
            onEdit={onEdit}
            onDelete={onDelete}
            onAddChild={onAddChild}
            editState={editState}
            onSaveEdit={onSaveEdit}
            onCancelEdit={onCancelEdit}
            onEditChange={onEditChange}
          />
        ))}
    </>
  );
}

export default function ProductCategoriesPage() {
  const t = useTranslations("settings");
  const tc = useTranslations("common");
  const { data: tree, isLoading } = useCategoryTree();
  const createCategory = useCreateCategory();
  const deleteCategory = useDeleteCategory();
  const queryClient = useQueryClient();

  const updateCategory = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateCategoryRequest }) =>
      apiClient<ProductCategory>(`/v1/categories/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
  });

  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [editState, setEditState] = useState<EditState | null>(null);
  const [newCategoryName, setNewCategoryName] = useState("");
  const [newCategoryColor, setNewCategoryColor] = useState("#6b7280");
  const [addingParentId, setAddingParentId] = useState<string | undefined>(undefined);
  const [showAddForm, setShowAddForm] = useState(false);

  const handleToggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleEdit = (cat: ProductCategory) => {
    setEditState({ id: cat.id, name: cat.name, color: cat.color });
  };

  const handleSaveEdit = async () => {
    if (!editState) return;
    if (!editState.name.trim()) {
      toast.error("Nazwa kategorii jest wymagana");
      return;
    }
    try {
      await updateCategory.mutateAsync({
        id: editState.id,
        data: { name: editState.name, color: editState.color },
      });
      toast.success("Kategoria zaktualizowana");
      setEditState(null);
    } catch {
      toast.error(t("bładpodczasaktualizacjikategorii"));
    }
  };

  const handleCancelEdit = () => {
    setEditState(null);
  };

  const handleEditChange = (field: "name" | "color", value: string) => {
    if (editState) {
      setEditState({ ...editState, [field]: value });
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteCategory.mutateAsync(id);
      toast.success(t("kategoriaUsunieta"));
    } catch {
      toast.error(t("bładpodczasusuwaniakategorii"));
    }
  };

  const handleAddChild = (parentId: string) => {
    setAddingParentId(parentId);
    setShowAddForm(true);
    setExpanded((prev) => new Set([...prev, parentId]));
  };

  const handleAddRoot = () => {
    setAddingParentId(undefined);
    setShowAddForm(true);
  };

  const handleCreateCategory = async () => {
    if (!newCategoryName.trim()) {
      toast.error("Nazwa kategorii jest wymagana");
      return;
    }
    try {
      await createCategory.mutateAsync({
        name: newCategoryName,
        parent_id: addingParentId,
        color: newCategoryColor,
      });
      toast.success("Kategoria utworzona");
      setNewCategoryName("");
      setNewCategoryColor("#6b7280");
      setShowAddForm(false);
    } catch {
      toast.error(t("bładpodczastworzeniakategorii"));
    }
  };

  if (isLoading) {
    return <div className="p-6">{tc("loading")}</div>;
  }

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{t("kategorieProduktow")}</h1>
            <p className="text-muted-foreground mt-1">
              {t("zarzadzajHierarchicznymDrzewemKategorii")}
            </p>
          </div>
          <Button onClick={handleAddRoot} size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Nowa kategoria
          </Button>
        </div>

        {showAddForm && (
          <Card>
            <CardContent className="pt-4">
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={newCategoryColor}
                  onChange={(e) => setNewCategoryColor(e.target.value)}
                  className="h-9 w-12 cursor-pointer rounded border border-input p-1"
                />
                <Input
                  placeholder={
                    addingParentId
                      ? "Nazwa podkategorii..."
                      : "Nazwa kategorii..."
                  }
                  value={newCategoryName}
                  onChange={(e) => setNewCategoryName(e.target.value)}
                  className="w-64"
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreateCategory();
                    if (e.key === "Escape") setShowAddForm(false);
                  }}
                />
                <Button
                  size="sm"
                  onClick={handleCreateCategory}
                  disabled={createCategory.isPending}
                >
                  {createCategory.isPending ? "Tworzenie..." : tc("create")}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowAddForm(false)}
                >
                  Anuluj
                </Button>
              </div>
              {addingParentId && (
                <p className="text-xs text-muted-foreground mt-2">
                  {t("dodajeszPodkategorie")}
                </p>
              )}
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Drzewo kategorii</CardTitle>
          </CardHeader>
          <CardContent>
            {!tree || tree.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">
                {t("brakKategoriiKliknijQuotnowaKategoriaquotAbyDodac")}
              </p>
            ) : (
              <div className="space-y-0.5">
                {tree.map((cat) => (
                  <CategoryRow
                    key={cat.id}
                    category={cat}
                    expanded={expanded}
                    onToggle={handleToggle}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                    onAddChild={handleAddChild}
                    editState={editState}
                    onSaveEdit={handleSaveEdit}
                    onCancelEdit={handleCancelEdit}
                    onEditChange={handleEditChange}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}
