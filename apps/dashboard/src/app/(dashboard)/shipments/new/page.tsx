"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ShipmentForm } from "@/components/shipments/shipment-form";
import { useCreateShipment } from "@/hooks/use-shipments";

export default function NewShipmentPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const defaultOrderId = searchParams.get("order_id") ?? undefined;
  const createShipment = useCreateShipment();

  const handleSubmit = (data: Parameters<typeof createShipment.mutate>[0]) => {
    createShipment.mutate(data, {
      onSuccess: (shipment) => {
        toast.success("Przesyłka została utworzona");
        router.push(`/shipments/${shipment.id}`);
      },
      onError: (error) => {
        toast.error(error.message || "Nie udało się utworzyć przesyłki");
      },
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/shipments">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Nowa przesyłka</h1>
          <p className="text-muted-foreground">
            Utwórz nową przesyłkę dla zamówienia
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Dane przesyłki</CardTitle>
        </CardHeader>
        <CardContent>
          <ShipmentForm
            defaultValues={defaultOrderId ? { order_id: defaultOrderId } : undefined}
            onSubmit={handleSubmit}
            isLoading={createShipment.isPending}
          />
        </CardContent>
      </Card>
    </div>
  );
}
