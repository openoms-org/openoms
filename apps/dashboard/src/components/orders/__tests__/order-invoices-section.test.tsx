import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import {
  renderWithProviders,
  screen,
  waitFor,
  userEvent,
} from "@/test/test-utils";
import { OrderInvoicesSection } from "@/components/orders/order-invoices-section";
import type { Invoice, Order } from "@/types/api";

const API_BASE = "*/v1";

const order: Order = {
  id: "ord-1",
  tenant_id: "t-1",
  source: "manual",
  status: "new",
  customer_name: "Jan Kowalski",
  customer_email: "jan@example.com",
  total_amount: 100,
  currency: "PLN",
  tags: [],
  payment_status: "paid",
  payment_method: "card",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

function makeInvoice(status: string): Invoice {
  return {
    id: `inv-${status}`,
    tenant_id: "t-1",
    order_id: "ord-1",
    provider: "fakturownia",
    status,
    invoice_type: "vat",
    currency: "PLN",
    metadata: {},
    ksef_status: "none",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  };
}

function mockInvoicing(provider: string) {
  server.use(
    http.get(`${API_BASE}/settings/invoicing`, () =>
      HttpResponse.json({
        provider,
        auto_create_on_status: [],
        default_tax_rate: 23,
        payment_days: 14,
        credentials: {},
      })
    )
  );
}

function mockOrderInvoices(invoices: Invoice[]) {
  server.use(
    http.get(`${API_BASE}/orders/:id/invoices`, () =>
      HttpResponse.json(invoices)
    )
  );
}

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("OrderInvoicesSection idempotency guard", () => {
  it("hides the issue button when an active invoice already exists", async () => {
    mockInvoicing("fakturownia");
    mockOrderInvoices([makeInvoice("issued")]);

    renderWithProviders(<OrderInvoicesSection order={order} />);

    // Wait for the invoice list to load (the active-invoice notice appears).
    await waitFor(() => expect(screen.getByText("activeExists")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /issue/i })).not.toBeInTheDocument();
  });

  it("shows the issue button when the only invoice is cancelled", async () => {
    mockInvoicing("fakturownia");
    mockOrderInvoices([makeInvoice("cancelled")]);

    renderWithProviders(<OrderInvoicesSection order={order} />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /issue/i })).toBeInTheDocument()
    );
  });

  it("issues an invoice via POST /v1/invoices with order_id when none is active", async () => {
    mockInvoicing("fakturownia");
    mockOrderInvoices([]);

    let posted: { order_id?: string; provider?: string } | null = null;
    server.use(
      http.post(`${API_BASE}/invoices`, async ({ request }) => {
        posted = (await request.json()) as { order_id?: string; provider?: string };
        return HttpResponse.json(makeInvoice("draft"), { status: 201 });
      })
    );

    const user = userEvent.setup();
    renderWithProviders(<OrderInvoicesSection order={order} />);

    const issueBtn = await screen.findByRole("button", { name: /^issue$/i });
    await user.click(issueBtn);

    const submitBtn = await screen.findByRole("button", { name: /submit/i });
    await user.click(submitBtn);

    await waitFor(() => expect(posted).not.toBeNull());
    expect(posted!.order_id).toBe("ord-1");
    expect(posted!.provider).toBe("fakturownia");
  });

  it("disables the issue button when no invoicing provider is configured", async () => {
    mockInvoicing("");
    mockOrderInvoices([]);

    renderWithProviders(<OrderInvoicesSection order={order} />);

    const issueBtn = await screen.findByRole("button", { name: /^issue$/i });
    expect(issueBtn).toBeDisabled();
  });
});
