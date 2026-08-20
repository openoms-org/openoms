import { describe, expect, it } from "vitest";
import { allegroSalesCenterCreateShipmentURL } from "../allegro-sales-center";

const checkoutFormId = "19829450-9c54-11f1-bd08-9328d2ed1733";
const sellerId = "110974929";

describe("allegroSalesCenterCreateShipmentURL", () => {
  it("uses the production Sales Center host when the integration is not sandbox", () => {
    const url = allegroSalesCenterCreateShipmentURL({
      checkoutFormId,
      sellerId,
      sandbox: false,
    });

    expect(url).toBe(
      "https://salescenter.allegro.com/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
    expect(url).not.toContain("allegrosandbox.pl");
    expect(url).not.toContain("nadaj-paczke");
    expect(url).not.toContain("orderId=");
  });

  it("uses the sandbox Sales Center host when the integration is sandbox", () => {
    const url = allegroSalesCenterCreateShipmentURL({
      checkoutFormId,
      sellerId,
      sandbox: true,
    });

    expect(url).toBe(
      "https://salescenter.allegro.com.allegrosandbox.pl/ship-with-allegro/swa/create-shipment/19829450-9c54-11f1-bd08-9328d2ed1733?sellerId=110974929"
    );
    expect(url).not.toContain("nadaj-paczke");
    expect(url).not.toContain("orderId=");
  });
});
