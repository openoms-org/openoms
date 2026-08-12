"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  ScanBarcode,
  Check,
  CheckCircle2,
  X,
  Package,
  PackageCheck,
  ArrowRight,
  Ban,
  ChevronLeft,
} from "lucide-react";
import { toast } from "sonner";
import {
  usePickPackSession,
  useScanItem,
  useMoveToPacking,
  useMarkItemPacked,
  useCompletePickPackSession,
  useCancelPickPackSession,
} from "@/hooks/use-pick-pack";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import type { PickPackItem } from "@/types/api";
import { useTranslations } from "next-intl";
import { enumLabel } from "@/lib/i18n-fallback";

export default function PickPackSessionPage() {
  const t = useTranslations("pickPack");
  const ts = useTranslations("statuses");
  const params = useParams();
  const router = useRouter();
  const sessionId = params.id as string;

  const [barcodeInput, setBarcodeInput] = useState("");
  const [lastScanResult, setLastScanResult] = useState<"success" | "error" | null>(null);
  const [lastScanMessage, setLastScanMessage] = useState("");
  const barcodeRef = useRef<HTMLInputElement>(null);

  const { data: session, isLoading, isError } = usePickPackSession(sessionId);
  const scanItem = useScanItem(sessionId);
  const moveToPacking = useMoveToPacking(sessionId);
  const markItemPacked = useMarkItemPacked(sessionId);
  const completeSession = useCompletePickPackSession(sessionId);
  const cancelSession = useCancelPickPackSession(sessionId);

  const items: PickPackItem[] = session?.items ?? [];
  const stats = session?.stats;
  const isPicking = session?.status === "picking";
  const isPacking = session?.status === "packing";
  const isActive = isPicking || isPacking;

  // Auto-focus barcode input when active
  useEffect(() => {
    if (isActive && barcodeRef.current) {
      barcodeRef.current.focus();
    }
  }, [isActive, session?.status]);

  // Clear scan result after 3 seconds
  useEffect(() => {
    if (lastScanResult) {
      const timer = setTimeout(() => {
        setLastScanResult(null);
        setLastScanMessage("");
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [lastScanResult]);

  const handleBarcodeScan = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const code = barcodeInput.trim();
      if (!code) return;

      setBarcodeInput("");

      try {
        const resp = await scanItem.mutateAsync({ barcode: code });
        setLastScanResult("success");
        setLastScanMessage(resp.message);
        // Play a simple success sound via AudioContext
        try {
          const ctx = new AudioContext();
          const osc = ctx.createOscillator();
          const gain = ctx.createGain();
          osc.connect(gain);
          gain.connect(ctx.destination);
          osc.frequency.value = 800;
          gain.gain.value = 0.1;
          osc.start();
          osc.stop(ctx.currentTime + 0.1);
        } catch {
          // Audio not available, ignore
        }
      } catch (err) {
        setLastScanResult("error");
        setLastScanMessage(getErrorMessage(err));
        // Play error sound
        try {
          const ctx = new AudioContext();
          const osc = ctx.createOscillator();
          const gain = ctx.createGain();
          osc.connect(gain);
          gain.connect(ctx.destination);
          osc.frequency.value = 300;
          gain.gain.value = 0.1;
          osc.start();
          osc.stop(ctx.currentTime + 0.3);
        } catch {
          // Audio not available, ignore
        }
      }

      barcodeRef.current?.focus();
    },
    [barcodeInput, scanItem]
  );

  const handleMoveToPacking = async () => {
    try {
      await moveToPacking.mutateAsync();
      toast.success(t("movedToPacking"));
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleMarkPacked = async (itemId: string, remaining: number) => {
    try {
      await markItemPacked.mutateAsync({
        itemId,
        data: { quantity: remaining },
      });
      toast.success(t("itemPacked"));
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleComplete = async () => {
    try {
      await completeSession.mutateAsync();
      toast.success(t("sessionCompleted"));
      router.push("/pick-pack");
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleCancel = async () => {
    try {
      await cancelSession.mutateAsync();
      toast.info(t("sessionCancelled"));
      router.push("/pick-pack");
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError || !session) {
    return (
      <div className="p-8 text-center">
        <p className="text-muted-foreground">Nie znaleziono sesji</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/pick-pack")}>
          Powrot do listy
        </Button>
      </div>
    );
  }

  const pickProgress = stats && stats.total_required > 0
    ? (stats.total_picked / stats.total_required) * 100
    : 0;
  const packProgress = stats && stats.total_required > 0
    ? (stats.total_packed / stats.total_required) * 100
    : 0;

  return (
    <div className="max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.push("/pick-pack")}>
            <ChevronLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("sessionTitle")}
            </h1>
            <p className="text-muted-foreground">
              {t(`sessionTypes.${session.session_type}`)} &middot;{" "}
              {enumLabel(ts, "pickPack", session.status)} &middot;{" "}
              {t("ordersCount", { count: stats?.order_count ?? 0 })}
            </p>
          </div>
        </div>
        {isActive && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive" size="lg">
                <Ban className="h-5 w-5 mr-2" />
                Anuluj sesje
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Anulowac sesje?</AlertDialogTitle>
                <AlertDialogDescription>
                  Wszystkie postepy kompletacji i pakowania zostana utracone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Nie</AlertDialogCancel>
                <AlertDialogAction onClick={handleCancel}>
                  Tak, anuluj
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      {/* Progress overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <Card className={isPicking ? "ring-2 ring-blue-500" : ""}>
          <CardHeader className="pb-2">
            <CardTitle className="text-base flex items-center gap-2">
              <ScanBarcode className="h-5 w-5" />
              {t("pickingProgress")}
              {stats?.all_picked && (
                <Check className="h-5 w-5 text-green-500 ml-auto" />
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-3 bg-muted rounded-full overflow-hidden mb-2">
              <div
                className="h-full bg-blue-500 rounded-full transition-all"
                style={{ width: `${pickProgress}%` }}
              />
            </div>
            <p className="text-2xl font-bold">
              {stats?.total_picked ?? 0}{" "}
              <span className="text-base font-normal text-muted-foreground">
                / {stats?.total_required ?? 0} sztuk
              </span>
            </p>
          </CardContent>
        </Card>

        <Card className={isPacking ? "ring-2 ring-green-500" : ""}>
          <CardHeader className="pb-2">
            <CardTitle className="text-base flex items-center gap-2">
              <PackageCheck className="h-5 w-5" />
              {t("packingProgress")}
              {stats?.all_packed && (
                <Check className="h-5 w-5 text-green-500 ml-auto" />
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-3 bg-muted rounded-full overflow-hidden mb-2">
              <div
                className="h-full bg-green-500 rounded-full transition-all"
                style={{ width: `${packProgress}%` }}
              />
            </div>
            <p className="text-2xl font-bold">
              {stats?.total_packed ?? 0}{" "}
              <span className="text-base font-normal text-muted-foreground">
                / {stats?.total_required ?? 0} sztuk
              </span>
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Scan feedback banner */}
      {lastScanResult && (
        <div
          className={`mb-4 p-4 rounded-lg text-lg font-medium transition-all ${
            lastScanResult === "success"
              ? "bg-green-100 text-green-800 border-2 border-green-300 dark:bg-green-900/30 dark:text-green-200 dark:border-green-700"
              : "bg-red-100 text-red-800 border-2 border-red-300 dark:bg-red-900/30 dark:text-red-200 dark:border-red-700"
          }`}
        >
          <div className="flex items-center gap-3">
            {lastScanResult === "success" ? (
              <Check className="h-6 w-6 flex-shrink-0" />
            ) : (
              <X className="h-6 w-6 flex-shrink-0" />
            )}
            {lastScanMessage}
          </div>
        </div>
      )}

      {/* Barcode input - only shown during picking phase */}
      {isPicking && (
        <Card className="mb-6">
          <CardContent className="pt-6">
            <form onSubmit={handleBarcodeScan} className="flex gap-3">
              <div className="relative flex-1">
                <ScanBarcode className="absolute left-4 top-1/2 -translate-y-1/2 h-6 w-6 text-muted-foreground" />
                <Input
                  ref={barcodeRef}
                  placeholder="Skanuj kod kreskowy lub wpisz SKU/EAN..."
                  value={barcodeInput}
                  onChange={(e) => setBarcodeInput(e.target.value)}
                  className="pl-14 text-xl h-16 font-mono"
                  autoFocus
                  autoComplete="off"
                />
              </div>
              <Button type="submit" size="lg" className="h-16 px-8 text-lg">
                Skanuj
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Items list */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Pozycje ({items.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              Brak pozycji w sesji
            </p>
          ) : (
            <div className="space-y-2">
              {items.map((item) => {
                const isPickedFull = item.quantity_picked >= item.quantity_required;
                const isPackedFull = item.quantity_packed >= item.quantity_required;
                const remainingToPack = item.quantity_required - item.quantity_packed;

                let borderColor = "border-border";
                let bgColor = "";
                if (isPicking) {
                  if (isPickedFull) {
                    borderColor = "border-green-300 dark:border-green-700";
                    bgColor = "bg-green-50 dark:bg-green-900/10";
                  }
                } else if (isPacking) {
                  if (isPackedFull) {
                    borderColor = "border-green-300 dark:border-green-700";
                    bgColor = "bg-green-50 dark:bg-green-900/10";
                  } else if (isPickedFull) {
                    borderColor = "border-yellow-300 dark:border-yellow-700";
                    bgColor = "bg-yellow-50 dark:bg-yellow-900/10";
                  }
                } else if (session.status === "completed") {
                  borderColor = "border-green-300 dark:border-green-700";
                  bgColor = "bg-green-50 dark:bg-green-900/10";
                }

                return (
                  <div
                    key={item.id}
                    className={`flex items-center justify-between p-4 rounded-lg border-2 ${borderColor} ${bgColor} transition-all`}
                  >
                    <div className="flex items-center gap-4 flex-1 min-w-0">
                      {isPackedFull ? (
                        <CheckCircle2 className="h-8 w-8 text-green-500 flex-shrink-0" />
                      ) : isPickedFull ? (
                        <Check className="h-8 w-8 text-blue-500 flex-shrink-0" />
                      ) : (
                        <Package className="h-8 w-8 text-muted-foreground flex-shrink-0" />
                      )}
                      <div className="min-w-0">
                        <p className="font-semibold text-lg truncate">
                          {item.product_name}
                        </p>
                        <div className="flex items-center gap-3 text-sm text-muted-foreground">
                          <span>SKU: {item.sku}</span>
                          {item.pick_location && (
                            <Badge variant="outline" className="text-xs">
                              {item.pick_location}
                            </Badge>
                          )}
                          <span className="text-xs">
                            Zamowienie: #{item.order_id.slice(0, 8)}
                          </span>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 flex-shrink-0">
                      {/* Pick progress */}
                      <div className="text-center min-w-[80px]">
                        <p className="text-xs text-muted-foreground mb-1">Zebrano</p>
                        <Badge
                          variant={isPickedFull ? "default" : "outline"}
                          className={`text-base px-3 py-1 ${isPickedFull ? "bg-blue-500" : ""}`}
                        >
                          {item.quantity_picked}/{item.quantity_required}
                        </Badge>
                      </div>

                      {/* Pack progress */}
                      <div className="text-center min-w-[80px]">
                        <p className="text-xs text-muted-foreground mb-1">Spakowano</p>
                        <Badge
                          variant={isPackedFull ? "default" : "outline"}
                          className={`text-base px-3 py-1 ${isPackedFull ? "bg-green-500" : ""}`}
                        >
                          {item.quantity_packed}/{item.quantity_required}
                        </Badge>
                      </div>

                      {/* Pack button */}
                      {isPacking && !isPackedFull && (
                        <Button
                          size="lg"
                          className="h-12 px-6 text-base"
                          onClick={() => handleMarkPacked(item.id, remainingToPack)}
                          disabled={markItemPacked.isPending}
                        >
                          <PackageCheck className="h-5 w-5 mr-2" />
                          Pakuj ({remainingToPack})
                        </Button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Action buttons */}
      {isActive && (
        <div className="flex justify-center gap-4 mt-6 mb-8">
          {isPicking && stats?.all_picked && (
            <Button
              size="lg"
              className="h-14 px-10 text-lg"
              onClick={handleMoveToPacking}
              disabled={moveToPacking.isPending}
            >
              <ArrowRight className="h-6 w-6 mr-3" />
              {moveToPacking.isPending ? "Przechodzenie..." : "Przejdz do pakowania"}
            </Button>
          )}

          {isPacking && stats?.all_packed && (
            <Button
              size="lg"
              className="h-14 px-10 text-lg bg-green-600 hover:bg-green-700"
              onClick={handleComplete}
              disabled={completeSession.isPending}
            >
              <CheckCircle2 className="h-6 w-6 mr-3" />
              {completeSession.isPending ? t("konczenie") : "Zakoncz i wyslij"}
            </Button>
          )}
        </div>
      )}

      {/* Completed/Cancelled state */}
      {(session.status === "completed" || session.status === "cancelled") && (
        <div className="flex justify-center mt-6 mb-8">
          <Button
            variant="outline"
            size="lg"
            onClick={() => router.push("/pick-pack")}
          >
            <ChevronLeft className="h-5 w-5 mr-2" />
            Powrot do listy
          </Button>
        </div>
      )}
    </div>
  );
}
