"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Warehouse, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWarehouses, useDeleteWarehouse, useCreateWarehouse } from "@/hooks/use-warehouses";
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

export default function WarehousesPage() {
  const router = useRouter();
  const { data, isLoading, isError, refetch } = useWarehouses();
  const deleteWarehouse = useDeleteWarehouse();
  const createWarehouse = useCreateWarehouse();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newCode, setNewCode] = useState("");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const warehouses = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteWarehouse.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Magazyn został usunięty");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    if (!newName.trim()) return;
    createWarehouse.mutate(
      { name: newName, code: newCode || undefined },
      {
        onSuccess: () => {
          toast.success("Magazyn został utworzony");
          setShowCreate(false);
          setNewName("");
          setNewCode("");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Magazyny</h1>
          <p className="text-muted-foreground">
            Zarządzaj magazynami i stanami magazynowymi
          </p>
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Nowy magazyn
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Nowy magazyn</DialogTitle>
              <DialogDescription>
                Dodaj nowy magazyn do systemu
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-name">Nazwa</Label>
                <Input
                  id="new-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="np. Magazyn Centralny"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-code">Kod</Label>
                <Input
                  id="new-code"
                  value={newCode}
                  onChange={(e) => setNewCode(e.target.value)}
                  placeholder="np. WH-01"
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
                disabled={!newName.trim() || createWarehouse.isPending}
              >
                {createWarehouse.isPending ? "Tworzenie..." : "Utwórz"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {warehouses.length === 0 ? (
        <EmptyState
          icon={Warehouse}
          title="Brak magazynów"
          description="Dodaj pierwszy magazyn, aby zarządzać stanami magazynowymi produktów."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Kod</TableHead>
                <TableHead>Domyślny</TableHead>
                <TableHead>Aktywny</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {warehouses.map((warehouse) => (
                <TableRow
                  key={warehouse.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() =>
                    router.push(`/settings/warehouses/${warehouse.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    {warehouse.name}
                  </TableCell>
                  <TableCell>{warehouse.code || "---"}</TableCell>
                  <TableCell>
                    {warehouse.is_default ? (
                      <Badge variant="default">Tak</Badge>
                    ) : (
                      <span className="text-muted-foreground">Nie</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {warehouse.active ? (
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
                  <TableCell>{formatDate(warehouse.created_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(warehouse.id);
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
        title="Usuń magazyn"
        description="Czy na pewno chcesz usunąć ten magazyn? Wszystkie stany magazynowe zostaną usunięte."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteWarehouse.isPending}
      />
    </AdminGuard>
  );
}
