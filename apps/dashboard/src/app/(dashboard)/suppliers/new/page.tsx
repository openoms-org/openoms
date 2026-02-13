"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateSupplier } from "@/hooks/use-suppliers";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

const supplierSchema = z.object({
  name: z.string().min(1, "Nazwa jest wymagana"),
  code: z.string().optional(),
  feed_url: z.string().optional(),
  feed_format: z.string().optional(),
});

type SupplierForm = z.infer<typeof supplierSchema>;

export default function NewSupplierPage() {
  const router = useRouter();
  const createSupplier = useCreateSupplier();

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<SupplierForm>({
    resolver: zodResolver(supplierSchema),
    defaultValues: { feed_format: "iof" },
  });

  const onSubmit = (data: SupplierForm) => {
    createSupplier.mutate(data, {
      onSuccess: () => {
        toast.success("Dostawca został utworzony");
        router.push("/suppliers");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
    <div className="mx-auto max-w-4xl">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="ghost" size="icon" onClick={() => router.push("/suppliers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold tracking-tight">Nowy dostawca</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Dane dostawcy</CardTitle>
          <CardDescription>Dodaj nowego dostawcę produktów</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nazwa *</Label>
              <Input id="name" {...register("name")} placeholder="np. Hurtownia ABC" />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="code">Kod dostawcy</Label>
              <Input id="code" {...register("code")} placeholder="np. ABC123" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feed_url">URL feeda produktów</Label>
              <Input
                id="feed_url"
                {...register("feed_url")}
                placeholder="https://example.com/feed.xml"
              />
            </div>
            <div className="space-y-2">
              <Label>Format feeda</Label>
              <Select
                defaultValue="iof"
                onValueChange={(v) => setValue("feed_format", v)}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="iof">IOF (Internet Offer Format)</SelectItem>
                  <SelectItem value="csv">CSV</SelectItem>
                  <SelectItem value="xml">XML</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex gap-2 pt-4">
              <Button type="submit" disabled={createSupplier.isPending}>
                {createSupplier.isPending ? "Tworzenie..." : "Utwórz dostawcę"}
              </Button>
              <Button type="button" variant="outline" onClick={() => router.push("/suppliers")}>
                Anuluj
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
    </AdminGuard>
  );
}
