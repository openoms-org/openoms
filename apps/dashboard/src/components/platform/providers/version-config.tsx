"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { EmptyState } from "@/components/shared/empty-state";
import { Boxes } from "lucide-react";
import { PublicationStateBadge } from "@/components/platform/publication-state-badge";
import { SchemaBuilder } from "@/components/platform/providers/schema-builder/schema-builder";
import { CapabilityMatrix } from "@/components/platform/providers/capability-matrix/capability-matrix";
import { StatusMappingWorkbench } from "@/components/platform/providers/status-mapping/status-mapping-workbench";
import { isPublishedState, type ProviderVersion } from "@/types/platform";

interface VersionConfigProps {
  providerId: string;
  versions: ProviderVersion[];
}

/**
 * The version configuration surface: pick a version, then edit its schema,
 * capabilities, and status mappings across three tabs. Published versions are
 * read-only — edits require creating a new draft version (handled elsewhere).
 */
export function VersionConfig({ providerId, versions }: VersionConfigProps) {
  const t = useTranslations("platform");
  // Only the user's explicit choice is stored; the effective selection is
  // derived each render so no effect is needed to track the versions list.
  const [chosenId, setChosenId] = useState<string | null>(null);

  if (versions.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("versionConfig.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={Boxes}
            title={t("versionConfig.noVersionsTitle")}
            description={t("versionConfig.noVersionsDescription")}
          />
        </CardContent>
      </Card>
    );
  }

  // Effective selection: the user's choice if still present, else the most
  // recent draft, else the first version.
  const chosen = chosenId
    ? versions.find((v) => v.id === chosenId)
    : undefined;
  const defaultVersion =
    versions.find((v) => !isPublishedState(v.publication_state)) ?? versions[0];
  const selected = chosen ?? defaultVersion;
  const readOnly = isPublishedState(selected.publication_state);

  return (
    <Card>
      <CardHeader className="flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <CardTitle>{t("versionConfig.title")}</CardTitle>
          <CardDescription>{t("versionConfig.description")}</CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <PublicationStateBadge state={selected.publication_state} />
          <Select value={selected.id} onValueChange={setChosenId}>
            <SelectTrigger className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {versions.map((version) => (
                <SelectItem key={version.id} value={version.id}>
                  {version.version} ·{" "}
                  {t(`publicationState.${version.publication_state}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="schema">
          <TabsList>
            <TabsTrigger value="schema">{t("versionConfig.tabs.schema")}</TabsTrigger>
            <TabsTrigger value="capabilities">
              {t("versionConfig.tabs.capabilities")}
            </TabsTrigger>
            <TabsTrigger value="status-mappings">
              {t("versionConfig.tabs.statusMappings")}
            </TabsTrigger>
          </TabsList>
          <TabsContent value="schema" className="pt-4">
            <SchemaBuilder
              key={`${selected.id}-schema`}
              providerId={providerId}
              versionId={selected.id}
              readOnly={readOnly}
            />
          </TabsContent>
          <TabsContent value="capabilities" className="pt-4">
            <CapabilityMatrix
              key={`${selected.id}-caps`}
              providerId={providerId}
              versionId={selected.id}
              readOnly={readOnly}
            />
          </TabsContent>
          <TabsContent value="status-mappings" className="pt-4">
            <StatusMappingWorkbench
              key={`${selected.id}-status`}
              providerId={providerId}
              versionId={selected.id}
              readOnly={readOnly}
            />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
