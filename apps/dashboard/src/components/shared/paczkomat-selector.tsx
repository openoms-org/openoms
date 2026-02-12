"use client";

import { useEffect, useRef, useState } from "react";
import { Check, ChevronsUpDown, MapPin, Loader2, Map } from "lucide-react";
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
} from "@/components/ui/dialog";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useInPostPointSearch, useGeowidgetToken } from "@/hooks/use-inpost-points";
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
  const { data: tokenData } = useGeowidgetToken();
  const token = tokenData?.geowidget_token || process.env.NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN || "";

  // Search mode — always works, no token needed
  if (mode === "search") {
    return <SearchMode value={value ?? ""} onPointSelect={onPointSelect} />;
  }

  // Inline/dialog — show search as primary, map as optional button
  return (
    <SearchWithMapFallback
      token={token}
      value={value ?? ""}
      onPointSelect={onPointSelect}
      layout={mode === "inline" ? "stacked" : "row"}
    />
  );
}

/** Primary: text search combobox + optional "Show map" button */
function SearchWithMapFallback({
  token,
  value,
  onPointSelect,
  layout,
}: {
  token: string;
  value: string;
  onPointSelect: (pointName: string) => void;
  layout: "row" | "stacked";
}) {
  const [mapOpen, setMapOpen] = useState(false);

  return (
    <div className={layout === "stacked" ? "space-y-3" : "flex gap-2 items-start"}>
      <div className={layout === "stacked" ? "" : "flex-1"}>
        <SearchMode value={value} onPointSelect={onPointSelect} />
      </div>
      {token && (
        <>
          <Button
            type="button"
            variant="outline"
            size={layout === "stacked" ? "default" : "icon"}
            onClick={() => setMapOpen(true)}
            title="Pokaż mapę paczkomatów"
          >
            <Map className="h-4 w-4" />
            {layout === "stacked" && <span className="ml-2">Pokaż na mapie</span>}
          </Button>
          <MapDialog
            token={token}
            open={mapOpen}
            onOpenChange={setMapOpen}
            onPointSelect={(name) => {
              onPointSelect(name);
              setMapOpen(false);
            }}
          />
        </>
      )}
    </div>
  );
}

/** Map dialog — only rendered when token is available */
function MapDialog({
  token,
  open,
  onOpenChange,
  onPointSelect,
}: {
  token: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPointSelect: (pointName: string) => void;
}) {
  const [widgetError, setWidgetError] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onPointSelect);
  callbackRef.current = onPointSelect;

  useEffect(() => {
    if (!open) return;
    setWidgetError(false);

    loadGeowidgetScript();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).onInpostPointSelected = (point: { name: string }) => {
      callbackRef.current(point.name);
    };

    const timer = setTimeout(() => {
      if (containerRef.current) {
        containerRef.current.innerHTML = "";
        const widget = document.createElement("inpost-geowidget");
        widget.setAttribute("onpoint", "onInpostPointSelected");
        widget.setAttribute("language", "pl");
        widget.setAttribute("config", "parcelcollect");
        widget.setAttribute("token", token);
        widget.style.display = "block";
        widget.style.width = "100%";
        widget.style.height = "100%";
        containerRef.current.appendChild(widget);

        // Detect widget auth errors after it loads
        const errorCheck = setTimeout(() => {
          const shadow = widget.shadowRoot;
          if (shadow) {
            const text = shadow.textContent || "";
            if (text.includes("Brak dostępu") || text.includes("nieprawidłow")) {
              setWidgetError(true);
            }
          }
        }, 3000);

        return () => clearTimeout(errorCheck);
      }
    }, 500);

    return () => {
      clearTimeout(timer);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (window as any).onInpostPointSelected;
    };
  }, [open, token]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl h-[80vh]">
        <DialogHeader>
          <DialogTitle>Wybierz paczkomat InPost</DialogTitle>
        </DialogHeader>
        {widgetError ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-4">
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-center max-w-md">
              <p className="font-medium text-destructive">Błąd tokenu GeoWidget</p>
              <p className="text-sm text-muted-foreground mt-2">
                Token GeoWidget jest przypisany do konkretnej domeny. Sprawdź w{" "}
                <a href="https://manager.paczkomaty.pl" target="_blank" rel="noopener noreferrer" className="underline">
                  panelu InPost
                </a>{" "}
                czy token jest wygenerowany dla domeny:{" "}
                <code className="bg-muted px-1 rounded text-xs">{typeof window !== "undefined" ? window.location.hostname : "localhost"}</code>
              </p>
            </div>
            <p className="text-sm text-muted-foreground">Zamknij dialog i użyj wyszukiwarki tekstowej.</p>
          </div>
        ) : (
          <div ref={containerRef} className="flex-1 min-h-[60vh]" />
        )}
      </DialogContent>
    </Dialog>
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
          type="button"
        >
          <span className="flex items-center gap-2 truncate">
            <MapPin className="h-4 w-4 shrink-0" />
            {value ? `Paczkomat: ${value}` : "Wybierz paczkomat..."}
          </span>
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
