import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { EvidenceDrawer } from "@/components/platform/providers/shared/evidence-drawer";
import type {
  PlatformMe,
  ProviderValidationRun,
} from "@/types/platform";

function runWithSecretObservation(): ProviderValidationRun {
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
        label: "auth probe",
        status: "passed",
        observation: "token password=hunter2supersecret accepted",
        payload_hash: "0123456789abcdef0123",
        findings: "",
        created_at: "2026-06-06T00:30:00Z",
      },
    ],
  };
}

const me = (perms: string[]): PlatformMe => ({ user_id: "u1", permissions: perms });

describe("EvidenceDrawer redaction", () => {
  it("redacts sensitive observation by default and offers no reveal without permission", () => {
    render(
      <EvidenceDrawer
        open
        onOpenChange={() => {}}
        run={runWithSecretObservation()}
        me={me(["providers:read"])}
      />,
    );
    // Raw secret never rendered.
    expect(screen.queryByText(/hunter2supersecret/)).not.toBeInTheDocument();
    expect(screen.getByText("evidence.redactedOnly")).toBeInTheDocument();
    // No reveal control without the secrets permission.
    expect(
      screen.queryByRole("button", { name: "evidence.reveal" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("evidence.redactedNotice")).toBeInTheDocument();
  });

  it("lets a providers:secrets viewer explicitly reveal the recorded observation", async () => {
    const user = userEvent.setup();
    render(
      <EvidenceDrawer
        open
        onOpenChange={() => {}}
        run={runWithSecretObservation()}
        me={me(["providers:read", "providers:secrets"])}
      />,
    );
    expect(screen.getByText("evidence.sensitiveAllowed")).toBeInTheDocument();
    // Still redacted until explicitly revealed.
    expect(screen.queryByText(/hunter2supersecret/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "evidence.reveal" }));
    expect(screen.getByText(/hunter2supersecret/)).toBeInTheDocument();
  });

  it("shows an empty state when the run has no results", () => {
    render(
      <EvidenceDrawer
        open
        onOpenChange={() => {}}
        run={{ ...runWithSecretObservation(), results: [] }}
        me={me(["providers:secrets"])}
      />,
    );
    expect(screen.getByText("evidence.empty")).toBeInTheDocument();
  });
});
