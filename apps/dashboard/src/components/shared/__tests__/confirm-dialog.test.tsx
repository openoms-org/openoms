import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";

describe("ConfirmDialog", () => {
  it("renders dialog content when open", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Confirm Delete"
        description="Are you sure you want to delete?"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText("Confirm Delete")).toBeInTheDocument();
    expect(screen.getByText("Are you sure you want to delete?")).toBeInTheDocument();
  });

  it("does not render content when closed", () => {
    render(
      <ConfirmDialog
        open={false}
        onOpenChange={vi.fn()}
        title="Confirm Delete"
        description="Are you sure?"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.queryByText("Confirm Delete")).not.toBeInTheDocument();
  });

  it("shows default confirm label (Potwierdź)", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText("Potwierdź")).toBeInTheDocument();
  });

  it("shows custom confirm label", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        confirmLabel="Delete"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("calls onConfirm when confirm button is clicked", async () => {
    const handleConfirm = vi.fn();
    const user = userEvent.setup();

    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        confirmLabel="Yes"
        onConfirm={handleConfirm}
      />
    );

    await user.click(screen.getByText("Yes"));
    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });

  it("shows cancel button with text 'Anuluj'", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        onConfirm={vi.fn()}
      />
    );

    expect(screen.getByText("Anuluj")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        confirmLabel="Yes"
        onConfirm={vi.fn()}
        isLoading={true}
      />
    );

    expect(screen.getByText("Przetwarzanie...")).toBeInTheDocument();
    expect(screen.queryByText("Yes")).not.toBeInTheDocument();
  });

  it("disables buttons when loading", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        onConfirm={vi.fn()}
        isLoading={true}
      />
    );

    expect(screen.getByText("Anuluj")).toBeDisabled();
    expect(screen.getByText("Przetwarzanie...")).toBeDisabled();
  });
});
