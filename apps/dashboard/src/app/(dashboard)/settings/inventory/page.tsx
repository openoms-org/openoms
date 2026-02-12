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
import { Button } from "@/components/ui/button";
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

export default function InventorySettingsPage() {
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
            toast.success("Tryb ścisłej kontroli magazynowej wyłączony");
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
          toast.success("Tryb ścisłej kontroli magazynowej włączony");
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
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Ustawienia magazynowe
          </h1>
          <p className="text-muted-foreground">
            Konfiguruj zachowanie systemu magazynowego
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Kontrola magazynowa</CardTitle>
            <CardDescription>
              Ustawienia dotyczące sposobu zarządzania stanami magazynowymi
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between rounded-lg border p-4">
              <div className="space-y-0.5">
                <Label htmlFor="strict-mode" className="text-base font-medium">
                  Tryb ścisłej kontroli magazynowej
                </Label>
                <p className="text-sm text-muted-foreground">
                  Gdy włączony, zmiany stanów magazynowych możliwe tylko przez dokumenty PZ/WZ/MM
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
              <div className="flex items-start gap-3 rounded-lg border border-yellow-300 bg-yellow-50 p-4 dark:border-yellow-700 dark:bg-yellow-950">
                <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                    Tryb ścisłej kontroli jest aktywny
                  </p>
                  <p className="text-sm text-yellow-700 dark:text-yellow-300">
                    Ręczna zmiana stanów magazynowych (PUT /warehouses/&#123;id&#125;/stock) jest zablokowana.
                    Wszystkie zmiany muszą przechodzić przez dokumenty magazynowe (PZ, WZ, MM) lub inwentaryzacje.
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
              Włączenie ścisłej kontroli magazynowej
            </AlertDialogTitle>
            <AlertDialogDescription>
              Uwaga: po włączeniu nie będzie możliwa ręczna zmiana stanów magazynowych.
              Wszystkie zmiany będą musiały przechodzić przez dokumenty magazynowe (PZ/WZ/MM).
              Czy na pewno chcesz włączyć ten tryb?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Anuluj</AlertDialogCancel>
            <AlertDialogAction onClick={confirmEnable}>
              {updateSettings.isPending ? "Włączanie..." : "Włącz"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AdminGuard>
  );
}
