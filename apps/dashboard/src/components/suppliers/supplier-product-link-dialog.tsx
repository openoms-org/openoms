"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { Search, Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useProducts } from "@/hooks/use-products";
import { useLinkSupplierProduct } from "@/hooks/use-suppliers";
import { getErrorMessage } from "@/lib/api-client";
import { formatCurrency } from "@/lib/utils";
import type { SupplierProduct } from "@/types/api";

/**
 * Links an unlinked supplier-catalog product to an existing tenant product
 * (POST /v1/suppliers/{id}/products/{spid}/link). The link is an FK upsert, so
 * re-linking is idempotent. Pre-seeds the search with the supplier product's EAN
 * (then name) to surface the most likely catalog match first.
 */
export function SupplierProductLinkDialog({
  supplierId,
  supplierProduct,
  onClose,
}: {
  supplierId: string;
  supplierProduct: SupplierProduct | null;
  onClose: () => void;
}) {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const linkProduct = useLinkSupplierProduct(supplierId);

  // Seed from the supplier product's EAN (then name). The parent remounts this
  // component via `key={supplierProduct.id}`, so the lazy initializer re-seeds
  // for each opened product without a setState-in-effect.
  const seed = supplierProduct ? supplierProduct.ean || supplierProduct.name || "" : "";
  const [search, setSearch] = useState(() => seed);
  const [debounced, setDebounced] = useState(() => seed);

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(search), 300);
    return () => clearTimeout(handle);
  }, [search]);

  const { data, isLoading } = useProducts(
    { search: debounced || undefined, limit: 20 },
    { enabled: !!supplierProduct }
  );
  const products = data?.items ?? [];

  const handleLink = (productId: string) => {
    if (!supplierProduct) return;
    linkProduct.mutate(
      { supplierProductId: supplierProduct.id, productId },
      {
        onSuccess: () => {
          toast.success(t("productLinked"));
          onClose();
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  return (
    <Dialog
      open={!!supplierProduct}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("linkToProduct")}</DialogTitle>
          <DialogDescription>
            {t("linkToProductDescription", { name: supplierProduct?.name ?? "" })}
          </DialogDescription>
        </DialogHeader>

        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("productSearchPlaceholder")}
            className="pl-9"
            autoFocus
          />
        </div>

        <div className="max-h-[320px] overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : products.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              {t("noProductsFound")}
            </p>
          ) : (
            <ul className="divide-y">
              {products.map((p) => (
                <li
                  key={p.id}
                  className="flex items-center justify-between gap-3 py-2"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{p.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {p.sku || "---"}
                      {p.ean ? ` · ${p.ean}` : ""} ·{" "}
                      {formatCurrency(p.price)}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={linkProduct.isPending}
                    onClick={() => handleLink(p.id)}
                  >
                    {t("linkAction")}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="flex justify-end">
          <Button variant="ghost" onClick={onClose}>
            {tc("cancel")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
