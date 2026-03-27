"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useCreateAutomationRule } from "@/hooks/use-automation";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AUTOMATION_TRIGGER_EVENTS,
  AUTOMATION_TRIGGER_LABELS,
  AUTOMATION_OPERATORS,
  AUTOMATION_OPERATOR_LABELS,
  AUTOMATION_ACTION_TYPES,
  AUTOMATION_ACTION_LABELS,
} from "@/lib/constants";
import { ArrowLeft, Save, Plus, Trash2, Loader2 } from "lucide-react";
import type { AutomationCondition, AutomationAction } from "@/types/api";
import { useTranslations } from "next-intl";

export default function NewAutomationRulePage() {
  const t = useTranslations("automation");
  const tc = useTranslations("common");
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const createRule = useCreateAutomationRule();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [priority, setPriority] = useState(0);
  const [triggerEvent, setTriggerEvent] = useState("");
  const [conditions, setConditions] = useState<AutomationCondition[]>([]);
  const [actions, setActions] = useState<AutomationAction[]>([]);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.replace("/");
    }
  }, [authLoading, isAdmin, router]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  const addCondition = () => {
    setConditions([...conditions, { field: "", operator: "eq", value: "" }]);
  };

  const updateCondition = (index: number, updates: Partial<AutomationCondition>) => {
    setConditions(conditions.map((c, i) => (i === index ? { ...c, ...updates } : c)));
  };

  const removeCondition = (index: number) => {
    setConditions(conditions.filter((_, i) => i !== index));
  };

  const addAction = () => {
    setActions([...actions, { type: "set_status", config: {}, delay_seconds: 0 }]);
  };

  const updateAction = (index: number, updates: Partial<AutomationAction>) => {
    setActions(actions.map((a, i) => (i === index ? { ...a, ...updates } : a)));
  };

  const removeAction = (index: number) => {
    setActions(actions.filter((_, i) => i !== index));
  };

  const handleSave = async () => {
    if (!name.trim()) {
      toast.error(t("ruleNameRequired"));
      return;
    }
    if (!triggerEvent) {
      toast.error(t("zdarzenieWyzwalajaceJestWymagane"));
      return;
    }

    try {
      await createRule.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
        enabled,
        priority,
        trigger_event: triggerEvent,
        conditions,
        actions,
      });
      toast.success(t("automationRuleCreated"));
      router.push("/settings/automation");
    } catch (err) {
      const message = err instanceof Error ? err.message : t("ruleCreateError");
      toast.error(message);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => router.push("/settings/automation")}>
          <ArrowLeft className="h-4 w-4" />
          {t("wroc")}
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{t("newRule")}</h1>
          <p className="text-muted-foreground">
            {t("createNewAutomationRule")}
          </p>
        </div>
      </div>

      {/* Basic info */}
      <Card>
        <CardHeader>
          <CardTitle>{t("basicInfo")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("nameRequired")}</Label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("npAutopotwierdzenieZamowienAllegro")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("priorityLabel")}</Label>
              <Input
                type="number"
                value={priority}
                onChange={(e) => setPriority(parseInt(e.target.value) || 0)}
                placeholder="0"
              />
              <p className="text-xs text-muted-foreground">
                {t("higherPriorityHint")}
              </p>
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("descriptionLabel")}</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("optionalRuleDescription")}
              rows={2}
            />
          </div>
          <div className="flex items-center gap-3">
            <Switch checked={enabled} onCheckedChange={setEnabled} />
            <Label>{t("ruleActive")}</Label>
          </div>
        </CardContent>
      </Card>

      {/* Trigger */}
      <Card>
        <CardHeader>
          <CardTitle>{t("zdarzenieWyzwalajace1")}</CardTitle>
        </CardHeader>
        <CardContent>
          <Select value={triggerEvent || "none"} onValueChange={(v) => setTriggerEvent(v === "none" ? "" : v)}>
            <SelectTrigger className="w-full max-w-md">
              <SelectValue placeholder={t("selectEvent")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">{t("selectEventPlaceholder")}</SelectItem>
              {AUTOMATION_TRIGGER_EVENTS.map((event) => (
                <SelectItem key={event} value={event}>
                  {AUTOMATION_TRIGGER_LABELS[event] || event}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Conditions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>{t("conditions")}</CardTitle>
            <Button variant="outline" size="sm" onClick={addCondition}>
              <Plus className="h-4 w-4" />
              {t("addCondition")}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {conditions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("noConditionsRuleTriggersOnEvery")}
            </p>
          ) : (
            conditions.map((condition, index) => (
              <div key={index} className="flex items-start gap-3 rounded-md border p-3">
                <div className="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
                  <div className="space-y-1">
                    <Label className="text-xs">{t("fieldLabel")}</Label>
                    <Input
                      value={condition.field}
                      onChange={(e) => updateCondition(index, { field: e.target.value })}
                      placeholder={t("fieldPlaceholder")}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t("operatorLabel")}</Label>
                    <Select
                      value={condition.operator}
                      onValueChange={(v) => updateCondition(index, { operator: v })}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {AUTOMATION_OPERATORS.map((op) => (
                          <SelectItem key={op} value={op}>
                            {AUTOMATION_OPERATOR_LABELS[op] || op}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t("detail.value")}</Label>
                    <Input
                      value={String(condition.value ?? "")}
                      onChange={(e) => updateCondition(index, { value: e.target.value })}
                      placeholder={t("wartosc")}
                    />
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-5"
                  onClick={() => removeCondition(index)}
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      {/* Actions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>{t("actions")}</CardTitle>
            <Button variant="outline" size="sm" onClick={addAction}>
              <Plus className="h-4 w-4" />
              {t("dodajAkcje")}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {actions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("brakAkcjiDodajCoNajmniejJednaAkcje")}
            </p>
          ) : (
            actions.map((action, index) => (
              <div key={index} className="flex items-start gap-3 rounded-md border p-3">
                <div className="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
                  <div className="space-y-1">
                    <Label className="text-xs">{t("actionTypeLabel")}</Label>
                    <Select
                      value={action.type}
                      onValueChange={(v) => updateAction(index, { type: v, config: {} })}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {AUTOMATION_ACTION_TYPES.map((at) => (
                          <SelectItem key={at} value={at}>
                            {AUTOMATION_ACTION_LABELS[at] || at}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">
                      {action.type === "set_status" && t("actionNewStatus")}
                      {action.type === "add_tag" && t("actionTag")}
                      {action.type === "send_email" && t("actionEmail")}
                      {action.type === "create_invoice" && t("actionInvoiceType")}
                      {action.type === "webhook" && t("actionWebhookUrl")}
                    </Label>
                    <Input
                      value={String(
                        action.type === "set_status"
                          ? action.config.status ?? ""
                          : action.type === "add_tag"
                          ? action.config.tag ?? ""
                          : action.type === "send_email"
                          ? action.config.to ?? ""
                          : action.type === "create_invoice"
                          ? action.config.invoice_type ?? "vat"
                          : action.type === "webhook"
                          ? action.config.url ?? ""
                          : ""
                      )}
                      onChange={(e) => {
                        const key =
                          action.type === "set_status"
                            ? "status"
                            : action.type === "add_tag"
                            ? "tag"
                            : action.type === "send_email"
                            ? "to"
                            : action.type === "create_invoice"
                            ? "invoice_type"
                            : action.type === "webhook"
                            ? "url"
                            : "value";
                        updateAction(index, {
                          config: { ...action.config, [key]: e.target.value },
                        });
                      }}
                      placeholder={
                        action.type === "set_status"
                          ? t("placeholderStatus")
                          : action.type === "add_tag"
                          ? t("placeholderTag")
                          : action.type === "send_email"
                          ? t("placeholderEmail")
                          : action.type === "create_invoice"
                          ? t("placeholderInvoice")
                          : action.type === "webhook"
                          ? "https://..."
                          : ""
                      }
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t("delayMinutes")}</Label>
                    <Input
                      type="number"
                      min={0}
                      value={Math.round((action.delay_seconds ?? 0) / 60)}
                      onChange={(e) => {
                        const minutes = parseInt(e.target.value) || 0;
                        updateAction(index, { delay_seconds: minutes * 60 });
                      }}
                      placeholder={t("delayImmediate")}
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("delayImmediate")}
                    </p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-5"
                  onClick={() => removeAction(index)}
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      {/* Save */}
      <div className="flex justify-end gap-3">
        <Button variant="outline" onClick={() => router.push("/settings/automation")}>
          {t("cancelButton")}
        </Button>
        <Button onClick={handleSave} disabled={createRule.isPending}>
          {createRule.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          {t("createAutomationRule")}
        </Button>
      </div>
    </div>
  );
}
