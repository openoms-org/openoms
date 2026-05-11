import type { NavItem } from "@/lib/nav-items";
import { getProvidersByCategory, type ProviderInfo } from "@/lib/provider-info";

export type FeatureReadiness = "ready" | "controlled" | "verify" | "beta" | "blocked";
export type DashboardSurfaceMode = "client-ready" | "full";

interface VisibilityOptions {
  mode?: DashboardSurfaceMode;
}

interface NavVisibilityOptions extends VisibilityOptions {
  isAdmin: boolean;
}

const DEFAULT_SURFACE_MODE: DashboardSurfaceMode = "client-ready";

export function getDashboardSurfaceMode(): DashboardSurfaceMode {
  return process.env.NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE === "full"
    ? "full"
    : DEFAULT_SURFACE_MODE;
}

const NAV_ROUTE_READINESS: Record<string, FeatureReadiness> = {
  "/": "ready",
  "/help": "ready",
  "/orders": "ready",
  "/orders/import": "verify",
  "/customers": "ready",
  "/customers/import": "verify",
  "/customers/segments": "beta",
  "/returns": "ready",
  "/invoices": "controlled",
  "/invoicing": "controlled",
  "/products": "ready",
  "/products/import": "verify",
  "/products/[id]/listings": "controlled",
  "/products/[id]/variants": "verify",
  "/settings/product-categories": "ready",
  "/settings/print-templates": "verify",
  "/shipments": "ready",
  "/carriers": "controlled",
  "/packing": "verify",
  "/pick-pack": "verify",
  "/settings/warehouses": "controlled",
  "/stocktakes": "verify",
  "/settings/warehouse-documents": "verify",
  "/stock-sync": "beta",
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
  "/settings/feeds": "beta",
  "/listing-sync": "beta",
  "/reports": "verify",
  "/reports/forecast": "beta",
  "/reports/carbon": "beta",
  "/reports/vat-oss": "beta",
  "/reconciliation": "beta",
  "/repricing": "beta",
  "/suppliers": "controlled",
  "/purchase-orders": "verify",
  "/dropship-orders": "beta",
  "/settings/automation": "controlled",
  "/workflows": "beta",
  "/tools/bg-removal": "beta",
  "/settings/marketing": "beta",
  "/settings/helpdesk": "beta",
  "/settings/currencies": "verify",
  "/recurring-orders": "beta",
  "/loyalty": "beta",
  "/settings/billing": "controlled",
  "/settings/company": "ready",
  "/settings/users": "ready",
  "/settings/roles": "ready",
  "/settings/security": "ready",
  "/settings/order-statuses": "controlled",
  "/settings/custom-fields": "controlled",
  "/settings/price-lists": "verify",
  "/settings/accounting": "controlled",
  "/settings/email": "controlled",
  "/settings/message-templates": "controlled",
  "/settings/ksef": "beta",
  "/settings/vat-oss": "beta",
  "/settings/inventory": "verify",
  "/settings/notifications": "blocked",
  "/settings/sms": "blocked",
  "/settings/webhooks": "controlled",
  "/settings/webhooks/deliveries": "controlled",
  "/settings/sync-jobs": "controlled",
  "/audit": "ready",
};

const PROVIDER_READINESS: Record<string, FeatureReadiness> = {
  allegro: "ready",
  olx: "controlled",
  amazon: "beta",
  ebay: "beta",
  woocommerce: "beta",
  erli: "beta",
  kaufland: "blocked",
  empik: "blocked",
  mirakl: "blocked",
  shopify: "blocked",
  prestashop: "blocked",
  shoper: "blocked",
  inpost: "ready",
  dhl: "controlled",
  dpd: "controlled",
  gls: "controlled",
  ups: "beta",
  fedex: "beta",
  poczta_polska: "beta",
  orlen_paczka: "beta",
  fakturownia: "controlled",
  wfirma: "beta",
  infakt: "beta",
  btp: "controlled",
};

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
