"use client";

import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ProductForm } from "@/components/products/product-form";
import { useCreateProduct } from "@/hooks/use-products";

export default function NewProductPage() {
  const router = useRouter();
  const createProduct = useCreateProduct();

  const handleSubmit = (data: Parameters<typeof createProduct.mutate>[0]) => {
    createProduct.mutate(data, {
      onSuccess: (product) => {
        toast.success("Produkt został utworzony");
        router.push(`/products/${product.id}`);
      },
      onError: (error) => {
        toast.error(error.message || "Nie udało się utworzyć produktu");
      },
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/products">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">Nowy produkt</h1>
          <p className="text-muted-foreground">
            Dodaj nowy produkt do katalogu
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Dane produktu</CardTitle>
        </CardHeader>
        <CardContent>
          <ProductForm
            onSubmit={handleSubmit}
            isLoading={createProduct.isPending}
          />
        </CardContent>
      </Card>
    </div>
  );
}
