import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SchemaBuilder } from "@/components/platform/providers/schema-builder/schema-builder";
import { emptyField } from "@/components/platform/providers/schema-builder/schema-helpers";
import type { ProviderField, ProviderFieldSchema } from "@/types/platform";

const state = {
  schema: undefined as ProviderFieldSchema | undefined,
};
const updateMutate = vi.fn();

vi.mock("@/hooks/use-platform-provider-config", () => ({
  useProviderSchema: () => ({
    data: state.schema,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useUpdateProviderSchema: () => ({ mutate: updateMutate, isPending: false }),
}));

function field(overrides: Partial<ProviderField>): ProviderField {
  return { ...emptyField(), ...overrides };
}

function schema(fields: ProviderField[]): ProviderFieldSchema {
  return {
    id: "s1",
    provider_version_id: "v1",
    groups: [{ key: "settings", label: "Settings", fields }],
    created_at: "2026-06-06T00:00:00Z",
    updated_at: "2026-06-06T00:00:00Z",
  };
}

beforeEach(() => {
  state.schema = schema([
    field({ key: "store_url", label: "Store URL", type: "url" }),
    field({ key: "locale", label: "Locale" }),
  ]);
  updateMutate.mockReset();
});

describe("SchemaBuilder field deletion", () => {
  it("renders an accessible delete button for every field row", () => {
    render(<SchemaBuilder providerId="p1" versionId="v1" readOnly={false} />);

    const deleteButtons = screen.getAllByRole("button", {
      name: "schemaBuilder.removeField",
    });
    expect(deleteButtons).toHaveLength(2);
  });

  it("deletes the field without selecting its row", async () => {
    const user = userEvent.setup();
    render(<SchemaBuilder providerId="p1" versionId="v1" readOnly={false} />);

    const [firstDelete] = screen.getAllByRole("button", {
      name: "schemaBuilder.removeField",
    });
    await user.click(firstDelete);

    // The field is gone from the list; the sibling survives.
    expect(screen.queryByText("store_url")).not.toBeInTheDocument();
    expect(screen.getByText("locale")).toBeInTheDocument();
    // Deleting must not select the row — the editor stays on its placeholder.
    expect(screen.getByText("schemaBuilder.selectField")).toBeInTheDocument();
  });

  it("does not render delete buttons in read-only mode", () => {
    render(<SchemaBuilder providerId="p1" versionId="v1" readOnly />);

    expect(
      screen.queryByRole("button", { name: "schemaBuilder.removeField" }),
    ).not.toBeInTheDocument();
  });
});
