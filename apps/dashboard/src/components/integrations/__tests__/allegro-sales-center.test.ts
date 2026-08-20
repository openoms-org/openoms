import { describe, expect, it } from "vitest";
import {
  allegroSalesCenterCreateShipmentURL,
  resolveWzASalesCenterURL,
} from "../allegro-sales-center";

const checkoutFormId = "19829450-9c54-11f1-bd08-9328d2ed1733";
const sellerId = "110974929";

describe("allegroSalesCenterCreateShipmentURL", () => {
  it("builds the sandbox Sales Center create-shipment URL for a checkout", () => {
    expect(
      allegroSalesCenterCreateShipmentURL({
        checkoutFormId,
        sellerId,
        sandbox: true,
      })
    ).toBe(
      "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
  });

  it("builds the production Sales Center host without the sandbox suffix", () => {
    expect(
      allegroSalesCenterCreateShipmentURL({
        checkoutFormId,
        sellerId,
        sandbox: false,
      })
    ).toBe(
      "https://salescenter.allegro.com/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
  });

  it("does not use the marketplace nadaj-paczke orderId pattern", () => {
    const url = allegroSalesCenterCreateShipmentURL({
      checkoutFormId,
      sellerId,
      sandbox: true,
    });
    expect(url).not.toContain("nadaj-paczke");
    expect(url).not.toContain("orderId=");
  });

  it("returns empty when checkoutFormId or sellerId is missing", () => {
    expect(
      allegroSalesCenterCreateShipmentURL({
        checkoutFormId: "",
        sellerId,
        sandbox: true,
      })
    ).toBe("");
    expect(
      allegroSalesCenterCreateShipmentURL({
        checkoutFormId,
        sellerId: "  ",
        sandbox: false,
      })
    ).toBe("");
  });
});

describe("resolveWzASalesCenterURL", () => {
  it("prefers the URL attached to empty-method delivery-proposals", () => {
    expect(
      resolveWzASalesCenterURL({
        proposalsUrl:
          "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929",
        checkoutFormId,
        sellerId: "other",
        sandbox: false,
      })
    ).toBe(
      "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
  });

  it("builds from checkoutFormId and sellerId when proposals omitted the link", () => {
    expect(
      resolveWzASalesCenterURL({
        checkoutFormId,
        sellerId,
        sandbox: true,
      })
    ).toBe(
      "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
  });

  it("does not guess a host before sandbox is known", () => {
    expect(
      resolveWzASalesCenterURL({
        checkoutFormId,
        sellerId,
      })
    ).toBe("");
  });
});
