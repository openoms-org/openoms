"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useCreateReturn } from "@/hooks/use-returns";
import { apiClient, getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { OrderSearchCombobox } from "@/components/shared/order-search-combobox";
import type { Order, OrderItem } from "@/types/api";

const returnSchema = z.object({
  order_id: z.string().min(1, "ID zamówienia jest wymagane"),
  reason: z.string().min(1, "Powód zwrotu jest wymagany"),
  refund_amount: z.number().min(0, "Kwota musi być dodatnia"),
  notes: z.string().optional(),
});

type ReturnFormValues = z.infer<typeof returnSchema>;

interface SelectedItem {
  name: string;
  sku?: string;
  quantity: number;
  maxQuantity: number;
  selected: boolean;
}

export default function NewReturnPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const defaultOrderId = searchParams.get("order_id") ?? "";
  const createReturn = useCreateReturn();

  const [orderItems, setOrderItems] = useState<SelectedItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ReturnFormValues>({
    resolver: zodResolver(returnSchema),
    defaultValues: {
      order_id: defaultOrderId,
      reason: "",
      refund_amount: 0,
      notes: "",
    },
  });

  const orderId = watch("order_id");

  // Fetch order items when order_id changes
  useEffect(() => {
    if (!orderId) {
      setOrderItems([]);
      return;
    }

    let cancelled = false;
    setLoadingItems(true);

    apiClient<Order>(`/v1/orders/${orderId}`)
      .then((order) => {
        if (cancelled) return;
        const items: SelectedItem[] = (order.items || []).map((item: OrderItem) => ({
          name: item.name,
          sku: item.sku,
          quantity: item.quantity,
          maxQuantity: item.quantity,
          selected: false,
        }));
        setOrderItems(items);
      })
      .catch(() => {
        if (!cancelled) setOrderItems([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingItems(false);
      });

    return () => {
      cancelled = true;
    };
  }, [orderId]);

  const toggleItem = useCallback((index: number, checked: boolean) => {
    setOrderItems((prev) =>
      prev.map((item, i) =>
        i === index ? { ...item, selected: checked, quantity: checked ? item.quantity : item.maxQuantity } : item
      )
    );
  }, []);

  const updateItemQuantity = useCallback((index: number, quantity: number) => {
    setOrderItems((prev) =>
      prev.map((item, i) =>
        i === index ? { ...item, quantity: Math.max(1, Math.min(quantity, item.maxQuantity)) } : item
      )
    );
  }, []);

  const onSubmit = async (data: ReturnFormValues) => {
    const selectedItems = orderItems
      .filter((item) => item.selected)
      .map((item) => ({ name: item.name, quantity: item.quantity }));

    try {
      const result = await createReturn.mutateAsync({
        order_id: data.order_id,
        reason: data.reason,
        refund_amount: data.refund_amount,
        notes: data.notes || undefined,
        items: selectedItems.length > 0 ? selectedItems : undefined,
      });
      toast.success("Zwrot został utworzony");
      router.push(`/returns/${result.id}`);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/returns">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Nowy zwrot</h1>
          <p className="text-muted-foreground">
            Wypełnij formularz, aby zgłosić nowy zwrot
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
              <Label>Zamówienie</Label>
              <OrderSearchCombobox
                value={watch("order_id")}
                onValueChange={(id) =>
                  setValue("order_id", id, { shouldValidate: true })
                }
              />
              {errors.order_id && (
                <p className="text-sm text-destructive">{errors.order_id.message}</p>
              )}
            </div>

            {orderId && (
              <div className="space-y-2">
                <Label>Produkty do zwrotu</Label>
                {loadingItems ? (
                  <div className="flex items-center gap-2 py-3 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Wczytywanie pozycji zamówienia...
                  </div>
                ) : orderItems.length === 0 ? (
                  <p className="text-sm text-muted-foreground py-2">
                    Brak pozycji w zamówieniu
                  </p>
                ) : (
                  <div className="rounded-md border">
                    <div className="grid grid-cols-[auto_1fr_auto_auto] items-center gap-x-4 gap-y-0 text-sm">
                      <div className="contents font-medium text-muted-foreground border-b">
                        <div className="px-3 py-2" />
                        <div className="px-3 py-2">Produkt</div>
                        <div className="px-3 py-2 text-center">W zamówieniu</div>
                        <div className="px-3 py-2 text-center">Do zwrotu</div>
                      </div>
                      {orderItems.map((item, index) => (
                        <div key={index} className="contents">
                          <div className="px-3 py-2">
                            <input
                              type="checkbox"
                              checked={item.selected}
                              onChange={(e) => toggleItem(index, e.target.checked)}
                              className="h-4 w-4 rounded border-border"
                            />
                          </div>
                          <div className="px-3 py-2">
                            <span className="font-medium">{item.name}</span>
                            {item.sku && (
                              <span className="ml-2 text-xs text-muted-foreground">
                                SKU: {item.sku}
                              </span>
                            )}
                          </div>
                          <div className="px-3 py-2 text-center text-muted-foreground">
                            {item.maxQuantity}
                          </div>
                          <div className="px-3 py-2">
                            <Input
                              type="number"
                              min={1}
                              max={item.maxQuantity}
                              value={item.selected ? item.quantity : ""}
                              disabled={!item.selected}
                              onChange={(e) =>
                                updateItemQuantity(index, parseInt(e.target.value, 10) || 1)
                              }
                              className="h-8 w-20 text-center"
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="reason">Powód</Label>
              <Textarea
                id="reason"
                placeholder="Podaj powód zwrotu"
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
                {createReturn.isPending ? "Tworzenie..." : "Utwórz zwrot"}
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
