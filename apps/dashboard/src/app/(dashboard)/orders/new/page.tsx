"use client";

import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useCreateOrder } from "@/hooks/use-orders";
import { OrderForm } from "@/components/orders/order-form";
import type { CreateOrderRequest } from "@/types/api";

export default function NewOrderPage() {
  const router = useRouter();
  const createOrder = useCreateOrder();

  const handleSubmit = async (data: CreateOrderRequest) => {
    try {
      const order = await createOrder.mutateAsync(data);
      toast.success("Zamowienie zostalo utworzone");
      router.push(`/orders/${order.id}`);
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Blad podczas tworzenia zamowienia"
      );
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Nowe zamowienie</h1>
        <p className="text-muted-foreground mt-1">
          Wypelnij formularz, aby utworzyc nowe zamowienie
        </p>
      </div>

      <div className="max-w-2xl">
        <OrderForm
          onSubmit={handleSubmit}
          isSubmitting={createOrder.isPending}
          onCancel={() => router.push("/orders")}
        />
      </div>
    </div>
  );
}
