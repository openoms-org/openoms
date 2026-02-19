import {
  Package,
  Truck,
  Receipt,
  Factory,
  Store,
} from "lucide-react";

export interface ProviderInfo {
  key: string;
  name: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  category: "marketplace" | "carrier" | "invoicing" | "supplier";
  /** Providers still in development / not fully verified */
  beta?: boolean;
}

export const PROVIDERS: ProviderInfo[] = [
  // ── Marketplace ──
  { key: "allegro", name: "Allegro", description: "Największa platforma e-commerce w Polsce", icon: Store, category: "marketplace" },
  { key: "amazon", name: "Amazon", description: "Globalny marketplace", icon: Store, category: "marketplace" },
  { key: "woocommerce", name: "WooCommerce", description: "Platforma e-commerce na WordPress", icon: Store, category: "marketplace", beta: true },
  { key: "shopify", name: "Shopify", description: "Platforma e-commerce SaaS", icon: Store, category: "marketplace", beta: true },
  { key: "prestashop", name: "PrestaShop", description: "Open-source e-commerce", icon: Store, category: "marketplace", beta: true },
  { key: "shoper", name: "Shoper", description: "Polska platforma sklepowa", icon: Store, category: "marketplace", beta: true },
  { key: "ebay", name: "eBay", description: "Globalny marketplace aukcyjny", icon: Store, category: "marketplace", beta: true },
  { key: "kaufland", name: "Kaufland", description: "Marketplace Kaufland (PL, DE, CZ)", icon: Store, category: "marketplace", beta: true },
  { key: "olx", name: "OLX", description: "Polska platforma ogłoszeniowa", icon: Store, category: "marketplace", beta: true },
  { key: "erli", name: "Erli", description: "Polski marketplace", icon: Store, category: "marketplace", beta: true },
  { key: "empik", name: "Empik (Mirakl)", description: "Empik Marketplace na platformie Mirakl", icon: Store, category: "marketplace", beta: true },

  // ── Carrier ──
  { key: "inpost", name: "InPost", description: "Paczkomaty i kurier InPost", icon: Truck, category: "carrier" },
  { key: "dhl", name: "DHL", description: "Kurier DHL Express i Parcel", icon: Truck, category: "carrier" },
  { key: "dpd", name: "DPD", description: "Kurier DPD Polska", icon: Truck, category: "carrier" },
  { key: "gls", name: "GLS", description: "Kurier GLS Poland", icon: Truck, category: "carrier" },
  { key: "ups", name: "UPS", description: "Kurier UPS Polska", icon: Truck, category: "carrier", beta: true },
  { key: "fedex", name: "FedEx", description: "Kurier FedEx Express", icon: Truck, category: "carrier", beta: true },
  { key: "poczta_polska", name: "Poczta Polska", description: "Elektroniczny Nadawca", icon: Truck, category: "carrier", beta: true },
  { key: "orlen_paczka", name: "Orlen Paczka", description: "Automaty paczkowe Orlen", icon: Truck, category: "carrier", beta: true },

  // ── Invoicing ──
  { key: "fakturownia", name: "Fakturownia", description: "System fakturowania online", icon: Receipt, category: "invoicing" },
  { key: "wfirma", name: "wFirma", description: "Księgowość i fakturowanie online", icon: Receipt, category: "invoicing", beta: true },
  { key: "infakt", name: "inFakt", description: "Fakturowanie i księgowość", icon: Receipt, category: "invoicing", beta: true },

  // ── Supplier ──
  { key: "btp", name: "BTP.pro", description: "Hurtownia BTP — integracja API", icon: Factory, category: "supplier" },
];

export function getProviderInfo(key: string): ProviderInfo | undefined {
  return PROVIDERS.find((p) => p.key === key);
}

export function getProvidersByCategory(category: ProviderInfo["category"]): ProviderInfo[] {
  return PROVIDERS.filter((p) => p.category === category);
}

export function getProviderDisplayName(key: string): string {
  return getProviderInfo(key)?.name ?? key.charAt(0).toUpperCase() + key.slice(1);
}
