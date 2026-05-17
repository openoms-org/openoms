import { readFileSync } from "node:fs";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";

interface TestRow {
  id: string;
  name: string;
  email: string;
}

const columns: ColumnDef<TestRow>[] = [
  { header: "Name", accessorKey: "name" },
  { header: "Email", accessorKey: "email" },
];

const testData: TestRow[] = [
  { id: "1", name: "Jan Kowalski", email: "jan@example.com" },
  { id: "2", name: "Anna Nowak", email: "anna@example.com" },
];

describe("DataTable", () => {
  it("renders column headers", () => {
    render(<DataTable columns={columns} data={testData} />);
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Email")).toBeInTheDocument();
  });

  it("renders data rows", () => {
    render(<DataTable columns={columns} data={testData} />);
    expect(screen.getByText("Jan Kowalski")).toBeInTheDocument();
    expect(screen.getByText("jan@example.com")).toBeInTheDocument();
    expect(screen.getByText("Anna Nowak")).toBeInTheDocument();
    expect(screen.getByText("anna@example.com")).toBeInTheDocument();
  });

  it("shows default empty message when data is empty", () => {
    render(<DataTable columns={columns} data={[]} />);
    // With mocked useTranslations, t("noData") returns "noData"
    expect(screen.getByText("noData")).toBeInTheDocument();
  });

  it("shows custom empty message", () => {
    render(<DataTable columns={columns} data={[]} emptyMessage="No records found" />);
    expect(screen.getByText("No records found")).toBeInTheDocument();
  });

  it("shows custom empty state element", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        emptyState={<div>Custom empty state</div>}
      />
    );
    expect(screen.getByText("Custom empty state")).toBeInTheDocument();
  });

  it("renders loading skeleton when isLoading is true", () => {
    render(<DataTable columns={columns} data={[]} isLoading={true} />);
    // Should still render column headers
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Email")).toBeInTheDocument();
    // Should not render data rows
    expect(screen.queryByText("Jan Kowalski")).not.toBeInTheDocument();
  });

  it("calls onRowClick when a row is clicked", async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();

    render(<DataTable columns={columns} data={testData} onRowClick={handleClick} />);
    await user.click(screen.getByText("Jan Kowalski"));

    expect(handleClick).toHaveBeenCalledTimes(1);
    expect(handleClick).toHaveBeenCalledWith(testData[0]);
  });

  it("renders custom cell renderer", () => {
    const columnsWithCell: ColumnDef<TestRow>[] = [
      {
        header: "Name",
        accessorKey: "name",
        cell: (row) => <strong data-testid="custom-cell">{row.name.toUpperCase()}</strong>,
      },
      { header: "Email", accessorKey: "email" },
    ];

    render(<DataTable columns={columnsWithCell} data={testData} />);
    expect(screen.getByText("JAN KOWALSKI")).toBeInTheDocument();
    expect(screen.getByText("ANNA NOWAK")).toBeInTheDocument();
    expect(screen.getAllByTestId("custom-cell")).toHaveLength(2);
  });

  it("handles nested accessor keys", () => {
    interface NestedRow {
      id: string;
      user: { name: string };
    }

    const nestedColumns: ColumnDef<NestedRow>[] = [
      { header: "User Name", accessorKey: "user.name" },
    ];

    const nestedData: NestedRow[] = [{ id: "1", user: { name: "Nested User" } }];

    render(<DataTable columns={nestedColumns} data={nestedData} />);
    expect(screen.getByText("Nested User")).toBeInTheDocument();
  });

  it("keeps nested accessor lookup typed without any escapes", () => {
    const source = readFileSync("src/components/shared/data-table.tsx", "utf8");

    expect(source).not.toContain("eslint-disable-next-line @typescript-eslint/no-explicit-any");
    expect(source).toMatch(
      /function getNestedValue\(\s*obj: Record<string, unknown>,\s*path: string\s*\): unknown/,
    );
    expect(source).not.toContain("function getNestedValue(obj: any");
  });

  it("labels the select-all checkbox and row checkboxes", () => {
    render(
      <DataTable
        columns={columns}
        data={testData}
        selectable
        selectedIds={new Set()}
        onSelectionChange={() => {}}
      />
    );

    expect(screen.getByRole("checkbox", { name: "dataTableSelectAll" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "dataTableSelectRow Jan Kowalski" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "dataTableSelectRow Anna Nowak" })).toBeInTheDocument();
  });

  it("uses table data for row checkbox labels when rows do not have names", () => {
    interface ShipmentRow {
      id: string;
      tracking_number: string;
      provider: string;
    }

    const shipmentColumns: ColumnDef<ShipmentRow>[] = [
      { header: "ID", accessorKey: "id" },
      { header: "Tracking", accessorKey: "tracking_number" },
      { header: "Provider", accessorKey: "provider" },
    ];

    render(
      <DataTable
        columns={shipmentColumns}
        data={[{ id: "shipment-1", tracking_number: "INP123", provider: "inpost" }]}
        selectable
        selectedIds={new Set()}
        onSelectionChange={() => {}}
      />
    );

    expect(screen.getByRole("checkbox", { name: "dataTableSelectRow INP123" })).toBeInTheDocument();
  });

  it("uses custom row labels when provided", () => {
    interface OrderRow {
      id: string;
      order_number: string;
    }

    const orderColumns: ColumnDef<OrderRow>[] = [
      { header: "ID", accessorKey: "id" },
      { header: "Order", accessorKey: "order_number" },
    ];

    render(
      <DataTable
        columns={orderColumns}
        data={[{ id: "order-1", order_number: "ORD-100" }]}
        getRowLabel={(row) => `Order ${row.order_number}`}
        selectable
        selectedIds={new Set()}
        onSelectionChange={() => {}}
      />
    );

    expect(screen.getByRole("checkbox", { name: "dataTableSelectRow Order ORD-100" })).toBeInTheDocument();
  });

  it("labels sortable header buttons", () => {
    const sortableColumns: ColumnDef<TestRow>[] = [
      { header: "Name", accessorKey: "name", sortable: true },
      { header: "Email", accessorKey: "email" },
    ];

    render(
      <DataTable
        columns={sortableColumns}
        data={testData}
        sortBy="name"
        sortOrder="asc"
        onSort={() => {}}
      />
    );

    expect(screen.getByRole("button", { name: "dataTableSortBy Name" })).toBeInTheDocument();
  });
});
