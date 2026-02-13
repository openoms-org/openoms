"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { RefreshCw, ArrowLeft, Link2, Search } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useSupplier,
  useUpdateSupplier,
  useSyncSupplier,
  useSupplierProducts,
  useLinkSupplierProduct,
} from "@/hooks/use-suppliers";
import { useProducts } from "@/hooks/use-products";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { SUPPLIER_STATUSES } from "@/lib/constants";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";

export default function SupplierDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const { data: supplier, isLoading } = useSupplier(id);
  const updateSupplier = useUpdateSupplier(id);
  const syncSupplier = useSyncSupplier();
  const { data: productsData } = useSupplierProducts(id);
  const linkProduct = useLinkSupplierProduct(id);

  const [name, setName] = useState("");
  const [linkingProductId, setLinkingProductId] = useState<string | null>(null);
  const [productSearch, setProductSearch] = useState("");
  const [selectedProductId, setSelectedProductId] = useState<string>("");
  const [code, setCode] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [feedFormat, setFeedFormat] = useState("iof");
  const [status, setStatus] = useState("active");

  const { data: localProducts } = useProducts({
    name: productSearch || undefined,
    limit: 20,
  });

  useEffect(() => {
    if (supplier) {
      setName(supplier.name);
      setCode(supplier.code || "");
      setFeedUrl(supplier.feed_url || "");
      setFeedFormat(supplier.feed_format);
      setStatus(supplier.status);
    }
  }, [supplier]);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!supplier) {
    return <div className="text-center py-8 text-muted-foreground">Dostawca nie znaleziony</div>;
  }

  const handleUpdate = () => {
    updateSupplier.mutate(
      { name, code: code || undefined, feed_url: feedUrl || undefined, feed_format: feedFormat, status },
      {
        onSuccess: () => toast.success("Dostawca zaktualizowany"),
        onError: (error) =>
          toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleSync = () => {
    syncSupplier.mutate(id, {
      onSuccess: () => toast.success("Synchronizacja zakończona"),
      onError: (error) =>
        toast.error(getErrorMessage(error)),
    });
  };

  const handleLink = () => {
    if (!linkingProductId || !selectedProductId) return;
    linkProduct.mutate(
      { supplierProductId: linkingProductId, productId: selectedProductId },
      {
        onSuccess: () => {
          toast.success("Produkt powiązany");
          setLinkingProductId(null);
          setSelectedProductId("");
          setProductSearch("");
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  const products = productsData?.items ?? [];

  return (
    <AdminGuard>
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/suppliers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{supplier.name}</h1>
          <p className="text-muted-foreground">
            Format: {supplier.feed_format.toUpperCase()} | Utworzono: {formatDate(supplier.created_at)}
          </p>
        </div>
        <StatusBadge status={supplier.status} statusMap={SUPPLIER_STATUSES} />
        <Button onClick={handleSync} disabled={syncSupplier.isPending}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Synchronizuj
        </Button>
      </div>

      {supplier.error_message && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {supplier.error_message}
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Dane dostawcy</CardTitle>
            <CardDescription>Edytuj informacje o dostawcy</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nazwa</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="code">Kod</Label>
              <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="np. ABC123" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feedUrl">URL feeda</Label>
              <Input id="feedUrl" value={feedUrl} onChange={(e) => setFeedUrl(e.target.value)} placeholder="https://example.com/feed.xml" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Format</Label>
                <Select value={feedFormat} onValueChange={setFeedFormat}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="iof">IOF</SelectItem>
                    <SelectItem value="csv">CSV</SelectItem>
                    <SelectItem value="xml">XML</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Aktywny</SelectItem>
                    <SelectItem value="inactive">Nieaktywny</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <Button onClick={handleUpdate} disabled={updateSupplier.isPending} className="w-full">
              {updateSupplier.isPending ? "Zapisywanie..." : "Zapisz zmiany"}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Informacje</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Ostatnia synchronizacja</span>
              <span>{supplier.last_sync_at ? formatDate(supplier.last_sync_at) : "Nigdy"}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Produkty dostawcy</span>
              <span>{productsData?.total ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Powiązane z OMS</span>
              <span>{products.filter((p) => p.product_id).length}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Utworzono</span>
              <span>{formatDate(supplier.created_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Zaktualizowano</span>
              <span>{formatDate(supplier.updated_at)}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Produkty dostawcy ({productsData?.total ?? 0})</CardTitle>
          <CardDescription>Produkty zaimportowane z feeda dostawcy</CardDescription>
        </CardHeader>
        <CardContent>
          {products.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              Brak produktów. Uruchom synchronizację, aby zaimportować produkty z feeda.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nazwa</TableHead>
                  <TableHead>EAN</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead className="text-right">Cena</TableHead>
                  <TableHead className="text-right">Stan</TableHead>
                  <TableHead>Powiązanie</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {products.map((sp) => (
                  <TableRow key={sp.id}>
                    <TableCell className="font-medium max-w-[250px] truncate">
                      {sp.name}
                    </TableCell>
                    <TableCell>{sp.ean || "---"}</TableCell>
                    <TableCell>{sp.sku || "---"}</TableCell>
                    <TableCell className="text-right">
                      {sp.price != null ? formatCurrency(sp.price) : "---"}
                    </TableCell>
                    <TableCell className="text-right">{sp.stock_quantity}</TableCell>
                    <TableCell>
                      {sp.product_id ? (
                        <Badge variant="outline" className="gap-1">
                          <Link2 className="h-3 w-3" />
                          Powiązany
                        </Badge>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setLinkingProductId(sp.id);
                            setSelectedProductId("");
                            setProductSearch("");
                          }}
                        >
                          <Link2 className="h-3 w-3 mr-1" />
                          Powiąż
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={!!linkingProductId}
        onOpenChange={(open) => {
          if (!open) {
            setLinkingProductId(null);
            setSelectedProductId("");
            setProductSearch("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Powiąż z produktem OMS</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Szukaj produktu</Label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  value={productSearch}
                  onChange={(e) => setProductSearch(e.target.value)}
                  placeholder="Wpisz nazwę produktu..."
                  className="pl-9"
                />
              </div>
            </div>
            <div className="max-h-64 overflow-auto border rounded-md">
              {localProducts?.items && localProducts.items.length > 0 ? (
                localProducts.items.map((product) => (
                  <button
                    key={product.id}
                    type="button"
                    onClick={() => setSelectedProductId(product.id)}
                    className={`w-full text-left px-3 py-2 text-sm hover:bg-muted/50 transition-colors flex items-center justify-between ${
                      selectedProductId === product.id
                        ? "bg-primary/10 border-l-2 border-primary"
                        : ""
                    }`}
                  >
                    <div>
                      <div className="font-medium">{product.name}</div>
                      <div className="text-xs text-muted-foreground">
                        SKU: {product.sku || "---"}
                      </div>
                    </div>
                    {selectedProductId === product.id && (
                      <Badge variant="default" className="ml-2">Wybrany</Badge>
                    )}
                  </button>
                ))
              ) : (
                <p className="text-sm text-muted-foreground text-center py-4">
                  {productSearch ? "Brak wyników" : "Wpisz nazwę, aby wyszukać"}
                </p>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setLinkingProductId(null)}
              >
                Anuluj
              </Button>
              <Button
                onClick={handleLink}
                disabled={!selectedProductId || linkProduct.isPending}
              >
                {linkProduct.isPending ? "Łączenie..." : "Powiąż"}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
    </AdminGuard>
  );
}
