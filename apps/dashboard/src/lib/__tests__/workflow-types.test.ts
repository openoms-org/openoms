import { describe, expect, it } from "vitest";
import { isWorkflowDefinition } from "@/lib/workflow-types";

describe("workflow definition guards", () => {
  it("accepts workflow definitions with nodes, edges, and viewport coordinates", () => {
    expect(
      isWorkflowDefinition({
        nodes: [],
        edges: [],
        viewport: { x: 0, y: 0, zoom: 1 },
      }),
    ).toBe(true);
  });

  it("rejects parseable but incomplete workflow drafts", () => {
    expect(isWorkflowDefinition({ nodes: [] })).toBe(false);
    expect(
      isWorkflowDefinition({
        nodes: [],
        edges: [],
        viewport: { x: 0, y: 0 },
      }),
    ).toBe(false);
  });
});
