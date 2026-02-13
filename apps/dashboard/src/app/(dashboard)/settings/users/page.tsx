"use client";

import { useState } from "react";
import { Users, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from "@/hooks/use-users";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { UserRoleBadge } from "@/components/users/user-role-badge";
import { UserForm } from "@/components/users/user-form";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { User, CreateUserRequest, UpdateUserRequest } from "@/types/api";

export default function UsersPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useUsers();
  const createUser = useCreateUser();
  const updateUser = useUpdateUser(editUser?.id || "");
  const deleteUser = useDeleteUser();

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const users = data || [];

  const handleCreate = (formData: CreateUserRequest | UpdateUserRequest) => {
    createUser.mutate(formData as CreateUserRequest, {
      onSuccess: () => {
        toast.success("Użytkownik został utworzony");
        setCreateOpen(false);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleEdit = (formData: CreateUserRequest | UpdateUserRequest) => {
    if (!editUser) return;
    const updateData: UpdateUserRequest = {
      name: formData.name,
      role: formData.role,
    };
    updateUser.mutate(updateData, {
      onSuccess: () => {
        toast.success("Użytkownik został zaktualizowany");
        setEditUser(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteUser.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Użytkownik został usunięty");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Użytkownicy</h1>
          <p className="text-muted-foreground">
            Zarządzaj użytkownikami w organizacji
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>Nowy użytkownik</Button>
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

      {users.length === 0 ? (
        <EmptyState
          icon={Users}
          title="Brak użytkowników"
          description="Dodaj pierwszego użytkownika do organizacji."
        />
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nazwa</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Rola</TableHead>
                  <TableHead>Ostatnie logowanie</TableHead>
                  <TableHead>Utworzono</TableHead>
                  <TableHead className="w-[80px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">{user.name}</TableCell>
                    <TableCell>{user.email}</TableCell>
                    <TableCell>
                      <UserRoleBadge role={user.role} />
                    </TableCell>
                    <TableCell>
                      {user.last_login_at
                        ? formatDate(user.last_login_at)
                        : "---"}
                    </TableCell>
                    <TableCell>{formatDate(user.created_at)}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => setEditUser(user)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => setDeleteId(user.id)}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

        </>
      )}

      {/* Create user dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Nowy użytkownik</DialogTitle>
          </DialogHeader>
          <UserForm
            mode="create"
            onSubmit={handleCreate}
            isLoading={createUser.isPending}
            onCancel={() => setCreateOpen(false)}
          />
        </DialogContent>
      </Dialog>

      {/* Edit user dialog */}
      <Dialog open={!!editUser} onOpenChange={(open) => !open && setEditUser(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edytuj użytkownika</DialogTitle>
          </DialogHeader>
          {editUser && (
            <UserForm
              mode="edit"
              defaultValues={{
                email: editUser.email,
                name: editUser.name,
                role: editUser.role,
              }}
              onSubmit={handleEdit}
              isLoading={updateUser.isPending}
              onCancel={() => setEditUser(null)}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usuń użytkownika"
        description="Czy na pewno chcesz usunąć tego użytkownika? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteUser.isPending}
      />
    </AdminGuard>
  );
}
