"use client";

import { createContext, useContext, useState, useEffect } from "react";

export type TableDensity = "compact" | "comfortable" | "spacious";

interface TableDensityContextType {
  density: TableDensity;
  setDensity: (density: TableDensity) => void;
}

const TableDensityContext = createContext<TableDensityContextType>({
  density: "comfortable",
  setDensity: () => {},
});

export function TableDensityProvider({ children }: { children: React.ReactNode }) {
  const [density, setDensity] = useState<TableDensity>("comfortable");

  useEffect(() => {
    const saved = localStorage.getItem("table-density") as TableDensity | null;
    if (saved && ["compact", "comfortable", "spacious"].includes(saved)) {
      setDensity(saved);
    }
  }, []);

  const handleSetDensity = (d: TableDensity) => {
    setDensity(d);
    localStorage.setItem("table-density", d);
  };

  return (
    <TableDensityContext.Provider value={{ density, setDensity: handleSetDensity }}>
      {children}
    </TableDensityContext.Provider>
  );
}

export function useTableDensity() {
  return useContext(TableDensityContext);
}

export const densityConfig: Record<TableDensity, { cellPadding: string; rowHeight: string }> = {
  compact: { cellPadding: "px-3 py-1.5", rowHeight: "h-10" },
  comfortable: { cellPadding: "px-4 py-2.5", rowHeight: "h-12" },
  spacious: { cellPadding: "px-4 py-3.5", rowHeight: "h-14" },
};
