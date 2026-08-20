export type WzACreateDecision =
  | { ok: true; deliveryMethodId: string }
  | {
      ok: false;
      reason: "no_proposal_method";
      checkoutMethodId: string;
      checkoutMethodName: string;
    };

// Only GET delivery-proposals.suggestedInput.deliveryMethodId may be sent to
// create-commands. Checkout metadata is shown in the empty state. A catalog
// row (Kurier One / WEDO) is never a substitute.
export function resolveWzACreateDeliveryMethod(input: {
  proposedDeliveryMethodId?: string;
  catalogFallbackId?: string;
  checkoutMethodId?: string;
  checkoutMethodName?: string;
}): WzACreateDecision {
  const proposed = input.proposedDeliveryMethodId?.trim() ?? "";
  if (proposed) {
    return { ok: true, deliveryMethodId: proposed };
  }
  return {
    ok: false,
    reason: "no_proposal_method",
    checkoutMethodId: input.checkoutMethodId?.trim() ?? "",
    checkoutMethodName: input.checkoutMethodName?.trim() ?? "",
  };
}

export function checkoutMethodLabel(decision: Extract<WzACreateDecision, { ok: false }>): string {
  if (decision.checkoutMethodName && decision.checkoutMethodId) {
    return `${decision.checkoutMethodName} (${decision.checkoutMethodId})`;
  }
  return decision.checkoutMethodName || decision.checkoutMethodId;
}
