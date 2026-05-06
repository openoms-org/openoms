import { describe, expect, it } from "vitest";

import { getSupplierPortalTokenHandoff } from "@/lib/supplier-portal-token";

describe("getSupplierPortalTokenHandoff", () => {
  it("extracts the portal token from the URL fragment", () => {
    const handoff = getSupplierPortalTokenHandoff("https://app.openoms.org/supplier-portal#token=abc123");

    expect(handoff.token).toBe("abc123");
    expect(handoff.cleanURL).toBe("/supplier-portal");
  });

  it("does not accept query-string portal tokens", () => {
    const handoff = getSupplierPortalTokenHandoff("https://app.openoms.org/supplier-portal?token=abc123");

    expect(handoff.token).toBe("");
    expect(handoff.cleanURL).toBe("/supplier-portal");
  });

  it("preserves unrelated URL state while removing token material", () => {
    const handoff = getSupplierPortalTokenHandoff(
      "https://app.openoms.org/supplier-portal?lang=pl&token=leaked#token=abc123&view=orders"
    );

    expect(handoff.token).toBe("abc123");
    expect(handoff.cleanURL).toBe("/supplier-portal?lang=pl#view=orders");
  });
});
