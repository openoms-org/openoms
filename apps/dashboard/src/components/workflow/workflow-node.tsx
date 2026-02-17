"use client";

import { useCallback } from "react";
import type { WorkflowNode as WFNode } from "@/types/api";
import { NODE_COLORS, type WorkflowNodeType } from "@/lib/workflow-types";
import {
  AUTOMATION_TRIGGER_LABELS,
  AUTOMATION_ACTION_LABELS,
} from "@/lib/constants";
import { Zap, GitBranch, Play, GripVertical } from "lucide-react";

interface WorkflowNodeProps {
  node: WFNode;
  selected: boolean;
  onSelect: (id: string) => void;
  onDragStart: (e: React.MouseEvent, nodeId: string) => void;
  scale: number;
}

const TYPE_ICONS: Record<WorkflowNodeType, React.ComponentType<{ className?: string }>> = {
  trigger: Zap,
  condition: GitBranch,
  action: Play,
};

const TYPE_LABELS: Record<WorkflowNodeType, string> = {
  trigger: "Wyzwalacz",
  condition: "Warunek",
  action: "Akcja",
};

function getNodeLabel(node: WFNode): string {
  const label = node.data.label as string | undefined;
  if (label) return label;

  if (node.type === "trigger") {
    const event = node.data.event as string;
    return AUTOMATION_TRIGGER_LABELS[event] || event || "Wyzwalacz";
  }
  if (node.type === "condition") {
    const field = node.data.field as string;
    const op = node.data.operator as string;
    const val = node.data.value;
    if (field) return `${field} ${op || "eq"} ${val ?? ""}`;
    return "Warunek";
  }
  if (node.type === "action") {
    const actionType = node.data.actionType as string;
    return AUTOMATION_ACTION_LABELS[actionType] || actionType || "Akcja";
  }
  return "Wezel";
}

function getNodeSubtext(node: WFNode): string | null {
  if (node.type === "trigger") {
    const event = node.data.event as string;
    return event || null;
  }
  if (node.type === "condition") {
    const field = node.data.field as string;
    const op = node.data.operator as string;
    const val = node.data.value;
    if (field) return `${field} ${op || "eq"} ${String(val ?? "")}`;
    return null;
  }
  if (node.type === "action") {
    const actionType = node.data.actionType as string;
    if (actionType === "set_status") return `Status: ${node.data.status || "?"}`;
    if (actionType === "add_tag") return `Tag: ${node.data.tag || "?"}`;
    if (actionType === "send_email") return `Do: ${node.data.to || "?"}`;
    if (actionType === "webhook") return `URL: ${node.data.url || "?"}`;
    if (actionType === "create_invoice") return `Typ: ${node.data.invoice_type || "vat"}`;
    return actionType;
  }
  return null;
}

export function WorkflowNodeComponent({
  node,
  selected,
  onSelect,
  onDragStart,
  scale,
}: WorkflowNodeProps) {
  const colors = NODE_COLORS[node.type];
  const Icon = TYPE_ICONS[node.type];
  const typeLabel = TYPE_LABELS[node.type];
  const label = getNodeLabel(node);
  const subtext = getNodeSubtext(node);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onSelect(node.id);
      onDragStart(e, node.id);
    },
    [node.id, onSelect, onDragStart]
  );

  const handleClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onSelect(node.id);
    },
    [node.id, onSelect]
  );

  return (
    <div
      className={`absolute select-none ${selected ? "z-20" : "z-10"}`}
      style={{
        left: node.position.x,
        top: node.position.y,
        transform: `scale(${1})`,
      }}
    >
      {/* Input handle */}
      {node.type !== "trigger" && (
        <div
          className="absolute -top-2 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-gray-400 dark:bg-gray-500 border-2 border-white dark:border-gray-900 z-30 cursor-crosshair hover:bg-gray-600 dark:hover:bg-gray-400 transition-colors"
          data-handle="input"
          data-node-id={node.id}
        />
      )}

      {/* Node body */}
      <div
        className={`
          w-[220px] rounded-lg border-2 shadow-md cursor-grab active:cursor-grabbing
          transition-shadow duration-150
          ${colors.bg} ${colors.border}
          ${selected ? "ring-2 ring-primary ring-offset-2 dark:ring-offset-gray-900 shadow-lg" : "hover:shadow-lg"}
        `}
        onMouseDown={handleMouseDown}
        onClick={handleClick}
      >
        {/* Header */}
        <div className={`${colors.header} px-3 py-1.5 rounded-t-md flex items-center gap-2`}>
          <GripVertical className="h-3 w-3 text-white/70" />
          <Icon className="h-3.5 w-3.5 text-white" />
          <span className="text-xs font-semibold text-white uppercase tracking-wide">
            {typeLabel}
          </span>
        </div>

        {/* Content */}
        <div className="px-3 py-2.5">
          <div className={`text-sm font-medium ${colors.text} truncate`}>
            {label}
          </div>
          {subtext && (
            <div className="text-xs text-muted-foreground mt-0.5 truncate">
              {subtext}
            </div>
          )}
        </div>
      </div>

      {/* Output handle */}
      <div
        className="absolute -bottom-2 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-gray-400 dark:bg-gray-500 border-2 border-white dark:border-gray-900 z-30 cursor-crosshair hover:bg-gray-600 dark:hover:bg-gray-400 transition-colors"
        data-handle="output"
        data-node-id={node.id}
      />
    </div>
  );
}
