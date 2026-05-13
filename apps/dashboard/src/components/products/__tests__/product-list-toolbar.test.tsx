import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ProductListToolbar } from "@/components/products/product-list-toolbar";
import { Button } from "@/components/ui/button";

describe("ProductListToolbar", () => {
  it("keeps product filters and actions in separate wrapping regions", () => {
    render(
      <ProductListToolbar
        filters={
          <>
            <input aria-label="Search products" />
            <button type="button">Category</button>
          </>
        }
        actions={
          <>
            <Button variant="outline">Export CSV</Button>
            <Button>Add product</Button>
          </>
        }
      />
    );

    expect(screen.getByTestId("product-list-toolbar-filters")).toHaveClass(
      "flex-wrap",
      "min-w-0"
    );
    expect(screen.getByTestId("product-list-toolbar-actions")).toHaveClass(
      "flex-wrap",
      "sm:justify-end"
    );
    expect(screen.getByRole("button", { name: "Add product" })).toBeVisible();
  });
});
