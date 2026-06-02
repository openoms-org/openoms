"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Contact, Trash2, Search, Upload } from "lucide-react";
import { toast } from "sonner";
import { useCustomers, useDeleteCustomer } from "@/hooks/use-customers";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DensityToggle } from "@/components/shared/density-toggle";
import { getErrorMessage } from "@/lib/api-client";
import { formatCurrency } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTranslations } from "next-intl";

const DEFAULT_LIMIT = 20;

export default function CustomersPage() {
  const t = useTranslations("customers");
  const tc = useTranslations("common");
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const { data, isLoading, isError, refetch } = useCustomers({ search: searchQuery, ...pagination });
  const deleteCustomer = useDeleteCustomer();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const customers = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteCustomer.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("customerDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchQuery(search);
    setPagination((prev) => ({ ...prev, offset: 0 }));
  };

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("title")}</h1>
          <p className="text-muted-foreground">{t("bazaKlientowIHistoriaZamowien")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" asChild>
            <Link href="/customers/import">
              <Upload className="h-4 w-4" />
              {t("importCsv")}
            </Link>
          </Button>
          <Button asChild>
            <Link href="/customers/new">{t("newCustomer")}</Link>
          </Button>
        </div>
      </div>

      <form onSubmit={handleSearch} className="flex items-center gap-2 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t("searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button type="submit" variant="outline" size="sm">
          {tc("search")}
        </Button>
        <div className="ml-auto">
          <DensityToggle />
        </div>
      </form>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("retry")}
          </Button>
        </div>
      )}

      {customers.length === 0 ? (
        <EmptyState
          icon={Contact}
          title={t("brakKlientow")}
          description={t("dodajPierwszegoKlientaAbySledzicZamowieniaIHistori")}
          action={{ label: t("newCustomer"), href: "/customers/new" }}
        />
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("form.fullName")}</TableHead>
                  <TableHead>{t("form.email")}</TableHead>
                  <TableHead>{t("form.phone")}</TableHead>
                  <TableHead>{t("form.company")}</TableHead>
                  <TableHead className="text-right">{t("zamowien")}</TableHead>
                  <TableHead className="text-right">{t("totalSpent")}</TableHead>
                  <TableHead>{tc("tags")}</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {customers.map((customer) => (
                  <TableRow
                    key={customer.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => router.push(`/customers/${customer.id}`)}
                  >
                    <TableCell className="font-medium">{customer.name}</TableCell>
                    <TableCell>{customer.email || "---"}</TableCell>
                    <TableCell>{customer.phone || "---"}</TableCell>
                    <TableCell>{customer.company_name || "---"}</TableCell>
                    <TableCell className="text-right">{customer.total_orders}</TableCell>
                    <TableCell className="text-right">
                      {formatCurrency(customer.total_spent)}
                    </TableCell>
                    <TableCell>
                      {customer.tags && customer.tags.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {customer.tags.slice(0, 3).map((tag) => (
                            <span
                              key={tag}
                              className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
                            >
                              {tag}
                            </span>
                          ))}
                          {customer.tags.length > 3 && (
                            <span className="text-xs text-muted-foreground">
                              +{customer.tags.length - 3}
                            </span>
                          )}
                        </div>
                      ) : (
                        "---"
                      )}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(customer.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {data && (
            <DataTablePagination
              total={data.total}
              limit={data.limit}
              offset={data.offset}
              onPageChange={(offset) =>
                setPagination((prev) => ({ ...prev, offset }))
              }
              onPageSizeChange={(limit) =>
                setPagination({ limit, offset: 0 })
              }
            />
          )}
        </>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("usunKlienta")}
        description={t("czyNaPewnoChceszUsunacTegoKlientaTaOperacjaJestNie")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteCustomer.isPending}
      />
    </>
  );
}
