"use client";

import { useState, useEffect } from "react";
import { Check, ChevronsUpDown, MapPin, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useInPostPointSearch } from "@/hooks/use-inpost-points";
import type { InPostPoint } from "@/types/api";

interface PaczkomatSearchProps {
  value: string;
  onValueChange: (value: string) => void;
}

export function PaczkomatSearch({ value, onValueChange }: PaczkomatSearchProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const { data, isLoading } = useInPostPointSearch(debouncedQuery);
  const points = data?.items ?? [];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
        >
          {value ? value : "Wybierz paczkomat..."}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Wpisz miasto lub kod paczkomatu..."
            value={searchQuery}
            onValueChange={setSearchQuery}
          />
          <CommandList>
            {isLoading && debouncedQuery.length >= 2 ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              </div>
            ) : debouncedQuery.length < 2 ? (
              <CommandEmpty>Wpisz min. 2 znaki, aby wyszukac</CommandEmpty>
            ) : points.length === 0 ? (
              <CommandEmpty>Nie znaleziono paczkomatow</CommandEmpty>
            ) : (
              <CommandGroup>
                {points.map((point: InPostPoint) => (
                  <CommandItem
                    key={point.name}
                    value={point.name}
                    onSelect={() => {
                      onValueChange(point.name);
                      setOpen(false);
                    }}
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        value === point.name ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <div className="flex flex-col">
                      <div className="flex items-center gap-1.5">
                        <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
                        <span className="font-medium">{point.name}</span>
                        <span className="text-muted-foreground text-xs">
                          {point.address_details?.city ?? ""}
                        </span>
                      </div>
                      <span className="text-xs text-muted-foreground ml-5">
                        {point.address.line1}
                        {point.location_description
                          ? ` — ${point.location_description}`
                          : ""}
                      </span>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
