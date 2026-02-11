"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { RefreshCw, ArrowLeft, Link2 } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import {
  useSupplier,
  useUpdateSupplier,
  useSyncSupplier,
  useSupplierProducts,
} from "@/hooks/use-suppliers";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
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
import { Badge } from "@/components/ui/badge";

const SUPPLIER_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "Aktywny", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" },
  inactive: { label: "Nieaktywny", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200" },
  error: { label: "Bład", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200" },
};

export default function SupplierDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: supplier, isLoading } = useSupplier(id);
  const updateSupplier = useUpdateSupplier(id);
  const syncSupplier = useSyncSupplier();
  const { data: productsData } = useSupplierProducts(id);

  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [feedFormat, setFeedFormat] = useState("iof");
  const [status, setStatus] = useState("active");

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (supplier) {
      setName(supplier.name);
      setCode(supplier.code || "");
      setFeedUrl(supplier.feed_url || "");
      setFeedFormat(supplier.feed_format);
      setStatus(supplier.status);
    }
  }, [supplier]);

  if (authLoading || !isAdmin || isLoading) {
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

  const products = productsData?.items ?? [];

  return (
    <div className="space-y-6">
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
                        <span className="text-muted-foreground text-sm">Brak</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
