import { describe, expect, it } from "vitest";
import { mapCreateOrderRequestToUpdateOrderRequest } from "../order-request-mappers";
import type { CreateOrderRequest } from "@/types/api";

describe("mapCreateOrderRequestToUpdateOrderRequest", () => {
  it("preserves fields supported by order updates", () => {
    const createRequest: CreateOrderRequest = {
      source: "manual",
      external_id: "EXT-123",
      customer_name: "Anna Nowak",
      customer_email: "anna@example.com",
      customer_phone: "+48123123123",
      shipping_address: {
        name: "Anna Nowak",
        street: "Prosta 1",
        city: "Warszawa",
        postal_code: "00-001",
        country: "PL",
      },
      billing_address: {
        name: "OpenOMS Sp. z o.o.",
        street: "Firmowa 2",
        city: "Krakow",
        postal_code: "30-001",
        country: "PL",
      },
      items: [
        {
          name: "Produkt",
          sku: "SKU-1",
          quantity: 2,
          price: 19.99,
        },
      ],
      total_amount: 39.98,
      currency: "PLN",
      notes: "Notatka klienta",
      internal_notes: "Notatka wewnetrzna",
      priority: "high",
      metadata: { source_note: "manual import" },
      tags: ["vip"],
      delivery_method: "Paczkomat InPost",
      pickup_point_id: "WAW01A",
      payment_status: "paid",
      payment_method: "blik",
    };

    const updateRequest = mapCreateOrderRequestToUpdateOrderRequest(createRequest);

    expect(updateRequest).toEqual({
      external_id: "EXT-123",
      customer_name: "Anna Nowak",
      customer_email: "anna@example.com",
      customer_phone: "+48123123123",
      shipping_address: createRequest.shipping_address,
      billing_address: createRequest.billing_address,
      items: createRequest.items,
      total_amount: 39.98,
      currency: "PLN",
      notes: "Notatka klienta",
      internal_notes: "Notatka wewnetrzna",
      priority: "high",
      metadata: { source_note: "manual import" },
      tags: ["vip"],
      delivery_method: "Paczkomat InPost",
      pickup_point_id: "WAW01A",
      payment_status: "paid",
      payment_method: "blik",
    });
  });

  it("omits create-only fields from edit payloads", () => {
    const createRequest: CreateOrderRequest = {
      source: "allegro",
      integration_id: "integration-1",
      ordered_at: "2026-05-17T10:00:00Z",
      shipment_provider: "inpost",
      auto_create_shipment: true,
      customer_name: "Jan Kowalski",
      total_amount: 10,
    };

    const updateRequest = mapCreateOrderRequestToUpdateOrderRequest(createRequest);

    expect(updateRequest).not.toHaveProperty("source");
    expect(updateRequest).not.toHaveProperty("integration_id");
    expect(updateRequest).not.toHaveProperty("ordered_at");
    expect(updateRequest).not.toHaveProperty("shipment_provider");
    expect(updateRequest).not.toHaveProperty("auto_create_shipment");
    expect(updateRequest).toEqual({
      customer_name: "Jan Kowalski",
      total_amount: 10,
    });
  });
});
