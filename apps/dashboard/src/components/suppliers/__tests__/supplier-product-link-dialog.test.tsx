import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { mockProducts } from "@/test/handlers";
import {
  renderWithProviders,
  screen,
  waitFor,
  userEvent,
} from "@/test/test-utils";
import { SupplierProductLinkDialog } from "@/components/suppliers/supplier-product-link-dialog";
import type { SupplierProduct } from "@/types/api";

const API_BASE = "*/v1";

const supplierProduct = {
  id: "sp-9",
  supplier_id: "sup-1",
  name: "Acme Widget",
  ean: "5901234123457",
  sku: "AW-1",
  price: 12,
  stock_quantity: 3,
  metadata: {},
  external_id: "ext-1",
} as unknown as SupplierProduct;

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("SupplierProductLinkDialog", () => {
  it("links the selected catalog product via POST .../link with product_id", async () => {
    server.use(
      http.get(`${API_BASE}/products`, () =>
        HttpResponse.json({
          items: mockProducts,
          total: mockProducts.length,
          limit: 20,
          offset: 0,
        })
      )
    );

    let posted: { product_id?: string } | null = null;
    let linkPath = "";
    server.use(
      http.post(
        `${API_BASE}/suppliers/:supplierId/products/:spId/link`,
        async ({ request, params }) => {
          posted = (await request.json()) as { product_id?: string };
          linkPath = `/v1/suppliers/${params.supplierId}/products/${params.spId}/link`;
          return HttpResponse.json({ id: params.spId, product_id: posted.product_id });
        }
      )
    );

    const user = userEvent.setup();
    renderWithProviders(
      <SupplierProductLinkDialog
        supplierId="sup-1"
        supplierProduct={supplierProduct}
        onClose={() => {}}
      />
    );

    // The first catalog product row renders with a "Link" action button.
    const linkButtons = await screen.findAllByRole("button", { name: /linkAction/i });
    await user.click(linkButtons[0]);

    await waitFor(() => expect(posted).not.toBeNull());
    expect(linkPath).toBe("/v1/suppliers/sup-1/products/sp-9/link");
    expect(posted!.product_id).toBe(mockProducts[0].id);
  });
});
