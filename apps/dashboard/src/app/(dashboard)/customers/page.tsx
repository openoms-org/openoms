"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Contact, Trash2, Search } from "lucide-react";
import { toast } from "sonner";
import { useCustomers, useDeleteCustomer } from "@/hooks/use-customers";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
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

export default function CustomersPage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const { data, isLoading, isError, refetch } = useCustomers({ search: searchQuery, limit: 50 });
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
        toast.success("Klient zostal usuniety");
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
  };

  return (
    <>
      <PageHeader
        title="Klienci"
        description="Baza klientow i historia zamowien"
        action={{ label: "Nowy klient", href: "/customers/new" }}
      />

      <form onSubmit={handleSearch} className="flex items-center gap-2 mb-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Szukaj po imieniu, e-mail, telefonie..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button type="submit" variant="outline" size="sm">
          Szukaj
        </Button>
      </form>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {customers.length === 0 ? (
        <EmptyState
          icon={Contact}
          title="Brak klientow"
          description="Dodaj pierwszego klienta, aby sledzic zamowienia i historie zakupow."
          action={{ label: "Nowy klient", href: "/customers/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Imie i nazwisko</TableHead>
                <TableHead>E-mail</TableHead>
                <TableHead>Telefon</TableHead>
                <TableHead>Firma</TableHead>
                <TableHead className="text-right">Zamowien</TableHead>
                <TableHead className="text-right">Wydano lacznie</TableHead>
                <TableHead>Tagi</TableHead>
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
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usun klienta"
        description="Czy na pewno chcesz usunac tego klienta? Ta operacja jest nieodwracalna."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteCustomer.isPending}
      />
    </>
  );
}
