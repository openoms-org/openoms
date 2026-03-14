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
  UsersRound,
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
  FileUp,
  BarChart3,
  RefreshCw,
  RotateCw,
  Warehouse,
  Contact,
  Printer,
  ScanBarcode,
  BadgePercent,
  ClipboardList,
  ListOrdered,
  ClipboardCheck,
  Coins,
  Shield,
  Send,
  Headphones,
  ShieldCheck,
  PackageSearch,
  Store,
  MessageSquare,
  Calculator,
  Rss,
  PackageCheck,
  Leaf,
  Globe,
  Landmark,
  Repeat,
  TrendingUp,
  Tag,
  Eraser,
  Award,
  Workflow,
  ArrowUpDown,
  HelpCircle,
  Settings,
  Wrench,
  Route,
  Share2,
  FileCheck,
  MailCheck,
} from "lucide-react";

export interface NavItem {
  href: string;
  /** Translation key under "navigation" namespace */
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
  /** Group key matching NavGroup.key — used for grouping and translated via "navigation.groups.*" */
  group?: string;
  children?: NavItem[];
}

export interface NavGroup {
  /** Stable key used for matching items and localStorage persistence */
  key: string;
  /** Translation key under "navigation.groups" namespace */
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  defaultExpanded?: boolean;
}

export const navGroups: NavGroup[] = [
  { key: "sales", label: "sales", icon: ShoppingCart, defaultExpanded: true },
  { key: "catalog", label: "catalog", icon: Package },
  { key: "logistics", label: "logistics", icon: Truck },
  { key: "salesChannels", label: "salesChannels", icon: Store },
  { key: "reports", label: "reports", icon: BarChart3 },
  { key: "procurement", label: "procurement", icon: Factory },
  { key: "tools", label: "tools", icon: Wrench },
  { key: "settings", label: "settings", icon: Settings },
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
  { href: "/", label: "dashboard", icon: LayoutDashboard },

  // ── Sales ──
  { href: "/orders", label: "orders", icon: ShoppingCart, group: "sales" },
  { href: "/customers", label: "customers", icon: Contact, group: "sales" },
  { href: "/returns", label: "returns", icon: RotateCcw, group: "sales" },
  { href: "/invoices", label: "invoices", icon: FileText, group: "sales" },
  { href: "/invoicing", label: "invoicing", icon: Receipt, adminOnly: true, group: "sales" },

  // ── Catalog ──
  { href: "/products", label: "products", icon: Package, group: "catalog" },
  { href: "/settings/product-categories", label: "categories", icon: FolderTree, group: "catalog" },
  { href: "/settings/print-templates", label: "printTemplates", icon: Printer, group: "catalog" },

  // ── Logistics ──
  { href: "/shipments", label: "shipments", icon: Truck, group: "logistics" },
  { href: "/carriers", label: "carriers", icon: Route, adminOnly: true, group: "logistics" },
  { href: "/packing", label: "packing", icon: ScanBarcode, group: "logistics" },
  { href: "/pick-pack", label: "pickPack", icon: PackageCheck, group: "logistics" },
  { href: "/settings/warehouses", label: "warehouses", icon: Warehouse, adminOnly: true, group: "logistics" },
  { href: "/stocktakes", label: "stocktakes", icon: ClipboardCheck, adminOnly: true, group: "logistics" },
  { href: "/settings/warehouse-documents", label: "warehouseDocuments", icon: ClipboardList, adminOnly: true, group: "logistics" },
  { href: "/stock-sync", label: "stockSync", icon: RefreshCw, adminOnly: true, group: "logistics" },

  // ── Sales channels ──
  { href: "/marketplaces", label: "marketplace", icon: Store, adminOnly: true, group: "salesChannels" },
  { href: "/settings/feeds", label: "productFeeds", icon: Rss, adminOnly: true, group: "salesChannels" },
  { href: "/listing-sync", label: "listingSync", icon: ArrowUpDown, adminOnly: true, group: "salesChannels" },

  // ── Reports ──
  { href: "/reports", label: "reports", icon: BarChart3, adminOnly: true, group: "reports" },
  { href: "/reports/forecast", label: "demandForecast", icon: TrendingUp, adminOnly: true, group: "reports" },
  { href: "/reports/carbon", label: "carbonFootprint", icon: Leaf, adminOnly: true, group: "reports" },
  { href: "/reports/vat-oss", label: "vatOssReport", icon: Globe, adminOnly: true, group: "reports" },
  { href: "/reconciliation", label: "reconciliation", icon: CreditCard, adminOnly: true, group: "reports" },
  { href: "/repricing", label: "repricing", icon: Tag, adminOnly: true, group: "reports" },

  // ── Procurement ──
  { href: "/suppliers", label: "suppliers", icon: Factory, adminOnly: true, group: "procurement" },
  { href: "/purchase-orders", label: "purchaseOrders", icon: ListOrdered, adminOnly: true, group: "procurement" },
  { href: "/dropship-orders", label: "dropshipping", icon: Share2, adminOnly: true, group: "procurement" },

  // ── Tools ──
  { href: "/settings/automation", label: "automation", icon: Zap, adminOnly: true, group: "tools" },
  { href: "/workflows", label: "workflowBuilder", icon: Workflow, adminOnly: true, group: "tools" },
  { href: "/orders/import", label: "orderImport", icon: Upload, group: "tools" },
  { href: "/products/import", label: "productImport", icon: FileUp, group: "tools" },
  { href: "/tools/bg-removal", label: "bgRemoval", icon: Eraser, group: "tools" },
  { href: "/settings/marketing", label: "marketing", icon: Send, adminOnly: true, group: "tools" },
  { href: "/settings/helpdesk", label: "helpdesk", icon: Headphones, adminOnly: true, group: "tools" },
  { href: "/settings/currencies", label: "currencies", icon: Coins, adminOnly: true, group: "tools" },
  { href: "/recurring-orders", label: "subscriptions", icon: Repeat, group: "tools" },
  { href: "/loyalty", label: "loyalty", icon: Award, group: "tools" },
  { href: "/customers/segments", label: "customerSegments", icon: UsersRound, group: "tools" },

  // ── Settings ──
  { href: "/settings/billing", label: "subscription", icon: CreditCard, adminOnly: true, group: "settings" },
  { href: "/settings/company", label: "company", icon: Building2, adminOnly: true, group: "settings" },
  { href: "/settings/users", label: "users", icon: Users, adminOnly: true, group: "settings" },
  { href: "/settings/roles", label: "roles", icon: Shield, adminOnly: true, group: "settings" },
  { href: "/settings/security", label: "security", icon: ShieldCheck, group: "settings" },
  { href: "/settings/order-statuses", label: "orderStatuses", icon: ListChecks, adminOnly: true, group: "settings" },
  { href: "/settings/custom-fields", label: "customFields", icon: TextCursorInput, adminOnly: true, group: "settings" },
  { href: "/settings/price-lists", label: "priceLists", icon: BadgePercent, adminOnly: true, group: "settings" },
  { href: "/settings/accounting", label: "accounting", icon: Calculator, adminOnly: true, group: "settings" },
  { href: "/settings/ksef", label: "ksef", icon: FileCheck, adminOnly: true, group: "settings" },
  { href: "/settings/vat-oss", label: "vatOss", icon: Landmark, adminOnly: true, group: "settings" },
  { href: "/settings/inventory", label: "inventoryControl", icon: PackageSearch, adminOnly: true, group: "settings" },
  { href: "/settings/notifications", label: "notifications", icon: Bell, adminOnly: true, group: "settings" },
  { href: "/settings/sms", label: "sms", icon: MessageSquare, adminOnly: true, group: "settings" },
  { href: "/settings/webhooks", label: "webhooks", icon: Webhook, adminOnly: true, group: "settings" },
  { href: "/settings/webhooks/deliveries", label: "webhookDeliveries", icon: MailCheck, adminOnly: true, group: "settings" },
  { href: "/settings/sync-jobs", label: "sync", icon: RotateCw, adminOnly: true, group: "settings" },
  { href: "/audit", label: "auditLog", icon: ScrollText, adminOnly: true, group: "settings" },

  // ── Help (flat — no group) ──
  { href: "/help", label: "help", icon: HelpCircle },
];
