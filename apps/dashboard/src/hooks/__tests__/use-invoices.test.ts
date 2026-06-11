import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import {
  useCreateInvoice,
  useOrderInvoices,
  isActiveInvoice,
} from "@/hooks/use-invoices";
import type { Invoice } from "@/types/api";

const API_BASE = "*/v1";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

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

describe("isActiveInvoice", () => {
  it("treats cancelled and error invoices as inactive, everything else active", () => {
    expect(isActiveInvoice(makeInvoice("draft"))).toBe(true);
    expect(isActiveInvoice(makeInvoice("issued"))).toBe(true);
    expect(isActiveInvoice(makeInvoice("paid"))).toBe(true);
    expect(isActiveInvoice(makeInvoice("cancelled"))).toBe(false);
    expect(isActiveInvoice(makeInvoice("error"))).toBe(false);
  });
});

describe("useOrderInvoices", () => {
  it("GETs /v1/orders/{id}/invoices", async () => {
    let path = "";
    server.use(
      http.get(`${API_BASE}/orders/:id/invoices`, ({ params }) => {
        path = `/v1/orders/${params.id}/invoices`;
        return HttpResponse.json([makeInvoice("issued")]);
      })
    );

    const { result } = renderHook(() => useOrderInvoices("ord-1"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(path).toBe("/v1/orders/ord-1/invoices");
    expect(result.current.data).toHaveLength(1);
  });

  it("does not fetch when disabled", () => {
    const { result } = renderHook(
      () => useOrderInvoices("ord-1", { enabled: false }),
      { wrapper: createWrapper() }
    );
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useCreateInvoice", () => {
  it("POSTs to /v1/invoices with order_id", async () => {
    let method = "";
    let body: { order_id?: string } = {};
    server.use(
      http.post(`${API_BASE}/invoices`, async ({ request }) => {
        method = request.method;
        body = (await request.json()) as { order_id?: string };
        return HttpResponse.json(makeInvoice("draft"), { status: 201 });
      })
    );

    const { result } = renderHook(() => useCreateInvoice(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({
      order_id: "ord-1",
      provider: "fakturownia",
      customer_name: "Jan Kowalski",
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(method).toBe("POST");
    expect(body.order_id).toBe("ord-1");
  });
});
