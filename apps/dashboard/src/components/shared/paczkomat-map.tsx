"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { MapPin } from "lucide-react";

interface PaczkomatMapProps {
  onSelect: (pointName: string) => void;
  selectedPoint?: string;
}

export function PaczkomatMap({ onSelect, selectedPoint }: PaczkomatMapProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onSelect);
  callbackRef.current = onSelect;

  const loadScript = useCallback(() => {
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
  }, []);

  useEffect(() => {
    if (!open) return;

    loadScript();

    // Global callback for the geowidget onpoint attribute
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).onInpostPointSelected = (point: { name: string }) => {
      callbackRef.current(point.name);
      setOpen(false);
    };

    // Wait for script to load, then create the widget element
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
  }, [open, loadScript]);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" type="button" className="w-full justify-start">
          <MapPin className="h-4 w-4 mr-2" />
          {selectedPoint
            ? `Paczkomat: ${selectedPoint}`
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
