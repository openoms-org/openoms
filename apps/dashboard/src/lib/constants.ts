export const ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  new: { label: "Nowe", color: "bg-blue-100 text-blue-800" },
  confirmed: { label: "Potwierdzone", color: "bg-indigo-100 text-indigo-800" },
  processing: { label: "W realizacji", color: "bg-yellow-100 text-yellow-800" },
  ready_to_ship: { label: "Gotowe do wysyłki", color: "bg-orange-100 text-orange-800" },
  shipped: { label: "Wysłane", color: "bg-purple-100 text-purple-800" },
  in_transit: { label: "W transporcie", color: "bg-purple-100 text-purple-800" },
  out_for_delivery: { label: "W doręczeniu", color: "bg-teal-100 text-teal-800" },
  delivered: { label: "Dostarczone", color: "bg-green-100 text-green-800" },
  completed: { label: "Zakończone", color: "bg-green-200 text-green-900" },
  on_hold: { label: "Wstrzymane", color: "bg-gray-100 text-gray-800" },
  cancelled: { label: "Anulowane", color: "bg-red-100 text-red-800" },
  refunded: { label: "Zwrócone", color: "bg-red-200 text-red-900" },
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
  created: { label: "Utworzona", color: "bg-blue-100 text-blue-800" },
  label_ready: { label: "Etykieta gotowa", color: "bg-indigo-100 text-indigo-800" },
  picked_up: { label: "Odebrana", color: "bg-yellow-100 text-yellow-800" },
  in_transit: { label: "W transporcie", color: "bg-purple-100 text-purple-800" },
  out_for_delivery: { label: "W doręczeniu", color: "bg-teal-100 text-teal-800" },
  delivered: { label: "Dostarczona", color: "bg-green-100 text-green-800" },
  returned: { label: "Zwrócona", color: "bg-red-100 text-red-800" },
  failed: { label: "Nieudana", color: "bg-red-200 text-red-900" },
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

export const INTEGRATION_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "Aktywna", color: "bg-green-100 text-green-800" },
  inactive: { label: "Nieaktywna", color: "bg-gray-100 text-gray-800" },
  error: { label: "Błąd", color: "bg-red-100 text-red-800" },
};

export const ROLES: Record<string, string> = {
  owner: "Właściciel",
  admin: "Administrator",
  member: "Członek",
};

export const ORDER_SOURCES = ["manual", "allegro", "woocommerce"] as const;

export const PAYMENT_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: "Oczekuje", color: "bg-yellow-100 text-yellow-800" },
  paid: { label: "Oplacone", color: "bg-green-100 text-green-800" },
  partially_paid: { label: "Czesciowo", color: "bg-orange-100 text-orange-800" },
  refunded: { label: "Zwrocone", color: "bg-red-100 text-red-800" },
  failed: { label: "Nieudane", color: "bg-red-200 text-red-900" },
};

export const PAYMENT_METHODS = ["przelew", "pobranie", "karta", "PayU", "Przelewy24", "BLIK"] as const;
export const SHIPMENT_PROVIDERS = ["inpost", "dhl", "dpd", "manual"] as const;
export const INTEGRATION_PROVIDERS = ["allegro", "inpost", "dhl", "dpd", "woocommerce"] as const;
