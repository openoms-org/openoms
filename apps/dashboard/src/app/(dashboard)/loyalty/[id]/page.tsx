"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Award, Trophy } from "lucide-react";
import { toast } from "sonner";
import {
  useLoyaltyProgram,
  useUpdateLoyaltyProgram,
  useDeleteLoyaltyProgram,
  useLeaderboard,
  useAwardPoints,
  useRedeemPoints,
} from "@/hooks/use-loyalty";
import { useCustomers } from "@/hooks/use-customers";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { UpdateLoyaltyProgramRequest } from "@/types/api";
import { useTranslations } from "next-intl";

export default function LoyaltyProgramDetailPage() {
  const t = useTranslations("loyalty");

  const PROGRAM_TYPE_LABELS: Record<string, string> = {
    points: t("type.points"),
    tier: t("type.tier"),
    discount_after_n: t("type.discountAfterN"),
  };

  const STATUS_LABELS: Record<string, { label: string; className: string }> = {
    active: {
      label: t("statusActive"),
      className: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    },
    paused: {
      label: t("statusPaused"),
      className: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    },
    ended: {
      label: t("zakonczony"),
      className: "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200",
    },
  };
  const params = useParams<{ id: string }>();
  const router = useRouter();

  const { data: program, isLoading } = useLoyaltyProgram(params.id);
  const updateProgram = useUpdateLoyaltyProgram(params.id);
  const deleteProgram = useDeleteLoyaltyProgram();
  const { data: leaderboard, isLoading: isLoadingLeaderboard } =
    useLeaderboard(params.id, 20);
  const awardPoints = useAwardPoints(params.id);
  const redeemPoints = useRedeemPoints(params.id);

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showAwardDialog, setShowAwardDialog] = useState(false);
  const [showRedeemDialog, setShowRedeemDialog] = useState(false);

  const [awardCustomerId, setAwardCustomerId] = useState("");
  const [awardPointsValue, setAwardPointsValue] = useState(0);
  const [awardReason, setAwardReason] = useState("");

  const [redeemCustomerId, setRedeemCustomerId] = useState("");
  const [redeemPointsValue, setRedeemPointsValue] = useState(0);

  const [customerSearch, setCustomerSearch] = useState("");
  const { data: customersData } = useCustomers({
    search: customerSearch,
    limit: 10,
  });
  const customers = customersData?.items ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!program) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">{t("programNotFound")}</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => router.push("/loyalty")}
        >
          {t("detail.backToList")}
        </Button>
      </div>
    );
  }

  const statusInfo = STATUS_LABELS[program.status] || STATUS_LABELS.active;

  const handleStatusChange = async (status: string) => {
    try {
      await updateProgram.mutateAsync({ status: status as UpdateLoyaltyProgramRequest["status"] });
      toast.success(t("statusprogramuzostałzmieniony"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteProgram.mutateAsync(params.id);
      toast.success(t("programzostałusuniety"));
      router.push("/loyalty");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleAward = async () => {
    try {
      await awardPoints.mutateAsync({
        customer_id: awardCustomerId,
        points: awardPointsValue,
        reason: awardReason,
      });
      toast.success(t("punktyzostałyprzyznane"));
      setShowAwardDialog(false);
      setAwardCustomerId("");
      setAwardPointsValue(0);
      setAwardReason("");
      setCustomerSearch("");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleRedeem = async () => {
    try {
      await redeemPoints.mutateAsync({
        customer_id: redeemCustomerId,
        points: redeemPointsValue,
      });
      toast.success(t("punktyzostaływymienione"));
      setShowRedeemDialog(false);
      setRedeemCustomerId("");
      setRedeemPointsValue(0);
      setCustomerSearch("");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const entries = leaderboard ?? [];

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push("/loyalty")}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">
            {program.name}
          </h1>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-sm text-muted-foreground">
              {PROGRAM_TYPE_LABELS[program.program_type]}
            </span>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusInfo.className}`}
            >
              {statusInfo.label}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {program.program_type === "points" && (
            <>
              <Button
                variant="outline"
                onClick={() => setShowAwardDialog(true)}
              >
                {t("awardPoints")}
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowRedeemDialog(true)}
              >
                {t("wymienPunkty")}
              </Button>
            </>
          )}
          <Button
            variant="destructive"
            onClick={() => setShowDeleteDialog(true)}
          >
            {t("delete")}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>{t("information")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm text-muted-foreground">{t("type.label")}</p>
              <p className="mt-1 text-sm font-medium">
                {PROGRAM_TYPE_LABELS[program.program_type]}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <Select
                value={program.status}
                onValueChange={handleStatusChange}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">{t("statusActive")}</SelectItem>
                  <SelectItem value="paused">{t("statusPaused")}</SelectItem>
                  <SelectItem value="ended">{t("zakonczony")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("createdAt")}</p>
              <p className="mt-1 text-sm">{formatDate(program.created_at)}</p>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t("configuration")}</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="rounded bg-muted p-3 text-sm font-mono overflow-auto">
              {JSON.stringify(program.config, null, 2)}
            </pre>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="flex items-center gap-2">
            <Trophy className="h-5 w-5" />
            Ranking (Top 20)
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoadingLeaderboard ? (
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
            </div>
          ) : entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <Award className="h-8 w-8 text-muted-foreground/50 mb-2" />
              <p className="text-sm text-muted-foreground">
                {t("brakUczestnikowWTymProgramie")}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[60px]">#</TableHead>
                  <TableHead>{t("customer")}</TableHead>
                  <TableHead className="text-right">{t("points")}</TableHead>
                  <TableHead className="text-right">{t("spent")}</TableHead>
                  <TableHead className="text-right">{t("zamowien")}</TableHead>
                  {program.program_type === "tier" && (
                    <TableHead>{t("tierLevel")}</TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map((entry, idx) => (
                  <TableRow
                    key={entry.customer_id}
                    className="cursor-pointer"
                    onClick={() =>
                      router.push(`/customers/${entry.customer_id}`)
                    }
                  >
                    <TableCell className="font-bold text-muted-foreground">
                      {idx + 1}
                    </TableCell>
                    <TableCell className="font-medium">
                      {entry.customer_name}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {entry.points.toLocaleString("pl-PL")}
                    </TableCell>
                    <TableCell className="text-right">
                      {formatCurrency(entry.total_spent)}
                    </TableCell>
                    <TableCell className="text-right">
                      {entry.order_count}
                    </TableCell>
                    {program.program_type === "tier" && (
                      <TableCell>
                        {entry.current_tier ? (
                          <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                            {entry.current_tier}
                          </span>
                        ) : (
                          "---"
                        )}
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Award points dialog */}
      <Dialog open={showAwardDialog} onOpenChange={setShowAwardDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("awardPoints")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("searchCustomer")}</Label>
              <Input
                value={customerSearch}
                onChange={(e) => setCustomerSearch(e.target.value)}
                placeholder={t("wpiszImieLubEmail")}
              />
            </div>
            {customers.length > 0 && (
              <Select
                value={awardCustomerId}
                onValueChange={setAwardCustomerId}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("selectCustomer")} />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} {c.email ? `(${c.email})` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <div className="space-y-2">
              <Label>{t("liczbaPunktow")}</Label>
              <Input
                type="number"
                min={1}
                value={awardPointsValue || ""}
                onChange={(e) =>
                  setAwardPointsValue(parseInt(e.target.value) || 0)
                }
              />
            </div>
            <div className="space-y-2">
              <Label>{t("detail.reason")}</Label>
              <Input
                value={awardReason}
                onChange={(e) => setAwardReason(e.target.value)}
                placeholder={t("npBonusZaLojalnosc")}
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowAwardDialog(false)}
              >
                {t("cancel")}
              </Button>
              <Button
                onClick={handleAward}
                disabled={
                  awardPoints.isPending ||
                  !awardCustomerId ||
                  awardPointsValue <= 0
                }
              >
                {awardPoints.isPending ? t("awarding") : t("award")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Redeem points dialog */}
      <Dialog open={showRedeemDialog} onOpenChange={setShowRedeemDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("wymienPunkty")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("searchCustomer")}</Label>
              <Input
                value={customerSearch}
                onChange={(e) => setCustomerSearch(e.target.value)}
                placeholder={t("wpiszImieLubEmail")}
              />
            </div>
            {customers.length > 0 && (
              <Select
                value={redeemCustomerId}
                onValueChange={setRedeemCustomerId}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("selectCustomer")} />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} {c.email ? `(${c.email})` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <div className="space-y-2">
              <Label>{t("liczbaPunktowDoWymiany")}</Label>
              <Input
                type="number"
                min={1}
                value={redeemPointsValue || ""}
                onChange={(e) =>
                  setRedeemPointsValue(parseInt(e.target.value) || 0)
                }
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowRedeemDialog(false)}
              >
                {t("cancel")}
              </Button>
              <Button
                onClick={handleRedeem}
                disabled={
                  redeemPoints.isPending ||
                  !redeemCustomerId ||
                  redeemPointsValue <= 0
                }
              >
                {redeemPoints.isPending ? t("redeeming") : t("wymien")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteProgram")}
        description={t("czyNaPewnoChceszUsunacTenProgramLojalnosciowyWszys")}
        confirmLabel={t("usunProgram")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteProgram.isPending}
      />
    </div>
  );
}
