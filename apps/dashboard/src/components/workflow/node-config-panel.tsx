"use client";

import { useTranslations } from "next-intl";
import type { WorkflowNode } from "@/types/api";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  TRIGGER_OPTIONS,
  OPERATOR_OPTIONS,
  ACTION_TYPE_OPTIONS,
  NODE_COLORS,
} from "@/lib/workflow-types";
import { X, Trash2, Zap, GitBranch, Play } from "lucide-react";

interface NodeConfigPanelProps {
  node: WorkflowNode | null;
  onUpdate: (nodeId: string, data: Record<string, unknown>) => void;
  onDelete: (nodeId: string) => void;
  onClose: () => void;
}

function TriggerConfig({
  node,
  onUpdate,
}: {
  node: WorkflowNode;
  onUpdate: (data: Record<string, unknown>) => void;
}) {
  const t = useTranslations("workflows.nodeConfig");
  const event = (node.data.event as string) || "";

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>{t("triggerEvent")}</Label>
        <Select
          value={event || "none"}
          onValueChange={(v) => {
            const selected = TRIGGER_OPTIONS.find((t) => t.value === v);
            onUpdate({
              ...node.data,
              event: v === "none" ? "" : v,
              label: selected?.label || v,
            });
          }}
        >
          <SelectTrigger>
            <SelectValue placeholder={t("selectEvent")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">{t("selectEvent")}</SelectItem>
            {TRIGGER_OPTIONS.map((t) => (
              <SelectItem key={t.value} value={t.value}>
                {t.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function ConditionConfig({
  node,
  onUpdate,
}: {
  node: WorkflowNode;
  onUpdate: (data: Record<string, unknown>) => void;
}) {
  const t = useTranslations("workflows.nodeConfig");
  const field = (node.data.field as string) || "";
  const operator = (node.data.operator as string) || "eq";
  const value = node.data.value ?? "";

  const updateField = (key: string, val: unknown) => {
    const newData = { ...node.data, [key]: val };
    newData.label = `${newData.field || ""} ${newData.operator || "eq"} ${String(newData.value ?? "")}`;
    onUpdate(newData);
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>{t("field")}</Label>
        <Input
          value={field}
          onChange={(e) => updateField("field", e.target.value)}
          placeholder={t("fieldPlaceholder")}
        />
      </div>
      <div className="space-y-2">
        <Label>{t("operator")}</Label>
        <Select value={operator} onValueChange={(v) => updateField("operator", v)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {OPERATOR_OPTIONS.map((op) => (
              <SelectItem key={op.value} value={op.value}>
                {op.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>{t("value")}</Label>
        <Input
          value={String(value)}
          onChange={(e) => updateField("value", e.target.value)}
          placeholder={t("valuePlaceholder")}
        />
      </div>
    </div>
  );
}

function ActionConfig({
  node,
  onUpdate,
}: {
  node: WorkflowNode;
  onUpdate: (data: Record<string, unknown>) => void;
}) {
  const t = useTranslations("workflows.nodeConfig");
  const actionType = (node.data.actionType as string) || "";
  const delaySeconds = (node.data.delay_seconds as number) || 0;

  const updateData = (key: string, val: unknown) => {
    const newData = { ...node.data, [key]: val };
    onUpdate(newData);
  };

  const setActionType = (type: string) => {
    const selected = ACTION_TYPE_OPTIONS.find((a) => a.value === type);
    onUpdate({
      actionType: type,
      label: selected?.label || type,
      delay_seconds: delaySeconds,
    });
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>{t("actionType")}</Label>
        <Select value={actionType || "none"} onValueChange={(v) => setActionType(v === "none" ? "" : v)}>
          <SelectTrigger>
            <SelectValue placeholder={t("selectActionType")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">{t("selectActionType")}</SelectItem>
            {ACTION_TYPE_OPTIONS.map((a) => (
              <SelectItem key={a.value} value={a.value}>
                {a.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Action-specific fields */}
      {actionType === "set_status" && (
        <div className="space-y-2">
          <Label>{t("newStatus")}</Label>
          <Input
            value={String(node.data.status ?? "")}
            onChange={(e) => updateData("status", e.target.value)}
            placeholder="np. confirmed, processing, shipped"
          />
        </div>
      )}

      {actionType === "add_tag" && (
        <div className="space-y-2">
          <Label>{t("tag")}</Label>
          <Input
            value={String(node.data.tag ?? "")}
            onChange={(e) => updateData("tag", e.target.value)}
            placeholder="np. VIP, pilne"
          />
        </div>
      )}

      {actionType === "send_email" && (
        <div className="space-y-2">
          <Label>{t("emailAddress")}</Label>
          <Input
            value={String(node.data.to ?? "")}
            onChange={(e) => updateData("to", e.target.value)}
            placeholder="np. admin@firma.pl"
          />
        </div>
      )}

      {actionType === "create_invoice" && (
        <div className="space-y-2">
          <Label>{t("invoiceType")}</Label>
          <Input
            value={String(node.data.invoice_type ?? "vat")}
            onChange={(e) => updateData("invoice_type", e.target.value)}
            placeholder="np. vat"
          />
        </div>
      )}

      {actionType === "webhook" && (
        <div className="space-y-2">
          <Label>{t("webhookUrl")}</Label>
          <Input
            value={String(node.data.url ?? "")}
            onChange={(e) => updateData("url", e.target.value)}
            placeholder="https://..."
          />
        </div>
      )}

      <div className="space-y-2">
        <Label>{t("delayMinutes")}</Label>
        <Input
          type="number"
          min={0}
          value={Math.round(delaySeconds / 60)}
          onChange={(e) => {
            const minutes = parseInt(e.target.value) || 0;
            updateData("delay_seconds", minutes * 60);
          }}
          placeholder={t("immediateHint")}
        />
        <p className="text-xs text-muted-foreground">{t("immediateHint")}</p>
      </div>
    </div>
  );
}

export function NodeConfigPanel({ node, onUpdate, onDelete, onClose }: NodeConfigPanelProps) {
  const t = useTranslations("workflows.nodeConfig");

  if (!node) return null;

  const colors = NODE_COLORS[node.type];
  const TypeIcon = node.type === "trigger" ? Zap : node.type === "condition" ? GitBranch : Play;
  const typeLabel =
    node.type === "trigger"
      ? t("trigger")
      : node.type === "condition"
        ? t("condition")
        : t("action");

  const handleUpdate = (data: Record<string, unknown>) => {
    onUpdate(node.id, data);
  };

  return (
    <div className="w-[300px] border-l bg-background flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className={`p-1.5 rounded ${colors.header}`}>
            <TypeIcon className="h-3.5 w-3.5 text-white" />
          </div>
          <span className="text-sm font-semibold">{typeLabel}</span>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Config form */}
      <ScrollArea className="flex-1 p-3">
        {node.type === "trigger" && <TriggerConfig node={node} onUpdate={handleUpdate} />}
        {node.type === "condition" && <ConditionConfig node={node} onUpdate={handleUpdate} />}
        {node.type === "action" && <ActionConfig node={node} onUpdate={handleUpdate} />}
      </ScrollArea>

      {/* Delete */}
      <div className="p-3 border-t">
        <Button
          variant="destructive"
          size="sm"
          className="w-full"
          onClick={() => onDelete(node.id)}
        >
          <Trash2 className="h-4 w-4" />
          {t("deleteNode")}
        </Button>
      </div>
    </div>
  );
}
