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
  { key: "allegro", name: "Allegro", description: "Zamówienia, oferty i podstawowa obsługa sprzedaży Allegro.", icon: Store, category: "marketplace" },
  { key: "amazon", name: "Amazon", description: "Synchronizacja z Amazon SP-API po certyfikacji konta seller.", icon: Store, category: "marketplace" },
  { key: "woocommerce", name: "WooCommerce", description: "Połączenie ze sklepem WooCommerce przez REST API.", icon: Store, category: "marketplace", beta: true },
  { key: "shopify", name: "Shopify", description: "Połączenie ze sklepem Shopify przez token administracyjny.", icon: Store, category: "marketplace", beta: true },
  { key: "prestashop", name: "PrestaShop", description: "Połączenie ze sklepem PrestaShop przez Webservice API.", icon: Store, category: "marketplace", beta: true },
  { key: "shoper", name: "Shoper", description: "Połączenie ze sklepem Shoper przez WebAPI.", icon: Store, category: "marketplace", beta: true },
  { key: "ebay", name: "eBay", description: "Synchronizacja konta eBay przez OAuth.", icon: Store, category: "marketplace", beta: true },
  { key: "kaufland", name: "Kaufland", description: "Integracja z Kaufland Marketplace API.", icon: Store, category: "marketplace", beta: true },
  { key: "olx", name: "OLX", description: "Ogłoszenia OLX i podstawowa synchronizacja ofert.", icon: Store, category: "marketplace" },
  { key: "erli", name: "Erli", description: "Synchronizacja ofert i zamówień Erli po teście sandbox/live.", icon: Store, category: "marketplace", beta: true },
  { key: "empik", name: "Empik (Mirakl)", description: "Integracja Mirakl/Empik wymaga potwierdzenia nazewnictwa providera.", icon: Store, category: "marketplace", beta: true },

  // ── Carrier ──
  { key: "inpost", name: "InPost", description: "Etykiety, śledzenie i punkty odbioru InPost.", icon: Truck, category: "carrier" },
  { key: "dhl", name: "DHL", description: "Etykiety DHL po weryfikacji konta i numeru płatnika.", icon: Truck, category: "carrier" },
  { key: "dpd", name: "DPD", description: "Etykiety DPD; automatyczny tracking wymaga osobnej ścieżki.", icon: Truck, category: "carrier" },
  { key: "gls", name: "GLS", description: "Etykiety i tracking GLS po weryfikacji API key.", icon: Truck, category: "carrier" },
  { key: "ups", name: "UPS", description: "Integracja UPS wymaga certyfikacji konta developerskiego.", icon: Truck, category: "carrier", beta: true },
  { key: "fedex", name: "FedEx", description: "Integracja FedEx wymaga certyfikacji konta developerskiego.", icon: Truck, category: "carrier", beta: true },
  { key: "poczta_polska", name: "Poczta Polska", description: "Integracja Poczty Polskiej wymaga konta partnerskiego.", icon: Truck, category: "carrier", beta: true },
  { key: "orlen_paczka", name: "Orlen Paczka", description: "Integracja Orlen Paczka wymaga konta partnerskiego.", icon: Truck, category: "carrier", beta: true },

  // ── Invoicing ──
  { key: "fakturownia", name: "Fakturownia", description: "Faktury i PDF przez API Fakturowni.", icon: Receipt, category: "invoicing" },
  { key: "wfirma", name: "wFirma", description: "Faktury przez API wFirma po osobnej walidacji.", icon: Receipt, category: "invoicing", beta: true },
  { key: "infakt", name: "inFakt", description: "Faktury przez API inFakt po osobnej walidacji.", icon: Receipt, category: "invoicing", beta: true },

  // ── Supplier ──
  { key: "btp", name: "BTP.pro", description: "Import XML oraz stany i ceny przez API BTP.", icon: Factory, category: "supplier" },
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
