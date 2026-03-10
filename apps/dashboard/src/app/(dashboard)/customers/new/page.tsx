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
import { useTranslations } from "next-intl";

const customerSchema = z.object({
  name: z.string().min(1),
  email: z.string().email().or(z.literal("")).optional(),
  phone: z.string().optional(),
  company_name: z.string().optional(),
  nip: z.string().optional(),
  notes: z.string().optional(),
});

type CustomerForm = z.infer<typeof customerSchema>;

export default function NewCustomerPage() {
  const t = useTranslations("customers");
  const tc = useTranslations("common");
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
        toast.success(t("created"));
        router.push("/customers");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <div className="mx-auto max-w-4xl">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/customers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold tracking-tight">{t("newCustomer")}</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("customerData")}</CardTitle>
          <CardDescription>{t("addDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t("form.fullName")}<span className="text-destructive">*</span></Label>
              <Input id="name" aria-invalid={!!errors.name} {...register("name")} placeholder={t("form.fullNamePlaceholder")} />
              {errors.name && (
                <p className="text-destructive text-xs mt-1">{errors.name.message}</p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t("form.email")}</Label>
                <Input
                  id="email"
                  type="email"
                  aria-invalid={!!errors.email}
                  {...register("email")}
                  placeholder="jan@example.com"
                />
                {errors.email && (
                  <p className="text-destructive text-xs mt-1">{errors.email.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">{t("form.phone")}</Label>
                <Input id="phone" {...register("phone")} placeholder="+48 123 456 789" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="company_name">{t("form.company")}</Label>
                <Input
                  id="company_name"
                  {...register("company_name")}
                  placeholder={t("form.companyPlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="nip">NIP</Label>
                <Input id="nip" {...register("nip")} placeholder="np. 1234567890" />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="notes">{t("form.notes")}</Label>
              <Textarea
                id="notes"
                {...register("notes")}
                placeholder={t("form.notesPlaceholder")}
                rows={3}
              />
            </div>
            <div className="flex gap-2 pt-4">
              <Button type="submit" disabled={createCustomer.isPending}>
                {createCustomer.isPending ? tc("creating") : t("form.submitCreate")}
              </Button>
              <Button type="button" variant="outline" onClick={() => router.push("/customers")}>
                {tc("cancel")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
