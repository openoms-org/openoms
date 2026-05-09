import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, userEvent } from "@/test/test-utils";
import TrackingPage from "./page";

vi.mock("next/navigation", () => ({
  useParams: () => ({ tenant_slug: "mercpart" }),
}));

describe("TrackingPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("submits tracking lookup with email in POST body instead of URL query string", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        order_number: "order-123",
        status: "new",
        status_label: "Nowe",
        customer_name: "Jan Kowalski",
        created_at: "2026-05-07T10:00:00Z",
        updated_at: "2026-05-07T10:00:00Z",
        total_amount: 123.45,
        currency: "PLN",
        items: [],
        shipments: [],
        timeline: [],
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<TrackingPage />);

    await userEvent.type(screen.getByLabelText("Numer zamowienia (ID)"), " order-123 ");
    await userEvent.type(screen.getByLabelText("Adres email"), " customer@example.com ");
    await userEvent.click(screen.getByRole("button", { name: "Sprawdz status" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];

    expect(url).toBe("/v1/tracking/mercpart/order-123");
    expect(url).not.toContain("email=");
    expect(init).toMatchObject({
      method: "POST",
      headers: { "Content-Type": "application/json" },
    });
    expect(JSON.parse(init.body as string)).toEqual({ email: "customer@example.com" });
  });
});
