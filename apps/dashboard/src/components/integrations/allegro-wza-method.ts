export type WzACreateDecision =
  | { ok: true; deliveryMethodId: string }
  | {
      ok: false;
      reason: "no_proposal_method";
      checkoutMethodId: string;
      checkoutMethodName: string;
    };

// Official WzA methods from Allegro's Nov 2025 InPost notice.
// Checkout delivery.method.id 9081532b is Allegro miniKurier24 InPost.
export const OFFICIAL_WZA_DELIVERY_METHODS = {
  "9081532b-5ad3-467d-80bc-9252982e9dd8": "Allegro miniKurier24 InPost",
} as const;

export function officialWzADeliveryMethodName(id?: string): string | undefined {
  const key = id?.trim() ?? "";
  if (!key) {
    return undefined;
  }
  return OFFICIAL_WZA_DELIVERY_METHODS[key as keyof typeof OFFICIAL_WZA_DELIVERY_METHODS];
}

export function officialWzADeliveryMethodID(name?: string): string | undefined {
  const want = name?.trim() ?? "";
  if (!want) {
    return undefined;
  }
  for (const [id, officialName] of Object.entries(OFFICIAL_WZA_DELIVERY_METHODS)) {
    if (officialName === want) {
      return id;
    }
  }
  return undefined;
}

// Prefer GET delivery-proposals.suggestedInput.deliveryMethodId. If that is
// empty, name the checkout method when it is an official WzA method or the
// exact id appears on GET delivery-services. A catalog row (Kurier One / WEDO)
// is never a substitute.
export function resolveWzACreateDeliveryMethod(input: {
  proposedDeliveryMethodId?: string;
  catalogFallbackId?: string;
  catalogServiceIds?: string[];
  checkoutMethodId?: string;
  checkoutMethodName?: string;
}): WzACreateDecision {
  const proposed = input.proposedDeliveryMethodId?.trim() ?? "";
  if (proposed) {
    return { ok: true, deliveryMethodId: proposed };
  }
  const checkoutMethodId = input.checkoutMethodId?.trim() ?? "";
  const checkoutMethodName = input.checkoutMethodName?.trim() ?? "";
  if (officialWzADeliveryMethodName(checkoutMethodId)) {
    return { ok: true, deliveryMethodId: checkoutMethodId };
  }
  const officialFromName = officialWzADeliveryMethodID(checkoutMethodName);
  if (officialFromName) {
    return { ok: true, deliveryMethodId: officialFromName };
  }
  if (
    checkoutMethodId &&
    (input.catalogServiceIds ?? []).some((id) => id.trim() === checkoutMethodId)
  ) {
    return { ok: true, deliveryMethodId: checkoutMethodId };
  }
  return {
    ok: false,
    reason: "no_proposal_method",
    checkoutMethodId,
    checkoutMethodName,
  };
}

export function checkoutMethodLabel(decision: Extract<WzACreateDecision, { ok: false }>): string {
  if (decision.checkoutMethodName && decision.checkoutMethodId) {
    return `${decision.checkoutMethodName} (${decision.checkoutMethodId})`;
  }
  return decision.checkoutMethodName || decision.checkoutMethodId;
}
