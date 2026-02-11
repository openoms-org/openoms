"use client";

import { useEffect, useRef } from "react";
import { loadInPostGeowidgetScript } from "@/components/shared/paczkomat-map";

interface InPostGeowidgetProps {
  onPointSelect: (pointName: string) => void;
}

export function InPostGeowidget({ onPointSelect }: InPostGeowidgetProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onPointSelect);
  callbackRef.current = onPointSelect;

  const token = process.env.NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN || "";

  // Register global callback for geowidget onpoint attribute
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

  // Create the custom element via DOM API (avoids JSX type issues with web components)
  useEffect(() => {
    loadInPostGeowidgetScript();

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
