import type { ComponentType } from "react";
import {
  Truck,
  Receipt,
  Factory,
  Store,
} from "lucide-react";

export interface ProviderBrand {
  wordmark: string;
  shortMark: string;
  className: string;
}

export interface ProviderInfo {
  key: string;
  name: string;
  description: string;
  icon: ComponentType<{ className?: string }>;
  category: "marketplace" | "carrier" | "invoicing" | "supplier";
  brand?: ProviderBrand;
  /** Providers still in development / not fully verified */
  beta?: boolean;
}

const brand = (
  wordmark: string,
  shortMark: string,
  className: string,
): ProviderBrand => ({
  wordmark,
  shortMark,
  className,
});

export const PROVIDERS: ProviderInfo[] = [
  // ── Marketplace ──
  { key: "allegro", name: "Allegro", description: "Zamówienia, oferty i podstawowa obsługa sprzedaży Allegro.", icon: Store, category: "marketplace", brand: brand("Allegro", "Al", "border-orange-200 bg-orange-50 text-orange-700") },
  { key: "amazon", name: "Amazon", description: "Synchronizacja z Amazon SP-API po certyfikacji konta seller.", icon: Store, category: "marketplace", brand: brand("Amazon", "Am", "border-slate-300 bg-slate-950 text-amber-300") },
  { key: "woocommerce", name: "WooCommerce", description: "Połączenie ze sklepem WooCommerce przez REST API.", icon: Store, category: "marketplace", brand: brand("Woo", "Woo", "border-violet-200 bg-violet-50 text-violet-700"), beta: true },
  { key: "shopify", name: "Shopify", description: "Połączenie ze sklepem Shopify przez token administracyjny.", icon: Store, category: "marketplace", brand: brand("Shopify", "Sh", "border-emerald-200 bg-emerald-50 text-emerald-700"), beta: true },
  { key: "prestashop", name: "PrestaShop", description: "Połączenie ze sklepem PrestaShop przez Webservice API.", icon: Store, category: "marketplace", brand: brand("Presta", "Ps", "border-pink-200 bg-pink-50 text-pink-700"), beta: true },
  { key: "shoper", name: "Shoper", description: "Połączenie ze sklepem Shoper przez WebAPI.", icon: Store, category: "marketplace", brand: brand("Shoper", "Sp", "border-sky-200 bg-sky-50 text-sky-700"), beta: true },
  { key: "ebay", name: "eBay", description: "Synchronizacja konta eBay przez OAuth.", icon: Store, category: "marketplace", brand: brand("eBay", "eB", "border-lime-200 bg-lime-50 text-lime-700"), beta: true },
  { key: "kaufland", name: "Kaufland", description: "Integracja z Kaufland Marketplace API.", icon: Store, category: "marketplace", brand: brand("Kaufland", "K", "border-red-200 bg-red-50 text-red-700"), beta: true },
  { key: "olx", name: "OLX", description: "Ogłoszenia OLX i podstawowa synchronizacja ofert.", icon: Store, category: "marketplace", brand: brand("OLX", "OLX", "border-cyan-200 bg-cyan-50 text-cyan-700") },
  { key: "erli", name: "Erli", description: "Synchronizacja ofert i zamówień Erli po teście sandbox/live.", icon: Store, category: "marketplace", brand: brand("Erli", "Er", "border-fuchsia-200 bg-fuchsia-50 text-fuchsia-700"), beta: true },
  { key: "empik", name: "Empik (Mirakl)", description: "Integracja Mirakl/Empik wymaga potwierdzenia nazewnictwa providera.", icon: Store, category: "marketplace", brand: brand("Empik", "Em", "border-red-200 bg-red-50 text-red-700"), beta: true },

  // ── Carrier ──
  { key: "inpost", name: "InPost", description: "Etykiety, śledzenie i punkty odbioru InPost.", icon: Truck, category: "carrier", brand: brand("InPost", "IP", "border-yellow-300 bg-yellow-100 text-neutral-950") },
  { key: "dhl", name: "DHL", description: "Etykiety DHL po weryfikacji konta i numeru płatnika.", icon: Truck, category: "carrier", brand: brand("DHL", "DHL", "border-red-200 bg-yellow-100 text-red-700") },
  { key: "dpd", name: "DPD", description: "Etykiety DPD; automatyczny tracking wymaga osobnej ścieżki.", icon: Truck, category: "carrier", brand: brand("DPD", "DPD", "border-red-200 bg-red-50 text-red-700") },
  { key: "gls", name: "GLS", description: "Etykiety i tracking GLS po weryfikacji API key.", icon: Truck, category: "carrier", brand: brand("GLS", "GLS", "border-blue-200 bg-yellow-50 text-blue-700") },
  { key: "ups", name: "UPS", description: "Integracja UPS wymaga certyfikacji konta developerskiego.", icon: Truck, category: "carrier", brand: brand("UPS", "UPS", "border-amber-300 bg-stone-100 text-stone-800"), beta: true },
  { key: "fedex", name: "FedEx", description: "Integracja FedEx wymaga certyfikacji konta developerskiego.", icon: Truck, category: "carrier", brand: brand("FedEx", "Fx", "border-purple-200 bg-purple-50 text-purple-700"), beta: true },
  { key: "poczta_polska", name: "Poczta Polska", description: "Integracja Poczty Polskiej wymaga konta partnerskiego.", icon: Truck, category: "carrier", brand: brand("Poczta", "PP", "border-red-200 bg-red-50 text-red-700"), beta: true },
  { key: "orlen_paczka", name: "Orlen Paczka", description: "Integracja Orlen Paczka wymaga konta partnerskiego.", icon: Truck, category: "carrier", brand: brand("Orlen", "OP", "border-red-200 bg-red-50 text-red-700"), beta: true },

  // ── Invoicing ──
  { key: "fakturownia", name: "Fakturownia", description: "Faktury i PDF przez API Fakturowni.", icon: Receipt, category: "invoicing", brand: brand("Fakturownia", "F", "border-blue-200 bg-blue-50 text-blue-700") },
  { key: "wfirma", name: "wFirma", description: "Faktury przez API wFirma po osobnej walidacji.", icon: Receipt, category: "invoicing", brand: brand("wFirma", "wF", "border-indigo-200 bg-indigo-50 text-indigo-700"), beta: true },
  { key: "infakt", name: "inFakt", description: "Faktury przez API inFakt po osobnej walidacji.", icon: Receipt, category: "invoicing", brand: brand("inFakt", "iF", "border-emerald-200 bg-emerald-50 text-emerald-700"), beta: true },

  // ── Supplier ──
  { key: "btp", name: "BTP.pro", description: "Import XML oraz stany i ceny przez API BTP.", icon: Factory, category: "supplier", brand: brand("BTP.pro", "BTP", "border-teal-200 bg-teal-50 text-teal-700") },
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
