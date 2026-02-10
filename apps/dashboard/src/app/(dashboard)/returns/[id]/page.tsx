"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { RETURN_STATUSES, RETURN_TRANSITIONS } from "@/lib/constants";
import { formatDate, formatCurrency } from "@/lib/utils";

const editSchema = z.object({
  reason: z.string().min(1, "Powód jest wymagany"),
  refund_amount: z.number().min(0, "Kwota musi być dodatnia"),
  notes: z.string().optional(),
});

type EditFormValues = z.infer<typeof editSchema>;

const TRANSITION_LABELS: Record<string, string> = {
  approved: "Zatwierdz",
  rejected: "Odrzuc",
  cancelled: "Anuluj",
  received: "Oznacz jako odebrane",
  refunded: "Zwroc srodki",
};

const TRANSITION_VARIANTS: Record<string, "default" | "destructive" | "outline" | "secondary"> = {
  approved: "default",
  rejected: "destructive",
  cancelled: "outline",
  received: "default",
  refunded: "default",
};

export default function ReturnDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

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
      toast.success("Zwrot został zaktualizowany");
      setIsEditing(false);
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Błąd podczas aktualizacji zwrotu"
      );
    }
  };

  const handleDelete = async () => {
    try {
      await deleteReturn.mutateAsync(params.id);
      toast.success("Zwrot został usunięty");
      router.push("/returns");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Błąd podczas usuwania zwrotu"
      );
    }
  };

  const handleTransition = async (newStatus: string) => {
    try {
      await transitionStatus.mutateAsync({ status: newStatus });
      toast.success("Status zwrotu został zmieniony");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Błąd podczas zmiany statusu"
      );
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
        <p className="text-muted-foreground">Nie znaleziono zwrotu</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/returns")}>
          Wróć do listy
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
              Zwrot #{params.id.slice(0, 8)}
            </h1>
            <StatusBadge status={returnData.status} statusMap={RETURN_STATUSES} />
          </div>
          <p className="text-muted-foreground mt-1">
            Utworzony {formatDate(returnData.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleEdit}>
            Edytuj
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            Usun
          </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>Edycja zwrotu</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(handleUpdate)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="reason">Powód</Label>
                <Textarea
                  id="reason"
                  {...register("reason")}
                />
                {errors.reason && (
                  <p className="text-sm text-destructive">{errors.reason.message}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="refund_amount">Kwota zwrotu (PLN)</Label>
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
                <Label htmlFor="notes">Notatki</Label>
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
                  {updateReturn.isPending ? "Zapisywanie..." : "Zapisz zmiany"}
                </Button>
                <Button variant="outline" type="button" onClick={() => setIsEditing(false)}>
                  Anuluj
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Dane zwrotu</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Zamowienie</p>
                    <Link
                      href={`/orders/${returnData.order_id}`}
                      className="mt-1 font-mono text-sm text-primary hover:underline"
                    >
                      {returnData.order_id.slice(0, 8)}
                    </Link>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Status</p>
                    <div className="mt-1">
                      <StatusBadge status={returnData.status} statusMap={RETURN_STATUSES} />
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Kwota zwrotu</p>
                    <p className="mt-1 font-medium">
                      {formatCurrency(returnData.refund_amount)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Utworzono</p>
                    <p className="mt-1 text-sm">{formatDate(returnData.created_at)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Zaktualizowano</p>
                    <p className="mt-1 text-sm">{formatDate(returnData.updated_at)}</p>
                  </div>
                </div>

                <Separator />

                <div>
                  <p className="text-sm text-muted-foreground">Powód</p>
                  <p className="mt-1 text-sm">{returnData.reason}</p>
                </div>

                {returnData.notes && (
                  <>
                    <Separator />
                    <div>
                      <p className="text-sm text-muted-foreground">Notatki</p>
                      <p className="mt-1 text-sm">{returnData.notes}</p>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            {allowedTransitions.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle>Zmiana statusu</CardTitle>
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
                        {TRANSITION_LABELS[status] || status}
                      </Button>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Usuwanie zwrotu</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz usunąć ten zwrot? Ta operacja jest nieodwracalna.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteReturn.isPending}
            >
              {deleteReturn.isPending ? "Usuwanie..." : "Usuń zwrot"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
