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
  merged: { label: "Scalone", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  split: { label: "Rozdzielone", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
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

export const SUPPLIER_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "Aktywny", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  inactive: { label: "Nieaktywny", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  error: { label: "Błąd", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
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

export const ORDER_PRIORITIES: Record<string, { label: string; color: string }> = {
  urgent: { label: "Pilne", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  high: { label: "Wysoki", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  normal: { label: "Normalny", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  low: { label: "Niski", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
};

export const PAYMENT_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: "Oczekuje", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  paid: { label: "Opłacone", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  partially_paid: { label: "Częściowo", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  refunded: { label: "Zwrócone", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  failed: { label: "Nieudane", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const PAYMENT_METHODS = ["przelew", "pobranie", "karta", "PayU", "Przelewy24", "BLIK"] as const;
export const SHIPMENT_PROVIDERS = ["inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "fedex", "manual"] as const;
export const INTEGRATION_PROVIDERS = ["allegro", "amazon", "woocommerce", "ebay", "kaufland", "olx", "erli", "empik", "inpost", "dhl", "dpd", "gls", "ups", "fedex", "poczta_polska", "orlen_paczka", "fakturownia"] as const;

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
  gls: "GLS",
  ups: "UPS",
  fedex: "FedEx",
  poczta_polska: "Poczta Polska",
  orlen_paczka: "Orlen Paczka",
  fakturownia: "Fakturownia",
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
  standard: "Standardowa",
  vat: "Faktura VAT",
  proforma: "Proforma",
  correction: "Korekta",
  receipt: "Paragon",
};

export const INVOICING_PROVIDERS = ["fakturownia"] as const;

export const INVOICING_PROVIDER_LABELS: Record<string, string> = {
  fakturownia: "Fakturownia",
};

// === KSeF ===
export const KSEF_STATUS_MAP: Record<string, { label: string; color: string }> = {
  not_sent: { label: "Nie wysłano", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  pending: { label: "Oczekuje", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  accepted: { label: "Zaakceptowana", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  rejected: { label: "Odrzucona", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export const KSEF_ENVIRONMENTS = [
  { value: "test", label: "Testowe (ksef-test.mf.gov.pl)" },
  { value: "production", label: "Produkcyjne (ksef.mf.gov.pl)" },
] as const;

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

// === Integration Provider Credential Fields ===

export interface CredentialField {
  key: string;
  label: string;
  placeholder: string;
  helpText?: string;
  type: "text" | "password" | "url" | "checkbox" | "select";
  required: boolean;
  options?: { value: string; label: string }[];
}

export const PROVIDER_CATEGORIES: Record<string, { label: string; providers: string[] }> = {
  marketplace: { label: "Marketplace", providers: ["allegro", "amazon", "woocommerce", "ebay", "kaufland", "olx", "erli", "empik"] },
  carrier: { label: "Kurierzy", providers: ["inpost", "dhl", "dpd", "gls", "ups", "fedex", "poczta_polska", "orlen_paczka"] },
  invoicing: { label: "Fakturowanie", providers: ["fakturownia"] },
};

/** Providers with dedicated setup pages — excluded from the generic "New Integration" form */
export const PROVIDERS_WITH_DEDICATED_PAGES: Record<string, string> = {
  allegro: "/integrations/allegro",
  amazon: "/integrations/amazon",
};

export const PROVIDER_CREDENTIAL_FIELDS: Record<string, CredentialField[]> = {
  allegro: [
    { key: "client_id", label: "Client ID", placeholder: "Twój Client ID z apps.developer.allegro.pl", helpText: "Znajdziesz w panelu deweloperskim Allegro: apps.developer.allegro.pl", type: "text", required: true },
    { key: "client_secret", label: "Client Secret", placeholder: "Twój Client Secret", type: "password", required: true },
    { key: "access_token", label: "Access Token", placeholder: "", helpText: "Zostanie pobrany automatycznie po autoryzacji OAuth2", type: "password", required: false },
    { key: "refresh_token", label: "Refresh Token", placeholder: "", type: "password", required: false },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  amazon: [
    { key: "client_id", label: "Client ID (LWA)", placeholder: "amzn1.application-oa2-client.xxx", helpText: "Login with Amazon (LWA) credentials z Seller Central > Develop Apps", type: "text", required: true },
    { key: "client_secret", label: "Client Secret (LWA)", placeholder: "", type: "password", required: true },
    { key: "refresh_token", label: "Refresh Token", placeholder: "", helpText: "Wygeneruj w Seller Central > Develop Apps > Authorize", type: "password", required: true },
    { key: "marketplace_id", label: "Marketplace ID", placeholder: "A1C3SOZRARQ6R3", helpText: "Amazon.pl: A1C3SOZRARQ6R3, Amazon.de: A1PA6795UKMFR9", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  woocommerce: [
    { key: "store_url", label: "Adres sklepu", placeholder: "https://twoj-sklep.pl", helpText: "Pełny adres URL Twojego sklepu WooCommerce", type: "url", required: true },
    { key: "consumer_key", label: "Consumer Key", placeholder: "ck_...", helpText: "WooCommerce > Ustawienia > Zaawansowane > REST API > Dodaj klucz", type: "password", required: true },
    { key: "consumer_secret", label: "Consumer Secret", placeholder: "cs_...", type: "password", required: true },
  ],
  ebay: [
    { key: "app_id", label: "App ID (Client ID)", placeholder: "Twój App ID z developer.ebay.com", helpText: "Znajdziesz w eBay Developer Program: developer.ebay.com", type: "text", required: true },
    { key: "cert_id", label: "Cert ID (Client Secret)", placeholder: "", type: "password", required: true },
    { key: "dev_id", label: "Dev ID", placeholder: "", type: "text", required: true },
    { key: "refresh_token", label: "Refresh Token", placeholder: "", helpText: "Wygeneruj przez eBay OAuth flow", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  kaufland: [
    { key: "api_key", label: "Klucz API", placeholder: "Twój klucz API Kaufland", helpText: "Seller Portal Kaufland > Ustawienia > API", type: "password", required: true },
    { key: "secret_key", label: "Klucz Secret", placeholder: "", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  olx: [
    { key: "client_id", label: "Client ID", placeholder: "Twój Client ID z OLX API", helpText: "Zarejestruj aplikację na developer.olx.pl", type: "text", required: true },
    { key: "client_secret", label: "Client Secret", placeholder: "", type: "password", required: true },
    { key: "access_token", label: "Access Token", placeholder: "", helpText: "Wygenerowany po autoryzacji OAuth2", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  erli: [
    { key: "api_token", label: "Token API", placeholder: "Twój token API Erli", helpText: "Panel sprzedawcy Erli > Ustawienia > API", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  empik: [
    { key: "base_url", label: "Adres API Mirakl", placeholder: "https://empikplace-prod.mirakl.net", helpText: "URL platformy Mirakl dla Empik Marketplace", type: "url", required: true },
    { key: "api_key", label: "Klucz API", placeholder: "", helpText: "Mój Profil > Klucze API w panelu Mirakl", type: "password", required: true },
  ],
  inpost: [
    { key: "api_token", label: "Token API", placeholder: "Twój token API InPost", helpText: "Manager Paczek InPost > Ustawienia > API", type: "password", required: true },
    { key: "organization_id", label: "ID organizacji", placeholder: "Twój numer organizacji InPost", type: "text", required: true },
    { key: "geowidget_token", label: "Token GeoWidget (mapa)", placeholder: "Twój token GeoWidget InPost", helpText: "Token do wyświetlania mapy paczkomatów. Wygeneruj na manager.paczkomaty.pl > API > GeoWidget. WAŻNE: token jest przypisany do domeny — dla dev wpisz 'localhost', dla produkcji domenę sklepu.", type: "text", required: false },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
    { key: "default_sending_method", label: "Domyślna metoda nadania", placeholder: "", helpText: "Metoda nadania używana domyślnie przy generowaniu etykiety InPost", type: "select", required: false, options: [
      { value: "dispatch_order", label: "Kurier odbierze (zlecenie odbioru)" },
      { value: "parcel_locker", label: "Nadam w paczkomacie" },
      { value: "pop", label: "Nadam w PaczkoPunkcie (POP)" },
      { value: "any_point", label: "Dowolny punkt (paczkomat/POP)" },
      { value: "pok", label: "Punkt Obsługi Klienta (POK)" },
      { value: "branch", label: "Oddział InPost" },
    ]},
  ],
  dhl: [
    { key: "username", label: "Nazwa użytkownika", placeholder: "Login do DHL WebAPI", type: "text", required: true },
    { key: "password", label: "Hasło", placeholder: "", type: "password", required: true },
    { key: "account_number", label: "Numer konta DHL", placeholder: "Twój numer konta nadawczego", helpText: "Numer konta nadawczego DHL (SAP)", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  dpd: [
    { key: "login", label: "Login", placeholder: "Login do DPD WebAPI", type: "text", required: true },
    { key: "password", label: "Hasło", placeholder: "", type: "password", required: true },
    { key: "master_fid", label: "Master FID", placeholder: "Numer Master FID", helpText: "Numer identyfikacyjny nadawcy DPD", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  gls: [
    { key: "api_key", label: "Klucz API", placeholder: "Twój klucz API GLS", helpText: "GLS Poland > Konto firmowe > API", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  ups: [
    { key: "client_id", label: "Client ID", placeholder: "Twój Client ID z developer.ups.com", helpText: "UPS Developer Kit: developer.ups.com", type: "text", required: true },
    { key: "client_secret", label: "Client Secret", placeholder: "", type: "password", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  fedex: [
    { key: "client_id", label: "Client ID (API Key)", placeholder: "Twój Client ID z developer.fedex.com", helpText: "FedEx Developer Portal: developer.fedex.com", type: "text", required: true },
    { key: "client_secret", label: "Client Secret (Secret Key)", placeholder: "", type: "password", required: true },
    { key: "account_number", label: "Numer konta FedEx", placeholder: "Twój 9-cyfrowy numer konta", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  poczta_polska: [
    { key: "api_key", label: "Klucz API", placeholder: "Twój klucz API Poczta Polska", helpText: "Elektroniczny Nadawca > Ustawienia > API", type: "password", required: true },
    { key: "partner_id", label: "ID Partnera", placeholder: "Twój identyfikator partnera", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  orlen_paczka: [
    { key: "api_key", label: "Klucz API", placeholder: "Twój klucz API Orlen Paczka", helpText: "Panel Orlen Paczka > Ustawienia > Dostęp API", type: "password", required: true },
    { key: "partner_id", label: "ID Partnera", placeholder: "Twój identyfikator partnera", type: "text", required: true },
    { key: "sandbox", label: "Tryb testowy (Sandbox)", placeholder: "", type: "checkbox", required: false },
  ],
  fakturownia: [
    { key: "api_token", label: "Token API", placeholder: "Twój token API Fakturownia", helpText: "Ustawienia > Ustawienia konta > Integracja > Kod autoryzacyjny API", type: "password", required: true },
    { key: "subdomain", label: "Subdomena", placeholder: "twoja-firma", helpText: "Nazwa Twojego konta, np. 'twoja-firma' z twoja-firma.fakturownia.pl", type: "text", required: true },
  ],
};

export const PROVIDER_SETTINGS_FIELDS: Record<string, string[]> = {
  inpost: ["geowidget_token", "default_sending_method"],
};
