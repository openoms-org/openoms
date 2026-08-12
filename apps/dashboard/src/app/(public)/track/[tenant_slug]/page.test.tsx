import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, userEvent } from "@/test/test-utils";
import TrackingPage from "./page";

vi.mock("next-intl", () => ({
  useTranslations: (namespace?: string) => (key: string) =>
    namespace === "statuses" && key === "shipment.label_ready"
      ? "Etykieta gotowa"
      : key,
}));

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

  async function lookup(overrides: Record<string, unknown>) {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        order_number: "order-123",
        status: "shipped",
        status_label: "Wysłane",
        customer_name: "Jan Kowalski",
        created_at: "2026-05-07T10:00:00Z",
        updated_at: "2026-05-07T10:00:00Z",
        total_amount: 123.45,
        currency: "PLN",
        items: [],
        shipments: [],
        timeline: [],
        ...overrides,
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<TrackingPage />);
    await userEvent.type(screen.getByLabelText("Numer zamowienia (ID)"), "order-123");
    await userEvent.type(screen.getByLabelText("Adres email"), "customer@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Sprawdz status" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
  }

  it("renders the order badge with the server label and the shared status colour", async () => {
    await lookup({});

    const badge = await screen.findByText("Wysłane");
    // ORDER_STATUSES.shipped
    expect(badge).toHaveClass("bg-violet-100");
    expect(badge).toHaveClass("text-violet-800");
  });

  it("keeps the server label for a tenant's custom status the catalog does not know", async () => {
    await lookup({ status: "awaiting_parts", status_label: "Czeka na części" });

    expect(await screen.findByText("Czeka na części")).toBeInTheDocument();
    expect(screen.queryByText("awaiting_parts")).not.toBeInTheDocument();
  });

  it("translates shipment badges from the shipment catalog", async () => {
    await lookup({
      shipments: [{ tracking_number: "TRK-1", carrier: "inpost", status: "label_ready" }],
    });

    const badge = await screen.findByText("Etykieta gotowa");
    // SHIPMENT_STATUSES.label_ready — a status the old local map had no colour for
    expect(badge).toHaveClass("bg-indigo-100");
    expect(badge).toHaveClass("text-indigo-800");
  });
});
