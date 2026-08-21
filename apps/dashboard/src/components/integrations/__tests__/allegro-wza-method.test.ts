import { describe, expect, it } from "vitest";
import {
  checkoutMethodLabel,
  officialWzADeliveryMethodID,
  officialWzADeliveryMethodName,
  resolveWzACreateDeliveryMethod,
} from "../allegro-wza-method";

const MINI_KURIER24_INPOST_ID = "9081532b-5ad3-467d-80bc-9252982e9dd8";

describe("official WzA delivery method map", () => {
  it("maps Allegro miniKurier24 InPost id 9081532b to its official name", () => {
    expect(officialWzADeliveryMethodName(MINI_KURIER24_INPOST_ID)).toBe(
      "Allegro miniKurier24 InPost"
    );
    expect(officialWzADeliveryMethodID("Allegro miniKurier24 InPost")).toBe(
      MINI_KURIER24_INPOST_ID
    );
    expect(officialWzADeliveryMethodName("c3066682-97a3-42fe-9eb5-3beeccab840c")).toBe(
      undefined
    );
    expect(officialWzADeliveryMethodID("Kurier One WEDO")).toBeUndefined();
  });
});

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
      checkoutMethodId: "unknown-checkout-method",
      checkoutMethodName: "Some other Allegro method",
    });

    expect(decision).toEqual({
      ok: false,
      reason: "no_proposal_method",
      checkoutMethodId: "unknown-checkout-method",
      checkoutMethodName: "Some other Allegro method",
    });
  });

  it("names official miniKurier24 InPost from checkout when proposals have no method", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "",
      catalogFallbackId: "kurier-one-wedo",
      checkoutMethodId: MINI_KURIER24_INPOST_ID,
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });

    expect(decision).toEqual({
      ok: true,
      deliveryMethodId: MINI_KURIER24_INPOST_ID,
    });
  });

  it("names official miniKurier24 InPost from the checkout method name alone", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "",
      checkoutMethodName: "Allegro miniKurier24 InPost",
    });

    expect(decision).toEqual({
      ok: true,
      deliveryMethodId: MINI_KURIER24_INPOST_ID,
    });
  });

  it("names the checkout method when delivery-services lists that exact id", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "",
      catalogFallbackId: "kurier-one-wedo",
      checkoutMethodId: MINI_KURIER24_INPOST_ID,
      catalogServiceIds: ["allegro-kurier-one-wedo", MINI_KURIER24_INPOST_ID],
    });

    expect(decision).toEqual({
      ok: true,
      deliveryMethodId: MINI_KURIER24_INPOST_ID,
    });
  });

  it("fails closed when checkout metadata also lacks a method id", () => {
    const decision = resolveWzACreateDeliveryMethod({
      proposedDeliveryMethodId: "   ",
      checkoutMethodName: "Unknown Allegro method",
    });

    expect(decision.ok).toBe(false);
    if (decision.ok) {
      return;
    }
    expect(decision.reason).toBe("no_proposal_method");
    expect(decision.checkoutMethodName).toBe("Unknown Allegro method");
    expect(decision.checkoutMethodId).toBe("");
  });

  it("labels the persisted checkout method for the empty state", () => {
    expect(
      checkoutMethodLabel({
        ok: false,
        reason: "no_proposal_method",
        checkoutMethodId: MINI_KURIER24_INPOST_ID,
        checkoutMethodName: "Allegro miniKurier24 InPost",
      })
    ).toBe(`Allegro miniKurier24 InPost (${MINI_KURIER24_INPOST_ID})`);
  });
});
