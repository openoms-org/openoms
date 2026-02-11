"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight, Home } from "lucide-react";

const segmentLabels: Record<string, string> = {
  orders: "Zamówienia",
  shipments: "Przesyłki",
  products: "Produkty",
  integrations: "Integracje",
  settings: "Ustawienia",
  users: "Użytkownicy",
  returns: "Zwroty",
  audit: "Dziennik aktywności",
  company: "Firma",
  email: "E-mail",
  "order-statuses": "Statusy",
  "custom-fields": "Pola",
  "product-categories": "Kategorie",
  webhooks: "Webhooki",
  allegro: "Allegro",
  listings: "Oferty marketplace",
  suppliers: "Dostawcy",
  deliveries: "Dostarczenia",
  new: "Nowe",
  import: "Import",
  "warehouse-documents": "Dokumenty magazynowe",
  "price-lists": "Cenniki",
  "sync-jobs": "Synchronizacja",
  "webhook-deliveries": "Dostarczenia webhooków",
  reports: "Raporty",
  packing: "Pakowanie",
  marketing: "Marketing",
  helpdesk: "Helpdesk",
  sms: "SMS",
  notifications: "Powiadomienia",
  currencies: "Waluty",
  "print-templates": "Szablony wydruków",
  categories: "Kategorie",
  automation: "Automatyzacja",
  variants: "Warianty",
  bundles: "Zestawy",
  "return-request": "Zgłoszenie zwrotu",
  customers: "Klienci",
  invoices: "Faktury",
  invoicing: "Fakturowanie",
  roles: "Role",
  warehouses: "Magazyny",
};

export function Breadcrumbs() {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  const formatSegment = (segment: string): string => {
    // If it looks like a UUID, show "Szczegóły" instead
    if (/^[0-9a-f]{8}-[0-9a-f]{4}-/.test(segment)) {
      return "Szczegóły";
    }
    return segmentLabels[segment] || segment;
  };

  const crumbs = segments.map((segment, index) => {
    const href = "/" + segments.slice(0, index + 1).join("/");
    const label = formatSegment(segment);
    const isLast = index === segments.length - 1;

    return { href, label, isLast };
  });

  return (
    <nav className="flex items-center gap-1 text-sm text-muted-foreground">
      <Link
        href="/"
        className="flex items-center hover:text-foreground transition-colors"
      >
        <Home className="h-4 w-4" />
      </Link>
      {crumbs.map((crumb) => (
        <span key={crumb.href} className="flex items-center gap-1">
          <ChevronRight className="h-3 w-3" />
          {crumb.isLast ? (
            <span className="text-foreground font-medium">{crumb.label}</span>
          ) : (
            <Link
              href={crumb.href}
              className="hover:text-foreground transition-colors"
            >
              {crumb.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
}
