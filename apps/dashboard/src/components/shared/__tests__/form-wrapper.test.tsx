import { describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormField, FormWrapper } from "@/components/shared/form-wrapper";
import { Input } from "@/components/ui/input";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
});

type TestFormValues = z.infer<typeof schema>;

describe("FormWrapper", () => {
  it("wires zod validation, field errors, and error summary", async () => {
    const handleSubmit = vi.fn();
    const user = userEvent.setup();

    renderTestForm(handleSubmit);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Name is required");
    expect(screen.getByLabelText(/^Name/)).toHaveAttribute("aria-invalid", "true");
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits typed form values when valid", async () => {
    const handleSubmit = vi.fn();
    const user = userEvent.setup();

    renderTestForm(handleSubmit);

    await user.type(screen.getByLabelText(/^Name/), "Ada");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(handleSubmit).toHaveBeenCalledWith({ name: "Ada" }, expect.anything());
    });
  });

  it("supports loading submit labels and cancel handling", async () => {
    const handleCancel = vi.fn();
    const user = userEvent.setup();

    renderTestForm(vi.fn(), { isSubmitting: true, onCancel: handleCancel });

    expect(screen.getByRole("button", { name: "Saving" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(handleCancel).toHaveBeenCalledTimes(1);
  });
});

function renderTestForm(
  onSubmit: (values: TestFormValues) => void,
  options: { isSubmitting?: boolean; onCancel?: () => void } = {}
) {
  return render(
    <FormWrapper<TestFormValues>
      schema={schema}
      defaultValues={{ name: "" }}
      onSubmit={onSubmit}
      submitLabel="Save"
      submittingLabel="Saving"
      isSubmitting={options.isSubmitting}
      onCancel={options.onCancel}
      cancelLabel="Cancel"
      errorSummaryTitle="Please fix these errors"
    >
      {({ register, formState: { errors } }) => (
        <FormField<TestFormValues>
          name="name"
          label="Name"
          error={errors.name}
          required
        >
          <Input
            id="name"
            aria-invalid={!!errors.name}
            {...register("name")}
          />
        </FormField>
      )}
    </FormWrapper>
  );
}
