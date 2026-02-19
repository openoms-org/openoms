import {
  LayoutDashboard,
  ShoppingCart,
  Truck,
  RotateCcw,
  FileText,
  Receipt,
  CreditCard,
  Package,
  Users,
  Building2,
  Bell,
  ListChecks,
  TextCursorInput,
  FolderTree,
  ScrollText,
  Webhook,
  Factory,
  Zap,
  Upload,
  BarChart3,
  RefreshCw,
  Warehouse,
  Contact,
  Printer,
  ScanBarcode,
  BadgePercent,
  ClipboardList,
  ClipboardCheck,
  Coins,
  Shield,
  Send,
  Headphones,
  ShieldCheck,
  Store,
  MessageSquare,
  Calculator,
  Rss,
  PackageCheck,
  Leaf,
  Globe,
  Repeat,
  TrendingUp,
  Eraser,
  Award,
  Workflow,
  ArrowUpDown,
  HelpCircle,
  Settings,
  Wrench,
} from "lucide-react";

export interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
  group?: string;
  children?: NavItem[];
}

export interface NavGroup {
  key: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  defaultExpanded?: boolean;
}

export const navGroups: NavGroup[] = [
  { key: "Sprzedaż", label: "Sprzedaż", icon: ShoppingCart, defaultExpanded: true },
  { key: "Katalog", label: "Katalog", icon: Package },
  { key: "Logistyka", label: "Logistyka", icon: Truck },
  { key: "Kanały sprzedaży", label: "Kanały sprzedaży", icon: Store },
  { key: "Kurierzy", label: "Kurierzy", icon: Truck },
  { key: "Fakturowanie", label: "Fakturowanie", icon: Receipt },
  { key: "Raporty", label: "Raporty", icon: BarChart3 },
  { key: "Zaopatrzenie", label: "Zaopatrzenie", icon: Factory },
  { key: "Narzędzia", label: "Narzędzia", icon: Wrench },
  { key: "Ustawienia", label: "Ustawienia", icon: Settings },
  { key: "Pomoc", label: "Pomoc", icon: HelpCircle },
];

/** Flatten nav items including children — for command palette search */
export function flattenNavItems(items: NavItem[]): NavItem[] {
  return items.flatMap((item) => [
    item,
    ...(item.children?.map((child) => ({
      ...child,
      group: child.group || item.label,
      adminOnly: child.adminOnly ?? item.adminOnly,
    })) || []),
  ]);
}

export const navItems: NavItem[] = [
  { href: "/", label: "Pulpit", icon: LayoutDashboard },

  // ── Sprzedaż ──
  { href: "/orders", label: "Zamówienia", icon: ShoppingCart, group: "Sprzedaż" },
  { href: "/customers", label: "Klienci", icon: Contact, group: "Sprzedaż" },
  { href: "/returns", label: "Zwroty", icon: RotateCcw, group: "Sprzedaż" },
  { href: "/invoices", label: "Faktury", icon: FileText, group: "Sprzedaż" },

  // ── Katalog ──
  { href: "/products", label: "Produkty", icon: Package, group: "Katalog" },
  { href: "/settings/product-categories", label: "Kategorie", icon: FolderTree, group: "Katalog" },
  { href: "/settings/print-templates", label: "Szablony druku", icon: Printer, group: "Katalog" },

  // ── Logistyka ──
  { href: "/shipments", label: "Przesyłki", icon: Truck, group: "Logistyka" },
  { href: "/packing", label: "Pakowanie", icon: ScanBarcode, group: "Logistyka" },
  { href: "/pick-pack", label: "Pick & Pack", icon: PackageCheck, group: "Logistyka" },
  { href: "/settings/warehouses", label: "Magazyny", icon: Warehouse, adminOnly: true, group: "Logistyka" },
  { href: "/stocktakes", label: "Inwentaryzacja", icon: ClipboardCheck, adminOnly: true, group: "Logistyka" },
  { href: "/settings/warehouse-documents", label: "Dokumenty magazynowe", icon: ClipboardList, adminOnly: true, group: "Logistyka" },
  { href: "/stock-sync", label: "Sync magazynu", icon: RefreshCw, adminOnly: true, group: "Logistyka" },

  // ── Kanały sprzedaży ──
  { href: "/marketplaces", label: "Marketplace", icon: Store, adminOnly: true, group: "Kanały sprzedaży" },
  { href: "/settings/feeds", label: "Feedy produktowe", icon: Rss, adminOnly: true, group: "Kanały sprzedaży" },
  { href: "/listing-sync", label: "Synchronizacja ofert", icon: ArrowUpDown, adminOnly: true, group: "Kanały sprzedaży" },

  // ── Kurierzy ──
  { href: "/carriers", label: "Kurierzy", icon: Truck, adminOnly: true, group: "Kurierzy" },

  // ── Fakturowanie ──
  { href: "/invoicing", label: "Fakturowanie", icon: Receipt, adminOnly: true, group: "Fakturowanie" },

  // ── Raporty ──
  { href: "/reports", label: "Raporty", icon: BarChart3, adminOnly: true, group: "Raporty" },
  { href: "/reports/forecast", label: "Prognoza popytu", icon: TrendingUp, adminOnly: true, group: "Raporty" },
  { href: "/reports/carbon", label: "Ślad węglowy", icon: Leaf, adminOnly: true, group: "Raporty" },
  { href: "/reports/vat-oss", label: "Raport VAT OSS", icon: Globe, adminOnly: true, group: "Raporty" },
  { href: "/reconciliation", label: "Rozliczenia", icon: CreditCard, adminOnly: true, group: "Raporty" },
  { href: "/repricing", label: "Repricing", icon: TrendingUp, adminOnly: true, group: "Raporty" },

  // ── Zaopatrzenie ──
  { href: "/suppliers", label: "Dostawcy", icon: Factory, adminOnly: true, group: "Zaopatrzenie" },
  { href: "/purchase-orders", label: "Zamówienia zakupu", icon: ClipboardList, adminOnly: true, group: "Zaopatrzenie" },
  { href: "/dropship-orders", label: "Dropshipping", icon: Truck, adminOnly: true, group: "Zaopatrzenie" },

  // ── Narzędzia ──
  { href: "/settings/automation", label: "Automatyzacja", icon: Zap, adminOnly: true, group: "Narzędzia" },
  { href: "/workflows", label: "Workflow Builder", icon: Workflow, adminOnly: true, group: "Narzędzia" },
  { href: "/orders/import", label: "Import zamówień", icon: Upload, group: "Narzędzia" },
  { href: "/products/import", label: "Import produktów", icon: Upload, group: "Narzędzia" },
  { href: "/tools/bg-removal", label: "Usuwanie tła", icon: Eraser, group: "Narzędzia" },
  { href: "/settings/marketing", label: "Marketing", icon: Send, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/helpdesk", label: "Helpdesk", icon: Headphones, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/currencies", label: "Waluty", icon: Coins, adminOnly: true, group: "Narzędzia" },
  { href: "/recurring-orders", label: "Subskrypcje", icon: Repeat, group: "Narzędzia" },
  { href: "/loyalty", label: "Program lojalnościowy", icon: Award, group: "Narzędzia" },
  { href: "/customers/segments", label: "Segmenty klientów", icon: Users, group: "Narzędzia" },

  // ── Ustawienia ──
  { href: "/settings/company", label: "Firma", icon: Building2, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/users", label: "Użytkownicy", icon: Users, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/roles", label: "Role", icon: Shield, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/security", label: "Bezpieczeństwo", icon: ShieldCheck, group: "Ustawienia" },
  { href: "/settings/order-statuses", label: "Statusy zamówień", icon: ListChecks, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/custom-fields", label: "Pola niestandardowe", icon: TextCursorInput, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/price-lists", label: "Cenniki", icon: BadgePercent, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/accounting", label: "Księgowość", icon: Calculator, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/ksef", label: "KSeF", icon: FileText, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/vat-oss", label: "VAT OSS", icon: Globe, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/inventory", label: "Kontrola magazynowa", icon: ShieldCheck, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/notifications", label: "Powiadomienia", icon: Bell, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/sms", label: "SMS", icon: MessageSquare, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/webhooks", label: "Webhooki", icon: Webhook, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/webhooks/deliveries", label: "Dostawy webhooków", icon: Webhook, adminOnly: true, group: "Ustawienia" },
  { href: "/settings/sync-jobs", label: "Synchronizacja", icon: RefreshCw, adminOnly: true, group: "Ustawienia" },
  { href: "/audit", label: "Dziennik aktywności", icon: ScrollText, adminOnly: true, group: "Ustawienia" },

  // ── Pomoc ──
  { href: "/help", label: "Pomoc", icon: HelpCircle, group: "Pomoc" },
];
