import { afterEach, describe, expect, it, vi } from "vitest";
import { navItems } from "@/lib/nav-items";
import { SHIPMENT_PROVIDERS } from "@/lib/constants";
import {
  getDashboardSurfaceMode,
  getFeatureReadiness,
  getSelectableCarrierShipmentProviders,
  getSelectableShipmentProviders,
  getRouteReadiness,
  getVisibleNavItems,
  getVisibleProviderKeys,
  getVisibleProvidersByCategory,
  isFeatureVisible,
  isRouteAccessible,
  setDashboardSurfaceModeOverride,
} from "@/lib/readiness";

const WAREHOUSE_HIDDEN_NAV_ROUTES = [
  "/packing",
  "/pick-pack",
  "/settings/warehouses",
  "/stocktakes",
  "/settings/warehouse-documents",
  "/stock-sync",
  "/settings/inventory",
] as const;

const WAREHOUSE_VERIFY_ROUTES = [
  "/packing",
  "/pick-pack",
  "/pick-pack/session-1",
  "/stocktakes",
  "/stocktakes/new",
  "/stocktakes/stocktake-1",
  "/settings/warehouse-documents",
  "/settings/warehouse-documents/new",
  "/settings/warehouse-documents/doc-1",
  "/settings/inventory",
] as const;

const WAREHOUSE_CONTROLLED_ROUTES = [
  "/settings/warehouses",
  "/settings/warehouses/warehouse-1",
] as const;

const WAREHOUSE_BETA_ROUTES = ["/stock-sync", "/stock-sync/events"] as const;

const WAREHOUSE_VALIDATION_ROUTES = [
  ...WAREHOUSE_VERIFY_ROUTES,
  ...WAREHOUSE_CONTROLLED_ROUTES,
  ...WAREHOUSE_BETA_ROUTES,
] as const;

describe("getDashboardSurfaceMode", () => {
  afterEach(() => {
    setDashboardSurfaceModeOverride(undefined);
    vi.unstubAllEnvs();
  });

  it("defaults to client-ready when no surface env is set", () => {
    expect(getDashboardSurfaceMode()).toBe("client-ready");
  });

  it("reads OPENOMS_DASHBOARD_SURFACE at runtime", () => {
    vi.stubEnv("OPENOMS_DASHBOARD_SURFACE", "full");
    expect(getDashboardSurfaceMode()).toBe("full");
  });

  it("keeps NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE as a local/build fallback", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "full");
    expect(getDashboardSurfaceMode()).toBe("full");
  });

  it("prefers the runtime env over a baked NEXT_PUBLIC value", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "full");
    vi.stubEnv("OPENOMS_DASHBOARD_SURFACE", "client-ready");
    expect(getDashboardSurfaceMode()).toBe("client-ready");
  });

  it("lets a provider override win so client nav matches the server value", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "client-ready");
    setDashboardSurfaceModeOverride("full");
    expect(getDashboardSurfaceMode()).toBe("full");
    expect(isRouteAccessible("/suppliers")).toBe(true);
  });

  it("treats unknown surface values as client-ready", () => {
    vi.stubEnv("OPENOMS_DASHBOARD_SURFACE", "maybe");
    expect(getDashboardSurfaceMode()).toBe("client-ready");
  });
});

describe("dashboard feature readiness", () => {
  it("keeps the client-ready navigation focused on certified core flows", () => {
    const visible = getVisibleNavItems(navItems, {
      isAdmin: true,
      mode: "client-ready",
    }).map((item) => item.href);

    expect(visible).toContain("/");
    expect(visible).toContain("/orders");
    expect(visible).toContain("/customers");
    expect(visible).toContain("/returns");
    expect(visible).toContain("/products");
    expect(visible).toContain("/settings/product-categories");
    expect(visible).toContain("/settings/company");
    expect(visible).toContain("/settings/users");
    expect(visible).toContain("/settings/roles");
    expect(visible).toContain("/settings/security");
    expect(visible).toContain("/marketplaces");
    expect(visible).toContain("/carriers");

    expect(visible).not.toContain("/invoices");
    expect(visible).not.toContain("/invoicing");
    expect(visible).not.toContain("/reports/forecast");
    expect(visible).not.toContain("/reports/carbon");
    expect(visible).not.toContain("/reports/vat-oss");
    expect(visible).not.toContain("/repricing");
    expect(visible).not.toContain("/suppliers");
    expect(visible).not.toContain("/listing-sync");
    for (const route of WAREHOUSE_HIDDEN_NAV_ROUTES) {
      expect(visible).not.toContain(route);
    }
    expect(visible).not.toContain("/settings/billing");
    expect(visible).not.toContain("/settings/accounting");
    expect(visible).not.toContain("/settings/vat-oss");
    expect(visible).not.toContain("/settings/sms");
    expect(visible).not.toContain("/settings/ksef");
    expect(visible).not.toContain("/settings/marketing");
    expect(visible).not.toContain("/settings/helpdesk");
    expect(visible).not.toContain("/settings/notifications");
    expect(visible).not.toContain("/workflows");
  });

  it("still respects admin-only navigation after readiness filtering", () => {
    const visibleForMember = getVisibleNavItems(navItems, {
      isAdmin: false,
      mode: "client-ready",
    }).map((item) => item.href);

    expect(visibleForMember).toContain("/orders");
    expect(visibleForMember).toContain("/settings/security");
    expect(visibleForMember).not.toContain("/settings/company");
    expect(visibleForMember).not.toContain("/settings/users");
    expect(visibleForMember).not.toContain("/marketplaces");
  });

  it("never exposes blocked routes even in full dashboard mode", () => {
    const visible = getVisibleNavItems(navItems, {
      isAdmin: true,
      mode: "full",
    }).map((item) => item.href);

    expect(visible).toContain("/suppliers");
    expect(visible).toContain("/repricing");
    expect(visible).not.toContain("/settings/sms");
    expect(visible).not.toContain("/settings/email");
    expect(visible).not.toContain("/settings/ksef");
    expect(visible).not.toContain("/settings/marketing");
    expect(visible).not.toContain("/settings/helpdesk");
    expect(visible).not.toContain("/settings/notifications");
  });

  it("classifies direct routes by the longest matching readiness rule", () => {
    expect(getRouteReadiness("/marketplaces/allegro")).toBe("ready");
    expect(getRouteReadiness("/marketplaces/allegro/offers")).toBe("controlled");
    expect(getRouteReadiness("/products/product-1/listings")).toBe("controlled");
    expect(getRouteReadiness("/products/product-1/variants")).toBe("verify");
    expect(getRouteReadiness("/customers/import")).toBe("verify");
    expect(getRouteReadiness("/carriers")).toBe("ready");
    expect(getRouteReadiness("/carriers/new")).toBe("ready");
    for (const route of WAREHOUSE_VERIFY_ROUTES) {
      expect(getRouteReadiness(route)).toBe("verify");
    }
    for (const route of WAREHOUSE_CONTROLLED_ROUTES) {
      expect(getRouteReadiness(route)).toBe("controlled");
    }
    for (const route of WAREHOUSE_BETA_ROUTES) {
      expect(getRouteReadiness(route)).toBe("beta");
    }
    expect(getRouteReadiness("/settings")).toBe("ready");
    expect(getRouteReadiness("/settings/email")).toBe("blocked");
    expect(getRouteReadiness("/settings/sms")).toBe("blocked");
    expect(getRouteReadiness("/settings/ksef")).toBe("blocked");
    expect(getRouteReadiness("/settings/billing")).toBe("controlled");
    expect(getRouteReadiness("/settings/accounting")).toBe("controlled");
    expect(getRouteReadiness("/settings/vat-oss")).toBe("beta");
    expect(getRouteReadiness("/settings/marketing")).toBe("blocked");
    expect(getRouteReadiness("/settings/helpdesk")).toBe("blocked");
    expect(getRouteReadiness("/settings/notifications")).toBe("blocked");
    expect(getRouteReadiness("/reports/forecast")).toBe("beta");
    expect(getRouteReadiness("/reports/vat-oss")).toBe("beta");
  });

  it("blocks direct access to beta and blocked routes in client-ready mode", () => {
    expect(isRouteAccessible("/orders/new", { mode: "client-ready" })).toBe(true);
    expect(isRouteAccessible("/marketplaces/allegro", { mode: "client-ready" })).toBe(true);
    expect(isRouteAccessible("/carriers/new", { mode: "client-ready" })).toBe(true);
    expect(isRouteAccessible("/shipments/new", { mode: "client-ready" })).toBe(true);
    expect(isRouteAccessible("/settings", { mode: "client-ready" })).toBe(true);
    expect(isRouteAccessible("/marketplaces/allegro/offers", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/products/product-1/listings", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/customers/import", { mode: "client-ready" })).toBe(false);
    for (const route of WAREHOUSE_VALIDATION_ROUTES) {
      expect(isRouteAccessible(route, { mode: "client-ready" })).toBe(false);
    }
    expect(isRouteAccessible("/reports/forecast", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/reports/vat-oss", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/invoices", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/invoices/inv-1", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/invoicing", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/billing", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/accounting", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/vat-oss", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/sms", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/ksef", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/marketing", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/helpdesk", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/settings/notifications", { mode: "client-ready" })).toBe(false);
  });

  it("keeps removed settings routes blocked even in full mode", () => {
    expect(isRouteAccessible("/settings/email", { mode: "full" })).toBe(false);
    expect(isRouteAccessible("/settings/sms", { mode: "full" })).toBe(false);
    expect(isRouteAccessible("/settings/ksef", { mode: "full" })).toBe(false);
    expect(isRouteAccessible("/settings/marketing", { mode: "full" })).toBe(false);
    expect(isRouteAccessible("/settings/helpdesk", { mode: "full" })).toBe(false);
    expect(isRouteAccessible("/settings/notifications", { mode: "full" })).toBe(false);
  });

  it("allows legal and accounting routes only in full validation mode", () => {
    expect(isRouteAccessible("/invoices", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/invoices/inv-1", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/invoicing", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/settings/billing", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/settings/accounting", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/settings/vat-oss", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/reports/vat-oss", { mode: "full" })).toBe(true);
    expect(isRouteAccessible("/settings/ksef", { mode: "full" })).toBe(false);
  });

  it("allows warehouse and fulfillment routes only in full validation mode", () => {
    for (const route of WAREHOUSE_VALIDATION_ROUTES) {
      expect(isRouteAccessible(route, { mode: "full" })).toBe(true);
    }
  });

  it("treats unreviewed direct routes as verify-only instead of client-ready", () => {
    expect(getRouteReadiness("/future-module")).toBe("verify");
    expect(isRouteAccessible("/future-module", { mode: "client-ready" })).toBe(false);
    expect(isRouteAccessible("/future-module", { mode: "full" })).toBe(true);
  });
});

describe("shipment provider readiness", () => {
  it("keeps unverified carriers out of shipment creation in client-ready mode", () => {
    expect(
      getSelectableShipmentProviders(SHIPMENT_PROVIDERS, {
        mode: "client-ready",
      }),
    ).toEqual(["inpost", "manual"]);
  });

  it("keeps internal carrier validation available in full mode", () => {
    expect(
      getSelectableShipmentProviders(SHIPMENT_PROVIDERS, {
        mode: "full",
      }),
    ).toEqual([
      "inpost",
      "dhl",
      "dpd",
      "gls",
      "ups",
      "poczta_polska",
      "orlen_paczka",
      "fedex",
      "manual",
    ]);
  });

  it("builds carrier-only provider lists without manual fallback entries", () => {
    expect(
      getSelectableCarrierShipmentProviders(SHIPMENT_PROVIDERS, {
        mode: "client-ready",
      }),
    ).toEqual(["inpost"]);

    expect(
      getSelectableCarrierShipmentProviders(SHIPMENT_PROVIDERS, {
        mode: "full",
      }),
    ).toEqual([
      "inpost",
      "dhl",
      "dpd",
      "gls",
      "ups",
      "poczta_polska",
      "orlen_paczka",
      "fedex",
    ]);
  });
});

describe("provider readiness", () => {
  it("does not render provider description translation keys as customer-facing text", () => {
    const providers = getVisibleProvidersByCategory("marketplace", {
      mode: "full",
    });

    for (const provider of providers) {
      expect(provider.description).not.toMatch(/^providers\./);
    }
  });

  it("only exposes certified marketplace providers in client-ready mode", () => {
    const providers = getVisibleProvidersByCategory("marketplace", {
      mode: "client-ready",
    }).map((provider) => provider.key);

    expect(providers).toEqual(["allegro"]);
  });

  it("only exposes certified carrier providers in client-ready mode", () => {
    const providers = getVisibleProvidersByCategory("carrier", {
      mode: "client-ready",
    }).map((provider) => provider.key);

    expect(providers).toEqual(["inpost"]);
  });

  it("keeps invoicing providers out of client-ready mode until certification", () => {
    const clientReadyProviders = getVisibleProvidersByCategory("invoicing", {
      mode: "client-ready",
    }).map((provider) => provider.key);
    const fullModeProviders = getVisibleProvidersByCategory("invoicing", {
      mode: "full",
    }).map((provider) => provider.key);

    expect(clientReadyProviders).toEqual([]);
    expect(fullModeProviders).toEqual(["fakturownia", "wfirma", "infakt"]);
    expect(
      getVisibleProviderKeys(["fakturownia", "wfirma", "infakt"], {
        mode: "client-ready",
      }),
    ).toEqual([]);
  });

  it("keeps blocked providers out of full mode while allowing beta review", () => {
    const providers = getVisibleProvidersByCategory("marketplace", {
      mode: "full",
    }).map((provider) => provider.key);

    expect(providers).toContain("amazon");
    expect(providers).toContain("olx");
    expect(providers).toContain("erli");
    expect(providers).not.toContain("empik");
    expect(providers).not.toContain("shopify");
    expect(providers).not.toContain("prestashop");
    expect(providers).not.toContain("shoper");
  });

  it("filters arbitrary provider key lists with the same readiness rules", () => {
    expect(
      getVisibleProviderKeys(["allegro", "amazon", "olx", "empik"], {
        mode: "client-ready",
      }),
    ).toEqual(["allegro"]);

    expect(
      getVisibleProviderKeys(["allegro", "amazon", "olx", "empik"], {
        mode: "full",
      }),
    ).toEqual(["allegro", "amazon", "olx"]);
  });
});

describe("feature visibility (ready pages guarding gated fetches)", () => {
  it("reads canonical feature states and falls back to verify for unknown ids", () => {
    expect(getFeatureReadiness("suppliers")).toBe("controlled");
    expect(getFeatureReadiness("dropship")).toBe("beta");
    expect(getFeatureReadiness("marketing")).toBe("blocked");
    expect(getFeatureReadiness("not_a_real_feature")).toBe("verify");
  });

  it("hides non-ready features in client-ready and reveals non-blocked ones in full", () => {
    // controlled/beta hidden in client-ready, shown in full
    expect(isFeatureVisible("suppliers", { mode: "client-ready" })).toBe(false);
    expect(isFeatureVisible("suppliers", { mode: "full" })).toBe(true);
    expect(isFeatureVisible("dropship", { mode: "client-ready" })).toBe(false);
    expect(isFeatureVisible("ai", { mode: "client-ready" })).toBe(false);
    expect(isFeatureVisible("repricing", { mode: "client-ready" })).toBe(false);
    // blocked is never visible
    expect(isFeatureVisible("marketing", { mode: "full" })).toBe(false);
  });
});
