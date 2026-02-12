"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import {
  useAutomationRule,
  useUpdateAutomationRule,
  useDeleteAutomationRule,
  useAutomationRuleLogs,
  useTestAutomationRule,
} from "@/hooks/use-automation";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AUTOMATION_TRIGGER_EVENTS,
  AUTOMATION_TRIGGER_LABELS,
  AUTOMATION_OPERATORS,
  AUTOMATION_OPERATOR_LABELS,
  AUTOMATION_ACTION_TYPES,
  AUTOMATION_ACTION_LABELS,
} from "@/lib/constants";
import { ArrowLeft, Save, Plus, Trash2, Loader2, Play } from "lucide-react";
import type { AutomationCondition, AutomationAction, AutomationRuleLog } from "@/types/api";

export default function AutomationRuleDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();

  const { data: rule, isLoading } = useAutomationRule(id);
  const { data: logsData, isLoading: logsLoading } = useAutomationRuleLogs(id);
  const updateRule = useUpdateAutomationRule(id);
  const deleteRule = useDeleteAutomationRule();
  const testRule = useTestAutomationRule(id);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [priority, setPriority] = useState(0);
  const [triggerEvent, setTriggerEvent] = useState("");
  const [conditions, setConditions] = useState<AutomationCondition[]>([]);
  const [actions, setActions] = useState<AutomationAction[]>([]);
  const [testData, setTestData] = useState("{}");
  const [testResult, setTestResult] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.replace("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (rule && !initialized) {
      setName(rule.name);
      setDescription(rule.description || "");
      setEnabled(rule.enabled);
      setPriority(rule.priority);
      setTriggerEvent(rule.trigger_event);
      setConditions(Array.isArray(rule.conditions) ? rule.conditions : []);
      setActions(Array.isArray(rule.actions) ? rule.actions : []);
      setInitialized(true);
    }
  }, [rule, initialized]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!rule) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" size="sm" onClick={() => router.push("/settings/automation")}>
          <ArrowLeft className="h-4 w-4" />
          Wróć
        </Button>
        <p className="text-muted-foreground">Reguła nie została znaleziona.</p>
      </div>
    );
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
      toast.error("Nazwa reguły jest wymagana");
      return;
    }

    try {
      await updateRule.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
        enabled,
        priority,
        trigger_event: triggerEvent || undefined,
        conditions,
        actions,
      });
      toast.success("Reguła została zaktualizowana");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Nie udało się zaktualizować reguły";
      toast.error(message);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Czy na pewno chcesz usunąć tę regułę?")) return;
    try {
      await deleteRule.mutateAsync(id);
      toast.success("Reguła została usunięta");
      router.push("/settings/automation");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Nie udało się usunąć reguły";
      toast.error(message);
    }
  };

  const handleTest = async () => {
    try {
      const data = JSON.parse(testData);
      const result = await testRule.mutateAsync({ data });
      setTestResult(JSON.stringify(result, null, 2));
    } catch (err) {
      if (err instanceof SyntaxError) {
        toast.error("Nieprawidłowy format JSON");
      } else {
        const message = err instanceof Error ? err.message : "Nie udało się przetestować reguły";
        toast.error(message);
      }
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("pl-PL", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.push("/settings/automation")}>
            <ArrowLeft className="h-4 w-4" />
            Wróć
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{rule.name}</h1>
            <p className="text-muted-foreground text-sm">
              Ostatnie wykonanie:{" "}
              {rule.last_fired_at ? formatDate(rule.last_fired_at) : "nigdy"}{" "}
              | Wykonania: {rule.fire_count}
            </p>
          </div>
        </div>
        <Button variant="destructive" size="sm" onClick={handleDelete} disabled={deleteRule.isPending}>
          <Trash2 className="h-4 w-4" />
          Usuń
        </Button>
      </div>

      <Tabs defaultValue="edit">
        <TabsList>
          <TabsTrigger value="edit">Edycja</TabsTrigger>
          <TabsTrigger value="logs">
            Historia wykonań
          </TabsTrigger>
          <TabsTrigger value="test">Test</TabsTrigger>
        </TabsList>

        <TabsContent value="edit" className="space-y-6 mt-4">
          {/* Basic info */}
          <Card>
            <CardHeader>
              <CardTitle>Podstawowe informacje</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>Nazwa *</Label>
                  <Input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Nazwa reguły"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Priorytet</Label>
                  <Input
                    type="number"
                    value={priority}
                    onChange={(e) => setPriority(parseInt(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Opis</Label>
                <Textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                />
              </div>
              <div className="flex items-center gap-3">
                <Switch checked={enabled} onCheckedChange={setEnabled} />
                <Label>Reguła aktywna</Label>
              </div>
            </CardContent>
          </Card>

          {/* Trigger */}
          <Card>
            <CardHeader>
              <CardTitle>Zdarzenie wyzwalające</CardTitle>
            </CardHeader>
            <CardContent>
              <Select value={triggerEvent || "none"} onValueChange={(v) => setTriggerEvent(v === "none" ? "" : v)}>
                <SelectTrigger className="w-full max-w-md">
                  <SelectValue placeholder="Wybierz zdarzenie" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Wybierz zdarzenie...</SelectItem>
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
                <CardTitle>Warunki</CardTitle>
                <Button variant="outline" size="sm" onClick={addCondition}>
                  <Plus className="h-4 w-4" />
                  Dodaj warunek
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {conditions.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Brak warunków - reguła uruchomi się przy każdym zdarzeniu wybranego typu.
                </p>
              ) : (
                conditions.map((condition, index) => (
                  <div key={index} className="flex items-start gap-3 rounded-md border p-3">
                    <div className="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
                      <div className="space-y-1">
                        <Label className="text-xs">Pole</Label>
                        <Input
                          value={condition.field}
                          onChange={(e) => updateCondition(index, { field: e.target.value })}
                          placeholder="np. status, total_amount"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Operator</Label>
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
                        <Label className="text-xs">Wartość</Label>
                        <Input
                          value={String(condition.value ?? "")}
                          onChange={(e) => updateCondition(index, { value: e.target.value })}
                          placeholder="wartość"
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
                <CardTitle>Akcje</CardTitle>
                <Button variant="outline" size="sm" onClick={addAction}>
                  <Plus className="h-4 w-4" />
                  Dodaj akcję
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {actions.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Brak akcji - dodaj co najmniej jedną akcję do wykonania.
                </p>
              ) : (
                actions.map((action, index) => (
                  <div key={index} className="flex items-start gap-3 rounded-md border p-3">
                    <div className="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
                      <div className="space-y-1">
                        <Label className="text-xs">Typ akcji</Label>
                        <Select
                          value={action.type}
                          onValueChange={(v) => updateAction(index, { type: v, config: {} })}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {AUTOMATION_ACTION_TYPES.map((t) => (
                              <SelectItem key={t} value={t}>
                                {AUTOMATION_ACTION_LABELS[t] || t}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">
                          {action.type === "set_status" && "Nowy status"}
                          {action.type === "add_tag" && "Tag"}
                          {action.type === "send_email" && "Adres e-mail"}
                          {action.type === "create_invoice" && "Typ faktury"}
                          {action.type === "webhook" && "URL webhooka"}
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
                              ? "np. confirmed"
                              : action.type === "add_tag"
                              ? "np. vip"
                              : action.type === "send_email"
                              ? "np. admin@firma.pl"
                              : action.type === "create_invoice"
                              ? "np. vat"
                              : action.type === "webhook"
                              ? "https://..."
                              : ""
                          }
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Opoznienie (minuty)</Label>
                        <Input
                          type="number"
                          min={0}
                          value={Math.round((action.delay_seconds ?? 0) / 60)}
                          onChange={(e) => {
                            const minutes = parseInt(e.target.value) || 0;
                            updateAction(index, { delay_seconds: minutes * 60 });
                          }}
                          placeholder="0 = natychmiast"
                        />
                        <p className="text-xs text-muted-foreground">
                          0 = natychmiast
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
          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={updateRule.isPending}>
              {updateRule.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              Zapisz zmiany
            </Button>
          </div>
        </TabsContent>

        <TabsContent value="logs" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>Historia wykonań</CardTitle>
            </CardHeader>
            <CardContent>
              {logsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Data</TableHead>
                        <TableHead>Zdarzenie</TableHead>
                        <TableHead>Encja</TableHead>
                        <TableHead>Warunki</TableHead>
                        <TableHead>Błąd</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {logsData?.items && logsData.items.length > 0 ? (
                        logsData.items.map((log: AutomationRuleLog) => (
                          <TableRow key={log.id}>
                            <TableCell className="whitespace-nowrap text-sm">
                              {formatDate(log.executed_at)}
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline">
                                {AUTOMATION_TRIGGER_LABELS[log.trigger_event] || log.trigger_event}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-sm">
                              {log.entity_type}/{log.entity_id.slice(0, 8)}...
                            </TableCell>
                            <TableCell>
                              <Badge
                                className={
                                  log.conditions_met
                                    ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                    : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                }
                              >
                                {log.conditions_met ? "Spełnione" : "Niespełnione"}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-sm text-destructive max-w-[200px] truncate">
                              {log.error_message || "-"}
                            </TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                            Brak historii wykonań
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="test" className="mt-4 space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Test reguły (dry-run)</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Podaj dane testowe w formacie JSON, aby sprawdzić jak reguła zareagowałaby na zdarzenie.
              </p>
              <div className="space-y-2">
                <Label>Dane testowe (JSON)</Label>
                <Textarea
                  value={testData}
                  onChange={(e) => setTestData(e.target.value)}
                  rows={6}
                  className="font-mono text-sm"
                  placeholder='{"status": "new", "total_amount": 150, "source": "allegro"}'
                />
              </div>
              <Button onClick={handleTest} disabled={testRule.isPending}>
                {testRule.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
                Uruchom test
              </Button>
              {testResult && (
                <div className="space-y-2">
                  <Label>Wynik testu</Label>
                  <pre className="rounded-md border bg-muted p-4 text-sm font-mono overflow-auto max-h-[400px]">
                    {testResult}
                  </pre>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
