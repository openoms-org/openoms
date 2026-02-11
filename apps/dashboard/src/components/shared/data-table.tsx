"use client";

import { useCallback, useMemo } from "react";
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

export type { EditableColumnConfig } from "@/components/shared/editable-cell";

export interface ColumnDef<T> {
  header: string;
  accessorKey: keyof T | string;
  cell?: (row: T) => React.ReactNode;
  sortable?: boolean;
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
}: DataTableProps<T>) {
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
              <TableHead className="w-10" />
            )}
            {columns.map((column) => (
              <TableHead key={String(column.accessorKey)}>
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
                <TableCell className="w-10">
                  <Skeleton className="h-4 w-4" />
                </TableCell>
              )}
              {columns.map((column) => (
                <TableCell key={String(column.accessorKey)}>
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

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {selectable && (
            <TableHead className="w-10">
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
          {columns.map((column) => (
            <TableHead key={String(column.accessorKey)}>
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
        {data.map((row, rowIndex) => {
          const id = getRowId(row);
          return (
            <TableRow
              key={rowIndex}
              className={`hover:bg-muted/50 transition-colors ${onRowClick ? "cursor-pointer" : ""}`}
              onClick={() => onRowClick?.(row)}
            >
              {selectable && (
                <TableCell className="w-10">
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
                    <TableCell key={key}>
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
                  <TableCell key={key}>
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
