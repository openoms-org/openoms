"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  PALETTE_TRIGGERS,
  PALETTE_CONDITIONS,
  PALETTE_ACTIONS,
  NODE_COLORS,
  type PaletteItem,
} from "@/lib/workflow-types";
import { Search, Zap, GitBranch, Play } from "lucide-react";

interface WorkflowSidebarProps {
  onDragStart: (e: React.DragEvent, item: PaletteItem) => void;
}

function PaletteGroup({
  title,
  icon: Icon,
  items,
  onDragStart,
}: {
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  items: PaletteItem[];
  onDragStart: (e: React.DragEvent, item: PaletteItem) => void;
}) {
  if (items.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 px-1">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </h3>
      </div>
      <div className="space-y-1">
        {items.map((item) => {
          const colors = NODE_COLORS[item.type];
          return (
            <div
              key={`${item.type}-${item.subType}`}
              draggable
              onDragStart={(e) => onDragStart(e, item)}
              className={`
                px-3 py-2 rounded-md border cursor-grab active:cursor-grabbing
                transition-all duration-150 hover:shadow-md
                ${colors.bg} ${colors.border}
              `}
            >
              <div className={`text-sm font-medium ${colors.text}`}>{item.label}</div>
              <div className="text-xs text-muted-foreground">{item.description}</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function WorkflowSidebar({ onDragStart }: WorkflowSidebarProps) {
  const [search, setSearch] = useState("");

  const filterItems = (items: PaletteItem[]) => {
    if (!search.trim()) return items;
    const q = search.toLowerCase();
    return items.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        item.description.toLowerCase().includes(q)
    );
  };

  const filteredTriggers = filterItems(PALETTE_TRIGGERS);
  const filteredConditions = filterItems(PALETTE_CONDITIONS);
  const filteredActions = filterItems(PALETTE_ACTIONS);

  return (
    <div className="w-[260px] border-r bg-background flex flex-col h-full">
      <div className="p-3 border-b">
        <h2 className="text-sm font-semibold mb-2">Elementy</h2>
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Szukaj..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8 h-9"
          />
        </div>
      </div>
      <ScrollArea className="flex-1 p-3">
        <div className="space-y-5">
          <PaletteGroup
            title="Wyzwalacze"
            icon={Zap}
            items={filteredTriggers}
            onDragStart={onDragStart}
          />
          <PaletteGroup
            title="Warunki"
            icon={GitBranch}
            items={filteredConditions}
            onDragStart={onDragStart}
          />
          <PaletteGroup
            title="Akcje"
            icon={Play}
            items={filteredActions}
            onDragStart={onDragStart}
          />
        </div>
      </ScrollArea>
    </div>
  );
}
