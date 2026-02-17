"use client";

import { useRef, useState, useCallback, useEffect } from "react";
import type { WorkflowNode, WorkflowEdge, WorkflowDefinition } from "@/types/api";
import { WorkflowNodeComponent } from "./workflow-node";
import { generateId, type PaletteItem } from "@/lib/workflow-types";

interface WorkflowCanvasProps {
  definition: WorkflowDefinition;
  onChange: (definition: WorkflowDefinition) => void;
  selectedNodeId: string | null;
  onSelectNode: (id: string | null) => void;
  onStartConnection: (sourceId: string) => void;
  connectionSource: string | null;
}

// Compute bezier path between two points (vertical flow)
function computeBezierPath(
  x1: number,
  y1: number,
  x2: number,
  y2: number
): string {
  const midY = (y1 + y2) / 2;
  return `M ${x1} ${y1} C ${x1} ${midY}, ${x2} ${midY}, ${x2} ${y2}`;
}

// Grid pattern size
const GRID_SIZE = 20;
const NODE_WIDTH = 220;

export function WorkflowCanvas({
  definition,
  onChange,
  selectedNodeId,
  onSelectNode,
  onStartConnection,
  connectionSource,
}: WorkflowCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const [pan, setPan] = useState({ x: definition.viewport.x, y: definition.viewport.y });
  const [zoom, setZoom] = useState(definition.viewport.zoom || 1);
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });
  const [draggingNodeId, setDraggingNodeId] = useState<string | null>(null);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });
  const [connectingFrom, setConnectingFrom] = useState<string | null>(null);
  const [connectMousePos, setConnectMousePos] = useState({ x: 0, y: 0 });

  // Expose zoom to parent via viewport
  useEffect(() => {
    const newDef = {
      ...definition,
      viewport: { x: pan.x, y: pan.y, zoom },
    };
    // Only update if viewport changed, avoid infinite loop
    if (
      definition.viewport.x !== pan.x ||
      definition.viewport.y !== pan.y ||
      definition.viewport.zoom !== zoom
    ) {
      onChange(newDef);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pan, zoom]);

  const getCanvasPoint = useCallback(
    (clientX: number, clientY: number) => {
      if (!canvasRef.current) return { x: 0, y: 0 };
      const rect = canvasRef.current.getBoundingClientRect();
      return {
        x: (clientX - rect.left - pan.x) / zoom,
        y: (clientY - rect.top - pan.y) / zoom,
      };
    },
    [pan, zoom]
  );

  // Pan handling
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      // Only pan on middle button or when clicking canvas background
      if (e.button === 1 || (e.button === 0 && e.target === canvasRef.current?.querySelector(".canvas-bg"))) {
        setIsPanning(true);
        setPanStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
        e.preventDefault();
      }
    },
    [pan]
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      setMousePos({ x: e.clientX, y: e.clientY });

      if (isPanning) {
        setPan({ x: e.clientX - panStart.x, y: e.clientY - panStart.y });
        return;
      }

      if (draggingNodeId) {
        const point = getCanvasPoint(e.clientX, e.clientY);
        const snappedX = Math.round((point.x - dragOffset.x) / GRID_SIZE) * GRID_SIZE;
        const snappedY = Math.round((point.y - dragOffset.y) / GRID_SIZE) * GRID_SIZE;

        onChange({
          ...definition,
          nodes: definition.nodes.map((n) =>
            n.id === draggingNodeId
              ? { ...n, position: { x: snappedX, y: snappedY } }
              : n
          ),
        });
        return;
      }

      if (connectingFrom) {
        const point = getCanvasPoint(e.clientX, e.clientY);
        setConnectMousePos(point);
      }
    },
    [isPanning, panStart, draggingNodeId, dragOffset, connectingFrom, getCanvasPoint, onChange, definition]
  );

  const handleMouseUp = useCallback(
    (e: React.MouseEvent) => {
      if (isPanning) {
        setIsPanning(false);
        return;
      }

      if (draggingNodeId) {
        setDraggingNodeId(null);
        return;
      }

      // Finish connection
      if (connectingFrom) {
        const target = (e.target as HTMLElement).closest("[data-handle='input']");
        if (target) {
          const targetNodeId = target.getAttribute("data-node-id");
          if (targetNodeId && targetNodeId !== connectingFrom) {
            // Check for duplicate edge
            const exists = definition.edges.some(
              (edge) => edge.source === connectingFrom && edge.target === targetNodeId
            );
            if (!exists) {
              const newEdge: WorkflowEdge = {
                id: generateId(),
                source: connectingFrom,
                target: targetNodeId,
              };
              onChange({
                ...definition,
                edges: [...definition.edges, newEdge],
              });
            }
          }
        }
        setConnectingFrom(null);
      }
    },
    [isPanning, draggingNodeId, connectingFrom, definition, onChange]
  );

  // Zoom with mouse wheel
  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY > 0 ? -0.1 : 0.1;
      const newZoom = Math.max(0.2, Math.min(3, zoom + delta));
      setZoom(newZoom);
    },
    [zoom]
  );

  // Node drag start
  const handleNodeDragStart = useCallback(
    (e: React.MouseEvent, nodeId: string) => {
      const node = definition.nodes.find((n) => n.id === nodeId);
      if (!node) return;
      const point = getCanvasPoint(e.clientX, e.clientY);
      setDraggingNodeId(nodeId);
      setDragOffset({
        x: point.x - node.position.x,
        y: point.y - node.position.y,
      });
    },
    [definition.nodes, getCanvasPoint]
  );

  // Handle connection start from output handle
  const handleCanvasClick = useCallback(
    (e: React.MouseEvent) => {
      // Check if clicked on output handle
      const outputHandle = (e.target as HTMLElement).closest("[data-handle='output']");
      if (outputHandle) {
        const nodeId = outputHandle.getAttribute("data-node-id");
        if (nodeId) {
          setConnectingFrom(nodeId);
          const point = getCanvasPoint(e.clientX, e.clientY);
          setConnectMousePos(point);
          return;
        }
      }

      // If clicking background, deselect
      const isBackground =
        e.target === canvasRef.current?.querySelector(".canvas-bg") ||
        (e.target as HTMLElement).classList.contains("canvas-bg");
      if (isBackground) {
        onSelectNode(null);
        setConnectingFrom(null);
      }
    },
    [getCanvasPoint, onSelectNode]
  );

  // Drop handler for sidebar palette
  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const data = e.dataTransfer.getData("application/workflow-node");
      if (!data) return;

      try {
        const item: PaletteItem = JSON.parse(data);
        const point = getCanvasPoint(e.clientX, e.clientY);
        const snappedX = Math.round(point.x / GRID_SIZE) * GRID_SIZE;
        const snappedY = Math.round(point.y / GRID_SIZE) * GRID_SIZE;

        // Build node data based on type
        let nodeData: Record<string, unknown> = {};
        if (item.type === "trigger") {
          nodeData = { event: item.subType, label: item.label };
        } else if (item.type === "condition") {
          nodeData = { field: "", operator: "eq", value: "", label: "Nowy warunek" };
        } else if (item.type === "action") {
          nodeData = { actionType: item.subType, label: item.label };
        }

        const newNode: WorkflowNode = {
          id: generateId(),
          type: item.type,
          position: { x: snappedX, y: snappedY },
          data: nodeData,
        };

        onChange({
          ...definition,
          nodes: [...definition.nodes, newNode],
        });

        onSelectNode(newNode.id);
      } catch {
        // ignore invalid drag data
      }
    },
    [getCanvasPoint, onChange, definition, onSelectNode]
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
  }, []);

  // Compute edge coordinates
  const getNodeCenter = (nodeId: string, handle: "output" | "input") => {
    const node = definition.nodes.find((n) => n.id === nodeId);
    if (!node) return { x: 0, y: 0 };
    const x = node.position.x + NODE_WIDTH / 2;
    const y = handle === "output" ? node.position.y + 68 : node.position.y;
    return { x, y };
  };

  // Delete edge on click
  const handleEdgeClick = useCallback(
    (edgeId: string) => {
      onChange({
        ...definition,
        edges: definition.edges.filter((e) => e.id !== edgeId),
      });
    },
    [definition, onChange]
  );

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Delete" || e.key === "Backspace") {
        // Don't delete if focus is in an input
        if (
          document.activeElement?.tagName === "INPUT" ||
          document.activeElement?.tagName === "TEXTAREA"
        ) {
          return;
        }
        if (selectedNodeId) {
          onChange({
            ...definition,
            nodes: definition.nodes.filter((n) => n.id !== selectedNodeId),
            edges: definition.edges.filter(
              (e) => e.source !== selectedNodeId && e.target !== selectedNodeId
            ),
          });
          onSelectNode(null);
        }
      }
      if (e.key === "Escape") {
        onSelectNode(null);
        setConnectingFrom(null);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [selectedNodeId, definition, onChange, onSelectNode]);

  return (
    <div
      ref={canvasRef}
      className="flex-1 overflow-hidden relative bg-muted/30 cursor-grab"
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onClick={handleCanvasClick}
      onWheel={handleWheel}
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      style={{ touchAction: "none" }}
    >
      {/* Grid background */}
      <svg className="absolute inset-0 w-full h-full pointer-events-none">
        <defs>
          <pattern
            id="grid"
            width={GRID_SIZE * zoom}
            height={GRID_SIZE * zoom}
            patternUnits="userSpaceOnUse"
            x={pan.x % (GRID_SIZE * zoom)}
            y={pan.y % (GRID_SIZE * zoom)}
          >
            <circle
              cx={1}
              cy={1}
              r={1}
              className="fill-muted-foreground/20"
            />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)" className="canvas-bg" />
      </svg>

      {/* Transformed layer */}
      <div
        className="absolute origin-top-left"
        style={{
          transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
        }}
      >
        {/* SVG for edges */}
        <svg
          className="absolute pointer-events-none"
          style={{
            width: "10000px",
            height: "10000px",
            left: "-5000px",
            top: "-5000px",
          }}
        >
          <defs>
            <marker
              id="arrowhead"
              viewBox="0 0 10 7"
              refX="9"
              refY="3.5"
              markerWidth="8"
              markerHeight="6"
              orient="auto"
            >
              <polygon points="0 0, 10 3.5, 0 7" className="fill-muted-foreground/50" />
            </marker>
          </defs>
          <g transform="translate(5000, 5000)">
            {definition.edges.map((edge) => {
              const from = getNodeCenter(edge.source, "output");
              const to = getNodeCenter(edge.target, "input");
              const path = computeBezierPath(from.x, from.y, to.x, to.y);

              return (
                <g key={edge.id}>
                  {/* Invisible wider stroke for easier clicking */}
                  <path
                    d={path}
                    fill="none"
                    stroke="transparent"
                    strokeWidth={12}
                    className="cursor-pointer pointer-events-auto"
                    onClick={() => handleEdgeClick(edge.id)}
                  />
                  <path
                    d={path}
                    fill="none"
                    className="stroke-muted-foreground/40"
                    strokeWidth={2}
                    markerEnd="url(#arrowhead)"
                  />
                </g>
              );
            })}

            {/* Connection being drawn */}
            {connectingFrom && (
              <path
                d={computeBezierPath(
                  getNodeCenter(connectingFrom, "output").x,
                  getNodeCenter(connectingFrom, "output").y,
                  connectMousePos.x,
                  connectMousePos.y
                )}
                fill="none"
                className="stroke-primary/60"
                strokeWidth={2}
                strokeDasharray="6 3"
              />
            )}
          </g>
        </svg>

        {/* Nodes */}
        {definition.nodes.map((node) => (
          <WorkflowNodeComponent
            key={node.id}
            node={node}
            selected={node.id === selectedNodeId}
            onSelect={onSelectNode}
            onDragStart={handleNodeDragStart}
            scale={zoom}
          />
        ))}
      </div>

      {/* Mini-map */}
      <div className="absolute bottom-3 right-3 w-[160px] h-[100px] border rounded bg-background/80 backdrop-blur-sm overflow-hidden pointer-events-none">
        <svg width="160" height="100" viewBox="-200 -100 1200 800">
          {definition.nodes.map((node) => {
            const color =
              node.type === "trigger"
                ? "#22c55e"
                : node.type === "condition"
                ? "#3b82f6"
                : "#f97316";
            return (
              <rect
                key={node.id}
                x={node.position.x}
                y={node.position.y}
                width={40}
                height={12}
                rx={2}
                fill={color}
                opacity={node.id === selectedNodeId ? 1 : 0.6}
              />
            );
          })}
          {definition.edges.map((edge) => {
            const from = definition.nodes.find((n) => n.id === edge.source);
            const to = definition.nodes.find((n) => n.id === edge.target);
            if (!from || !to) return null;
            return (
              <line
                key={edge.id}
                x1={from.position.x + 20}
                y1={from.position.y + 12}
                x2={to.position.x + 20}
                y2={to.position.y}
                stroke="#9ca3af"
                strokeWidth={1}
                opacity={0.5}
              />
            );
          })}
        </svg>
      </div>

      {/* Empty state */}
      {definition.nodes.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="text-center text-muted-foreground">
            <p className="text-lg font-medium">Przeciagnij elementy z panelu po lewej</p>
            <p className="text-sm mt-1">aby rozpoczac budowanie workflow</p>
          </div>
        </div>
      )}
    </div>
  );
}

// Export zoom helpers
export function zoomIn(current: number): number {
  return Math.min(3, current + 0.2);
}

export function zoomOut(current: number): number {
  return Math.max(0.2, current - 0.2);
}
