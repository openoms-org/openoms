"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { BadgePercent, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import {
  usePriceLists,
  useDeletePriceList,
  useCreatePriceList,
} from "@/hooks/use-price-lists";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const discountTypeLabels: Record<string, string> = {
  percentage: "Procentowy",
  fixed: "Kwotowy",
  override: "Cena nadpisana",
};

export default function PriceListsPage() {
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data, isLoading, isError, refetch } = usePriceLists();
  const deletePriceList = useDeletePriceList();
  const createPriceList = useCreatePriceList();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [newDiscountType, setNewDiscountType] = useState("percentage");
  const [newCurrency, setNewCurrency] = useState("PLN");

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const priceLists = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deletePriceList.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Cennik zostal usuniety");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    if (!newName.trim()) return;
    createPriceList.mutate(
      {
        name: newName,
        description: newDescription || undefined,
        discount_type: newDiscountType as "percentage" | "fixed" | "override",
        currency: newCurrency,
      },
      {
        onSuccess: () => {
          toast.success("Cennik zostal utworzony");
          setShowCreate(false);
          setNewName("");
          setNewDescription("");
          setNewDiscountType("percentage");
          setNewCurrency("PLN");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Cenniki B2B</h1>
          <p className="text-muted-foreground">
            Zarzadzaj cennikami i rabatami dla klientow biznesowych
          </p>
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Nowy cennik
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Nowy cennik</DialogTitle>
              <DialogDescription>
                Utworz nowy cennik z rabatami dla klientow
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-name">Nazwa</Label>
                <Input
                  id="new-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="np. Cennik hurtowy"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-desc">Opis</Label>
                <Input
                  id="new-desc"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="Opcjonalny opis cennika"
                />
              </div>
              <div className="space-y-2">
                <Label>Typ rabatu</Label>
                <Select
                  value={newDiscountType}
                  onValueChange={setNewDiscountType}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="percentage">Procentowy</SelectItem>
                    <SelectItem value="fixed">Kwotowy</SelectItem>
                    <SelectItem value="override">Cena nadpisana</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-currency">Waluta</Label>
                <Input
                  id="new-currency"
                  value={newCurrency}
                  onChange={(e) => setNewCurrency(e.target.value)}
                  placeholder="PLN"
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowCreate(false)}
              >
                Anuluj
              </Button>
              <Button
                onClick={handleCreate}
                disabled={!newName.trim() || createPriceList.isPending}
              >
                {createPriceList.isPending ? "Tworzenie..." : "Utworz"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

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

      {priceLists.length === 0 ? (
        <EmptyState
          icon={BadgePercent}
          title="Brak cennikow"
          description="Utworz pierwszy cennik, aby oferowac indywidualne ceny dla klientow biznesowych."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Typ rabatu</TableHead>
                <TableHead>Waluta</TableHead>
                <TableHead>Domyslny</TableHead>
                <TableHead>Aktywny</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {priceLists.map((pl) => (
                <TableRow
                  key={pl.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() =>
                    router.push(`/settings/price-lists/${pl.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    <div>
                      <p>{pl.name}</p>
                      {pl.description && (
                        <p className="text-xs text-muted-foreground">
                          {pl.description}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {discountTypeLabels[pl.discount_type] || pl.discount_type}
                  </TableCell>
                  <TableCell>{pl.currency}</TableCell>
                  <TableCell>
                    {pl.is_default ? (
                      <Badge variant="default">Tak</Badge>
                    ) : (
                      <span className="text-muted-foreground">Nie</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {pl.active ? (
                      <Badge
                        variant="outline"
                        className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      >
                        Aktywny
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                      >
                        Nieaktywny
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{formatDate(pl.created_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(pl.id);
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
        title="Usun cennik"
        description="Czy na pewno chcesz usunac ten cennik? Wszystkie przypisane pozycje cennikowe zostana usuniete."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deletePriceList.isPending}
      />
    </>
  );
}
