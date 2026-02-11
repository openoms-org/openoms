"use client";

import { useEffect, useRef, useState } from "react";
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useInPostPointSearch } from "@/hooks/use-inpost-points";
import type { InPostPoint } from "@/types/api";

function loadGeowidgetScript() {
  const existing = document.getElementById("inpost-geowidget-script");
  if (existing) return;

  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = "https://geowidget.inpost.pl/inpost-geowidget.css";
  document.head.appendChild(link);

  const script = document.createElement("script");
  script.id = "inpost-geowidget-script";
  script.src = "https://geowidget.inpost.pl/inpost-geowidget.js";
  script.defer = true;
  document.head.appendChild(script);
}

interface PaczkomatSelectorProps {
  mode?: "dialog" | "inline" | "search";
  value?: string;
  onPointSelect: (pointName: string) => void;
}

export function PaczkomatSelector({
  mode = "dialog",
  value,
  onPointSelect,
}: PaczkomatSelectorProps) {
  if (mode === "search") {
    return <SearchMode value={value ?? ""} onPointSelect={onPointSelect} />;
  }

  if (mode === "inline") {
    return <InlineMode onPointSelect={onPointSelect} />;
  }

  return <DialogMode value={value} onPointSelect={onPointSelect} />;
}

function DialogMode({
  value,
  onPointSelect,
}: {
  value?: string;
  onPointSelect: (pointName: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onPointSelect);
  callbackRef.current = onPointSelect;

  useEffect(() => {
    if (!open) return;

    loadGeowidgetScript();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).onInpostPointSelected = (point: { name: string }) => {
      callbackRef.current(point.name);
      setOpen(false);
    };

    const timer = setTimeout(() => {
      if (containerRef.current) {
        containerRef.current.innerHTML = "";
        const widget = document.createElement("inpost-geowidget");
        widget.setAttribute("onpoint", "onInpostPointSelected");
        widget.setAttribute("language", "pl");
        widget.setAttribute("config", "parcelcollect");
        const token = process.env.NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN;
        if (token) {
          widget.setAttribute("token", token);
        }
        widget.style.display = "block";
        widget.style.width = "100%";
        widget.style.height = "100%";
        containerRef.current.appendChild(widget);
      }
    }, 500);

    return () => {
      clearTimeout(timer);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (window as any).onInpostPointSelected;
    };
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" type="button" className="w-full justify-start">
          <MapPin className="h-4 w-4 mr-2" />
          {value
            ? `Paczkomat: ${value}`
            : "Wybierz paczkomat na mapie"}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-4xl h-[80vh]">
        <DialogHeader>
          <DialogTitle>Wybierz paczkomat InPost</DialogTitle>
        </DialogHeader>
        <div ref={containerRef} className="flex-1 min-h-[60vh]" />
      </DialogContent>
    </Dialog>
  );
}

function InlineMode({
  onPointSelect,
}: {
  onPointSelect: (pointName: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onPointSelect);
  callbackRef.current = onPointSelect;

  const token = process.env.NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN || "";

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__inpostGeowidgetCallback = (point: { name: string }) => {
      callbackRef.current(point.name);
    };
    return () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (window as any).__inpostGeowidgetCallback;
    };
  }, []);

  useEffect(() => {
    loadGeowidgetScript();

    if (!containerRef.current || !token) return;

    const el = document.createElement("inpost-geowidget");
    el.setAttribute("token", token);
    el.setAttribute("language", "pl");
    el.setAttribute("config", "parcelCollect");
    el.setAttribute("onpoint", "__inpostGeowidgetCallback");
    el.style.display = "block";
    el.style.width = "100%";
    el.style.height = "100%";

    containerRef.current.appendChild(el);

    return () => {
      if (containerRef.current?.contains(el)) {
        containerRef.current.removeChild(el);
      }
    };
  }, [token]);

  if (!token) {
    return (
      <div className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
        <p>Brak tokenu GeoWidget InPost.</p>
        <p className="text-xs mt-1">
          Ustaw NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN w pliku .env
        </p>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="h-[400px] w-full rounded-md overflow-hidden border" />
  );
}

function SearchMode({
  value,
  onPointSelect,
}: {
  value: string;
  onPointSelect: (pointName: string) => void;
}) {
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
              <CommandEmpty>Wpisz min. 2 znaki, aby wyszukać</CommandEmpty>
            ) : points.length === 0 ? (
              <CommandEmpty>Nie znaleziono paczkomatów</CommandEmpty>
            ) : (
              <CommandGroup>
                {points.map((point: InPostPoint) => (
                  <CommandItem
                    key={point.name}
                    value={point.name}
                    onSelect={() => {
                      onPointSelect(point.name);
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
