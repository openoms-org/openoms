"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { Loader2, Pencil } from "lucide-react";
import { toast } from "sonner";

export interface EditableColumnConfig<T> {
  accessorKey: string;
  type: "text" | "number" | "select";
  options?: { label: string; value: string }[];
  onSave: (row: T, value: unknown) => Promise<void>;
}

interface EditableCellProps<T> {
  row: T;
  value: unknown;
  config: EditableColumnConfig<T>;
  displayContent: React.ReactNode;
}

export function EditableCell<T>({
  row,
  value,
  config,
  displayContent,
}: EditableCellProps<T>) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(String(value ?? ""));
  const [isSaving, setIsSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLSelectElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      if (inputRef.current instanceof HTMLInputElement) {
        inputRef.current.select();
      }
    }
  }, [isEditing]);

  const startEditing = useCallback(() => {
    if (isSaving) return;
    setEditValue(String(value ?? ""));
    setIsEditing(true);
  }, [isSaving, value]);

  const cancelEditing = useCallback(() => {
    setEditValue(String(value ?? ""));
    setIsEditing(false);
  }, [value]);

  const saveValue = useCallback(async () => {
    const newValue =
      config.type === "number" ? Number(editValue) : editValue;

    // Skip save if value hasn't changed
    if (String(newValue) === String(value ?? "")) {
      setIsEditing(false);
      return;
    }

    setIsSaving(true);
    try {
      await config.onSave(row, newValue);
      setIsEditing(false);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Nie udało się zapisać zmian";
      toast.error(message);
    } finally {
      setIsSaving(false);
    }
  }, [config, editValue, row, value]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        saveValue();
      } else if (e.key === "Escape") {
        e.preventDefault();
        cancelEditing();
      }
    },
    [saveValue, cancelEditing]
  );

  if (isSaving) {
    return (
      <div className="flex items-center gap-2">
        <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
        <span className="text-xs text-muted-foreground">Zapisywanie...</span>
      </div>
    );
  }

  if (!isEditing) {
    return (
      <div
        className="group relative cursor-pointer rounded px-1 py-0.5 -mx-1 -my-0.5 hover:bg-muted/40 transition-colors"
        onDoubleClick={(e) => {
          e.stopPropagation();
          startEditing();
        }}
        title="Kliknij dwukrotnie, aby edytować"
      >
        {displayContent}
        <Pencil className="absolute top-1 right-1 h-3 w-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
    );
  }

  if (config.type === "select" && config.options) {
    return (
      <select
        ref={inputRef as React.RefObject<HTMLSelectElement>}
        value={editValue}
        onChange={(e) => {
          setEditValue(e.target.value);
          // Auto-save on select change
          const newValue = e.target.value;
          setIsSaving(true);
          setIsEditing(false);
          config
            .onSave(row, newValue)
            .catch((err) => {
              const message =
                err instanceof Error
                  ? err.message
                  : "Nie udało się zapisać zmian";
              toast.error(message);
            })
            .finally(() => {
              setIsSaving(false);
            });
        }}
        onKeyDown={handleKeyDown}
        onBlur={cancelEditing}
        onClick={(e) => e.stopPropagation()}
        className="h-7 w-full min-w-[120px] rounded border border-input bg-background px-2 text-xs focus:border-ring focus:ring-1 focus:ring-ring/50 outline-none"
      >
        {config.options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    );
  }

  return (
    <input
      ref={inputRef as React.RefObject<HTMLInputElement>}
      type={config.type === "number" ? "number" : "text"}
      value={editValue}
      onChange={(e) => setEditValue(e.target.value)}
      onKeyDown={handleKeyDown}
      onBlur={saveValue}
      onClick={(e) => e.stopPropagation()}
      step={config.type === "number" ? "0.01" : undefined}
      className="h-7 w-full min-w-[80px] rounded border border-input bg-background px-2 text-sm focus:border-ring focus:ring-1 focus:ring-ring/50 outline-none"
    />
  );
}
