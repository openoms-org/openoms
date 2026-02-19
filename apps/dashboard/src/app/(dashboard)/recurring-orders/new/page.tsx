"use client";

import { useRouter } from "next/navigation";
import { useForm, useFieldArray } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { useCreateRecurringOrder } from "@/hooks/use-recurring-orders";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const itemSchema = z.object({
  sku: z.string().min(1, "SKU jest wymagane"),
  product_name: z.string().min(1, "Nazwa produktu jest wymagana"),
  quantity: z.number().min(1, "Ilosc musi byc wieksza od 0"),
  unit_price: z.number().min(0, "Cena nie moze byc ujemna"),
  product_id: z.string().optional(),
});

const formSchema = z.object({
  customer_name: z.string().min(1, "Nazwa klienta jest wymagana"),
  customer_email: z.string().email("Nieprawidlowy adres e-mail").or(z.literal("")).optional(),
  frequency: z.string().min(1, "Czestotliwosc jest wymagana"),
  next_order_date: z.string().min(1, "Data rozpoczecia jest wymagana"),
  end_date: z.string().optional(),
  max_orders: z.number().min(1).optional().or(z.literal(0).transform(() => undefined)),
  notes: z.string().optional(),
  items: z.array(itemSchema).min(1, "Wymagana co najmniej jedna pozycja"),
});

type FormValues = z.infer<typeof formSchema>;

export default function NewRecurringOrderPage() {
  const router = useRouter();
  const createRecurringOrder = useCreateRecurringOrder();

  const {
    register,
    handleSubmit,
    control,
    setValue,
    watch,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      customer_name: "",
      customer_email: "",
      frequency: "monthly",
      next_order_date: new Date().toISOString().split("T")[0],
      end_date: "",
      notes: "",
      items: [{ sku: "", product_name: "", quantity: 1, unit_price: 0 }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "items",
  });

  const frequency = watch("frequency");

  const onSubmit = (data: FormValues) => {
    const payload = {
      customer_name: data.customer_name,
      customer_email: data.customer_email || undefined,
      frequency: data.frequency,
      next_order_date: data.next_order_date,
      end_date: data.end_date || undefined,
      max_orders: data.max_orders || undefined,
      notes: data.notes || undefined,
      items: data.items.map((item) => ({
        sku: item.sku,
        product_name: item.product_name,
        quantity: item.quantity,
        unit_price: item.unit_price,
        product_id: item.product_id || undefined,
      })),
    };

    createRecurringOrder.mutate(payload, {
      onSuccess: () => {
        toast.success("Subskrypcja zostala utworzona");
        router.push("/recurring-orders");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <div className="mx-auto max-w-4xl">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/recurring-orders")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold tracking-tight">Nowa subskrypcja</h1>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Dane klienta</CardTitle>
            <CardDescription>Klient, dla ktorego tworzona jest subskrypcja</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="customer_name">
                  Nazwa klienta <span className="text-destructive">*</span>
                </Label>
                <Input
                  id="customer_name"
                  aria-invalid={!!errors.customer_name}
                  {...register("customer_name")}
                  placeholder="np. Firma ABC Sp. z o.o."
                />
                {errors.customer_name && (
                  <p className="text-destructive text-xs mt-1">{errors.customer_name.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="customer_email">E-mail klienta</Label>
                <Input
                  id="customer_email"
                  type="email"
                  aria-invalid={!!errors.customer_email}
                  {...register("customer_email")}
                  placeholder="firma@example.com"
                />
                {errors.customer_email && (
                  <p className="text-destructive text-xs mt-1">{errors.customer_email.message}</p>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Harmonogram</CardTitle>
            <CardDescription>Czestotliwosc i daty realizacji zamowien</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>
                  Czestotliwosc <span className="text-destructive">*</span>
                </Label>
                <Select
                  value={frequency}
                  onValueChange={(value) => setValue("frequency", value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="daily">Codziennie</SelectItem>
                    <SelectItem value="weekly">Co tydzien</SelectItem>
                    <SelectItem value="biweekly">Co 2 tygodnie</SelectItem>
                    <SelectItem value="monthly">Co miesiac</SelectItem>
                    <SelectItem value="quarterly">Co kwartal</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="next_order_date">
                  Data pierwszego zamowienia <span className="text-destructive">*</span>
                </Label>
                <Input
                  id="next_order_date"
                  type="date"
                  aria-invalid={!!errors.next_order_date}
                  {...register("next_order_date")}
                />
                {errors.next_order_date && (
                  <p className="text-destructive text-xs mt-1">{errors.next_order_date.message}</p>
                )}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="end_date">Data zakonczenia (opcjonalnie)</Label>
                <Input
                  id="end_date"
                  type="date"
                  {...register("end_date")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max_orders">Maks. liczba zamowien (opcjonalnie)</Label>
                <Input
                  id="max_orders"
                  type="number"
                  min={0}
                  {...register("max_orders", { valueAsNumber: true })}
                  placeholder="np. 12"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Pozycje zamowienia</CardTitle>
            <CardDescription>
              Produkty, ktore beda zamawiane w kazdym cyklu
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {fields.map((field, index) => (
              <div key={field.id} className="flex gap-3 items-start">
                <div className="flex-1 space-y-2">
                  <Label>SKU</Label>
                  <Input
                    {...register(`items.${index}.sku`)}
                    placeholder="SKU-001"
                  />
                  {errors.items?.[index]?.sku && (
                    <p className="text-destructive text-xs">{errors.items[index].sku?.message}</p>
                  )}
                </div>
                <div className="flex-[2] space-y-2">
                  <Label>Nazwa produktu</Label>
                  <Input
                    {...register(`items.${index}.product_name`)}
                    placeholder="Nazwa produktu"
                  />
                  {errors.items?.[index]?.product_name && (
                    <p className="text-destructive text-xs">{errors.items[index].product_name?.message}</p>
                  )}
                </div>
                <div className="w-24 space-y-2">
                  <Label>Ilosc</Label>
                  <Input
                    type="number"
                    min={1}
                    {...register(`items.${index}.quantity`, { valueAsNumber: true })}
                  />
                </div>
                <div className="w-32 space-y-2">
                  <Label>Cena jedn.</Label>
                  <Input
                    type="number"
                    step="0.01"
                    min={0}
                    {...register(`items.${index}.unit_price`, { valueAsNumber: true })}
                  />
                </div>
                <div className="pt-7">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    disabled={fields.length <= 1}
                    onClick={() => remove(index)}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </div>
            ))}
            {errors.items && typeof errors.items.message === "string" && (
              <p className="text-destructive text-xs">{errors.items.message}</p>
            )}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => append({ sku: "", product_name: "", quantity: 1, unit_price: 0 })}
            >
              <Plus className="h-4 w-4 mr-1" />
              Dodaj pozycje
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Dodatkowe</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="notes">Notatki</Label>
              <Textarea
                id="notes"
                {...register("notes")}
                placeholder="Dodatkowe informacje o subskrypcji..."
                rows={3}
              />
            </div>
          </CardContent>
        </Card>

        <div className="flex gap-2">
          <Button type="submit" disabled={createRecurringOrder.isPending}>
            {createRecurringOrder.isPending ? "Tworzenie..." : "Utworz subskrypcje"}
          </Button>
          <Button type="button" variant="outline" onClick={() => router.push("/recurring-orders")}>
            Anuluj
          </Button>
        </div>
      </form>
    </div>
  );
}
