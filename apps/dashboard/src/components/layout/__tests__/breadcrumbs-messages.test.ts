import { describe, expect, it } from "vitest";

import enLayoutMessages from "../../../../messages/en/layout.json";
import plLayoutMessages from "../../../../messages/pl/layout.json";

describe("breadcrumb messages", () => {
  it("localizes the client-ready carrier routes", () => {
    expect(plLayoutMessages.breadcrumbs.carriers).toBe("Kurierzy");
    expect(enLayoutMessages.breadcrumbs.carriers).toBe("Carriers");
  });
});
