"use client";

import { useCallback, useMemo, useState, useRef, useEffect } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowUp, ArrowDown, ArrowUpDown } from "lucide-react";
import { EditableCell, type EditableColumnConfig } from "@/components/shared/editable-cell";
import { useTableDensity, densityConfig } from "@/lib/table-density";
import { cn } from "@/lib/utils";

export type { EditableColumnConfig } from "@/components/shared/editable-cell";

export interface ColumnDef<T> {
  header: string;
  accessorKey: keyof T | string;
  cell?: (row: T) => React.ReactNode;
  sortable?: boolean;
  className?: string;
}

interface DataTableProps<T> {
  columns: ColumnDef<T>[];
  data: T[];
  isLoading?: boolean;
  emptyMessage?: string;
  emptyState?: React.ReactNode;
  onRowClick?: (row: T) => void;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
  rowId?: (row: T) => string;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  onSort?: (column: string) => void;
  editableColumns?: EditableColumnConfig<T>[];
  resizable?: boolean;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getNestedValue(obj: any, path: string): unknown {
  return path.split(".").reduce((acc, part) => {
    if (acc && typeof acc === "object") {
      return acc[part];
    }
    return undefined;
  }, obj);
}

export function DataTable<T>({
  columns,
  data,
  isLoading = false,
  emptyMessage = "Brak danych",
  emptyState,
  onRowClick,
  selectable = false,
  selectedIds,
  onSelectionChange,
  rowId,
  sortBy,
  sortOrder,
  onSort,
  editableColumns,
  resizable = false,
}: DataTableProps<T>) {
  const { density } = useTableDensity();
  const cellPx = densityConfig[density].cellPadding;

  // Column resize state
  const [colWidths, setColWidths] = useState<Record<string, number>>({});
  const resizeRef = useRef<{
    key: string;
    startX: number;
    startWidth: number;
  } | null>(null);
  const tableRef = useRef<HTMLTableElement>(null);

  // Initialize column widths from rendered table on first load
  const headersRef = useRef<HTMLTableRowElement>(null);
  const initialized = useRef(false);
  useEffect(() => {
    if (!resizable || initialized.current || !headersRef.current || isLoading) return;
    const cells = headersRef.current.querySelectorAll("th");
    const widths: Record<string, number> = {};
    let colIdx = selectable ? 1 : 0; // skip checkbox column
    for (const col of columns) {
      const cell = cells[colIdx];
      if (cell) {
        widths[String(col.accessorKey)] = cell.getBoundingClientRect().width;
      }
      colIdx++;
    }
    if (Object.keys(widths).length > 0) {
      setColWidths(widths);
      initialized.current = true;
    }
  }, [resizable, columns, isLoading, selectable]);

  // Reset initialized flag when columns change
  useEffect(() => {
    initialized.current = false;
  }, [columns.length]);

  useEffect(() => {
    if (!resizable) return;
    const onMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return;
      const delta = e.clientX - resizeRef.current.startX;
      const newWidth = Math.max(40, resizeRef.current.startWidth + delta);
      setColWidths((prev) => ({
        ...prev,
        [resizeRef.current!.key]: newWidth,
      }));
    };
    const onMouseUp = () => {
      resizeRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [resizable]);

  const onResizeStart = (key: string, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const currentWidth = colWidths[key] || 100;
    resizeRef.current = { key, startX: e.clientX, startWidth: currentWidth };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  };

  const getRowId = useCallback(
    (row: T) => (rowId ? rowId(row) : (row as Record<string, unknown>).id as string),
    [rowId]
  );

  const editableMap = useMemo(() => {
    if (!editableColumns) return null;
    const map = new Map<string, EditableColumnConfig<T>>();
    for (const ec of editableColumns) {
      map.set(ec.accessorKey, ec);
    }
    return map;
  }, [editableColumns]);

  const allRowIds = data.map((row) => getRowId(row));
  const allSelected = allRowIds.length > 0 && allRowIds.every((id) => selectedIds?.has(id));
  const someSelected = allRowIds.some((id) => selectedIds?.has(id));

  const toggleAll = () => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange(new Set());
    } else {
      onSelectionChange(new Set(allRowIds));
    }
  };

  const toggleRow = (id: string) => {
    if (!onSelectionChange || !selectedIds) return;
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelectionChange(next);
  };

  if (isLoading) {
    return (
      <Table>
        <TableHeader>
          <TableRow>
            {selectable && (
              <TableHead className={cn("w-10", cellPx)} />
            )}
            {columns.map((column) => (
              <TableHead key={String(column.accessorKey)} className={cn(cellPx, column.className)}>
                {column.sortable && onSort ? (
                  <button
                    className="flex items-center gap-1 hover:text-foreground"
                    onClick={() => onSort(String(column.accessorKey))}
                  >
                    {column.header}
                    {sortBy === String(column.accessorKey) ? (
                      sortOrder === "asc" ? (
                        <ArrowUp className="h-4 w-4" />
                      ) : (
                        <ArrowDown className="h-4 w-4" />
                      )
                    ) : (
                      <ArrowUpDown className="h-4 w-4 opacity-50" />
                    )}
                  </button>
                ) : (
                  column.header
                )}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 5 }).map((_, rowIndex) => (
            <TableRow key={rowIndex}>
              {selectable && (
                <TableCell className={cn("w-10", cellPx)}>
                  <Skeleton className="h-4 w-4" />
                </TableCell>
              )}
              {columns.map((column) => (
                <TableCell key={String(column.accessorKey)} className={cn(cellPx, column.className)}>
                  <Skeleton className="h-4 w-full" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    );
  }

  if (!data || data.length === 0) {
    if (emptyState) {
      return <>{emptyState}</>;
    }
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground text-sm">{emptyMessage}</p>
      </div>
    );
  }

  const hasWidths = resizable && Object.keys(colWidths).length > 0;

  return (
    <Table
      ref={tableRef}
      style={hasWidths ? { tableLayout: "fixed" } : undefined}
    >
      <TableHeader>
        <TableRow ref={headersRef}>
          {selectable && (
            <TableHead className={cn("w-10", cellPx)} style={hasWidths ? { width: 40 } : undefined}>
              <input
                type="checkbox"
                className="cursor-pointer"
                checked={allSelected}
                ref={(el) => {
                  if (el) el.indeterminate = someSelected && !allSelected;
                }}
                onChange={toggleAll}
              />
            </TableHead>
          )}
          {columns.map((column) => {
            const key = String(column.accessorKey);
            const width = colWidths[key];
            return (
              <TableHead
                key={key}
                className={cn("relative", cellPx, column.className)}
                style={hasWidths && width ? { width } : undefined}
              >
                {column.sortable && onSort ? (
                  <button
                    className="flex items-center gap-1 hover:text-foreground"
                    onClick={() => onSort(key)}
                  >
                    {column.header}
                    {sortBy === key ? (
                      sortOrder === "asc" ? (
                        <ArrowUp className="h-4 w-4" />
                      ) : (
                        <ArrowDown className="h-4 w-4" />
                      )
                    ) : (
                      <ArrowUpDown className="h-4 w-4 opacity-50" />
                    )}
                  </button>
                ) : (
                  column.header
                )}
                {resizable && (
                  <div
                    className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/30 active:bg-primary/50"
                    onMouseDown={(e) => onResizeStart(key, e)}
                  />
                )}
              </TableHead>
            );
          })}
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.map((row, rowIndex) => {
          const id = getRowId(row);
          return (
            <TableRow
              key={rowIndex}
              className={`hover:bg-muted/50 transition-colors ${onRowClick ? "cursor-pointer" : ""}`}
              onClick={() => onRowClick?.(row)}
            >
              {selectable && (
                <TableCell className={cn("w-10", cellPx)}>
                  <input
                    type="checkbox"
                    className="cursor-pointer"
                    checked={selectedIds?.has(id) || false}
                    onChange={() => toggleRow(id)}
                    onClick={(e) => e.stopPropagation()}
                  />
                </TableCell>
              )}
              {columns.map((column) => {
                const key = String(column.accessorKey);
                const editConfig = editableMap?.get(key);
                const rawValue = getNestedValue(row, key);
                const displayContent = column.cell
                  ? column.cell(row)
                  : String(rawValue ?? "");

                if (editConfig) {
                  return (
                    <TableCell key={key} className={cn(cellPx, column.className)}>
                      <EditableCell<T>
                        row={row}
                        value={rawValue}
                        config={editConfig}
                        displayContent={displayContent}
                      />
                    </TableCell>
                  );
                }

                return (
                  <TableCell key={key} className={cn(cellPx, column.className)}>
                    {displayContent}
                  </TableCell>
                );
              })}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
