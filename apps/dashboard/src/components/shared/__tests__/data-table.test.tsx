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
    expect(screen.getByText("Brak danych")).toBeInTheDocument();
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
});
