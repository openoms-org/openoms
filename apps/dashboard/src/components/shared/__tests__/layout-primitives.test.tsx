import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Surface } from "@/components/shared/surface";
import { PageSection } from "@/components/shared/page-section";
import { ActionBar } from "@/components/shared/action-bar";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { PackagePlus } from "lucide-react";
import {
  FormActions,
  FormSection,
} from "@/components/shared/form-layout";
import {
  DetailLayout,
  DetailMain,
  DetailSidebar,
} from "@/components/shared/detail-layout";
import {
  SettingsLayout,
  SettingsNav,
  SettingsPanel,
} from "@/components/shared/settings-layout";

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

  it("uses the dashboard primary treatment for default buttons", () => {
    render(<Button>Primary action</Button>);

    expect(screen.getByRole("button", { name: "Primary action" })).toHaveClass("bg-info");
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
        action={{ label: "Add product", href: "/products/new", variant: "soft" }}
        variant="compact"
      />
    );

    expect(screen.getByRole("heading", { name: "No products" })).toBeInTheDocument();
    expect(screen.getByText("Create the first catalog item.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add product" })).toHaveAttribute("href", "/products/new");
    expect(screen.getByRole("link", { name: "Add product" })).toHaveClass("bg-sidebar-accent");
    expect(screen.getByTestId("empty-state-icon")).toHaveAttribute("aria-hidden", "true");
  });

  it("renders FormSection and FormActions", () => {
    render(
      <FormSection title="Carrier credentials" description="Production API settings">
        <label htmlFor="token">Token</label>
        <input id="token" />
        <FormActions primary={<Button>Save</Button>} secondary={<Button variant="outline">Cancel</Button>} />
      </FormSection>
    );

    expect(screen.getByRole("heading", { name: "Carrier credentials" })).toBeInTheDocument();
    expect(screen.getByText("Production API settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Token")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
  });

  it("renders DetailLayout with main and sidebar regions", () => {
    render(
      <DetailLayout>
        <DetailMain aria-label="Order activity">Activity</DetailMain>
        <DetailSidebar aria-label="Order metadata">Metadata</DetailSidebar>
      </DetailLayout>
    );

    expect(screen.getByRole("region", { name: "Order activity" })).toHaveTextContent("Activity");
    expect(screen.getByRole("complementary", { name: "Order metadata" })).toHaveTextContent("Metadata");
  });

  it("renders SettingsLayout with navigation and panel", () => {
    render(
      <SettingsLayout>
        <SettingsNav aria-label="Settings sections">
          <a href="/settings/company">Company</a>
        </SettingsNav>
        <SettingsPanel title="Company" description="Company profile">
          Settings content
        </SettingsPanel>
      </SettingsLayout>
    );

    expect(screen.getByRole("navigation", { name: "Settings sections" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Company" })).toBeInTheDocument();
    expect(screen.getByText("Settings content")).toBeInTheDocument();
  });
});
