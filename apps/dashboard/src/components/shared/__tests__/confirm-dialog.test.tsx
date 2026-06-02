import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

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

  it("shows default confirm label", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        onConfirm={vi.fn()}
      />
    );

    // With mocked useTranslations, t("confirm") returns "confirm"
    expect(screen.getByText("confirm")).toBeInTheDocument();
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

  it("shows cancel button", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Title"
        description="Description"
        onConfirm={vi.fn()}
      />
    );

    // With mocked useTranslations, t("cancel") returns "cancel"
    expect(screen.getByText("cancel")).toBeInTheDocument();
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
        isPending={true}
      />
    );

    // With mocked useTranslations, t("processing") returns "processing"
    expect(screen.getByText("processing")).toBeInTheDocument();
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
        isPending={true}
      />
    );

    expect(screen.getByText("cancel")).toBeDisabled();
    expect(screen.getByText("processing")).toBeDisabled();
  });
});
