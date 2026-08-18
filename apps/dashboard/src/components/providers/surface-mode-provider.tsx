"use client";

import type { ReactNode } from "react";
import {
  setDashboardSurfaceModeOverride,
  type DashboardSurfaceMode,
} from "@/lib/readiness";

export function SurfaceModeProvider({
  mode,
  children,
}: {
  mode: DashboardSurfaceMode;
  children: ReactNode;
}) {
  setDashboardSurfaceModeOverride(mode);
  return children;
}
