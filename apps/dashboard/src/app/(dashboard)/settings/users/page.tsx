"use client";

import { useState } from "react";
import { Users, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from "@/hooks/use-users";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { UserRoleBadge } from "@/components/users/user-role-badge";
import { UserForm, type UserFormValues } from "@/components/users/user-form";
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
import { useTranslations } from "next-intl";

export default function UsersPage() {
  const t = useTranslations("users");
  const tc = useTranslations("common");
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

  const handleCreate = (formData: UserFormValues) => {
    const createData: CreateUserRequest = {
      email: formData.email,
      name: formData.name,
      role: formData.role,
      password: formData.password || "",
    };
    createUser.mutate(createData, {
      onSuccess: () => {
        toast.success(t("created"));
        setCreateOpen(false);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleEdit = (formData: UserFormValues) => {
    if (!editUser) return;
    const updateData: UpdateUserRequest = {
      name: formData.name,
      role: formData.role,
    };
    updateUser.mutate(updateData, {
      onSuccess: () => {
        toast.success(t("updated"));
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
        toast.success(t("deleted"));
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
          <h1 className="text-2xl font-bold tracking-tight">{t("title")}</h1>
          <p className="text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>{t("newUser")}</Button>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4 mb-6">
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

      {users.length === 0 ? (
        <EmptyState
          icon={Users}
          title={t("empty.title")}
          description={t("empty.description")}
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
            <DialogTitle>{t("newUser")}</DialogTitle>
          </DialogHeader>
          <UserForm
            mode="create"
            onSubmit={handleCreate}
            isPending={createUser.isPending}
            onCancel={() => setCreateOpen(false)}
          />
        </DialogContent>
      </Dialog>

      {/* Edit user dialog */}
      <Dialog open={!!editUser} onOpenChange={(open) => !open && setEditUser(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("editUser")}</DialogTitle>
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
              isPending={updateUser.isPending}
              onCancel={() => setEditUser(null)}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("deleteConfirm.title")}
        description={t("deleteConfirm.description")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteUser.isPending}
      />
    </AdminGuard>
  );
}
