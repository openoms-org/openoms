import { describe, expect, it } from "vitest";

import enLayoutMessages from "../../../../messages/en/layout.json";
import plLayoutMessages from "../../../../messages/pl/layout.json";

describe("breadcrumb messages", () => {
  it("localizes the client-ready carrier routes", () => {
    expect(plLayoutMessages.breadcrumbs.carriers).toBe("Kurierzy");
    expect(enLayoutMessages.breadcrumbs.carriers).toBe("Carriers");
  });

  it("localizes the client-ready marketplace routes", () => {
    expect(plLayoutMessages.breadcrumbs.marketplaces).toBe("Marketplace");
    expect(enLayoutMessages.breadcrumbs.marketplaces).toBe("Marketplaces");
  });
});
