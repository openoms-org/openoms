"use client";

import {
  useState,
  useCallback,
  useMemo,
  useRef,
  type SetStateAction,
} from "react";
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
import {
  createEmptyWorkflow,
  isWorkflowDefinition,
  type PaletteItem,
} from "@/lib/workflow-types";
import type { WorkflowDefinition, WorkflowNode } from "@/types/api";
import { useHydratedState } from "@/hooks/use-effect-synced-state";

const MAX_HISTORY = 50;

interface NewWorkflowEditorState {
  name: string;
  definition: WorkflowDefinition;
  history: WorkflowDefinition[];
  historyIndex: number;
}

function resolveStateAction<T>(next: SetStateAction<T>, current: T): T {
  return typeof next === "function" ? (next as (value: T) => T)(current) : next;
}

export default function NewWorkflowEditorPage() {
  const t = useTranslations("workflows");
  const router = useRouter();
  const createRule = useCreateAutomationRule();
  const convertWorkflow = useConvertWorkflow();
  const validateWorkflow = useValidateWorkflow();

  const defaultEditorState = useMemo<NewWorkflowEditorState>(() => {
    const definition = createEmptyWorkflow();

    return {
      name: t("editor.defaultName"),
      definition,
      history: [definition],
      historyIndex: 0,
    };
  }, [t]);
  const readStoredEditorState = useCallback(() => {
    const stored = sessionStorage.getItem("workflow-new");
    if (!stored) return defaultEditorState;

    try {
      const parsed = JSON.parse(stored) as {
        name?: string;
        definition?: unknown;
      };
      const definition = isWorkflowDefinition(parsed.definition)
        ? parsed.definition
        : createEmptyWorkflow();

      return {
        name: parsed.name || t("editor.defaultName"),
        definition,
        history: [definition],
        historyIndex: 0,
      };
    } catch {
      return defaultEditorState;
    } finally {
      sessionStorage.removeItem("workflow-new");
    }
  }, [defaultEditorState, t]);
  const [editorState, setEditorState] = useHydratedState(
    defaultEditorState,
    readStoredEditorState,
  );
  const { name, definition, history, historyIndex } = editorState;
  const setName = useCallback(
    (next: SetStateAction<string>) => {
      setEditorState((current) => ({
        ...current,
        name: resolveStateAction(next, current.name),
      }));
    },
    [setEditorState],
  );
  const setDefinition = useCallback(
    (next: SetStateAction<WorkflowDefinition>) => {
      setEditorState((current) => ({
        ...current,
        definition: resolveStateAction(next, current.definition),
      }));
    },
    [setEditorState],
  );
  const setHistory = useCallback(
    (next: SetStateAction<WorkflowDefinition[]>) => {
      setEditorState((current) => ({
        ...current,
        history: resolveStateAction(next, current.history),
      }));
    },
    [setEditorState],
  );
  const setHistoryIndex = useCallback(
    (next: SetStateAction<number>) => {
      setEditorState((current) => ({
        ...current,
        historyIndex: resolveStateAction(next, current.historyIndex),
      }));
    },
    [setEditorState],
  );
  const [isActive, setIsActive] = useState(false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [connectionSource, setConnectionSource] = useState<string | null>(null);
  const skipHistoryRef = useRef(false);

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
