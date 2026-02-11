import { describe, it, expect } from "vitest";
import {
  ORDER_STATUSES,
  ORDER_TRANSITIONS,
  SHIPMENT_STATUSES,
  SHIPMENT_TRANSITIONS,
  RETURN_STATUSES,
  RETURN_TRANSITIONS,
  INTEGRATION_STATUSES,
  ROLES,
  ORDER_SOURCES,
  PAYMENT_STATUSES,
  PAYMENT_METHODS,
  SHIPMENT_PROVIDERS,
  INTEGRATION_PROVIDERS,
  ORDER_SOURCE_LABELS,
  SHIPMENT_PROVIDER_LABELS,
  INTEGRATION_PROVIDER_LABELS,
  INVOICE_STATUS_MAP,
  INVOICE_TYPE_LABELS,
  AUTOMATION_TRIGGER_EVENTS,
  AUTOMATION_TRIGGER_LABELS,
  AUTOMATION_ACTION_TYPES,
  AUTOMATION_ACTION_LABELS,
} from "@/lib/constants";

describe("ORDER_STATUSES", () => {
  it("has all expected status keys", () => {
    const expectedKeys = [
      "new", "confirmed", "processing", "ready_to_ship",
      "shipped", "in_transit", "out_for_delivery", "delivered",
      "completed", "on_hold", "cancelled", "refunded",
    ];
    for (const key of expectedKeys) {
      expect(ORDER_STATUSES).toHaveProperty(key);
    }
  });

  it("each status has label and color", () => {
    for (const [, value] of Object.entries(ORDER_STATUSES)) {
      expect(value).toHaveProperty("label");
      expect(value).toHaveProperty("color");
      expect(typeof value.label).toBe("string");
      expect(typeof value.color).toBe("string");
      expect(value.label.length).toBeGreaterThan(0);
      expect(value.color.length).toBeGreaterThan(0);
    }
  });
});

describe("ORDER_TRANSITIONS", () => {
  it("has transitions for all order statuses", () => {
    for (const key of Object.keys(ORDER_STATUSES)) {
      expect(ORDER_TRANSITIONS).toHaveProperty(key);
      expect(Array.isArray(ORDER_TRANSITIONS[key])).toBe(true);
    }
  });

  it("refunded has no transitions (terminal state)", () => {
    expect(ORDER_TRANSITIONS.refunded).toEqual([]);
  });

  it("new can transition to confirmed, cancelled, on_hold", () => {
    expect(ORDER_TRANSITIONS.new).toContain("confirmed");
    expect(ORDER_TRANSITIONS.new).toContain("cancelled");
    expect(ORDER_TRANSITIONS.new).toContain("on_hold");
  });
});

describe("SHIPMENT_STATUSES", () => {
  it("has expected keys", () => {
    const expectedKeys = [
      "created", "label_ready", "picked_up", "in_transit",
      "out_for_delivery", "delivered", "returned", "failed",
    ];
    for (const key of expectedKeys) {
      expect(SHIPMENT_STATUSES).toHaveProperty(key);
    }
  });

  it("each status has label and color", () => {
    for (const [, value] of Object.entries(SHIPMENT_STATUSES)) {
      expect(value).toHaveProperty("label");
      expect(value).toHaveProperty("color");
    }
  });
});

describe("SHIPMENT_TRANSITIONS", () => {
  it("returned has no transitions (terminal state)", () => {
    expect(SHIPMENT_TRANSITIONS.returned).toEqual([]);
  });
});

describe("RETURN_STATUSES", () => {
  it("has expected keys", () => {
    const expectedKeys = ["requested", "approved", "received", "refunded", "rejected", "cancelled"];
    for (const key of expectedKeys) {
      expect(RETURN_STATUSES).toHaveProperty(key);
    }
  });
});

describe("RETURN_TRANSITIONS", () => {
  it("refunded has no transitions (terminal state)", () => {
    expect(RETURN_TRANSITIONS.refunded).toEqual([]);
  });

  it("rejected has no transitions (terminal state)", () => {
    expect(RETURN_TRANSITIONS.rejected).toEqual([]);
  });
});

describe("INTEGRATION_STATUSES", () => {
  it("has active, inactive, and error statuses", () => {
    expect(INTEGRATION_STATUSES).toHaveProperty("active");
    expect(INTEGRATION_STATUSES).toHaveProperty("inactive");
    expect(INTEGRATION_STATUSES).toHaveProperty("error");
  });
});

describe("ROLES", () => {
  it("has owner, admin, and member", () => {
    expect(ROLES).toHaveProperty("owner");
    expect(ROLES).toHaveProperty("admin");
    expect(ROLES).toHaveProperty("member");
  });
});

describe("ORDER_SOURCES", () => {
  it("contains manual and marketplace sources", () => {
    expect(ORDER_SOURCES).toContain("manual");
    expect(ORDER_SOURCES).toContain("allegro");
    expect(ORDER_SOURCES).toContain("amazon");
  });
});

describe("PAYMENT_STATUSES", () => {
  it("has expected keys", () => {
    expect(PAYMENT_STATUSES).toHaveProperty("pending");
    expect(PAYMENT_STATUSES).toHaveProperty("paid");
    expect(PAYMENT_STATUSES).toHaveProperty("refunded");
  });
});

describe("PAYMENT_METHODS", () => {
  it("is a non-empty array", () => {
    expect(PAYMENT_METHODS.length).toBeGreaterThan(0);
  });
});

describe("Provider labels", () => {
  it("ORDER_SOURCE_LABELS has labels for all ORDER_SOURCES", () => {
    for (const source of ORDER_SOURCES) {
      expect(ORDER_SOURCE_LABELS).toHaveProperty(source);
      expect(typeof ORDER_SOURCE_LABELS[source]).toBe("string");
    }
  });

  it("SHIPMENT_PROVIDER_LABELS has labels for all SHIPMENT_PROVIDERS", () => {
    for (const provider of SHIPMENT_PROVIDERS) {
      expect(SHIPMENT_PROVIDER_LABELS).toHaveProperty(provider);
    }
  });

  it("INTEGRATION_PROVIDER_LABELS has labels for all INTEGRATION_PROVIDERS", () => {
    for (const provider of INTEGRATION_PROVIDERS) {
      expect(INTEGRATION_PROVIDER_LABELS).toHaveProperty(provider);
    }
  });
});

describe("INVOICE_STATUS_MAP", () => {
  it("has expected keys", () => {
    expect(INVOICE_STATUS_MAP).toHaveProperty("draft");
    expect(INVOICE_STATUS_MAP).toHaveProperty("issued");
    expect(INVOICE_STATUS_MAP).toHaveProperty("paid");
    expect(INVOICE_STATUS_MAP).toHaveProperty("cancelled");
  });
});

describe("INVOICE_TYPE_LABELS", () => {
  it("has labels for invoice types", () => {
    expect(INVOICE_TYPE_LABELS).toHaveProperty("vat");
    expect(INVOICE_TYPE_LABELS).toHaveProperty("proforma");
    expect(INVOICE_TYPE_LABELS).toHaveProperty("correction");
    expect(INVOICE_TYPE_LABELS).toHaveProperty("receipt");
  });
});

describe("Automation constants", () => {
  it("AUTOMATION_TRIGGER_EVENTS is non-empty", () => {
    expect(AUTOMATION_TRIGGER_EVENTS.length).toBeGreaterThan(0);
  });

  it("AUTOMATION_TRIGGER_LABELS has labels for all events", () => {
    for (const event of AUTOMATION_TRIGGER_EVENTS) {
      expect(AUTOMATION_TRIGGER_LABELS).toHaveProperty(event);
    }
  });

  it("AUTOMATION_ACTION_TYPES is non-empty", () => {
    expect(AUTOMATION_ACTION_TYPES.length).toBeGreaterThan(0);
  });

  it("AUTOMATION_ACTION_LABELS has labels for all action types", () => {
    for (const action of AUTOMATION_ACTION_TYPES) {
      expect(AUTOMATION_ACTION_LABELS).toHaveProperty(action);
    }
  });
});
