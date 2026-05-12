import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ProviderCard } from "@/components/shared/provider-card";
import { ProviderLogo } from "@/components/shared/provider-logo";
import { getProviderInfo } from "@/lib/provider-info";

describe("provider identity", () => {
  it("renders a recognizable accessible logo for a known carrier", () => {
    const provider = getProviderInfo("inpost");
    expect(provider).toBeDefined();

    render(<ProviderLogo provider={provider!} />);

    const logo = screen.getByLabelText("InPost logo");
    expect(logo).toHaveAttribute("data-provider-key", "inpost");
    expect(logo).toHaveTextContent("InPost");
  });

  it("falls back to initials for unknown providers", () => {
    render(
      <ProviderLogo
        providerKey="custom_carrier"
        fallbackName="Custom Carrier"
        category="carrier"
      />
    );

    const logo = screen.getByLabelText("Custom Carrier logo");
    expect(logo).toHaveAttribute("data-provider-key", "custom_carrier");
    expect(logo).toHaveTextContent("CC");
  });

  it("uses provider logos inside selectable provider cards", () => {
    const provider = getProviderInfo("allegro");
    expect(provider).toBeDefined();

    render(<ProviderCard provider={provider!} />);

    expect(screen.getByRole("button", { name: /Allegro/ })).toBeInTheDocument();
    expect(screen.getByLabelText("Allegro logo")).toHaveAttribute(
      "data-provider-key",
      "allegro",
    );
  });
});
