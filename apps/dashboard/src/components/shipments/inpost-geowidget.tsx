"use client";

import { useEffect, useRef, useCallback } from "react";

interface InPostGeowidgetProps {
  onPointSelect: (pointName: string) => void;
}

let scriptLoaded = false;

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

  // Load geowidget script + CSS once
  const loadScript = useCallback(() => {
    if (scriptLoaded) return;
    scriptLoaded = true;

    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = "https://geowidget.inpost.pl/inpost-geowidget.css";
    document.head.appendChild(link);

    const script = document.createElement("script");
    script.src = "https://geowidget.inpost.pl/inpost-geowidget.js";
    script.defer = true;
    document.head.appendChild(script);
  }, []);

  // Create the custom element via DOM API (avoids JSX type issues with web components)
  useEffect(() => {
    loadScript();

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
  }, [loadScript, token]);

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
