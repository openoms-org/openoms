"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { useCreateReturn } from "@/hooks/use-returns";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const returnSchema = z.object({
  order_id: z.string().min(1, "ID zamowienia jest wymagane"),
  reason: z.string().min(1, "Powod zwrotu jest wymagany"),
  refund_amount: z.number().min(0, "Kwota musi byc dodatnia"),
  notes: z.string().optional(),
});

type ReturnFormValues = z.infer<typeof returnSchema>;

export default function NewReturnPage() {
  const router = useRouter();
  const createReturn = useCreateReturn();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ReturnFormValues>({
    resolver: zodResolver(returnSchema),
    defaultValues: {
      order_id: "",
      reason: "",
      refund_amount: 0,
      notes: "",
    },
  });

  const onSubmit = async (data: ReturnFormValues) => {
    try {
      const result = await createReturn.mutateAsync({
        order_id: data.order_id,
        reason: data.reason,
        refund_amount: data.refund_amount,
        notes: data.notes || undefined,
      });
      toast.success("Zwrot zostal utworzony");
      router.push(`/returns/${result.id}`);
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas tworzenia zwrotu"
      );
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/returns">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Nowy zwrot</h1>
          <p className="text-muted-foreground">
            Wypelnij formularz, aby zglosic nowy zwrot
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Dane zwrotu</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="order_id">Zamowienie</Label>
              <Input
                id="order_id"
                placeholder="ID zamowienia"
                {...register("order_id")}
              />
              {errors.order_id && (
                <p className="text-sm text-destructive">{errors.order_id.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="reason">Powod</Label>
              <Textarea
                id="reason"
                placeholder="Podaj powod zwrotu"
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
                placeholder="0.00"
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
                placeholder="Dodatkowe informacje (opcjonalne)"
                {...register("notes")}
              />
              {errors.notes && (
                <p className="text-sm text-destructive">{errors.notes.message}</p>
              )}
            </div>

            <div className="flex items-center gap-2">
              <Button type="submit" disabled={createReturn.isPending}>
                {createReturn.isPending ? "Tworzenie..." : "Utworz zwrot"}
              </Button>
              <Button variant="outline" type="button" onClick={() => router.push("/returns")}>
                Anuluj
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
