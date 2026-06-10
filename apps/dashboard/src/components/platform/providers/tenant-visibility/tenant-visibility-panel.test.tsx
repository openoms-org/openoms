import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TenantVisibilityPanel } from "@/components/platform/providers/tenant-visibility/tenant-visibility-panel";
import type { ProviderPublicationState, ProviderVersion } from "@/types/platform";

const enableMutate = vi.fn();

vi.mock("@/hooks/use-platform-provider-validation", () => ({
  useEnableTenantForVersion: () => ({ mutate: enableMutate, isPending: false }),
}));

function version(state: ProviderPublicationState): ProviderVersion {
  return {
    id: "v1",
    version: "1.0.0",
    publication_state: state,
    changelog: "",
    created_at: "2026-06-06T00:00:00Z",
  };
}

beforeEach(() => enableMutate.mockReset());

describe("TenantVisibilityPanel", () => {
  it("offers tenant enablement only in private beta", () => {
    const { rerender } = render(
      <TenantVisibilityPanel providerId="p1" version={version("available")} />,
    );
    expect(
      screen.queryByRole("button", { name: "tenantVisibility.enableTenant" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("tenantVisibility.onlyPrivateBeta")).toBeInTheDocument();

    rerender(
      <TenantVisibilityPanel providerId="p1" version={version("private_beta")} />,
    );
    expect(
      screen.getByRole("button", { name: "tenantVisibility.enableTenant" }),
    ).toBeInTheDocument();
  });

  it("validates the tenant UUID and enables the tenant on confirm", async () => {
    const user = userEvent.setup();
    render(
      <TenantVisibilityPanel providerId="p1" version={version("private_beta")} />,
    );

    await user.click(
      screen.getByRole("button", { name: "tenantVisibility.enableTenant" }),
    );
    const dialog = await screen.findByRole("dialog");
    const confirm = within(dialog).getByRole("button", {
      name: "tenantVisibility.enableTenantAction",
    });
    expect(confirm).toBeDisabled();

    const input = within(dialog).getByLabelText("tenantVisibility.tenantId");
    await user.type(input, "not-a-uuid");
    expect(confirm).toBeDisabled();
    expect(
      within(dialog).getByText("tenantVisibility.tenantIdInvalid"),
    ).toBeInTheDocument();

    await user.clear(input);
    await user.type(input, "11111111-2222-3333-4444-555555555555");
    expect(confirm).toBeEnabled();

    await user.click(confirm);
    expect(enableMutate).toHaveBeenCalledTimes(1);
    expect(enableMutate.mock.calls[0][0]).toMatchObject({
      tenant_id: "11111111-2222-3333-4444-555555555555",
    });
  });

  it("surfaces the not-yet-available allowlist and downgrade states", () => {
    render(
      <TenantVisibilityPanel providerId="p1" version={version("private_beta")} />,
    );
    expect(
      screen.getByText("tenantVisibility.allowlistUnavailable"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("tenantVisibility.downgradesUnavailable"),
    ).toBeInTheDocument();
  });
});
