import type { NavItem } from "@/lib/nav-items";
import { getProvidersByCategory, type ProviderInfo } from "@/lib/provider-info";
// Canonical feature-readiness registry, synced from the api-server source of
// truth (apps/api-server/internal/readiness/readiness.json) by
// scripts/sync-readiness.mjs. See that script for why a committed copy is used.
import registry from "@/lib/readiness.generated.json";

export type FeatureReadiness = "ready" | "controlled" | "verify" | "beta" | "blocked";
export type DashboardSurfaceMode = "client-ready" | "full";

interface VisibilityOptions {
  mode?: DashboardSurfaceMode;
}

interface NavVisibilityOptions extends VisibilityOptions {
  isAdmin: boolean;
}

const DEFAULT_SURFACE_MODE: DashboardSurfaceMode = "client-ready";

// Helm/runtime override for the current process. The public image stays
// client-ready because NEXT_PUBLIC_* is baked at build time; a server layout
// reads OPENOMS_DASHBOARD_SURFACE and a client provider installs that value
// here so nav and ReadinessRouteGuard see the same mode after hydration.
let runtimeSurfaceModeOverride: DashboardSurfaceMode | undefined;

export function setDashboardSurfaceModeOverride(
  mode: DashboardSurfaceMode | undefined,
): void {
  runtimeSurfaceModeOverride = mode;
}

function parseDashboardSurfaceMode(value: string | undefined): DashboardSurfaceMode {
  return value === "full" ? "full" : DEFAULT_SURFACE_MODE;
}

export function getDashboardSurfaceMode(): DashboardSurfaceMode {
  if (runtimeSurfaceModeOverride !== undefined) {
    return runtimeSurfaceModeOverride;
  }

  return parseDashboardSurfaceMode(
    process.env.OPENOMS_DASHBOARD_SURFACE ??
      process.env.NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE,
  );
}

type Registry = {
  providers: Record<string, FeatureReadiness>;
  features: Record<
    string,
    { state: FeatureReadiness; routes: string[]; endpoints: string[] }
  >;
};

const reg = registry as Registry;

// Dashboard-only route readiness. The canonical registry covers only the routes
// that are gated server-side (one feature entry per gated /v1 route group). The
// dashboard navigation also has routes that are either always `ready` (core
// flows that are never gated) or whose readiness is purely a UI concern with no
// backend gate (e.g. settings sub-pages, import flows, marketplace sub-views).
// These are declared here and take precedence over the derived registry values.
const ROUTE_READINESS_OVERRIDES: Record<string, FeatureReadiness> = {
  "/": "ready",
  "/help": "ready",
  "/orders": "ready",
  "/orders/import": "verify",
  "/customers": "ready",
  "/customers/import": "verify",
  "/returns": "ready",
  "/products": "ready",
  "/products/import": "verify",
  "/settings/product-categories": "ready",
  "/settings/print-templates": "verify",
  "/shipments": "ready",
  "/carriers": "ready",
  "/carriers/new": "ready",
  "/marketplaces": "ready",
  "/marketplaces/new": "ready",
  "/marketplaces/allegro": "ready",
  "/marketplaces/allegro/catalog": "controlled",
  "/marketplaces/allegro/delivery": "controlled",
  "/marketplaces/allegro/disputes": "controlled",
  "/marketplaces/allegro/finance": "controlled",
  "/marketplaces/allegro/messages": "controlled",
  "/marketplaces/allegro/offers": "controlled",
  "/marketplaces/allegro/policies": "controlled",
  "/marketplaces/allegro/promotions": "controlled",
  "/marketplaces/allegro/ratings": "controlled",
  "/marketplaces/allegro/returns": "controlled",
  "/marketplaces/allegro/shipments": "controlled",
  "/marketplaces/allegro/import": "controlled",
  "/marketplaces/olx": "controlled",
  "/marketplaces/amazon": "beta",
  "/marketplaces/ebay": "beta",
  "/marketplaces/ebay/offers": "beta",
  "/marketplaces/shopify": "blocked",
  "/marketplaces/prestashop": "blocked",
  "/marketplaces/shoper": "blocked",
  "/integrations": "controlled",
  // Feed configuration UI gates on its own (beta) regardless of the suppliers
  // backend feature, which is "controlled".
  "/settings/feeds": "beta",
  "/reports": "verify",
  "/reports/forecast": "beta",
  "/reports/carbon": "beta",
  "/reports/vat-oss": "beta",
  "/tools/bg-removal": "beta",
  "/settings/marketing": "blocked",
  "/settings/helpdesk": "blocked",
  "/settings/billing": "controlled",
  "/settings": "ready",
  "/settings/company": "ready",
  "/settings/users": "ready",
  "/settings/roles": "ready",
  "/settings/security": "ready",
  "/settings/order-statuses": "controlled",
  "/settings/custom-fields": "controlled",
  "/settings/accounting": "controlled",
  "/settings/email": "blocked",
  "/settings/ksef": "blocked",
  "/settings/vat-oss": "beta",
  "/settings/inventory": "verify",
  "/settings/notifications": "blocked",
  "/settings/sms": "blocked",
  "/settings/webhooks": "controlled",
  "/settings/webhooks/deliveries": "controlled",
  "/audit": "ready",
  // OPE-419 / OPE-423c: process-backed operations control tower. The read
  // endpoints are always registered server-side (RLS-scoped) and return
  // empty/zero while FULFILLMENT_PROCESS_ENABLED is off, so the surface is safe
  // to expose and simply renders intentional empty states until recording is
  // enabled. The page is operator-guarded (AdminGuard) regardless of readiness.
  //
  // ROUTE-READINESS DECISION (OPE-423c): this route stays unconditionally
  // "ready" — i.e. directly reachable in BOTH the heuristic (default) and
  // process-backed modes — and is NOT coupled to the cutover flag. Rationale:
  //   1. It is the dedicated full process-backed surface (the home-page panel is
  //      only a preview), so an operator must be able to open it to read the
  //      parity-readiness indicator and decide whether to cut over BEFORE the
  //      flag is flipped. Gating it behind the flag would create a chicken/egg:
  //      you could not inspect parity until after you had already cut over.
  //   2. ReadinessRouteGuard blocks direct navigation to non-"ready" routes in
  //      client-ready surface mode; the existing Playwright spec
  //      (e2e/fulfillment-operations.spec.ts) navigates straight to this route,
  //      so demoting it would render the lock screen and break that spec.
  // The cutover flag therefore governs only which section is PRIMARY on the
  // home page, not whether this dedicated route exists.
  "/operations/fulfillment": "ready",
};

// Derive route -> readiness from the canonical registry, then layer the
// dashboard-only overrides on top.
const NAV_ROUTE_READINESS: Record<string, FeatureReadiness> = (() => {
  const map: Record<string, FeatureReadiness> = {};
  for (const feature of Object.values(reg.features)) {
    for (const route of feature.routes) {
      map[route] = feature.state;
    }
  }
  return { ...map, ...ROUTE_READINESS_OVERRIDES };
})();

const PROVIDER_READINESS: Record<string, FeatureReadiness> = reg.providers;

export function isReadinessVisible(
  readiness: FeatureReadiness,
  options: VisibilityOptions = {},
): boolean {
  const mode = options.mode ?? getDashboardSurfaceMode();

  if (readiness === "blocked") {
    return false;
  }

  if (mode === "full") {
    return true;
  }

  return readiness === "ready";
}

export function getFeatureReadiness(featureId: string): FeatureReadiness {
  return reg.features[featureId]?.state ?? "verify";
}

// isFeatureVisible reports whether a canonical-registry feature is reachable under
// the active (or supplied) surface mode. Ready pages use this to avoid firing
// requests to gated (non-ready) endpoints, which return feature_not_available 404
// server-side in client-ready mode. Mirrors the server gate (IsFeatureEnabled).
export function isFeatureVisible(
  featureId: string,
  options: VisibilityOptions = {},
): boolean {
  return isReadinessVisible(getFeatureReadiness(featureId), options);
}

export function getRouteReadiness(pathname: string): FeatureReadiness {
  const normalized = normalizePathname(pathname);
  const match = Object.keys(NAV_ROUTE_READINESS)
    .filter((route) => routePatternMatches(route, normalized))
    .sort((a, b) => getRouteSpecificity(b) - getRouteSpecificity(a))[0];

  return match ? NAV_ROUTE_READINESS[match] : "verify";
}

export function isRouteAccessible(
  pathname: string,
  options: VisibilityOptions = {},
): boolean {
  return isReadinessVisible(getRouteReadiness(pathname), options);
}

export function getVisibleNavItems(
  items: NavItem[],
  options: NavVisibilityOptions,
): NavItem[] {
  return items.flatMap((item) => {
    if (item.hidden || (item.adminOnly && !options.isAdmin)) {
      return [];
    }

    if (!isRouteAccessible(item.href, options)) {
      return [];
    }

    const children = item.children
      ? getVisibleNavItems(item.children, options)
      : undefined;

    return [{ ...item, ...(children ? { children } : {}) }];
  });
}

export function getProviderReadiness(provider: string): FeatureReadiness {
  return PROVIDER_READINESS[provider] ?? "verify";
}

export function getVisibleProviderKeys(
  providers: string[],
  options: VisibilityOptions = {},
): string[] {
  return providers.filter((provider) =>
    isReadinessVisible(getProviderReadiness(provider), options),
  );
}

export function isShipmentProviderSelectable(
  provider: string,
  options: VisibilityOptions = {},
): boolean {
  if (provider === "manual") {
    return true;
  }

  return isReadinessVisible(getProviderReadiness(provider), options);
}

export function getSelectableShipmentProviders<T extends string>(
  providers: readonly T[],
  options: VisibilityOptions = {},
): T[] {
  return providers.filter((provider) =>
    isShipmentProviderSelectable(provider, options),
  );
}

export function getSelectableCarrierShipmentProviders<T extends string>(
  providers: readonly T[],
  options: VisibilityOptions = {},
): T[] {
  return getSelectableShipmentProviders(providers, options).filter(
    (provider) => provider !== "manual",
  );
}

export function getVisibleProvidersByCategory(
  category: ProviderInfo["category"],
  options: VisibilityOptions = {},
): ProviderInfo[] {
  return getProvidersByCategory(category).filter((provider) =>
    isReadinessVisible(getProviderReadiness(provider.key), options),
  );
}

function normalizePathname(pathname: string): string {
  if (pathname === "") return "/";
  const withoutQuery = pathname.split("?")[0]?.split("#")[0] ?? pathname;
  if (withoutQuery.length > 1 && withoutQuery.endsWith("/")) {
    return withoutQuery.slice(0, -1);
  }
  return withoutQuery;
}

function routePatternMatches(pattern: string, pathname: string): boolean {
  if (pattern === "/") return pathname === "/";

  const patternSegments = toSegments(pattern);
  const pathnameSegments = toSegments(pathname);

  if (patternSegments.length > pathnameSegments.length) {
    return false;
  }

  return patternSegments.every((segment, index) =>
    isDynamicSegment(segment) || segment === pathnameSegments[index],
  );
}

function getRouteSpecificity(pattern: string): number {
  return toSegments(pattern).reduce((score, segment) =>
    score + (isDynamicSegment(segment) ? 1 : 3),
  0);
}

function toSegments(pathname: string): string[] {
  return pathname.split("/").filter(Boolean);
}

function isDynamicSegment(segment: string): boolean {
  return segment.startsWith("[") && segment.endsWith("]");
}
