import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ProviderCard } from "@/components/shared/provider-card";
import { ProviderLogo } from "@/components/shared/provider-logo";
import { getProviderInfo } from "@/lib/provider-info";

describe("provider identity", () => {
  it("renders the approved official asset for InPost", () => {
    const provider = getProviderInfo("inpost");
    expect(provider).toBeDefined();

    render(<ProviderLogo provider={provider!} />);

    const logo = screen.getByRole("img", { name: "InPost logo" });
    expect(logo).toHaveAttribute("src", "/logos/official/inpost.svg");
    expect(logo).toHaveAttribute("alt", "InPost logo");
    expect(logo).toHaveAttribute("data-provider-key", "inpost");
  });

  it("keeps non-approved providers on the safe wordmark fallback", () => {
    const provider = getProviderInfo("allegro");
    expect(provider).toBeDefined();

    render(<ProviderLogo provider={provider!} />);

    const logo = screen.getByLabelText("Allegro logo");
    expect(logo).toHaveAttribute("data-provider-key", "allegro");
    expect(logo).toHaveTextContent("Allegro");
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
