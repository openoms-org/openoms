import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PublicationReview } from "@/components/platform/providers/publication-review/publication-review";
import type {
  ProviderCapability,
  ProviderFieldSchema,
  ProviderIntegrationGap,
  ProviderPublicationState,
  ProviderValidationRun,
  ProviderVersion,
} from "@/types/platform";

// Mutable fixtures the mocked hooks read from, so each test can tune readiness.
const state = {
  schema: undefined as ProviderFieldSchema | undefined,
  capabilities: [] as ProviderCapability[],
  gaps: [] as ProviderIntegrationGap[],
  runs: [] as ProviderValidationRun[],
  events: [] as unknown[],
};

const publishMutate = vi.fn();
const disableMutate = vi.fn();

vi.mock("@/hooks/use-platform-provider-config", () => ({
  useProviderSchema: () => ({ data: state.schema }),
  useProviderCapabilities: () => ({ data: state.capabilities }),
  useProviderGaps: () => ({ data: state.gaps }),
}));

vi.mock("@/hooks/use-platform-provider-validation", () => ({
  useProviderValidationRuns: () => ({ data: state.runs }),
  useProviderPublicationEvents: () => ({ data: state.events }),
  usePublishProviderVersion: () => ({ mutate: publishMutate, isPending: false }),
  useEmergencyDisableProviderVersion: () => ({
    mutate: disableMutate,
    isPending: false,
  }),
}));

function version(s: ProviderPublicationState): ProviderVersion {
  return {
    id: "v1",
    version: "1.0.0",
    publication_state: s,
    changelog: "",
    created_at: "2026-06-06T00:00:00Z",
  };
}

function schemaWithFields(): ProviderFieldSchema {
  return {
    id: "s",
    provider_version_id: "v1",
    groups: [
      {
        key: "secret_credentials",
        label: "Secrets",
        fields: [
          {
            key: "token",
            label: "Token",
            type: "password",
            required: true,
            secret: true,
            validation: {},
          },
        ],
      },
    ],
    created_at: "",
    updated_at: "",
  };
}

function passingRun(): ProviderValidationRun {
  return {
    id: "run1",
    provider_version_id: "v1",
    environment: "sandbox",
    verdict: "passed",
    started_at: "2026-06-06T00:00:00Z",
    finished_at: "2026-06-06T01:00:00Z",
    notes: "",
    results: [
      {
        id: "r1",
        run_id: "run1",
        probe_type: "auth_check",
        label: "auth",
        status: "passed",
        observation: "ok",
        payload_hash: "abc",
        findings: "",
        created_at: "2026-06-06T00:30:00Z",
      },
    ],
  };
}

function renderReview(s: ProviderPublicationState) {
  const v = version(s);
  return render(
    <PublicationReview providerId="p1" version={v} versions={[v]} />,
  );
}

beforeEach(() => {
  state.schema = undefined;
  state.capabilities = [];
  state.gaps = [];
  state.runs = [];
  state.events = [];
  publishMutate.mockReset();
  disableMutate.mockReset();
});

describe("PublicationReview gating", () => {
  it("enables private-beta promotion when ready and calls publish via the dialog", async () => {
    state.schema = schemaWithFields();
    state.capabilities = [
      {
        capability_key: "marketplace.order.read",
        support_status: "supported",
        channel: "",
        mode: "",
        freshness: "",
        required_inputs: [],
        provided_outputs: [],
        latency_sla_seconds: null,
        notes: "",
      },
    ];
    state.runs = [passingRun()];

    const user = userEvent.setup();
    renderReview("internal_validation");

    const promote = screen.getByRole("button", {
      name: "publicationReview.transitionTo.private_beta",
    });
    expect(promote).toBeEnabled();

    await user.click(promote);
    // Confirmation dialog opens; confirm fires the publish mutation.
    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", {
        name: "publicationReview.confirmAction",
      }),
    );
    expect(publishMutate).toHaveBeenCalledTimes(1);
    expect(publishMutate.mock.calls[0][0]).toMatchObject({
      to_state: "private_beta",
    });
  });

  it("disables exposure-increasing promotion and shows the concrete blocked-gate reason", () => {
    // No schema, no run -> G2/G3 blocked.
    renderReview("internal_validation");

    const promote = screen.getByRole("button", {
      name: "publicationReview.transitionTo.private_beta",
    });
    expect(promote).toBeDisabled();
    expect(
      screen.getByText("publicationReview.disabledReason.blockedGates"),
    ).toBeInTheDocument();
  });

  it("explains blocking gaps on a disabled promotion", () => {
    state.schema = schemaWithFields();
    state.runs = [passingRun()];
    state.capabilities = [
      {
        capability_key: "marketplace.order.read",
        support_status: "supported",
        channel: "",
        mode: "",
        freshness: "",
        required_inputs: [],
        provided_outputs: [],
        latency_sla_seconds: null,
        notes: "",
      },
    ];
    state.gaps = [
      {
        id: "g1",
        provider_version_id: "v1",
        gap_type: "missing_status_mapping",
        severity: "system_error",
        status: "open",
        description: "x",
        created_at: "",
        updated_at: "",
      },
    ];

    renderReview("private_beta");
    const promote = screen.getByRole("button", {
      name: "publicationReview.transitionTo.available",
    });
    expect(promote).toBeDisabled();
    // Reasons are joined into one explanatory line; the blocking-gaps reason
    // must be present (a blocking gap also blocks G3, so both reasons appear).
    expect(
      screen.getByText(/publicationReview\.disabledReason\.blockingGaps/),
    ).toBeInTheDocument();
  });
});

describe("PublicationReview emergency disable", () => {
  it("requires a reason before disabling and traps focus in the dialog", async () => {
    const user = userEvent.setup();
    renderReview("available");

    await user.click(
      screen.getByRole("button", { name: "publicationReview.emergencyDisable" }),
    );
    const dialog = await screen.findByRole("dialog");

    const confirm = within(dialog).getByRole("button", {
      name: "publicationReview.emergencyDisableAction",
    });
    // Disabled until a reason is entered.
    expect(confirm).toBeDisabled();

    const textarea = within(dialog).getByLabelText(
      "publicationReview.reasonRequired",
    );
    await user.type(textarea, "rolling back faulty release");
    expect(confirm).toBeEnabled();

    await user.click(confirm);
    expect(disableMutate).toHaveBeenCalledTimes(1);
    expect(disableMutate.mock.calls[0][0]).toMatchObject({
      reason: "rolling back faulty release",
    });
  });

  it("does not offer emergency disable from internal validation", () => {
    renderReview("internal_validation");
    expect(
      screen.queryByRole("button", {
        name: "publicationReview.emergencyDisable",
      }),
    ).not.toBeInTheDocument();
  });
});
