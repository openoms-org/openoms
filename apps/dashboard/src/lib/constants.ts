export const ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  new: { label: "Nowe", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  confirmed: { label: "Potwierdzone", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  processing: { label: "W realizacji", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  ready_to_ship: { label: "Gotowe do wysyłki", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  shipped: { label: "Wysłane", color: "bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200" },
  in_transit: { label: "W transporcie", color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200" },
  out_for_delivery: { label: "W doręczeniu", color: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200" },
  delivered: { label: "Dostarczone", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  completed: { label: "Zakończone", color: "bg-green-200 text-green-900 dark:bg-green-900 dark:text-green-200" },
  on_hold: { label: "Wstrzymane", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  cancelled: { label: "Anulowane", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  refunded: { label: "Zwrócone", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const ORDER_TRANSITIONS: Record<string, string[]> = {
  new: ["confirmed", "cancelled", "on_hold"],
  confirmed: ["processing", "cancelled", "on_hold"],
  processing: ["ready_to_ship", "cancelled", "on_hold"],
  ready_to_ship: ["shipped", "cancelled", "on_hold"],
  shipped: ["in_transit", "delivered", "refunded"],
  in_transit: ["out_for_delivery", "delivered", "refunded"],
  out_for_delivery: ["delivered", "refunded"],
  delivered: ["completed", "refunded"],
  completed: ["refunded"],
  on_hold: ["confirmed", "processing", "cancelled"],
  cancelled: ["refunded"],
  refunded: [],
};

export const SHIPMENT_STATUSES: Record<string, { label: string; color: string }> = {
  created: { label: "Utworzona", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  label_ready: { label: "Etykieta gotowa", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  picked_up: { label: "Odebrana", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  in_transit: { label: "W transporcie", color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200" },
  out_for_delivery: { label: "W doręczeniu", color: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200" },
  delivered: { label: "Dostarczona", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  returned: { label: "Zwrócona", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  failed: { label: "Nieudana", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const SHIPMENT_TRANSITIONS: Record<string, string[]> = {
  created: ["label_ready", "failed"],
  label_ready: ["picked_up", "failed"],
  picked_up: ["in_transit", "failed"],
  in_transit: ["out_for_delivery", "delivered", "returned", "failed"],
  out_for_delivery: ["delivered", "returned", "failed"],
  delivered: ["returned"],
  returned: [],
  failed: ["created"],
};

export const RETURN_STATUSES: Record<string, { label: string; color: string }> = {
  requested: { label: "Zgłoszone", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  approved: { label: "Zatwierdzone", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  received: { label: "Odebrane", color: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200" },
  refunded: { label: "Zwrócone", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  rejected: { label: "Odrzucone", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  cancelled: { label: "Anulowane", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
};

export const RETURN_TRANSITIONS: Record<string, string[]> = {
  requested: ["approved", "rejected", "cancelled"],
  approved: ["received", "cancelled"],
  received: ["refunded", "cancelled"],
  refunded: [],
  rejected: [],
  cancelled: [],
};

export const INTEGRATION_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "Aktywna", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  inactive: { label: "Nieaktywna", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  error: { label: "Błąd", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export const ROLES: Record<string, string> = {
  owner: "Właściciel",
  admin: "Administrator",
  member: "Członek",
};

export const ORDER_SOURCES = ["manual", "allegro", "amazon", "empik", "erli", "ebay", "kaufland", "olx", "woocommerce"] as const;

export const PAYMENT_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: "Oczekuje", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  paid: { label: "Opłacone", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  partially_paid: { label: "Częściowo", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  refunded: { label: "Zwrócone", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  failed: { label: "Nieudane", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const PAYMENT_METHODS = ["przelew", "pobranie", "karta", "PayU", "Przelewy24", "BLIK"] as const;
export const SHIPMENT_PROVIDERS = ["inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "fedex", "manual"] as const;
export const INTEGRATION_PROVIDERS = ["allegro", "amazon", "empik", "erli", "ebay", "kaufland", "olx", "inpost", "dhl", "dpd", "woocommerce"] as const;

export const ORDER_SOURCE_LABELS: Record<string, string> = {
  manual: "Ręczne",
  allegro: "Allegro",
  amazon: "Amazon",
  empik: "Empik",
  erli: "Erli",
  ebay: "eBay",
  kaufland: "Kaufland",
  olx: "OLX",
  woocommerce: "WooCommerce",
};

export const SHIPMENT_PROVIDER_LABELS: Record<string, string> = {
  inpost: "InPost",
  dhl: "DHL",
  dpd: "DPD",
  gls: "GLS",
  ups: "UPS",
  poczta_polska: "Poczta Polska",
  orlen_paczka: "Orlen Paczka",
  fedex: "FedEx",
  manual: "Ręczna",
};

export const INTEGRATION_PROVIDER_LABELS: Record<string, string> = {
  allegro: "Allegro",
  amazon: "Amazon",
  empik: "Empik",
  erli: "Erli",
  ebay: "eBay",
  kaufland: "Kaufland",
  olx: "OLX",
  inpost: "InPost",
  dhl: "DHL",
  dpd: "DPD",
  woocommerce: "WooCommerce",
};

export const INVOICE_STATUS_MAP: Record<string, { label: string; color: string }> = {
  draft: { label: "Szkic", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  issued: { label: "Wystawiona", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  sent: { label: "Wysłana", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  paid: { label: "Opłacona", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  partially_paid: { label: "Częściowo opłacona", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  cancelled: { label: "Anulowana", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  error: { label: "Błąd", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const INVOICE_TYPE_LABELS: Record<string, string> = {
  vat: "Faktura VAT",
  proforma: "Proforma",
  correction: "Korekta",
  receipt: "Paragon",
};

export const INVOICING_PROVIDERS = ["fakturownia"] as const;

export const INVOICING_PROVIDER_LABELS: Record<string, string> = {
  fakturownia: "Fakturownia",
};

// === Automation ===
export const AUTOMATION_TRIGGER_EVENTS = [
  "order.created",
  "order.status_changed",
  "order.updated",
  "shipment.created",
  "shipment.status_changed",
  "return.created",
  "return.status_changed",
  "product.created",
  "product.updated",
] as const;

export const AUTOMATION_TRIGGER_LABELS: Record<string, string> = {
  "order.created": "Zamówienie utworzone",
  "order.status_changed": "Zmiana statusu zamówienia",
  "order.updated": "Zamówienie zaktualizowane",
  "shipment.created": "Przesyłka utworzona",
  "shipment.status_changed": "Zmiana statusu przesyłki",
  "return.created": "Zwrot utworzony",
  "return.status_changed": "Zmiana statusu zwrotu",
  "product.created": "Produkt utworzony",
  "product.updated": "Produkt zaktualizowany",
};

export const AUTOMATION_OPERATORS = [
  "eq",
  "neq",
  "in",
  "not_in",
  "gt",
  "gte",
  "lt",
  "lte",
  "contains",
  "not_contains",
  "starts_with",
] as const;

export const AUTOMATION_OPERATOR_LABELS: Record<string, string> = {
  eq: "Równe (=)",
  neq: "Różne (!=)",
  in: "Zawiera się w",
  not_in: "Nie zawiera się w",
  gt: "Większe (>)",
  gte: "Większe lub równe (>=)",
  lt: "Mniejsze (<)",
  lte: "Mniejsze lub równe (<=)",
  contains: "Zawiera",
  not_contains: "Nie zawiera",
  starts_with: "Zaczyna się od",
};

export const AUTOMATION_ACTION_TYPES = [
  "set_status",
  "add_tag",
  "send_email",
  "create_invoice",
  "webhook",
] as const;

export const AUTOMATION_ACTION_LABELS: Record<string, string> = {
  set_status: "Ustaw status",
  add_tag: "Dodaj tag",
  send_email: "Wyślij e-mail",
  create_invoice: "Utwórz fakturę",
  webhook: "Wywołaj webhook",
};
