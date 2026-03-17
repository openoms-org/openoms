"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAutomationRule,
  useUpdateAutomationRule,
} from "@/hooks/use-automation";
import {
  useWorkflowForRule,
  useConvertWorkflow,
  useValidateWorkflow,
} from "@/hooks/use-workflows";
import { WorkflowCanvas, zoomIn, zoomOut } from "@/components/workflow/workflow-canvas";
import { WorkflowSidebar } from "@/components/workflow/workflow-sidebar";
import { WorkflowToolbar } from "@/components/workflow/workflow-toolbar";
import { NodeConfigPanel } from "@/components/workflow/node-config-panel";
import { createEmptyWorkflow, type PaletteItem } from "@/lib/workflow-types";
import type { WorkflowDefinition } from "@/types/api";
import { Loader2 } from "lucide-react";

const MAX_HISTORY = 50;

export default function WorkflowEditorPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();

  const { data: rule, isLoading: ruleLoading } = useAutomationRule(id);
  const { data: workflowDef, isLoading: workflowLoading } = useWorkflowForRule(id);
  const updateRule = useUpdateAutomationRule(id);
  const convertWorkflow = useConvertWorkflow();
  const validateWorkflow = useValidateWorkflow();

  const [name, setName] = useState("");
  const [isActive, setIsActive] = useState(false);
  const [definition, setDefinition] = useState<WorkflowDefinition>(createEmptyWorkflow());
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [connectionSource, setConnectionSource] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  // Undo/redo history
  const [history, setHistory] = useState<WorkflowDefinition[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const skipHistoryRef = useRef(false);

  // Load rule and workflow
  useEffect(() => {
    if (rule && workflowDef && !initialized) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setName(rule.name);
      setIsActive(rule.enabled);
      setDefinition(workflowDef);
      setHistory([workflowDef]);
      setHistoryIndex(0);
      setInitialized(true);
    }
  }, [rule, workflowDef, initialized]);

  const handleDefinitionChange = useCallback(
    (newDef: WorkflowDefinition) => {
      setDefinition(newDef);

      if (skipHistoryRef.current) {
        skipHistoryRef.current = false;
        return;
      }

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
      toast.error("Nazwa workflow jest wymagana");
      return;
    }

    // Validate
    try {
      const validation = await validateWorkflow.mutateAsync(definition);
      if (!validation.valid) {
        toast.error(validation.errors[0] || "Workflow zawiera bledy");
        return;
      }
    } catch {
      toast.error("Blad walidacji workflow");
      return;
    }

    // Convert to automation rule
    try {
      const result = await convertWorkflow.mutateAsync({
        definition,
        name: name.trim(),
        enabled: isActive,
      });

      // Update the automation rule
      await updateRule.mutateAsync({
        name: name.trim(),
        enabled: isActive,
        trigger_event: result.trigger_event,
        conditions: result.conditions,
        actions: result.actions,
      });

      toast.success("Workflow zostal zaktualizowany");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Nie udalo sie zapisac workflow";
      toast.error(message);
    }
  };

  const handleTest = () => {
    router.push(`/settings/automation/${id}`);
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

  if (ruleLoading || workflowLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-64px)]">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!rule) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-64px)]">
        <p className="text-muted-foreground">Workflow nie zostal znaleziony.</p>
      </div>
    );
  }

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
          isSaving={updateRule.isPending}
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
