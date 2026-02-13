import {
  LayoutDashboard,
  ShoppingCart,
  Truck,
  RotateCcw,
  FileText,
  Receipt,
  Package,
  Plug,
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
  Tag,
  MessageSquare,
  Megaphone,
  Layers,
  Star,
  AlertTriangle,
  Scale,
  Calculator,
  Ruler,
} from "lucide-react";

export interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
  group?: string;
  children?: NavItem[];
}

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
  // Sprzedaż
  { href: "/orders", label: "Zamówienia", icon: ShoppingCart, group: "Sprzedaż" },
  { href: "/shipments", label: "Przesyłki", icon: Truck, group: "Sprzedaż" },
  { href: "/returns", label: "Zwroty", icon: RotateCcw, group: "Sprzedaż" },
  { href: "/invoices", label: "Faktury", icon: FileText, group: "Sprzedaż" },
  { href: "/orders/import", label: "Import", icon: Upload, group: "Sprzedaż" },
  { href: "/customers", label: "Klienci", icon: Contact, group: "Sprzedaż" },
  { href: "/packing", label: "Pakowanie", icon: ScanBarcode, group: "Sprzedaż" },
  { href: "/reports", label: "Raporty", icon: BarChart3, adminOnly: true, group: "Sprzedaż" },
  // Katalog
  { href: "/products", label: "Produkty", icon: Package, group: "Katalog" },
  { href: "/products/import", label: "Import produktów", icon: Upload, group: "Katalog" },
  { href: "/settings/product-categories", label: "Kategorie", icon: FolderTree, group: "Katalog" },
  { href: "/settings/print-templates", label: "Szablony druku", icon: Printer, group: "Katalog" },
  // Kanały sprzedaży — marketplace integrations with sub-pages
  {
    href: "/integrations/allegro",
    label: "Allegro",
    icon: Store,
    adminOnly: true,
    group: "Kanały sprzedaży",
    children: [
      { href: "/integrations/allegro/offers", label: "Oferty", icon: Tag },
      { href: "/integrations/allegro/catalog", label: "Katalog", icon: Layers },
      { href: "/integrations/allegro/promotions", label: "Promocje", icon: Megaphone },
      { href: "/integrations/allegro/messages", label: "Wiadomości", icon: MessageSquare },
      { href: "/integrations/allegro/returns", label: "Zwroty", icon: RotateCcw },
      { href: "/integrations/allegro/disputes", label: "Spory", icon: AlertTriangle },
      { href: "/integrations/allegro/delivery", label: "Dostawa", icon: Truck },
      { href: "/integrations/allegro/policies", label: "Polityki", icon: Scale },
      { href: "/integrations/allegro/finance", label: "Finanse", icon: Calculator },
      { href: "/integrations/allegro/ratings", label: "Oceny", icon: Star },
    ],
  },
  { href: "/integrations/amazon", label: "Amazon", icon: Store, adminOnly: true, group: "Kanały sprzedaży" },
  // Ogólne
  { href: "/settings/security", label: "Bezpieczeństwo", icon: ShieldCheck, group: "Ogólne" },
  { href: "/settings/company", label: "Firma", icon: Building2, adminOnly: true, group: "Ogólne" },
  { href: "/settings/users", label: "Użytkownicy", icon: Users, adminOnly: true, group: "Ogólne" },
  { href: "/settings/roles", label: "Role", icon: Shield, adminOnly: true, group: "Ogólne" },
  // Sprzedaż - ustawienia
  { href: "/settings/order-statuses", label: "Statusy zamówień", icon: ListChecks, adminOnly: true, group: "Sprzedaż - ustawienia" },
  { href: "/settings/custom-fields", label: "Pola niestandardowe", icon: TextCursorInput, adminOnly: true, group: "Sprzedaż - ustawienia" },
  { href: "/settings/price-lists", label: "Cenniki", icon: BadgePercent, adminOnly: true, group: "Sprzedaż - ustawienia" },
  { href: "/settings/invoicing", label: "Fakturowanie", icon: Receipt, adminOnly: true, group: "Sprzedaż - ustawienia" },
  { href: "/settings/ksef", label: "KSeF", icon: FileText, adminOnly: true, group: "Sprzedaż - ustawienia" },
  // Powiadomienia
  { href: "/settings/notifications", label: "Powiadomienia", icon: Bell, adminOnly: true, group: "Powiadomienia" },
  { href: "/settings/webhooks", label: "Webhooki", icon: Webhook, adminOnly: true, group: "Powiadomienia" },
  // Magazyn
  { href: "/settings/inventory", label: "Kontrola magazynowa", icon: ShieldCheck, adminOnly: true, group: "Magazyn" },
  { href: "/settings/warehouses", label: "Magazyny", icon: Warehouse, adminOnly: true, group: "Magazyn" },
  { href: "/settings/warehouse-documents", label: "Dokumenty magazynowe", icon: ClipboardList, adminOnly: true, group: "Magazyn" },
  { href: "/stocktakes", label: "Inwentaryzacja", icon: ClipboardCheck, adminOnly: true, group: "Magazyn" },
  // Narzędzia
  { href: "/integrations", label: "Połączenia", icon: Plug, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/automation", label: "Automatyzacja", icon: Zap, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/currencies", label: "Waluty", icon: Coins, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/marketing", label: "Marketing", icon: Send, adminOnly: true, group: "Narzędzia" },
  { href: "/settings/helpdesk", label: "Helpdesk", icon: Headphones, adminOnly: true, group: "Narzędzia" },
  { href: "/suppliers", label: "Dostawcy", icon: Factory, adminOnly: true, group: "Narzędzia" },
  // Monitoring
  { href: "/settings/sync-jobs", label: "Synchronizacja", icon: RefreshCw, adminOnly: true, group: "Monitoring" },
  { href: "/settings/webhooks/deliveries", label: "Dostawy webhooków", icon: Webhook, adminOnly: true, group: "Monitoring" },
  { href: "/audit", label: "Dziennik aktywności", icon: ScrollText, adminOnly: true, group: "Monitoring" },
];
