"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import Link from "next/link";
import { ArrowLeft, ImageIcon, Package, Pencil, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { ProductForm } from "@/components/products/product-form";
import {
  useProduct,
  useUpdateProduct,
  useDeleteProduct,
} from "@/hooks/use-products";
import { formatCurrency, formatDate } from "@/lib/utils";

const SOURCE_LABELS: Record<string, string> = {
  manual: "Reczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

export default function ProductDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { data: product, isLoading } = useProduct(params.id);
  const updateProduct = useUpdateProduct(params.id);
  const deleteProduct = useDeleteProduct();

  const handleUpdate = (data: any) => {
    updateProduct.mutate(
      {
        name: data.name || undefined,
        sku: data.sku || undefined,
        ean: data.ean || undefined,
        price: data.price,
        stock_quantity: data.stock_quantity,
        source: data.source || undefined,
        image_url: data.image_url,
        images: data.images,
      },
      {
        onSuccess: () => {
          toast.success("Produkt zostal zaktualizowany");
          setIsEditing(false);
        },
        onError: (error) => {
          toast.error(error.message || "Nie udalo sie zaktualizowac produktu");
        },
      }
    );
  };

  const handleDelete = () => {
    deleteProduct.mutate(params.id, {
      onSuccess: () => {
        toast.success("Produkt zostal usuniety");
        router.push("/products");
      },
      onError: (error) => {
        toast.error(error.message || "Nie udalo sie usunac produktu");
      },
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!product) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Nie znaleziono produktu</h1>
        <Button asChild variant="outline">
          <Link href="/products">Wroc do listy</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/products">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{product.name}</h1>
            <p className="text-muted-foreground">
              Utworzony {formatDate(product.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            <Pencil className="h-4 w-4" />
            {isEditing ? "Anuluj edycje" : "Edytuj"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="h-4 w-4" />
            Usun
          </Button>
        </div>
      </div>

      {isEditing ? (
        <Card>
          <CardHeader>
            <CardTitle>Edycja produktu</CardTitle>
          </CardHeader>
          <CardContent>
            <ProductForm
              product={product}
              onSubmit={handleUpdate}
              isLoading={updateProduct.isPending}
            />
          </CardContent>
        </Card>
      ) : (
        <>
        <Card>
          <CardHeader>
            <CardTitle>Zdjecia</CardTitle>
          </CardHeader>
          <CardContent>
            {product.image_url ? (
              <div className="space-y-4">
                <img
                  src={product.image_url}
                  alt={product.name}
                  className="max-w-sm rounded-lg border object-cover"
                  onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                />
                {product.images && product.images.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {product.images.map((img, i) => (
                      <img
                        key={i}
                        src={img.url}
                        alt={img.alt || `Zdjecie ${i + 1}`}
                        className="h-20 w-20 rounded border object-cover"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                      />
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="flex h-32 w-32 items-center justify-center rounded-lg border bg-muted">
                <Package className="h-12 w-12 text-muted-foreground" />
              </div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Szczegoly produktu</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <p className="text-sm text-muted-foreground">Nazwa</p>
                <p className="text-sm font-medium">{product.name}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">ID</p>
                <p className="font-mono text-sm">{product.id}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">SKU</p>
                <p className="font-mono text-sm">{product.sku || "-"}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">EAN</p>
                <p className="font-mono text-sm">{product.ean || "-"}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Cena</p>
                <p className="text-sm font-medium">
                  {formatCurrency(product.price)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Stan magazynowy</p>
                <p
                  className={`text-sm font-medium ${
                    product.stock_quantity === 0
                      ? "text-destructive"
                      : product.stock_quantity < 10
                        ? "text-yellow-600"
                        : ""
                  }`}
                >
                  {product.stock_quantity}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Zrodlo</p>
                <p className="text-sm">
                  {SOURCE_LABELS[product.source] ?? product.source}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Identyfikator zewnetrzny
                </p>
                <p className="font-mono text-sm">
                  {product.external_id || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Data utworzenia</p>
                <p className="text-sm">{formatDate(product.created_at)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">
                  Ostatnia aktualizacja
                </p>
                <p className="text-sm">{formatDate(product.updated_at)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
        </>
      )}

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Usun produkt</DialogTitle>
            <DialogDescription>
              Czy na pewno chcesz usunac produkt &quot;{product.name}&quot;? Ta
              operacja jest nieodwracalna.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteProduct.isPending}
            >
              {deleteProduct.isPending ? "Usuwanie..." : "Usun"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
