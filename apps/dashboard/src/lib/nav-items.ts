import {
  LayoutDashboard,
  ShoppingCart,
  Truck,
  RotateCcw,
  Package,
  Plug,
  Users,
  Building2,
  Mail,
  ListChecks,
  TextCursorInput,
  FolderTree,
  ScrollText,
  Webhook,
  Factory,
} from "lucide-react";

export interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
}

export const navItems: NavItem[] = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/orders", label: "Zamowienia", icon: ShoppingCart },
  { href: "/shipments", label: "Przesyłki", icon: Truck },
  { href: "/returns", label: "Zwroty", icon: RotateCcw },
  { href: "/products", label: "Produkty", icon: Package },
  { href: "/integrations", label: "Integracje", icon: Plug, adminOnly: true },
  { href: "/suppliers", label: "Dostawcy", icon: Factory, adminOnly: true },
  { href: "/audit", label: "Dziennik", icon: ScrollText, adminOnly: true },
  { href: "/settings/users", label: "Uzytkownicy", icon: Users, adminOnly: true },
  { href: "/settings/company", label: "Firma", icon: Building2, adminOnly: true },
  { href: "/settings/email", label: "Powiadomienia", icon: Mail, adminOnly: true },
  { href: "/settings/order-statuses", label: "Statusy", icon: ListChecks, adminOnly: true },
  { href: "/settings/custom-fields", label: "Pola", icon: TextCursorInput, adminOnly: true },
  { href: "/settings/product-categories", label: "Kategorie", icon: FolderTree, adminOnly: true },
  { href: "/settings/webhooks", label: "Webhooki", icon: Webhook, adminOnly: true },
];
