import type { WorkflowNode, WorkflowEdge, WorkflowDefinition, WorkflowViewport } from "@/types/api";

export type { WorkflowNode, WorkflowEdge, WorkflowDefinition, WorkflowViewport };

export type WorkflowNodeType = "trigger" | "condition" | "action";

// Trigger event options for the trigger node
export interface TriggerOption {
  value: string;
  label: string;
}

export const TRIGGER_OPTIONS: TriggerOption[] = [
  { value: "order.created", label: "Zamowienie utworzone" },
  { value: "order.status_changed", label: "Zmiana statusu zamowienia" },
  { value: "order.updated", label: "Zamowienie zaktualizowane" },
  { value: "shipment.created", label: "Przesylka utworzona" },
  { value: "shipment.status_changed", label: "Zmiana statusu przesylki" },
  { value: "return.created", label: "Zwrot utworzony" },
  { value: "return.status_changed", label: "Zmiana statusu zwrotu" },
  { value: "product.created", label: "Produkt utworzony" },
  { value: "product.updated", label: "Produkt zaktualizowany" },
];

// Condition operator options
export interface OperatorOption {
  value: string;
  label: string;
}

export const OPERATOR_OPTIONS: OperatorOption[] = [
  { value: "eq", label: "Rowne (=)" },
  { value: "neq", label: "Rozne (!=)" },
  { value: "gt", label: "Wieksze (>)" },
  { value: "gte", label: "Wieksze lub rowne (>=)" },
  { value: "lt", label: "Mniejsze (<)" },
  { value: "lte", label: "Mniejsze lub rowne (<=)" },
  { value: "contains", label: "Zawiera" },
  { value: "not_contains", label: "Nie zawiera" },
  { value: "starts_with", label: "Zaczyna sie od" },
  { value: "in", label: "Zawiera sie w" },
  { value: "not_in", label: "Nie zawiera sie w" },
];

// Action type options
export interface ActionTypeOption {
  value: string;
  label: string;
}

export const ACTION_TYPE_OPTIONS: ActionTypeOption[] = [
  { value: "set_status", label: "Ustaw status" },
  { value: "add_tag", label: "Dodaj tag" },
  { value: "send_email", label: "Wyslij e-mail" },
  { value: "create_invoice", label: "Utworz fakture" },
  { value: "webhook", label: "Wywolaj webhook" },
];

// Palette items for sidebar drag
export interface PaletteItem {
  type: WorkflowNodeType;
  subType: string;
  label: string;
  description: string;
}

export const PALETTE_TRIGGERS: PaletteItem[] = TRIGGER_OPTIONS.map((t) => ({
  type: "trigger" as const,
  subType: t.value,
  label: t.label,
  description: t.value,
}));

export const PALETTE_CONDITIONS: PaletteItem[] = [
  { type: "condition", subType: "field_check", label: "Warunek pola", description: "Sprawdz wartosc pola" },
];

export const PALETTE_ACTIONS: PaletteItem[] = ACTION_TYPE_OPTIONS.map((a) => ({
  type: "action" as const,
  subType: a.value,
  label: a.label,
  description: a.value,
}));

// Node colors by type
export const NODE_COLORS: Record<WorkflowNodeType, { bg: string; border: string; header: string; text: string }> = {
  trigger: {
    bg: "bg-green-50 dark:bg-green-950",
    border: "border-green-300 dark:border-green-700",
    header: "bg-green-500 dark:bg-green-700",
    text: "text-green-800 dark:text-green-200",
  },
  condition: {
    bg: "bg-blue-50 dark:bg-blue-950",
    border: "border-blue-300 dark:border-blue-700",
    header: "bg-blue-500 dark:bg-blue-700",
    text: "text-blue-800 dark:text-blue-200",
  },
  action: {
    bg: "bg-orange-50 dark:bg-orange-950",
    border: "border-orange-300 dark:border-orange-700",
    header: "bg-orange-500 dark:bg-orange-700",
    text: "text-orange-800 dark:text-orange-200",
  },
};

// Default empty workflow definition
export function createEmptyWorkflow(): WorkflowDefinition {
  return {
    nodes: [],
    edges: [],
    viewport: { x: 0, y: 0, zoom: 1 },
  };
}

export function isWorkflowDefinition(value: unknown): value is WorkflowDefinition {
  if (!value || typeof value !== "object") return false;

  const candidate = value as Partial<WorkflowDefinition>;
  return (
    Array.isArray(candidate.nodes) &&
    Array.isArray(candidate.edges) &&
    typeof candidate.viewport?.x === "number" &&
    typeof candidate.viewport?.y === "number" &&
    typeof candidate.viewport?.zoom === "number"
  );
}

// Generate a unique ID for new nodes/edges
export function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}
