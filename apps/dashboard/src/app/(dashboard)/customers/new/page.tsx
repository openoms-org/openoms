"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import { useCreateCustomer } from "@/hooks/use-customers";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const customerSchema = z.object({
  name: z.string().min(1, "Imię i nazwisko jest wymagane"),
  email: z.string().email("Nieprawidłowy adres e-mail").or(z.literal("")).optional(),
  phone: z.string().optional(),
  company_name: z.string().optional(),
  nip: z.string().optional(),
  notes: z.string().optional(),
});

type CustomerForm = z.infer<typeof customerSchema>;

export default function NewCustomerPage() {
  const router = useRouter();
  const createCustomer = useCreateCustomer();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CustomerForm>({
    resolver: zodResolver(customerSchema),
  });

  const onSubmit = (data: CustomerForm) => {
    const payload = {
      name: data.name,
      email: data.email || undefined,
      phone: data.phone || undefined,
      company_name: data.company_name || undefined,
      nip: data.nip || undefined,
      notes: data.notes || undefined,
    };

    createCustomer.mutate(payload, {
      onSuccess: () => {
        toast.success("Klient został utworzony");
        router.push("/customers");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/customers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold tracking-tight">Nowy klient</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Dane klienta</CardTitle>
          <CardDescription>Dodaj nowego klienta do bazy</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Imię i nazwisko *</Label>
              <Input id="name" {...register("name")} placeholder="np. Jan Kowalski" />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="email">E-mail</Label>
                <Input
                  id="email"
                  type="email"
                  {...register("email")}
                  placeholder="jan@example.com"
                />
                {errors.email && (
                  <p className="text-sm text-destructive">{errors.email.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">Telefon</Label>
                <Input id="phone" {...register("phone")} placeholder="+48 123 456 789" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="company_name">Firma</Label>
                <Input
                  id="company_name"
                  {...register("company_name")}
                  placeholder="np. Firma Sp. z o.o."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="nip">NIP</Label>
                <Input id="nip" {...register("nip")} placeholder="np. 1234567890" />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="notes">Notatki</Label>
              <Textarea
                id="notes"
                {...register("notes")}
                placeholder="Dodatkowe informacje o kliencie..."
                rows={3}
              />
            </div>
            <div className="flex gap-2 pt-4">
              <Button type="submit" disabled={createCustomer.isPending}>
                {createCustomer.isPending ? "Tworzenie..." : "Utwórz klienta"}
              </Button>
              <Button type="button" variant="outline" onClick={() => router.push("/customers")}>
                Anuluj
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
