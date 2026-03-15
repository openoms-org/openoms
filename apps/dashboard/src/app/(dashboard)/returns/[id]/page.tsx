"use client";

import { useState, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ExternalLink, Copy, Check } from "lucide-react";
import {
  useReturn,
  useUpdateReturn,
  useDeleteReturn,
  useTransitionReturnStatus,
} from "@/hooks/use-returns";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { RETURN_STATUSES, RETURN_TRANSITIONS } from "@/lib/constants";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { useTranslations } from "next-intl";

function createEditSchema(t: (key: string) => string) {
  return z.object({
    reason: z.string().min(1, t("validation.reasonRequired")),
    refund_amount: z.number().min(0, t("validation.amountPositive")),
    notes: z.string().optional(),
  });
}

type EditFormValues = z.infer<ReturnType<typeof createEditSchema>>;

const TRANSITION_LABEL_KEYS: Record<string, string> = {
  approved: "transitions.approved",
  rejected: "transitions.rejected",
  cancelled: "transitions.cancelled",
  received: "transitions.received",
  refunded: "transitions.refunded",
};

const TRANSITION_VARIANTS: Record<string, "default" | "destructive" | "outline" | "secondary"> = {
  approved: "default",
  rejected: "destructive",
  cancelled: "outline",
  received: "default",
  refunded: "default",
};

export default function ReturnDetailPage() {
  const t = useTranslations("returns");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [tokenCopied, setTokenCopied] = useState(false);

  const editSchema = useMemo(() => createEditSchema(t), [t]);
  const { data: returnData, isLoading } = useReturn(params.id);
  const updateReturn = useUpdateReturn(params.id);
  const deleteReturn = useDeleteReturn();
  const transitionStatus = useTransitionReturnStatus(params.id);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<EditFormValues>({
    resolver: zodResolver(editSchema),
  });

  const handleEdit = () => {
    if (returnData) {
      reset({
        reason: returnData.reason,
        refund_amount: returnData.refund_amount,
        notes: returnData.notes || "",
      });
    }
    setIsEditing(true);
  };

  const handleUpdate = async (data: EditFormValues) => {
    try {
      await updateReturn.mutateAsync({
        reason: data.reason,
        refund_amount: data.refund_amount,
        notes: data.notes || undefined,
      });
      toast.success(t("returnUpdated"));
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteReturn.mutateAsync(params.id);
      toast.success(t("returnDeleted"));
      router.push("/returns");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleTransition = async (newStatus: string) => {
    try {
      await transitionStatus.mutateAsync({ status: newStatus });
      toast.success(t("statusChanged"));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

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

  if (!returnData) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">{t("notFound")}</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/returns")}>
          {t("detail.backToList")}
        </Button>
      </div>
    );
  }

  const allowedTransitions = RETURN_TRANSITIONS[returnData.status] || [];

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">
              {t("returnTitle", { id: shortId(params.id) })}
            </h1>
            <StatusBadge status={returnData.status} statusMap={RETURN_STATUSES} />
          </div>
          <p className="text-muted-foreground mt-1">
            {t("createdOn", { date: formatDate(returnData.created_at) })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleEdit}>
            {t("edit")}
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            {t("delete")}
          </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("editReturn")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(handleUpdate)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="reason">{t("columns.reason")}</Label>
                <Textarea
                  id="reason"
                  {...register("reason")}
                />
                {errors.reason && (
                  <p className="text-sm text-destructive">{errors.reason.message}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="refund_amount">{t("refundAmount")}</Label>
                <Input
                  id="refund_amount"
                  type="number"
                  step="0.01"
                  {...register("refund_amount", { valueAsNumber: true })}
                />
                {errors.refund_amount && (
                  <p className="text-sm text-destructive">{errors.refund_amount.message}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="notes">{t("notes")}</Label>
                <Textarea
                  id="notes"
                  {...register("notes")}
                />
                {errors.notes && (
                  <p className="text-sm text-destructive">{errors.notes.message}</p>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Button type="submit" disabled={updateReturn.isPending}>
                  {updateReturn.isPending ? t("saving") : t("saveChanges")}
                </Button>
                <Button variant="outline" type="button" onClick={() => setIsEditing(false)}>
                  {t("cancel")}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div className="md:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>{t("returnData")}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">{t("columns.order")}</p>
                    <Link
                      href={`/orders/${returnData.order_id}`}
                      className="mt-1 font-mono text-sm text-primary hover:underline"
                    >
                      {shortId(returnData.order_id)}
                    </Link>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{t("columns.status")}</p>
                    <div className="mt-1">
                      <StatusBadge status={returnData.status} statusMap={RETURN_STATUSES} />
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{t("refundAmount")}</p>
                    <p className="mt-1 font-medium">
                      {formatCurrency(returnData.refund_amount)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{t("created")}</p>
                    <p className="mt-1 text-sm">{formatDate(returnData.created_at)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{t("updated")}</p>
                    <p className="mt-1 text-sm">{formatDate(returnData.updated_at)}</p>
                  </div>
                </div>

                <Separator />

                <div>
                  <p className="text-sm text-muted-foreground">{t("columns.reason")}</p>
                  <p className="mt-1 text-sm">{returnData.reason}</p>
                </div>

                {returnData.notes && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm text-muted-foreground">{t("notes")}</p>
                      <p className="mt-1 text-sm">{returnData.notes}</p>
                    </div>
                  </>
                )}

                {returnData.customer_email && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm text-muted-foreground">{t("customerEmail")}</p>
                      <p className="mt-1 text-sm">{returnData.customer_email}</p>
                    </div>
                  </>
                )}

                {returnData.customer_notes && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm text-muted-foreground">{t("customerNotes")}</p>
                      <p className="mt-1 text-sm">{returnData.customer_notes}</p>
                    </div>
                  </>
                )}

                {returnData.return_token && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm text-muted-foreground mb-1">{t("trackingLinkForCustomer")}</p>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 rounded bg-muted px-2 py-1 text-xs font-mono break-all">
                          {typeof window !== "undefined"
                            ? `${window.location.origin}/return-request/${returnData.return_token}`
                            : `/return-request/${returnData.return_token}`}
                        </code>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            const url = `${window.location.origin}/return-request/${returnData.return_token}`;
                            navigator.clipboard.writeText(url).then(() => {
                              setTokenCopied(true);
                              setTimeout(() => setTokenCopied(false), 2000);
                            });
                          }}
                        >
                          {tokenCopied ? (
                            <Check className="h-4 w-4 text-green-600" />
                          ) : (
                            <Copy className="h-4 w-4" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          asChild
                        >
                          <a
                            href={`/return-request/${returnData.return_token}`}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            <ExternalLink className="h-4 w-4" />
                          </a>
                        </Button>
                      </div>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            {allowedTransitions.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle>{t("changeStatus")}</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {allowedTransitions.map((status) => (
                      <Button
                        key={status}
                        variant={TRANSITION_VARIANTS[status] || "outline"}
                        onClick={() => handleTransition(status)}
                        disabled={transitionStatus.isPending}
                      >
                        {TRANSITION_LABEL_KEYS[status] ? t(TRANSITION_LABEL_KEYS[status]) : status}
                      </Button>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteReturn")}
        description={t("deleteReturnConfirm")}
        confirmLabel={t("deleteReturn")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteReturn.isPending}
      />
    </div>
  );
}
