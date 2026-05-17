"use client";

import { use, useState } from "react";
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
import { useTranslations } from "next-intl";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";

export default function RoleDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const t = useTranslations("roles");
  const td = useTranslations("roles.detail");
  const { id } = use(params);
  const router = useRouter();
  const { data: role, isLoading } = useRole(id);
  const { data: permGroups } = usePermissionGroups();
  const updateRole = useUpdateRole(id);

  const PERMISSION_LABELS: Record<string, string> = {
    "orders.view": td("permView"),
    "orders.create": td("permCreate"),
    "orders.edit": td("permEdit"),
    "orders.delete": td("permDelete"),
    "orders.export": td("permExport"),
    "products.view": td("permView"),
    "products.create": td("permCreate"),
    "products.edit": td("permEdit"),
    "products.delete": td("permDelete"),
    "shipments.view": td("permView"),
    "shipments.create": td("permCreate"),
    "shipments.edit": td("permEdit"),
    "shipments.delete": td("permDelete"),
    "returns.view": td("permView"),
    "returns.create": td("permCreate"),
    "returns.edit": td("permEdit"),
    "returns.delete": td("permDelete"),
    "customers.view": td("permView"),
    "customers.create": td("permCreate"),
    "customers.edit": td("permEdit"),
    "customers.delete": td("permDelete"),
    "invoices.view": td("permView"),
    "invoices.create": td("permCreate"),
    "invoices.delete": td("permDelete"),
    "integrations.manage": td("permManageIntegrations"),
    "settings.manage": td("permManageSettings"),
    "users.manage": td("permManageUsers"),
    "reports.view": td("permViewReports"),
    "audit.view": td("permViewAudit"),
    "automation.manage": td("permManageAutomation"),
    "warehouses.manage": td("permManageWarehouses"),
  };

  const roleKey = role?.id ?? null;
  const [name, setName] = useEffectSyncedState(role?.name ?? "", roleKey);
  const [description, setDescription] = useEffectSyncedState(
    role?.description || "",
    roleKey,
  );
  const [permissions, setPermissions] = useEffectSyncedState(
    role?.permissions || [],
    roleKey,
  );
  const [dirty, setDirty] = useState(false);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!role) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">{td("notFound")}</p>
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
          toast.success(t("roleSaved"));
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
            {role.is_system ? td("systemRole") : td("customRole")} &middot;{" "}
            {t("permissionsCount", { count: role.permissions.length })}
          </p>
        </div>
        <Button onClick={handleSave} disabled={!dirty || updateRole.isPending}>
          <Save className="h-4 w-4 mr-2" />
          {updateRole.isPending ? td("saving") : td("saveButton")}
        </Button>
      </div>

      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="role-name">{td("nameLabel")}</Label>
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
            <Label htmlFor="role-desc">{td("descriptionLabel")}</Label>
            <Input
              id="role-desc"
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                setDirty(true);
              }}
              placeholder={td("descriptionPlaceholder")}
            />
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-4">{td("permissions")}</h2>
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
