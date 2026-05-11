import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ActionDialog } from "@/components/shared/action-dialog";

describe("ActionDialog", () => {
  it("renders title, description, and content when open", () => {
    render(
      <ActionDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Confirm action"
        description="This action needs confirmation."
        onConfirm={vi.fn()}
      >
        <p>Extra form content</p>
      </ActionDialog>
    );

    expect(screen.getByText("Confirm action")).toBeInTheDocument();
    expect(screen.getByText("This action needs confirmation.")).toBeInTheDocument();
    expect(screen.getByText("Extra form content")).toBeInTheDocument();
  });

  it("calls onConfirm from the primary action", async () => {
    const handleConfirm = vi.fn();
    const user = userEvent.setup();

    render(
      <ActionDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        confirmLabel="Run"
        onConfirm={handleConfirm}
      />
    );

    await user.click(screen.getByRole("button", { name: "Run" }));

    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });

  it("uses custom cancel handler before falling back to close", async () => {
    const handleCancel = vi.fn();
    const handleOpenChange = vi.fn();
    const user = userEvent.setup();

    render(
      <ActionDialog
        open={true}
        onOpenChange={handleOpenChange}
        title="Title"
        cancelLabel="Back"
        onCancel={handleCancel}
        onConfirm={vi.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: "Back" }));

    expect(handleCancel).toHaveBeenCalledTimes(1);
    expect(handleOpenChange).not.toHaveBeenCalled();
  });

  it("shows loading label and disables actions while loading", () => {
    render(
      <ActionDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        confirmLabel="Run"
        isLoading={true}
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByRole("button", { name: "processing" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "cancel" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Run" })).not.toBeInTheDocument();
  });

  it("renders inline error feedback", () => {
    render(
      <ActionDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        error="Could not complete action"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText("Could not complete action")).toBeInTheDocument();
  });

  it("confirms with Enter when focus is on the dialog chrome", () => {
    const handleConfirm = vi.fn();

    render(
      <ActionDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        confirmLabel="Run"
        onConfirm={handleConfirm}
      />
    );

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Enter" });

    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });
});
