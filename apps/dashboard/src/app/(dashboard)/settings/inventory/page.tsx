"use client";

import { useState } from "react";
import { AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useInventorySettings, useUpdateInventorySettings } from "@/hooks/use-settings";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTranslations } from "next-intl";

export default function InventorySettingsPage() {
  const t = useTranslations("settings");
  const tc = useTranslations("common");
  const { data, isLoading } = useInventorySettings();
  const updateSettings = useUpdateInventorySettings();
  const [showWarning, setShowWarning] = useState(false);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const strictMode = data?.strict_mode ?? false;

  const handleToggle = (checked: boolean) => {
    if (checked) {
      // Show warning before enabling
      setShowWarning(true);
    } else {
      // Disable directly
      updateSettings.mutate(
        { strict_mode: false },
        {
          onSuccess: () => {
            toast.success(t("strictInventoryControlDisabled"));
          },
          onError: (error) => {
            toast.error(getErrorMessage(error));
          },
        }
      );
    }
  };

  const confirmEnable = () => {
    updateSettings.mutate(
      { strict_mode: true },
      {
        onSuccess: () => {
          toast.success(t("strictInventoryControlEnabled"));
          setShowWarning(false);
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
          setShowWarning(false);
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("inventory.title")}
          </h1>
          <p className="text-muted-foreground">
            {t("inventory.subtitle")}
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t("inventory.controlTitle")}</CardTitle>
            <CardDescription>
              {t("ustawieniaDotyczaceSposobuZarzadzaniaStanamiMagazy")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between rounded-lg border p-4">
              <div className="space-y-0.5">
                <Label htmlFor="strict-mode" className="text-base font-medium">
                  {t("strictInventoryControlMode")}
                </Label>
                <p className="text-sm text-muted-foreground">
                  {t("whenEnabledStockChangesOnlyViaDocuments")}
                </p>
              </div>
              <Switch
                id="strict-mode"
                checked={strictMode}
                onCheckedChange={handleToggle}
                disabled={updateSettings.isPending}
              />
            </div>

            {strictMode && (
              <div className="flex items-start gap-3 rounded-lg border border-warning/30 bg-warning/15 p-4">
                <AlertTriangle className="h-5 w-5 text-warning mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-warning">
                    {t("strictInventoryControlActive")}
                  </p>
                  <p className="text-sm text-warning">
                    {t("inventory.strictModeWarning")}
                  </p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <AlertDialog open={showWarning} onOpenChange={setShowWarning}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("enableStrictInventoryControl")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("warningNoManualStockChanges")}
              {t("inventory.allChangesMustGoThrough")}
              {t("confirmEnableThisMode")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={confirmEnable}>
              {updateSettings.isPending ? t("enabling") : t("enable1")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AdminGuard>
  );
}
