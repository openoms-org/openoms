import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { SurfaceModeProvider } from "@/components/providers/surface-mode-provider";
import {
  getDashboardSurfaceMode,
  isRouteAccessible,
  setDashboardSurfaceModeOverride,
} from "@/lib/readiness";

describe("SurfaceModeProvider", () => {
  afterEach(() => {
    cleanup();
    setDashboardSurfaceModeOverride(undefined);
  });

  it("installs the server surface mode so client readiness helpers see it", () => {
    render(
      <SurfaceModeProvider mode="full">
        <div />
      </SurfaceModeProvider>,
    );

    expect(getDashboardSurfaceMode()).toBe("full");
    expect(isRouteAccessible("/suppliers")).toBe(true);
  });

  it("keeps the default locked when the server passes client-ready", () => {
    render(
      <SurfaceModeProvider mode="client-ready">
        <div />
      </SurfaceModeProvider>,
    );

    expect(getDashboardSurfaceMode()).toBe("client-ready");
    expect(isRouteAccessible("/suppliers")).toBe(false);
  });
});
