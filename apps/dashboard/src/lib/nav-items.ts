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
  Mail,
  MessageSquare,
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
} from "lucide-react";

export interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
  group?: string;
}

export const navItems: NavItem[] = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/orders", label: "Zamówienia", icon: ShoppingCart, group: "Sprzedaż" },
  { href: "/shipments", label: "Przesyłki", icon: Truck, group: "Sprzedaż" },
  { href: "/returns", label: "Zwroty", icon: RotateCcw, group: "Sprzedaż" },
  { href: "/invoices", label: "Faktury", icon: FileText, group: "Sprzedaż" },
  { href: "/orders/import", label: "Import", icon: Upload, group: "Sprzedaż" },
  { href: "/customers", label: "Klienci", icon: Contact, group: "Sprzedaż" },
  { href: "/packing", label: "Pakowanie", icon: ScanBarcode, group: "Sprzedaż" },
  { href: "/reports", label: "Raporty", icon: BarChart3, group: "Sprzedaż" },
  { href: "/products", label: "Produkty", icon: Package, group: "Katalog" },
  { href: "/settings/warehouses", label: "Magazyny", icon: Warehouse, adminOnly: true, group: "Katalog" },
  { href: "/settings/product-categories", label: "Kategorie", icon: FolderTree, group: "Katalog" },
  { href: "/settings/price-lists", label: "Cenniki", icon: BadgePercent, adminOnly: true, group: "Katalog" },
  { href: "/integrations", label: "Integracje", icon: Plug, adminOnly: true, group: "Połączenia" },
  { href: "/suppliers", label: "Dostawcy", icon: Factory, adminOnly: true, group: "Połączenia" },
  { href: "/settings/webhooks", label: "Webhooki", icon: Webhook, adminOnly: true, group: "Połączenia" },
  { href: "/settings/sync-jobs", label: "Synchronizacja", icon: RefreshCw, adminOnly: true, group: "Połączenia" },
  { href: "/audit", label: "Dziennik", icon: ScrollText, adminOnly: true, group: "Administracja" },
  { href: "/settings/users", label: "Użytkownicy", icon: Users, adminOnly: true, group: "Administracja" },
  { href: "/settings/company", label: "Firma", icon: Building2, adminOnly: true, group: "Administracja" },
  { href: "/settings/email", label: "Powiadomienia", icon: Mail, adminOnly: true, group: "Administracja" },
  { href: "/settings/sms", label: "SMS", icon: MessageSquare, adminOnly: true, group: "Administracja" },
  { href: "/settings/order-statuses", label: "Statusy", icon: ListChecks, adminOnly: true, group: "Administracja" },
  { href: "/settings/custom-fields", label: "Pola", icon: TextCursorInput, adminOnly: true, group: "Administracja" },
  { href: "/settings/invoicing", label: "Fakturowanie", icon: Receipt, adminOnly: true, group: "Administracja" },
  { href: "/settings/automation", label: "Automatyzacja", icon: Zap, adminOnly: true, group: "Administracja" },
  { href: "/settings/print-templates", label: "Szablony", icon: Printer, adminOnly: true, group: "Administracja" },
];
