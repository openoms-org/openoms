"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateAutomationRule } from "@/hooks/use-automation";
import { useConvertWorkflow, useValidateWorkflow } from "@/hooks/use-workflows";
import { WorkflowCanvas, zoomIn, zoomOut } from "@/components/workflow/workflow-canvas";
import { WorkflowSidebar } from "@/components/workflow/workflow-sidebar";
import { WorkflowToolbar } from "@/components/workflow/workflow-toolbar";
import { NodeConfigPanel } from "@/components/workflow/node-config-panel";
import { createEmptyWorkflow, type PaletteItem } from "@/lib/workflow-types";
import type { WorkflowDefinition, WorkflowNode } from "@/types/api";

const MAX_HISTORY = 50;

export default function NewWorkflowEditorPage() {
  const t = useTranslations("workflows");
  const router = useRouter();
  const createRule = useCreateAutomationRule();
  const convertWorkflow = useConvertWorkflow();
  const validateWorkflow = useValidateWorkflow();

  const [name, setName] = useState(t("editor.defaultName"));
  const [isActive, setIsActive] = useState(false);
  const [definition, setDefinition] = useState<WorkflowDefinition>(createEmptyWorkflow());
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [connectionSource, setConnectionSource] = useState<string | null>(null);

  // Undo/redo history
  const [history, setHistory] = useState<WorkflowDefinition[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const skipHistoryRef = useRef(false);

  // Load from sessionStorage (template or blank)
  useEffect(() => {
    const stored = sessionStorage.getItem("workflow-new");
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        setName(parsed.name || t("editor.defaultName"));
        setDefinition(parsed.definition || createEmptyWorkflow());
        setHistory([parsed.definition || createEmptyWorkflow()]);
        setHistoryIndex(0);
      } catch {
        // ignore
      }
      sessionStorage.removeItem("workflow-new");
    } else {
      const empty = createEmptyWorkflow();
      setHistory([empty]);
      setHistoryIndex(0);
    }
  }, []);

  // Record changes to history
  const handleDefinitionChange = useCallback(
    (newDef: WorkflowDefinition) => {
      setDefinition(newDef);

      if (skipHistoryRef.current) {
        skipHistoryRef.current = false;
        return;
      }

      // Only record if nodes/edges changed (not just viewport)
      const currentDef = history[historyIndex];
      if (
        currentDef &&
        JSON.stringify(currentDef.nodes) === JSON.stringify(newDef.nodes) &&
        JSON.stringify(currentDef.edges) === JSON.stringify(newDef.edges)
      ) {
        return;
      }

      const newHistory = history.slice(0, historyIndex + 1);
      newHistory.push(newDef);
      if (newHistory.length > MAX_HISTORY) {
        newHistory.shift();
      }
      setHistory(newHistory);
      setHistoryIndex(newHistory.length - 1);
    },
    [history, historyIndex]
  );

  const handleUndo = useCallback(() => {
    if (historyIndex > 0) {
      skipHistoryRef.current = true;
      setHistoryIndex(historyIndex - 1);
      setDefinition(history[historyIndex - 1]);
    }
  }, [history, historyIndex]);

  const handleRedo = useCallback(() => {
    if (historyIndex < history.length - 1) {
      skipHistoryRef.current = true;
      setHistoryIndex(historyIndex + 1);
      setDefinition(history[historyIndex + 1]);
    }
  }, [history, historyIndex]);

  const handleSave = async () => {
    if (!name.trim()) {
      toast.error(t("editor.nameRequired"));
      return;
    }

    // Validate
    try {
      const validation = await validateWorkflow.mutateAsync(definition);
      if (!validation.valid) {
        toast.error(validation.errors[0] || t("editor.validationError"));
        return;
      }
    } catch {
      toast.error(t("editor.validationFailed"));
      return;
    }

    // Convert to automation rule
    try {
      const result = await convertWorkflow.mutateAsync({
        definition,
        name: name.trim(),
        enabled: isActive,
      });

      // Create the automation rule
      await createRule.mutateAsync({
        name: name.trim(),
        enabled: isActive,
        trigger_event: result.trigger_event,
        conditions: result.conditions,
        actions: result.actions,
      });

      toast.success(t("editor.createSuccess"));
      router.push("/workflows");
    } catch (err) {
      const message = err instanceof Error ? err.message : t("editor.createError");
      toast.error(message);
    }
  };

  const handleTest = () => {
    toast.info(t("editor.testHint"));
  };

  const handleNodeUpdate = useCallback(
    (nodeId: string, data: Record<string, unknown>) => {
      handleDefinitionChange({
        ...definition,
        nodes: definition.nodes.map((n) =>
          n.id === nodeId ? { ...n, data } : n
        ),
      });
    },
    [definition, handleDefinitionChange]
  );

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      handleDefinitionChange({
        ...definition,
        nodes: definition.nodes.filter((n) => n.id !== nodeId),
        edges: definition.edges.filter(
          (e) => e.source !== nodeId && e.target !== nodeId
        ),
      });
      setSelectedNodeId(null);
    },
    [definition, handleDefinitionChange]
  );

  const handleSidebarDragStart = useCallback(
    (e: React.DragEvent, item: PaletteItem) => {
      e.dataTransfer.setData("application/workflow-node", JSON.stringify(item));
      e.dataTransfer.effectAllowed = "copy";
    },
    []
  );

  const selectedNode = definition.nodes.find((n) => n.id === selectedNodeId) || null;

  return (
    <AdminGuard>
      <div className="flex flex-col h-[calc(100vh-64px)] -m-6">
        <WorkflowToolbar
          name={name}
          onNameChange={setName}
          isActive={isActive}
          onSave={handleSave}
          onActivate={() => setIsActive(true)}
          onDeactivate={() => setIsActive(false)}
          onUndo={handleUndo}
          onRedo={handleRedo}
          canUndo={historyIndex > 0}
          canRedo={historyIndex < history.length - 1}
          onZoomIn={() => {
            setDefinition((prev) => ({
              ...prev,
              viewport: { ...prev.viewport, zoom: zoomIn(prev.viewport.zoom) },
            }));
          }}
          onZoomOut={() => {
            setDefinition((prev) => ({
              ...prev,
              viewport: { ...prev.viewport, zoom: zoomOut(prev.viewport.zoom) },
            }));
          }}
          onFitToScreen={() => {
            setDefinition((prev) => ({
              ...prev,
              viewport: { x: 0, y: 0, zoom: 1 },
            }));
          }}
          onTest={handleTest}
          onBack={() => router.push("/workflows")}
          isSaving={createRule.isPending}
          zoom={definition.viewport.zoom}
        />

        <div className="flex flex-1 overflow-hidden">
          <WorkflowSidebar onDragStart={handleSidebarDragStart} />

          <WorkflowCanvas
            definition={definition}
            onChange={handleDefinitionChange}
            selectedNodeId={selectedNodeId}
            onSelectNode={setSelectedNodeId}
            onStartConnection={setConnectionSource}
            connectionSource={connectionSource}
          />

          {selectedNode && (
            <NodeConfigPanel
              node={selectedNode}
              onUpdate={handleNodeUpdate}
              onDelete={handleNodeDelete}
              onClose={() => setSelectedNodeId(null)}
            />
          )}
        </div>
      </div>
    </AdminGuard>
  );
}
