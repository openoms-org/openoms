import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { FieldEditor } from "@/components/platform/providers/schema-builder/field-editor";
import { SetupPreview } from "@/components/platform/providers/schema-builder/setup-preview";
import { emptyField } from "@/components/platform/providers/schema-builder/schema-helpers";
import { validateField } from "@/components/platform/providers/schema-builder/schema-helpers";
import type { ProviderField, ProviderFieldGroup } from "@/types/platform";

function field(overrides: Partial<ProviderField>): ProviderField {
  return { ...emptyField(), ...overrides };
}

describe("FieldEditor", () => {
  it("auto-derives the key from the label while the key is untouched", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <FieldEditor
        field={field({ key: "", label: "" })}
        issues={[]}
        readOnly={false}
        onChange={onChange}
      />,
    );

    await user.type(screen.getByLabelText("schemaBuilder.field.label"), "A");

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ label: "A", key: "a" }),
    );
  });

  it("disables inputs when read-only", () => {
    render(
      <FieldEditor
        field={field({ key: "api_key", label: "API key" })}
        issues={[]}
        readOnly
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByLabelText("schemaBuilder.field.label")).toBeDisabled();
  });

  it("renders the secret warning for a sensitive field that is not secret", () => {
    render(
      <FieldEditor
        field={field({ key: "api_secret", label: "API secret", secret: false })}
        issues={validateField(
          field({ key: "api_secret", label: "API secret", secret: false }),
          new Set(),
        )}
        readOnly={false}
        onChange={vi.fn()}
      />,
    );
    expect(
      screen.getByText("schemaBuilder.field.secretWarning"),
    ).toBeInTheDocument();
  });

  it("renders validation messages for blocking issues", () => {
    render(
      <FieldEditor
        field={field({ key: "", label: "" })}
        issues={validateField(field({ key: "", label: "" }), new Set())}
        readOnly={false}
        onChange={vi.fn()}
      />,
    );
    expect(
      screen.getByText("schemaBuilder.validation.keyRequired"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("schemaBuilder.validation.labelRequired"),
    ).toBeInTheDocument();
  });
});

describe("SetupPreview", () => {
  const groups: ProviderFieldGroup[] = [
    {
      key: "secret_credentials",
      label: "Secret credentials",
      fields: [
        field({
          key: "api_key",
          label: "API key",
          type: "password",
          secret: true,
          required: true,
        }),
      ],
    },
  ];

  it("renders a secret field without exposing a raw value", () => {
    render(<SetupPreview groups={groups} />);
    expect(screen.getByText("API key")).toBeInTheDocument();
    expect(screen.getByText("schemaBuilder.preview.secret")).toBeInTheDocument();
    // The masked placeholder, never a real value.
    expect(screen.getByText("••••••••")).toBeInTheDocument();
  });

  it("renders the empty state when no complete fields exist", () => {
    render(
      <SetupPreview
        groups={[
          { key: "settings", label: "Settings", fields: [field({ key: "", label: "" })] },
        ]}
      />,
    );
    expect(screen.getByText("schemaBuilder.preview.empty")).toBeInTheDocument();
  });
});
