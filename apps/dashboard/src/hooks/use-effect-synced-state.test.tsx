import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import {
  useEffectComputedState,
  useEffectSyncedState,
  useHydratedState,
  useHydratedValue,
} from "@/hooks/use-effect-synced-state";

function HydratedValueProbe({ readValue }: { readValue: () => string }) {
  const value = useHydratedValue("fallback", readValue);

  return <div>{value}</div>;
}

function HydratedStateProbe({ readValue }: { readValue: () => string }) {
  const [value, setValue, hydrated] = useHydratedState("fallback", readValue);

  return (
    <div>
      <span>{value}</span>
      <span>{hydrated ? "hydrated" : "pending"}</span>
      <button type="button" onClick={() => setValue("manual")}>
        update
      </button>
    </div>
  );
}

function SyncedStateProbe({
  sourceValue,
  resetKey,
}: {
  sourceValue: string;
  resetKey: string;
}) {
  const [value, setValue] = useEffectSyncedState(sourceValue, resetKey);

  return (
    <div>
      <span>{value}</span>
      <button type="button" onClick={() => setValue("draft")}>
        edit
      </button>
    </div>
  );
}

function ComputedStateProbe({ computeValue }: { computeValue: () => string | null }) {
  const [value, setValue] = useEffectComputedState("initial", computeValue);

  return (
    <div>
      <span>{value}</span>
      <button type="button" onClick={() => setValue("manual")}>
        update
      </button>
    </div>
  );
}

describe("useHydratedValue", () => {
  it("uses the browser-only reader after mount", async () => {
    const readValue = vi.fn(() => "stored");

    render(<HydratedValueProbe readValue={readValue} />);

    expect(await screen.findByText("stored")).toBeInTheDocument();
    expect(readValue).toHaveBeenCalledTimes(1);
  });
});

describe("useHydratedState", () => {
  it("hydrates state and then allows local updates", async () => {
    const user = userEvent.setup();
    const readValue = vi.fn(() => "stored");

    render(<HydratedStateProbe readValue={readValue} />);

    expect(await screen.findByText("stored")).toBeInTheDocument();
    expect(screen.getByText("hydrated")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "update" }));

    expect(screen.getByText("manual")).toBeInTheDocument();
  });

  it("does not rehydrate when the reader identity changes", async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <HydratedStateProbe readValue={() => "stored"} />,
    );

    expect(await screen.findByText("stored")).toBeInTheDocument();
    expect(screen.getByText("hydrated")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "update" }));
    expect(screen.getByText("manual")).toBeInTheDocument();

    rerender(<HydratedStateProbe readValue={() => "fresh"} />);

    expect(screen.getByText("manual")).toBeInTheDocument();
  });
});

describe("useEffectSyncedState", () => {
  it("preserves local edits while the reset key is unchanged", async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <SyncedStateProbe sourceValue="api-one" resetKey="one" />,
    );

    expect(screen.getByText("api-one")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "edit" }));
    expect(screen.getByText("draft")).toBeInTheDocument();

    rerender(<SyncedStateProbe sourceValue="api-refetch" resetKey="one" />);

    expect(screen.getByText("draft")).toBeInTheDocument();
  });

  it("resets local state when the external reset key changes", async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <SyncedStateProbe sourceValue="api-one" resetKey="one" />,
    );

    await user.click(screen.getByRole("button", { name: "edit" }));
    expect(screen.getByText("draft")).toBeInTheDocument();

    rerender(<SyncedStateProbe sourceValue="api-two" resetKey="two" />);

    expect(await screen.findByText("api-two")).toBeInTheDocument();
  });
});

describe("useEffectComputedState", () => {
  it("updates from the computed effect value", async () => {
    const computeValue = vi.fn(() => "measured");

    render(<ComputedStateProbe computeValue={computeValue} />);

    expect(await screen.findByText("measured")).toBeInTheDocument();
    expect(computeValue).toHaveBeenCalledTimes(1);
  });

  it("allows local updates after the effect computed value", async () => {
    const user = userEvent.setup();

    render(<ComputedStateProbe computeValue={() => "measured"} />);

    expect(await screen.findByText("measured")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "update" }));

    expect(screen.getByText("manual")).toBeInTheDocument();
  });

  it("does not recompute when the compute function identity changes", async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <ComputedStateProbe computeValue={() => "measured"} />,
    );

    expect(await screen.findByText("measured")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "update" }));
    expect(screen.getByText("manual")).toBeInTheDocument();

    rerender(<ComputedStateProbe computeValue={() => "fresh"} />);

    expect(screen.getByText("manual")).toBeInTheDocument();
  });
});
