"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Shield, Trash2, Plus, Lock } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useRoles, useDeleteRole, useCreateRole } from "@/hooks/use-roles";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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

export default function RolesPage() {
  const router = useRouter();
  const { data, isLoading, isError, refetch } = useRoles({ limit: 100 });
  const deleteRole = useDeleteRole();
  const createRole = useCreateRole();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const roles = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteRole.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Rola została usunięta");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    if (!newName.trim()) return;
    createRole.mutate(
      {
        name: newName,
        description: newDesc || undefined,
        permissions: [],
      },
      {
        onSuccess: (role) => {
          toast.success("Rola została utworzona");
          setShowCreate(false);
          setNewName("");
          setNewDesc("");
          router.push(`/settings/roles/${role.id}`);
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
          <h1 className="text-2xl font-bold tracking-tight">Role</h1>
          <p className="text-muted-foreground">
            Zarządzaj rolami i uprawnieniami użytkowników
          </p>
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Nowa rola
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Nowa rola</DialogTitle>
              <DialogDescription>
                Utwórz nową rolę i przypisz uprawnienia
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-name">Nazwa</Label>
                <Input
                  id="new-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="np. Magazynier"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-desc">Opis</Label>
                <Input
                  id="new-desc"
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  placeholder="Opcjonalny opis roli"
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
                disabled={!newName.trim() || createRole.isPending}
              >
                {createRole.isPending ? "Tworzenie..." : "Utwórz"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4 mb-6">
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

      {roles.length === 0 ? (
        <EmptyState
          icon={Shield}
          title="Brak ról"
          description="Utwórz pierwszą rolę, aby zarządzać uprawnieniami użytkowników."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Opis</TableHead>
                <TableHead>Uprawnienia</TableHead>
                <TableHead>Typ</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {roles.map((role) => (
                <TableRow
                  key={role.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/settings/roles/${role.id}`)}
                >
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      {role.is_system && (
                        <Lock className="h-3.5 w-3.5 text-muted-foreground" />
                      )}
                      {role.name}
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {role.description || "---"}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {role.permissions.length} uprawnień
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {role.is_system ? (
                      <Badge variant="outline">Systemowa</Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                      >
                        Niestandardowa
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{formatDate(role.created_at)}</TableCell>
                  <TableCell>
                    {!role.is_system && (
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(role.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    )}
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
        title="Usuń rolę"
        description="Czy na pewno chcesz usunąć tę rolę? Użytkownicy z tą rolą stracą przypisane uprawnienia."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteRole.isPending}
      />
    </AdminGuard>
  );
}
