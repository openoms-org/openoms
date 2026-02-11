export const DESTRUCTIVE_ORDER_STATUSES = ["cancelled", "refunded"];

export function isDestructiveOrderStatus(status: string): boolean {
  return DESTRUCTIVE_ORDER_STATUSES.includes(status);
}
