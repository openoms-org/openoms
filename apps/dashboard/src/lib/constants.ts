export const ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  new: { label: "new", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  confirmed: { label: "confirmed", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  processing: { label: "processing", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  ready_to_ship: { label: "ready_to_ship", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  shipped: { label: "shipped", color: "bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200" },
  in_transit: { label: "in_transit", color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200" },
  out_for_delivery: { label: "out_for_delivery", color: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200" },
  delivered: { label: "delivered", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  completed: { label: "completed", color: "bg-green-200 text-green-900 dark:bg-green-900 dark:text-green-200" },
  on_hold: { label: "on_hold", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  refunded: { label: "refunded", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
  merged: { label: "merged", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  split: { label: "split", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
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
  merged: [],
  split: [],
};

export const SHIPMENT_STATUSES: Record<string, { label: string; color: string }> = {
  created: { label: "created", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  label_ready: { label: "label_ready", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  picked_up: { label: "picked_up", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  in_transit: { label: "in_transit", color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200" },
  out_for_delivery: { label: "out_for_delivery", color: "bg-teal-100 text-teal-800 dark:bg-teal-900 dark:text-teal-200" },
  delivered: { label: "delivered", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  returned: { label: "returned", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  failed: { label: "failed", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
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
  requested: { label: "requested", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  approved: { label: "approved", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  received: { label: "received", color: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200" },
  refunded: { label: "refunded", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  rejected: { label: "rejected", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  cancelled: { label: "cancelled", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
};

export const RETURN_TRANSITIONS: Record<string, string[]> = {
  requested: ["approved", "rejected", "cancelled"],
  approved: ["received", "cancelled"],
  received: ["refunded", "cancelled"],
  refunded: [],
  rejected: [],
  cancelled: [],
};

export const PURCHASE_ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  draft: { label: "draft", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  sent: { label: "sent", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  partially_received: { label: "partially_received", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  received: { label: "received", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export const SUPPLIER_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "active", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  inactive: { label: "inactive", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  error: { label: "error", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export const INTEGRATION_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "active", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  inactive: { label: "inactive", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  error: { label: "error", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

// Status families below were hoisted out of individual pages, where each list/detail
// pair kept its own duplicate colour map. Colours are transcribed 1:1 from those maps —
// this is a move, not a palette redesign. As everywhere else in this file, `label` holds
// the raw status key; the display string comes from messages/*/statuses.json via
// StatusBadge's translationPrefix.

export const DROPSHIP_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: "pending", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300" },
  sent: { label: "sent", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300" },
  confirmed: { label: "confirmed", color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300" },
  shipped: { label: "shipped", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
  delivered: { label: "delivered", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
};

export const LOYALTY_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "active", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  paused: { label: "paused", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  ended: { label: "ended", color: "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200" },
};

export const STOCKTAKE_STATUSES: Record<string, { label: string; color: string }> = {
  draft: { label: "draft", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  in_progress: { label: "in_progress", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  completed: { label: "completed", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

// Recurring orders use a translucent-fill idiom (bg-*-500/10) rather than the -100/-800
// pairing the other families use. Kept verbatim so this stays a pure move; normalising
// the two idioms is a separate call.
export const RECURRING_ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "active", color: "bg-green-500/10 text-green-700 dark:text-green-400" },
  paused: { label: "paused", color: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400" },
  cancelled: { label: "cancelled", color: "bg-red-500/10 text-red-700 dark:text-red-400" },
  completed: { label: "completed", color: "bg-gray-500/10 text-gray-700 dark:text-gray-400" },
};

export const WAREHOUSE_DOCUMENT_STATUSES: Record<string, { label: string; color: string }> = {
  draft: { label: "draft", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  confirmed: { label: "confirmed", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  cancelled: { label: "cancelled", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
};

export const PICK_PACK_STATUSES: Record<string, { label: string; color: string }> = {
  picking: { label: "picking", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  packing: { label: "packing", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  completed: { label: "completed", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export const REPRICING_RULE_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "active", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  paused: { label: "paused", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  archived: { label: "archived", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
};

export const ROLES: Record<string, string> = {
  owner: "owner",
  admin: "admin",
  member: "member",
};

export const ORDER_SOURCES = ["manual", "supplier", "allegro", "amazon", "empik", "erli", "ebay", "kaufland", "olx", "woocommerce"] as const;

export const ORDER_PRIORITIES: Record<string, { label: string; color: string }> = {
  urgent: { label: "urgent", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  high: { label: "high", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  normal: { label: "normal", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  low: { label: "low", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
};

export const PAYMENT_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: "pending", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  paid: { label: "paid", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  partially_paid: { label: "partially_paid", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  refunded: { label: "refunded", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  failed: { label: "failed", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const PAYMENT_METHODS = ["transfer", "cod", "card", "PayU", "Przelewy24", "BLIK"] as const;
export const SHIPMENT_PROVIDERS = ["inpost", "dhl", "dpd", "gls", "ups", "poczta_polska", "orlen_paczka", "fedex", "manual"] as const;
export const INTEGRATION_PROVIDERS = ["allegro", "amazon", "woocommerce", "ebay", "kaufland", "olx", "erli", "empik", "inpost", "dhl", "dpd", "gls", "ups", "fedex", "poczta_polska", "orlen_paczka", "fakturownia", "btp"] as const;

export const ORDER_SOURCE_LABELS: Record<string, string> = {
  manual: "manual",
  supplier: "supplier",
  allegro: "allegro",
  amazon: "amazon",
  empik: "empik",
  erli: "erli",
  ebay: "ebay",
  kaufland: "kaufland",
  olx: "olx",
  woocommerce: "woocommerce",
};

export const SHIPMENT_PROVIDER_LABELS: Record<string, string> = {
  inpost: "inpost",
  dhl: "dhl",
  dpd: "dpd",
  gls: "gls",
  ups: "ups",
  poczta_polska: "poczta_polska",
  orlen_paczka: "orlen_paczka",
  fedex: "fedex",
  manual: "manual",
};

export const INTEGRATION_PROVIDER_LABELS: Record<string, string> = {
  allegro: "allegro",
  amazon: "amazon",
  empik: "empik",
  erli: "erli",
  ebay: "ebay",
  kaufland: "kaufland",
  olx: "olx",
  inpost: "inpost",
  dhl: "dhl",
  dpd: "dpd",
  woocommerce: "woocommerce",
  gls: "gls",
  ups: "ups",
  fedex: "fedex",
  poczta_polska: "poczta_polska",
  orlen_paczka: "orlen_paczka",
  fakturownia: "fakturownia",
  btp: "btp",
};

export const INVOICE_STATUS_MAP: Record<string, { label: string; color: string }> = {
  draft: { label: "draft", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  issued: { label: "issued", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200" },
  sent: { label: "sent", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200" },
  paid: { label: "paid", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  partially_paid: { label: "partially_paid", color: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200" },
  cancelled: { label: "cancelled", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
  error: { label: "error", color: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-200" },
};

export const INVOICE_TYPE_LABELS: Record<string, string> = {
  standard: "standard",
  vat: "vat",
  proforma: "proforma",
  correction: "correction",
  receipt: "receipt",
};

export const INVOICING_PROVIDER_LABELS: Record<string, string> = {
  fakturownia: "fakturownia",
  wfirma: "wfirma",
  infakt: "infakt",
};

// === KSeF ===
export const KSEF_STATUS_MAP: Record<string, { label: string; color: string }> = {
  not_sent: { label: "not_sent", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  pending: { label: "pending", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" },
  accepted: { label: "accepted", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  rejected: { label: "rejected", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
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
  "product.stock_restored",
  "product.out_of_stock",
] as const;

export const AUTOMATION_TRIGGER_LABELS: Record<string, string> = {
  "order.created": "order.created",
  "order.status_changed": "order.status_changed",
  "order.updated": "order.updated",
  "shipment.created": "shipment.created",
  "shipment.status_changed": "shipment.status_changed",
  "return.created": "return.created",
  "return.status_changed": "return.status_changed",
  "product.created": "product.created",
  "product.updated": "product.updated",
  "product.stock_restored": "product.stock_restored",
  "product.out_of_stock": "product.out_of_stock",
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
  eq: "eq",
  neq: "neq",
  in: "in",
  not_in: "not_in",
  gt: "gt",
  gte: "gte",
  lt: "lt",
  lte: "lte",
  contains: "contains",
  not_contains: "not_contains",
  starts_with: "starts_with",
};

export const AUTOMATION_ACTION_TYPES = [
  "set_status",
  "add_tag",
  "send_email",
  "create_invoice",
  "webhook",
  "activate_listing",
  "deactivate_listing",
] as const;

export const AUTOMATION_ACTION_LABELS: Record<string, string> = {
  set_status: "set_status",
  add_tag: "add_tag",
  send_email: "send_email",
  create_invoice: "create_invoice",
  webhook: "webhook",
  activate_listing: "activate_listing",
  deactivate_listing: "deactivate_listing",
};

// === Integration Provider Credential Fields ===

export interface CredentialField {
  key: string;
  labelKey: string;
  placeholderKey?: string;
  helpTextKey?: string;
  type: "text" | "password" | "url" | "checkbox" | "select";
  required: boolean;
  options?: { value: string; labelKey: string }[];
}

export const PROVIDER_CATEGORIES: Record<string, { labelKey: string; providers: string[] }> = {
  marketplace: { labelKey: "category.marketplace", providers: ["allegro", "amazon", "woocommerce", "ebay", "olx", "erli"] },
  carrier: { labelKey: "category.carrier", providers: ["inpost", "dhl", "dpd", "gls", "ups", "fedex", "poczta_polska", "orlen_paczka"] },
  invoicing: { labelKey: "category.invoicing", providers: ["fakturownia"] },
  supplier: { labelKey: "category.supplier", providers: ["btp"] },
};

/** Providers with dedicated setup pages — excluded from the generic "New Integration" form */
export const PROVIDERS_WITH_DEDICATED_PAGES: Record<string, string> = {
  allegro: "/marketplaces/allegro",
  amazon: "/marketplaces/amazon",
  olx: "/marketplaces/olx",
};

export const PROVIDER_CREDENTIAL_FIELDS: Record<string, CredentialField[]> = {
  allegro: [
    { key: "client_id", labelKey: "credentials.allegro.clientId", placeholderKey: "credentials.allegro.clientIdPlaceholder", helpTextKey: "credentials.allegro.clientIdHelp", type: "text", required: true },
    { key: "client_secret", labelKey: "credentials.allegro.clientSecret", placeholderKey: "credentials.allegro.clientSecretPlaceholder", type: "password", required: true },
    { key: "webhook_secret", labelKey: "credentials.allegro.webhookSecret", placeholderKey: "credentials.allegro.webhookSecretPlaceholder", helpTextKey: "credentials.allegro.webhookSecretHelp", type: "password", required: false },
    { key: "access_token", labelKey: "credentials.allegro.accessToken", helpTextKey: "credentials.allegro.accessTokenHelp", type: "password", required: false },
    { key: "refresh_token", labelKey: "credentials.allegro.refreshToken", type: "password", required: false },
    { key: "sandbox", labelKey: "credentials.allegro.sandbox", type: "checkbox", required: false },
  ],
  amazon: [
    { key: "application_id", labelKey: "credentials.amazon.applicationId", placeholderKey: "credentials.amazon.applicationIdPlaceholder", helpTextKey: "credentials.amazon.applicationIdHelp", type: "text", required: true },
    { key: "client_id", labelKey: "credentials.amazon.clientId", placeholderKey: "credentials.amazon.clientIdPlaceholder", helpTextKey: "credentials.amazon.clientIdHelp", type: "text", required: true },
    { key: "client_secret", labelKey: "credentials.amazon.clientSecret", type: "password", required: true },
    { key: "refresh_token", labelKey: "credentials.amazon.refreshToken", helpTextKey: "credentials.amazon.refreshTokenHelp", type: "password", required: false },
    { key: "marketplace_id", labelKey: "credentials.amazon.marketplaceId", placeholderKey: "credentials.amazon.marketplaceIdPlaceholder", helpTextKey: "credentials.amazon.marketplaceIdHelp", type: "text", required: true },
    { key: "sandbox", labelKey: "credentials.amazon.sandbox", type: "checkbox", required: false },
  ],
  woocommerce: [
    { key: "store_url", labelKey: "credentials.woocommerce.storeUrl", placeholderKey: "credentials.woocommerce.storeUrlPlaceholder", helpTextKey: "credentials.woocommerce.storeUrlHelp", type: "url", required: true },
    { key: "consumer_key", labelKey: "credentials.woocommerce.consumerKey", placeholderKey: "credentials.woocommerce.consumerKeyPlaceholder", helpTextKey: "credentials.woocommerce.consumerKeyHelp", type: "password", required: true },
    { key: "consumer_secret", labelKey: "credentials.woocommerce.consumerSecret", placeholderKey: "credentials.woocommerce.consumerSecretPlaceholder", type: "password", required: true },
  ],
  ebay: [
    { key: "app_id", labelKey: "credentials.ebay.appId", placeholderKey: "credentials.ebay.appIdPlaceholder", helpTextKey: "credentials.ebay.appIdHelp", type: "text", required: true },
    { key: "cert_id", labelKey: "credentials.ebay.certId", type: "password", required: true },
    { key: "dev_id", labelKey: "credentials.ebay.devId", type: "text", required: true },
    { key: "refresh_token", labelKey: "credentials.ebay.refreshToken", helpTextKey: "credentials.ebay.refreshTokenHelp", type: "password", required: true },
    { key: "sandbox", labelKey: "credentials.ebay.sandbox", type: "checkbox", required: false },
  ],
  kaufland: [
    { key: "api_key", labelKey: "credentials.kaufland.apiKey", placeholderKey: "credentials.kaufland.apiKeyPlaceholder", helpTextKey: "credentials.kaufland.apiKeyHelp", type: "password", required: true },
    { key: "secret_key", labelKey: "credentials.kaufland.secretKey", type: "password", required: true },
    { key: "sandbox", labelKey: "credentials.kaufland.sandbox", type: "checkbox", required: false },
  ],
  olx: [
    { key: "client_id", labelKey: "credentials.olx.clientId", placeholderKey: "credentials.olx.clientIdPlaceholder", helpTextKey: "credentials.olx.clientIdHelp", type: "text", required: true },
    { key: "client_secret", labelKey: "credentials.olx.clientSecret", type: "password", required: true },
    { key: "access_token", labelKey: "credentials.olx.accessToken", helpTextKey: "credentials.olx.accessTokenHelp", type: "password", required: false },
    { key: "refresh_token", labelKey: "credentials.olx.refreshToken", helpTextKey: "credentials.olx.refreshTokenHelp", type: "password", required: false },
  ],
  erli: [
    { key: "api_token", labelKey: "credentials.erli.apiToken", placeholderKey: "credentials.erli.apiTokenPlaceholder", helpTextKey: "credentials.erli.apiTokenHelp", type: "password", required: true },
    { key: "sandbox", labelKey: "credentials.erli.sandbox", type: "checkbox", required: false },
  ],
  empik: [
    { key: "base_url", labelKey: "credentials.empik.baseUrl", placeholderKey: "credentials.empik.baseUrlPlaceholder", helpTextKey: "credentials.empik.baseUrlHelp", type: "url", required: true },
    { key: "api_key", labelKey: "credentials.empik.apiKey", helpTextKey: "credentials.empik.apiKeyHelp", type: "password", required: true },
  ],
  inpost: [
    { key: "api_token", labelKey: "credentials.inpost.apiToken", placeholderKey: "credentials.inpost.apiTokenPlaceholder", helpTextKey: "credentials.inpost.apiTokenHelp", type: "password", required: true },
    { key: "organization_id", labelKey: "credentials.inpost.organizationId", placeholderKey: "credentials.inpost.organizationIdPlaceholder", type: "text", required: true },
    { key: "geowidget_token", labelKey: "credentials.inpost.geowidgetToken", placeholderKey: "credentials.inpost.geowidgetTokenPlaceholder", helpTextKey: "credentials.inpost.geowidgetTokenHelp", type: "text", required: false },
    { key: "webhook_secret", labelKey: "credentials.inpost.webhookSecret", placeholderKey: "credentials.inpost.webhookSecretPlaceholder", helpTextKey: "credentials.inpost.webhookSecretHelp", type: "password", required: false },
    { key: "sandbox", labelKey: "credentials.inpost.sandbox", type: "checkbox", required: false },
    { key: "default_sending_method", labelKey: "credentials.inpost.defaultSendingMethod", helpTextKey: "credentials.inpost.defaultSendingMethodHelp", type: "select", required: false, options: [
      { value: "dispatch_order", labelKey: "credentials.inpost.sendingMethods.dispatch_order" },
      { value: "parcel_locker", labelKey: "credentials.inpost.sendingMethods.parcel_locker" },
      { value: "pop", labelKey: "credentials.inpost.sendingMethods.pop" },
      { value: "any_point", labelKey: "credentials.inpost.sendingMethods.any_point" },
      { value: "pok", labelKey: "credentials.inpost.sendingMethods.pok" },
      { value: "branch", labelKey: "credentials.inpost.sendingMethods.branch" },
    ]},
  ],
  dhl: [
    { key: "username", labelKey: "credentials.dhl.username", placeholderKey: "credentials.dhl.usernamePlaceholder", type: "text", required: true },
    { key: "password", labelKey: "credentials.dhl.password", type: "password", required: true },
    { key: "account_number", labelKey: "credentials.dhl.accountNumber", placeholderKey: "credentials.dhl.accountNumberPlaceholder", helpTextKey: "credentials.dhl.accountNumberHelp", type: "text", required: true },
    { key: "sandbox", labelKey: "credentials.dhl.sandbox", type: "checkbox", required: false },
  ],
  dpd: [
    { key: "login", labelKey: "credentials.dpd.login", placeholderKey: "credentials.dpd.loginPlaceholder", type: "text", required: true },
    { key: "password", labelKey: "credentials.dpd.password", type: "password", required: true },
    { key: "master_fid", labelKey: "credentials.dpd.masterFid", placeholderKey: "credentials.dpd.masterFidPlaceholder", helpTextKey: "credentials.dpd.masterFidHelp", type: "text", required: true },
    { key: "info_channel", labelKey: "credentials.dpd.infoChannel", placeholderKey: "credentials.dpd.infoChannelPlaceholder", helpTextKey: "credentials.dpd.infoChannelHelp", type: "text", required: false },
    { key: "info_login", labelKey: "credentials.dpd.infoLogin", placeholderKey: "credentials.dpd.infoLoginPlaceholder", helpTextKey: "credentials.dpd.infoLoginHelp", type: "text", required: false },
    { key: "info_password", labelKey: "credentials.dpd.infoPassword", helpTextKey: "credentials.dpd.infoPasswordHelp", type: "password", required: false },
    { key: "sandbox", labelKey: "credentials.dpd.sandbox", type: "checkbox", required: false },
  ],
  gls: [
    { key: "api_key", labelKey: "credentials.gls.apiKey", placeholderKey: "credentials.gls.apiKeyPlaceholder", helpTextKey: "credentials.gls.apiKeyHelp", type: "password", required: true },
    { key: "sandbox", labelKey: "credentials.gls.sandbox", type: "checkbox", required: false },
  ],
  ups: [
    { key: "client_id", labelKey: "credentials.ups.clientId", placeholderKey: "credentials.ups.clientIdPlaceholder", helpTextKey: "credentials.ups.clientIdHelp", type: "text", required: true },
    { key: "client_secret", labelKey: "credentials.ups.clientSecret", type: "password", required: true },
    { key: "sandbox", labelKey: "credentials.ups.sandbox", type: "checkbox", required: false },
  ],
  fedex: [
    { key: "client_id", labelKey: "credentials.fedex.clientId", placeholderKey: "credentials.fedex.clientIdPlaceholder", helpTextKey: "credentials.fedex.clientIdHelp", type: "text", required: true },
    { key: "client_secret", labelKey: "credentials.fedex.clientSecret", type: "password", required: true },
    { key: "account_number", labelKey: "credentials.fedex.accountNumber", placeholderKey: "credentials.fedex.accountNumberPlaceholder", type: "text", required: true },
    { key: "sandbox", labelKey: "credentials.fedex.sandbox", type: "checkbox", required: false },
  ],
  poczta_polska: [
    { key: "api_key", labelKey: "credentials.poczta_polska.apiKey", placeholderKey: "credentials.poczta_polska.apiKeyPlaceholder", helpTextKey: "credentials.poczta_polska.apiKeyHelp", type: "password", required: true },
    { key: "partner_id", labelKey: "credentials.poczta_polska.partnerId", placeholderKey: "credentials.poczta_polska.partnerIdPlaceholder", type: "text", required: true },
    { key: "sandbox", labelKey: "credentials.poczta_polska.sandbox", type: "checkbox", required: false },
  ],
  orlen_paczka: [
    { key: "api_key", labelKey: "credentials.orlen_paczka.apiKey", placeholderKey: "credentials.orlen_paczka.apiKeyPlaceholder", helpTextKey: "credentials.orlen_paczka.apiKeyHelp", type: "password", required: true },
    { key: "partner_id", labelKey: "credentials.orlen_paczka.partnerId", placeholderKey: "credentials.orlen_paczka.partnerIdPlaceholder", type: "text", required: true },
    { key: "sandbox", labelKey: "credentials.orlen_paczka.sandbox", type: "checkbox", required: false },
  ],
  fakturownia: [
    { key: "api_token", labelKey: "credentials.fakturownia.apiToken", placeholderKey: "credentials.fakturownia.apiTokenPlaceholder", helpTextKey: "credentials.fakturownia.apiTokenHelp", type: "password", required: true },
    { key: "subdomain", labelKey: "credentials.fakturownia.subdomain", placeholderKey: "credentials.fakturownia.subdomainPlaceholder", helpTextKey: "credentials.fakturownia.subdomainHelp", type: "text", required: true },
  ],
  btp: [
    { key: "username", labelKey: "credentials.btp.username", placeholderKey: "credentials.btp.usernamePlaceholder", helpTextKey: "credentials.btp.usernameHelp", type: "text", required: true },
    { key: "password", labelKey: "credentials.btp.password", placeholderKey: "credentials.btp.passwordPlaceholder", helpTextKey: "credentials.btp.passwordHelp", type: "password", required: true },
    { key: "base_url", labelKey: "credentials.btp.baseUrl", placeholderKey: "credentials.btp.baseUrlPlaceholder", helpTextKey: "credentials.btp.baseUrlHelp", type: "url", required: false },
  ],
};

export const PROVIDER_SETTINGS_FIELDS: Record<string, string[]> = {
  inpost: ["geowidget_token", "default_sending_method"],
};
