"use client";

import { use, useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Save } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useRole, useUpdateRole, usePermissionGroups } from "@/hooks/use-roles";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";

const PERMISSION_LABELS: Record<string, string> = {
  "orders.view": "Podgląd",
  "orders.create": "Tworzenie",
  "orders.edit": "Edycja",
  "orders.delete": "Usuwanie",
  "orders.export": "Eksport",
  "products.view": "Podgląd",
  "products.create": "Tworzenie",
  "products.edit": "Edycja",
  "products.delete": "Usuwanie",
  "shipments.view": "Podgląd",
  "shipments.create": "Tworzenie",
  "shipments.edit": "Edycja",
  "shipments.delete": "Usuwanie",
  "returns.view": "Podgląd",
  "returns.create": "Tworzenie",
  "returns.edit": "Edycja",
  "returns.delete": "Usuwanie",
  "customers.view": "Podgląd",
  "customers.create": "Tworzenie",
  "customers.edit": "Edycja",
  "customers.delete": "Usuwanie",
  "invoices.view": "Podgląd",
  "invoices.create": "Tworzenie",
  "invoices.delete": "Usuwanie",
  "integrations.manage": "Zarządzanie integracjami",
  "settings.manage": "Zarządzanie ustawieniami",
  "users.manage": "Zarządzanie użytkownikami",
  "reports.view": "Podgląd raportów",
  "audit.view": "Podgląd dziennika",
  "automation.manage": "Zarządzanie automatyzacją",
  "warehouses.manage": "Zarządzanie magazynami",
};

export default function RoleDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { data: role, isLoading } = useRole(id);
  const { data: permGroups } = usePermissionGroups();
  const updateRole = useUpdateRole(id);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [permissions, setPermissions] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (role) {
      setName(role.name);
      setDescription(role.description || "");
      setPermissions(role.permissions || []);
    }
  }, [role]);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!role) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">Nie znaleziono roli</p>
      </div>
    );
  }

  const togglePermission = (perm: string) => {
    setDirty(true);
    setPermissions((prev) =>
      prev.includes(perm) ? prev.filter((p) => p !== perm) : [...prev, perm]
    );
  };

  const toggleGroupAll = (groupPerms: string[]) => {
    setDirty(true);
    const allSelected = groupPerms.every((p) => permissions.includes(p));
    if (allSelected) {
      setPermissions((prev) => prev.filter((p) => !groupPerms.includes(p)));
    } else {
      setPermissions((prev) => [
        ...prev,
        ...groupPerms.filter((p) => !prev.includes(p)),
      ]);
    }
  };

  const handleSave = () => {
    updateRole.mutate(
      {
        name: name !== role.name ? name : undefined,
        description: description !== (role.description || "") ? description : undefined,
        permissions,
      },
      {
        onSuccess: () => {
          toast.success("Rola została zapisana");
          setDirty(false);
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  const groups = permGroups || [];

  return (
    <AdminGuard>
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/settings/roles")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{role.name}</h1>
          <p className="text-muted-foreground">
            {role.is_system ? "Rola systemowa" : "Rola niestandardowa"} &middot;{" "}
            {role.permissions.length} uprawnień
          </p>
        </div>
        <Button onClick={handleSave} disabled={!dirty || updateRole.isPending}>
          <Save className="h-4 w-4 mr-2" />
          {updateRole.isPending ? "Zapisywanie..." : "Zapisz"}
        </Button>
      </div>

      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="role-name">Nazwa</Label>
            <Input
              id="role-name"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setDirty(true);
              }}
              disabled={role.is_system}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="role-desc">Opis</Label>
            <Input
              id="role-desc"
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                setDirty(true);
              }}
              placeholder="Opcjonalny opis roli"
            />
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-4">Uprawnienia</h2>
          <div className="space-y-6">
            {groups.map((group) => {
              const groupPerms = group.permissions;
              const allSelected = groupPerms.every((p) =>
                permissions.includes(p)
              );
              const someSelected =
                !allSelected && groupPerms.some((p) => permissions.includes(p));

              return (
                <div
                  key={group.group}
                  className="rounded-lg border p-4"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <Checkbox
                      checked={allSelected}
                      ref={someSelected ? (el) => {
                        if (el) {
                          (el as unknown as HTMLInputElement).indeterminate = true;
                        }
                      } : undefined}
                      onCheckedChange={() => toggleGroupAll(groupPerms)}
                    />
                    <h3 className="font-medium">{group.group}</h3>
                    <Badge variant="secondary" className="text-xs">
                      {groupPerms.filter((p) => permissions.includes(p)).length}/
                      {groupPerms.length}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3 ml-7">
                    {groupPerms.map((perm) => (
                      <label
                        key={perm}
                        className="flex items-center gap-2 text-sm cursor-pointer"
                      >
                        <Checkbox
                          checked={permissions.includes(perm)}
                          onCheckedChange={() => togglePermission(perm)}
                        />
                        <span>{PERMISSION_LABELS[perm] || perm}</span>
                      </label>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </AdminGuard>
  );
}
