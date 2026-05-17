"use client";

import { useTranslations } from "next-intl";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Save,
  Power,
  PowerOff,
  Undo2,
  Redo2,
  ZoomIn,
  ZoomOut,
  Maximize,
  Play,
  ArrowLeft,
  Loader2,
} from "lucide-react";

interface WorkflowToolbarProps {
  name: string;
  onNameChange: (name: string) => void;
  isActive: boolean;
  onSave: () => void;
  onActivate: () => void;
  onDeactivate: () => void;
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFitToScreen: () => void;
  onTest: () => void;
  onBack: () => void;
  isSaving: boolean;
  zoom: number;
}

export function WorkflowToolbar({
  name,
  onNameChange,
  isActive,
  onSave,
  onActivate,
  onDeactivate,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onZoomIn,
  onZoomOut,
  onFitToScreen,
  onTest,
  onBack,
  isSaving,
  zoom,
}: WorkflowToolbarProps) {
  const t = useTranslations("workflows.toolbar");

  return (
    <div className="h-14 border-b bg-background flex items-center gap-2 px-3 shrink-0">
      <Button variant="ghost" size="sm" onClick={onBack} className="mr-1">
        <ArrowLeft className="h-4 w-4" />
      </Button>

      <Input
        value={name}
        onChange={(e) => onNameChange(e.target.value)}
        className="w-[250px] h-8 text-sm font-medium"
        placeholder={t("namePlaceholder")}
      />

      <Badge
        variant="outline"
        className={
          isActive
            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
            : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
        }
      >
        {isActive ? t("active") : t("draft")}
      </Badge>

      <div className="h-6 w-px bg-border mx-1" />

      {/* Undo/Redo */}
      <Button variant="ghost" size="sm" onClick={onUndo} disabled={!canUndo} title={t("undo")}>
        <Undo2 className="h-4 w-4" />
      </Button>
      <Button variant="ghost" size="sm" onClick={onRedo} disabled={!canRedo} title={t("redo")}>
        <Redo2 className="h-4 w-4" />
      </Button>

      <div className="h-6 w-px bg-border mx-1" />

      {/* Zoom */}
      <Button variant="ghost" size="sm" onClick={onZoomOut} title={t("zoomOut")}>
        <ZoomOut className="h-4 w-4" />
      </Button>
      <span className="text-xs text-muted-foreground w-12 text-center tabular-nums">
        {Math.round(zoom * 100)}%
      </span>
      <Button variant="ghost" size="sm" onClick={onZoomIn} title={t("zoomIn")}>
        <ZoomIn className="h-4 w-4" />
      </Button>
      <Button variant="ghost" size="sm" onClick={onFitToScreen} title={t("fitToScreen")}>
        <Maximize className="h-4 w-4" />
      </Button>

      <div className="flex-1" />

      {/* Test */}
      <Button variant="outline" size="sm" onClick={onTest}>
        <Play className="h-4 w-4" />
        {t("test")}
      </Button>

      {/* Activate/Deactivate */}
      {isActive ? (
        <Button variant="outline" size="sm" onClick={onDeactivate}>
          <PowerOff className="h-4 w-4" />
          {t("deactivate")}
        </Button>
      ) : (
        <Button variant="outline" size="sm" onClick={onActivate}>
          <Power className="h-4 w-4" />
          {t("activate")}
        </Button>
      )}

      {/* Save */}
      <Button size="sm" onClick={onSave} disabled={isSaving}>
        {isSaving ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Save className="h-4 w-4" />
        )}
        {t("save")}
      </Button>
    </div>
  );
}
