"use client";

import { useState } from "react";
import { Check, ChevronsUpDown, FolderTree } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { useCategoryTree } from "@/hooks/use-categories";
import type { ProductCategory } from "@/types/api";

interface CategoryTreePickerProps {
  value?: string;
  onChange: (value: string | undefined) => void;
  placeholder?: string;
  className?: string;
  allowClear?: boolean;
}

function flattenTree(
  categories: ProductCategory[],
  prefix = ""
): { category: ProductCategory; label: string }[] {
  const result: { category: ProductCategory; label: string }[] = [];
  for (const cat of categories) {
    const indent = prefix ? `${prefix} / ${cat.name}` : cat.name;
    result.push({ category: cat, label: indent });
    if (cat.children?.length) {
      result.push(...flattenTree(cat.children, indent));
    }
  }
  return result;
}

export function CategoryTreePicker({
  value,
  onChange,
  placeholder = "Wybierz kategorię...",
  className,
  allowClear = true,
}: CategoryTreePickerProps) {
  const [open, setOpen] = useState(false);
  const { data: tree } = useCategoryTree();

  const flatItems = tree ? flattenTree(tree) : [];
  const selected = flatItems.find((item) => item.category.id === value);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn("justify-between font-normal", className)}
        >
          {selected ? (
            <span className="flex items-center gap-2 truncate">
              <span
                className="inline-block h-2.5 w-2.5 rounded-full shrink-0"
                style={{ backgroundColor: selected.category.color }}
              />
              {selected.label}
            </span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="start">
        <Command>
          <CommandInput placeholder="Szukaj kategorii..." />
          <CommandList>
            <CommandEmpty>Brak kategorii</CommandEmpty>
            <CommandGroup>
              {allowClear && value && (
                <CommandItem
                  onSelect={() => {
                    onChange(undefined);
                    setOpen(false);
                  }}
                >
                  <FolderTree className="mr-2 h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Brak kategorii</span>
                </CommandItem>
              )}
              {flatItems.map((item) => (
                <CommandItem
                  key={item.category.id}
                  value={item.label}
                  onSelect={() => {
                    onChange(item.category.id);
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value === item.category.id ? "opacity-100" : "opacity-0"
                    )}
                  />
                  <span
                    className="inline-block h-2.5 w-2.5 rounded-full shrink-0 mr-2"
                    style={{
                      backgroundColor: item.category.color,
                      marginLeft: `${item.category.depth * 12}px`,
                    }}
                  />
                  {item.category.name}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
