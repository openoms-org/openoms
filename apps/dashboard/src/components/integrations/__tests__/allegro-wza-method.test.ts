import { describe, expect, it } from "vitest";
import {
  checkoutMethodLabel,
  resolveWzACreateDeliveryMethod,
} from "../allegro-wza-method";

describe("resolveWzACreateDeliveryMethod", () => {
  it("uses only suggestedInput.deliveryMethodId when Allegro proposes one", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "c3066682-97a3-42fe-9eb5-3beeccab840c",
      checkoutMethodId: "checkout-only-id",
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });

    expect(decision).toEqual({
      ok: true,
      deliveryMethodId: "c3066682-97a3-42fe-9eb5-3beeccab840c",
    });
  });

  it("fails closed when proposals return an empty method and does not use a catalog fallback", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "",
      catalogFallbackId: "kurier-one-wedo",
      checkoutMethodId: "c3066682-97a3-42fe-9eb5-3beeccab840c",
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });

    expect(decision).toEqual({
      ok: false,
      reason: "no_proposal_method",
      checkoutMethodId: "c3066682-97a3-42fe-9eb5-3beeccab840c",
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });
  });

  it("fails closed when checkout metadata also lacks a method id", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "   ",
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });

    expect(decision.ok).toBe(false);
    if (decision.ok) {
      return;
    }
    expect(decision.reason).toBe("no_proposal_method");
    expect(decision.checkoutMethodName).toBe("Allegro miniKurier24 InPost");
    expect(decision.checkoutMethodId).toBe("");
  });

  it("labels the persisted checkout method for the empty state", () => {
    expect(
      checkoutMethodLabel({
        ok: false,
        reason: "no_proposal_method",
        checkoutMethodId: "c3066682-97a3-42fe-9eb5-3beeccab840c",
        checkoutMethodName: "Allegro miniKurier24 InPost",
      })
    ).toBe("Allegro miniKurier24 InPost (c3066682-97a3-42fe-9eb5-3beeccab840c)");
  });
});
