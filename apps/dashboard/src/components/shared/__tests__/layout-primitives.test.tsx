import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Surface } from "@/components/shared/surface";
import { PageSection } from "@/components/shared/page-section";
import { ActionBar } from "@/components/shared/action-bar";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { PackagePlus } from "lucide-react";

describe("layout primitives", () => {
  it("renders Surface as a named region when aria-label is provided", () => {
    render(
      <Surface aria-label="Order summary">
        <p>Summary content</p>
      </Surface>
    );

    expect(screen.getByRole("region", { name: "Order summary" })).toBeInTheDocument();
    expect(screen.getByText("Summary content")).toBeInTheDocument();
  });

  it("preserves an explicit Surface role when an accessible name is provided", () => {
    render(
      <Surface role="status" aria-label="Sync status">
        Synced
      </Surface>
    );

    expect(screen.getByRole("status", { name: "Sync status" })).toHaveTextContent("Synced");
  });

  it("renders PageSection with title, description, actions, and children", () => {
    render(
      <PageSection
        title="Shipments"
        description="Labels and tracking"
        actions={<Button>New shipment</Button>}
      >
        <p>Shipment table</p>
      </PageSection>
    );

    expect(screen.getByRole("heading", { name: "Shipments" })).toBeInTheDocument();
    expect(screen.getByText("Labels and tracking")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New shipment" })).toBeInTheDocument();
    expect(screen.getByText("Shipment table")).toBeInTheDocument();
  });

  it("renders ActionBar with primary and secondary action areas", () => {
    render(
      <ActionBar
        secondary={<Button variant="outline">Cancel</Button>}
        primary={<Button>Save</Button>}
      />
    );

    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
  });

  it("renders PageHeader with legacy action and additional action slots", () => {
    render(
      <PageHeader
        title="Products"
        description="Catalog orchestration"
        action={{ label: "Add product", href: "/products/new" }}
        actions={<Button variant="outline">Import</Button>}
        meta={<span>12 active</span>}
      />
    );

    expect(screen.getByRole("heading", { name: "Products" })).toBeInTheDocument();
    expect(screen.getByText("Catalog orchestration")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add product" })).toHaveAttribute("href", "/products/new");
    expect(screen.getByRole("button", { name: "Import" })).toBeInTheDocument();
    expect(screen.getByText("12 active")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Import" }).compareDocumentPosition(
        screen.getByRole("link", { name: "Add product" })
      )
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it("renders EmptyState compact variant with accessible icon wrapper", () => {
    render(
      <EmptyState
        icon={PackagePlus}
        title="No products"
        description="Create the first catalog item."
        action={{ label: "Add product", href: "/products/new" }}
        variant="compact"
      />
    );

    expect(screen.getByRole("heading", { name: "No products" })).toBeInTheDocument();
    expect(screen.getByText("Create the first catalog item.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add product" })).toHaveAttribute("href", "/products/new");
    expect(screen.getByTestId("empty-state-icon")).toHaveAttribute("aria-hidden", "true");
  });
});
