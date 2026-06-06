"use client";

import { useMemo, useState } from "react";
import { Plus, Trash2, Lock } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
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
import { cn } from "@/lib/utils";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";
import { ApiClientError } from "@/lib/api-client";
import {
  useProviderSchema,
  useUpdateProviderSchema,
} from "@/hooks/use-platform-provider-config";
import { FieldEditor } from "@/components/platform/providers/schema-builder/field-editor";
import { SetupPreview } from "@/components/platform/providers/schema-builder/setup-preview";
import {
  emptyField,
  validateSchema,
} from "@/components/platform/providers/schema-builder/schema-helpers";
import {
  PROVIDER_FIELD_GROUP_KEYS,
  type ProviderField,
  type ProviderFieldGroup,
  type ProviderFieldGroupKey,
} from "@/types/platform";

interface SchemaBuilderProps {
  providerId: string;
  versionId: string;
  readOnly: boolean;
}

interface Selection {
  groupIndex: number;
  fieldIndex: number;
}

function defaultGroups(): ProviderFieldGroup[] {
  return PROVIDER_FIELD_GROUP_KEYS.map((key) => ({
    key,
    label: key,
    fields: [],
  }));
}

/** Merge the server schema into the canonical group ordering for editing. */
function normalizeGroups(groups: ProviderFieldGroup[]): ProviderFieldGroup[] {
  const byKey = new Map<ProviderFieldGroupKey, ProviderFieldGroup>();
  for (const g of groups) {
    byKey.set(g.key, g);
  }
  return PROVIDER_FIELD_GROUP_KEYS.map((key) => {
    const existing = byKey.get(key);
    return existing ?? { key, label: key, fields: [] };
  });
}

/** Drop empty groups before sending — the backend rejects empty field keys. */
function prunedGroups(groups: ProviderFieldGroup[]): ProviderFieldGroup[] {
  return groups
    .map((g) => ({ ...g, fields: g.fields.filter((f) => f.key.trim() !== "") }))
    .filter((g) => g.fields.length > 0);
}

export function SchemaBuilder({ providerId, versionId, readOnly }: SchemaBuilderProps) {
  const t = useTranslations("platform");
  const { data, isLoading, isError, refetch } = useProviderSchema(providerId, versionId);
  const updateSchema = useUpdateProviderSchema(providerId, versionId);

  // Reset the editable draft whenever a different version's schema loads.
  const [groups, setGroups] = useEffectSyncedState<ProviderFieldGroup[]>(
    data ? normalizeGroups(data.groups ?? []) : defaultGroups(),
    `${versionId}:${data?.updated_at ?? ""}`,
  );
  const [selected, setSelected] = useState<Selection | null>(null);

  const issuesByKey = useMemo(() => validateSchema(groups), [groups]);
  const blockingCount = useMemo(() => {
    let n = 0;
    for (const issues of issuesByKey.values()) {
      if (issues.some((i) => !i.warning)) n += 1;
    }
    return n;
  }, [issuesByKey]);

  const selectedField: ProviderField | null =
    selected != null
      ? (groups[selected.groupIndex]?.fields[selected.fieldIndex] ?? null)
      : null;

  const selectedFieldId = selectedField
    ? selectedField.key || `${groups[selected!.groupIndex].key}:${selectedField.label}`
    : "";

  const addField = (groupIndex: number) => {
    setGroups((prev) => {
      const next = prev.map((g) => ({ ...g, fields: [...g.fields] }));
      next[groupIndex].fields.push(emptyField());
      return next;
    });
    setSelected({ groupIndex, fieldIndex: groups[groupIndex].fields.length });
  };

  const updateField = (field: ProviderField) => {
    if (!selected) return;
    setGroups((prev) => {
      const next = prev.map((g) => ({ ...g, fields: [...g.fields] }));
      next[selected.groupIndex].fields[selected.fieldIndex] = field;
      return next;
    });
  };

  const removeField = (groupIndex: number, fieldIndex: number) => {
    setGroups((prev) => {
      const next = prev.map((g) => ({ ...g, fields: [...g.fields] }));
      next[groupIndex].fields.splice(fieldIndex, 1);
      return next;
    });
    setSelected(null);
  };

  const handleSave = () => {
    if (blockingCount > 0) {
      toast.error(t("schemaBuilder.fixErrors"));
      return;
    }
    updateSchema.mutate(
      { groups: prunedGroups(groups) },
      {
        onSuccess: () => toast.success(t("schemaBuilder.saved")),
        onError: (err) =>
          toast.error(
            err instanceof ApiClientError ? err.message : t("saveError"),
          ),
      },
    );
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <div className="rounded-md border border-destructive bg-destructive/10 p-4">
        <p className="text-sm text-destructive">{t("loadError")}</p>
        <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
          {t("retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {readOnly && (
        <div className="flex items-center gap-2 rounded-md border border-warning/40 bg-warning/10 p-3 text-sm text-warning">
          <Lock className="h-4 w-4" />
          {t("schemaBuilder.readOnlyNotice")}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[1fr_1.2fr]">
        {/* Left: field list grouped by group key */}
        <Card>
          <CardHeader>
            <CardTitle>{t("schemaBuilder.fieldsTitle")}</CardTitle>
            <CardDescription>{t("schemaBuilder.fieldsDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            {groups.map((group, gi) => (
              <div key={group.key} className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    {t(`schemaBuilder.group.${group.key}`)}
                  </span>
                  {!readOnly && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => addField(gi)}
                    >
                      <Plus className="mr-1 h-3 w-3" />
                      {t("schemaBuilder.addField")}
                    </Button>
                  )}
                </div>
                {group.fields.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    {t("schemaBuilder.noFields")}
                  </p>
                ) : (
                  <ul className="space-y-1">
                    {group.fields.map((field, fi) => {
                      const id = field.key || `${group.key}:${field.label}`;
                      const fieldIssues = issuesByKey.get(id) ?? [];
                      const hasBlocking = fieldIssues.some((i) => !i.warning);
                      const isSelected =
                        selected?.groupIndex === gi && selected?.fieldIndex === fi;
                      return (
                        <li key={`${group.key}-${fi}`}>
                          <button
                            type="button"
                            onClick={() => setSelected({ groupIndex: gi, fieldIndex: fi })}
                            className={cn(
                              "flex w-full items-center justify-between rounded-md border px-3 py-2 text-left text-sm",
                              isSelected
                                ? "border-primary bg-primary/5"
                                : "border-border hover:bg-accent",
                            )}
                          >
                            <span className="min-w-0 truncate">
                              <span className="font-mono text-xs text-muted-foreground">
                                {field.key || t("schemaBuilder.unnamedKey")}
                              </span>
                              <span className="ml-2">{field.label}</span>
                            </span>
                            <span className="flex items-center gap-1.5">
                              {field.secret && (
                                <Badge variant="warning">
                                  {t("schemaBuilder.field.secret")}
                                </Badge>
                              )}
                              {hasBlocking && (
                                <Badge variant="destructive">
                                  {t("schemaBuilder.hasError")}
                                </Badge>
                              )}
                              {!readOnly && (
                                <Trash2
                                  className="h-3.5 w-3.5 text-muted-foreground hover:text-destructive"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    removeField(gi, fi);
                                  }}
                                />
                              )}
                            </span>
                          </button>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Right: editor for the selected field */}
        <Card>
          <CardHeader>
            <CardTitle>{t("schemaBuilder.editorTitle")}</CardTitle>
          </CardHeader>
          <CardContent>
            {selectedField ? (
              <FieldEditor
                field={selectedField}
                issues={issuesByKey.get(selectedFieldId) ?? []}
                readOnly={readOnly}
                onChange={updateField}
              />
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("schemaBuilder.selectField")}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Bottom: tenant setup preview */}
      <Card>
        <CardHeader>
          <CardTitle>{t("schemaBuilder.preview.title")}</CardTitle>
          <CardDescription>{t("schemaBuilder.preview.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <SchemaPreviewBody groups={groups} />
        </CardContent>
      </Card>

      {!readOnly && (
        <div className="flex items-center justify-end gap-3">
          {blockingCount > 0 && (
            <span className="text-sm text-destructive">
              {t("schemaBuilder.errorCount", { count: blockingCount })}
            </span>
          )}
          <Button
            type="button"
            onClick={handleSave}
            disabled={updateSchema.isPending || blockingCount > 0}
          >
            {updateSchema.isPending ? t("saving") : t("schemaBuilder.save")}
          </Button>
        </div>
      )}
    </div>
  );
}

// The preview's environment filter is local UI state, isolated here so it
// doesn't re-render the whole builder on every change.
function SchemaPreviewBody({ groups }: { groups: ProviderFieldGroup[] }) {
  const t = useTranslations("platform");
  const [env, setEnv] = useState<"all" | "production" | "sandbox">("all");
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground">
          {t("schemaBuilder.preview.environment")}
        </span>
        <Select value={env} onValueChange={(v) => setEnv(v as typeof env)}>
          <SelectTrigger className="h-8 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("schemaBuilder.envScope.all")}</SelectItem>
            <SelectItem value="production">
              {t("schemaBuilder.envScope.production")}
            </SelectItem>
            <SelectItem value="sandbox">
              {t("schemaBuilder.envScope.sandbox")}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <SetupPreview groups={groups} environment={env} />
    </div>
  );
}
