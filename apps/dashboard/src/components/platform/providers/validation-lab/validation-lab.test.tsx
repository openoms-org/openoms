import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ValidationLab } from "@/components/platform/providers/validation-lab/validation-lab";
import type {
  PlatformMe,
  ProviderValidationProbe,
} from "@/types/platform";

const state = {
  probes: [] as ProviderValidationProbe[],
};
const startMutate = vi.fn();

vi.mock("@/hooks/use-platform-provider-validation", () => ({
  useProviderProbes: () => ({ data: state.probes, isLoading: false }),
  useProviderValidationRuns: () => ({
    data: [],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  // Used by the run timeline child; runId is null so it stays disabled.
  useProviderValidationRun: () => ({ data: undefined, isLoading: false }),
  useStartValidationRun: () => ({ mutate: startMutate, isPending: false }),
}));

function probe(
  overrides: Partial<ProviderValidationProbe>,
): ProviderValidationProbe {
  return {
    id: overrides.id ?? "p",
    provider_version_id: "v1",
    probe_type: overrides.probe_type ?? "auth_check",
    label: overrides.label ?? "Auth",
    destructive: overrides.destructive ?? false,
    required: overrides.required ?? false,
    created_at: "2026-06-06T00:00:00Z",
    updated_at: "2026-06-06T00:00:00Z",
    ...overrides,
  };
}

const me = (perms: string[]): PlatformMe => ({ user_id: "u1", permissions: perms });

beforeEach(() => {
  state.probes = [];
  startMutate.mockReset();
});

describe("ValidationLab", () => {
  it("starts a sandbox run directly for read-only probes", async () => {
    state.probes = [probe({ id: "p1", probe_type: "auth_check", label: "Auth" })];
    const user = userEvent.setup();
    render(<ValidationLab providerId="p1" versionId="v1" me={me(["providers:validate"])} />);

    await user.click(
      screen.getByRole("button", { name: "validationLab.startRun" }),
    );
    expect(startMutate).toHaveBeenCalledTimes(1);
    expect(startMutate.mock.calls[0][0]).toMatchObject({
      environment: "sandbox",
      allow_destructive: false,
    });
  });

  it("requires explicit confirmation before running destructive probes", async () => {
    state.probes = [
      probe({ id: "p1", probe_type: "auth_check", label: "Auth" }),
      probe({
        id: "p2",
        probe_type: "sandbox_order_create",
        label: "Cancel order",
        destructive: true,
      }),
    ];
    const user = userEvent.setup();
    render(<ValidationLab providerId="p1" versionId="v1" me={me(["providers:validate"])} />);

    // Opt in to destructive probes.
    await user.click(
      screen.getByRole("checkbox", { name: "validationLab.allowDestructive" }),
    );
    await user.click(
      screen.getByRole("button", { name: "validationLab.startRun" }),
    );

    // A confirmation dialog must appear before the run starts.
    expect(startMutate).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", {
        name: "validationLab.destructiveConfirmAction",
      }),
    );
    expect(startMutate).toHaveBeenCalledTimes(1);
    expect(startMutate.mock.calls[0][0]).toMatchObject({ allow_destructive: true });
  });

  it("blocks production runs without the providers:secrets permission", async () => {
    state.probes = [probe({ id: "p1", probe_type: "auth_check", label: "Auth" })];
    const user = userEvent.setup();
    render(<ValidationLab providerId="p1" versionId="v1" me={me(["providers:validate"])} />);

    await user.click(screen.getByRole("combobox"));
    await user.click(
      await screen.findByRole("option", { name: "validationLab.env.production" }),
    );

    expect(
      screen.getByText("validationLab.productionPermissionRequired"),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "validationLab.startRun" }),
    ).toBeDisabled();
  });
});
