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
  { key: "allegro", name: "Allegro", description: "providers.allegro.description", icon: Store, category: "marketplace" },
  { key: "amazon", name: "Amazon", description: "providers.amazon.description", icon: Store, category: "marketplace" },
  { key: "woocommerce", name: "WooCommerce", description: "providers.woocommerce.description", icon: Store, category: "marketplace", beta: true },
  { key: "shopify", name: "Shopify", description: "providers.shopify.description", icon: Store, category: "marketplace", beta: true },
  { key: "prestashop", name: "PrestaShop", description: "providers.prestashop.description", icon: Store, category: "marketplace", beta: true },
  { key: "shoper", name: "Shoper", description: "providers.shoper.description", icon: Store, category: "marketplace", beta: true },
  { key: "ebay", name: "eBay", description: "providers.ebay.description", icon: Store, category: "marketplace", beta: true },
  { key: "kaufland", name: "Kaufland", description: "providers.kaufland.description", icon: Store, category: "marketplace", beta: true },
  { key: "olx", name: "OLX", description: "providers.olx.description", icon: Store, category: "marketplace" },
  { key: "erli", name: "Erli", description: "providers.erli.description", icon: Store, category: "marketplace", beta: true },
  { key: "empik", name: "Empik (Mirakl)", description: "providers.empik.description", icon: Store, category: "marketplace", beta: true },

  // ── Carrier ──
  { key: "inpost", name: "InPost", description: "providers.inpost.description", icon: Truck, category: "carrier" },
  { key: "dhl", name: "DHL", description: "providers.dhl.description", icon: Truck, category: "carrier" },
  { key: "dpd", name: "DPD", description: "providers.dpd.description", icon: Truck, category: "carrier" },
  { key: "gls", name: "GLS", description: "providers.gls.description", icon: Truck, category: "carrier" },
  { key: "ups", name: "UPS", description: "providers.ups.description", icon: Truck, category: "carrier", beta: true },
  { key: "fedex", name: "FedEx", description: "providers.fedex.description", icon: Truck, category: "carrier", beta: true },
  { key: "poczta_polska", name: "Poczta Polska", description: "providers.poczta_polska.description", icon: Truck, category: "carrier", beta: true },
  { key: "orlen_paczka", name: "Orlen Paczka", description: "providers.orlen_paczka.description", icon: Truck, category: "carrier", beta: true },

  // ── Invoicing ──
  { key: "fakturownia", name: "Fakturownia", description: "providers.fakturownia.description", icon: Receipt, category: "invoicing" },
  { key: "wfirma", name: "wFirma", description: "providers.wfirma.description", icon: Receipt, category: "invoicing", beta: true },
  { key: "infakt", name: "inFakt", description: "providers.infakt.description", icon: Receipt, category: "invoicing", beta: true },

  // ── Supplier ──
  { key: "btp", name: "BTP.pro", description: "providers.btp.description", icon: Factory, category: "supplier" },
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
