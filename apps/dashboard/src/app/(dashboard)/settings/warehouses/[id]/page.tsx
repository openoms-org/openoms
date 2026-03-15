"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Plus } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import {
  useWarehouse,
  useUpdateWarehouse,
  useWarehouseStock,
  useUpsertWarehouseStock,
} from "@/hooks/use-warehouses";
import { useProducts } from "@/hooks/use-products";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
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
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTranslations } from "next-intl";

export default function WarehouseDetailPage() {
  const t = useTranslations("warehouses");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const { isAdmin, isLoading: authLoading } = useAuth();
  const { data: warehouse, isLoading } = useWarehouse(id);
  const updateWarehouse = useUpdateWarehouse(id);
  const { data: stockData, isLoading: stockLoading } = useWarehouseStock(id, {
    limit: 100,
  });
  const upsertStock = useUpsertWarehouseStock(id);
  const { data: productsData } = useProducts({ limit: 100 });

  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [active, setActive] = useState(true);

  const [showAddStock, setShowAddStock] = useState(false);
  const [selectedProductId, setSelectedProductId] = useState("");
  const [stockQuantity, setStockQuantity] = useState("0");
  const [stockReserved, setStockReserved] = useState("0");
  const [stockMinStock, setStockMinStock] = useState("0");

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  useEffect(() => {
    if (warehouse) {
      setName(warehouse.name);
      setCode(warehouse.code || "");
      setIsDefault(warehouse.is_default);
      setActive(warehouse.active);
    }
  }, [warehouse]);

  if (authLoading || !isAdmin || isLoading) {
    return <LoadingSkeleton />;
  }

  if (!warehouse) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        {t("notFound")}
      </div>
    );
  }

  const handleUpdate = () => {
    updateWarehouse.mutate(
      {
        name,
        code: code || undefined,
        is_default: isDefault,
        active,
      },
      {
        onSuccess: () => toast.success(t("updated")),
        onError: (error) =>
          toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleAddStock = () => {
    if (!selectedProductId) return;
    upsertStock.mutate(
      {
        product_id: selectedProductId,
        quantity: parseInt(stockQuantity) || 0,
        reserved: parseInt(stockReserved) || 0,
        min_stock: parseInt(stockMinStock) || 0,
      },
      {
        onSuccess: () => {
          toast.success(t("stockUpdated"));
          setShowAddStock(false);
          setSelectedProductId("");
          setStockQuantity("0");
          setStockReserved("0");
          setStockMinStock("0");
        },
        onError: (error) =>
          toast.error(getErrorMessage(error)),
      }
    );
  };

  const stocks = stockData?.items ?? [];
  const products = productsData?.items ?? [];

  const getProductName = (productId: string) => {
    const product = products.find((p) => p.id === productId);
    return product?.name ?? productId;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push("/settings/warehouses")}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">
            {warehouse.name}
          </h1>
          <p className="text-muted-foreground">
            {warehouse.code ? `${t("codeLabel")} ${warehouse.code} | ` : ""}{t("createdLabel")}{" "}
            {formatDate(warehouse.created_at)}
          </p>
        </div>
        {warehouse.is_default && <Badge variant="default">{t("columns.default")}</Badge>}
        {warehouse.active ? (
          <Badge
            variant="outline"
            className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
          >
            {t("columns.active")}
          </Badge>
        ) : (
          <Badge
            variant="outline"
            className="bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
          >
            {t("inactive")}
          </Badge>
        )}
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("warehouseData")}</CardTitle>
            <CardDescription>{t("editWarehouseDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t("columns.name")}</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="code">{t("columns.code")}</Label>
              <Input
                id="code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder={t("codePlaceholder")}
              />
            </div>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Switch
                  checked={isDefault}
                  onCheckedChange={setIsDefault}
                  id="is-default"
                />
                <Label htmlFor="is-default">{t("columns.default")}</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  checked={active}
                  onCheckedChange={setActive}
                  id="active"
                />
                <Label htmlFor="active">{t("columns.active")}</Label>
              </div>
            </div>
            <Button
              onClick={handleUpdate}
              disabled={updateWarehouse.isPending}
              className="w-full"
            >
              {updateWarehouse.isPending ? t("saving") : t("saveChanges")}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("info")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("stockItems")}</span>
              <span>{stockData?.total ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("columns.createdAt")}</span>
              <span>{formatDate(warehouse.created_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("updatedLabel")}</span>
              <span>{formatDate(warehouse.updated_at)}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>{t("stockTitle", { count: stockData?.total ?? 0 })}</CardTitle>
            <CardDescription>
              {t("stanyMagazynoweProduktowWTymMagazynie")}
            </CardDescription>
          </div>
          <Dialog open={showAddStock} onOpenChange={setShowAddStock}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-2" />
                {t("addProduct")}
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{t("addStockTitle")}</DialogTitle>
                <DialogDescription>
                  {t("addStockDescription")}
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>{t("product")}</Label>
                  <Select
                    value={selectedProductId}
                    onValueChange={setSelectedProductId}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder={t("selectProduct")} />
                    </SelectTrigger>
                    <SelectContent>
                      {products.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.name}
                          {p.sku ? ` (${p.sku})` : ""}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="stock-qty">{t("quantity")}</Label>
                    <Input
                      id="stock-qty"
                      type="number"
                      min="0"
                      value={stockQuantity}
                      onChange={(e) => setStockQuantity(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="stock-reserved">{t("reserved")}</Label>
                    <Input
                      id="stock-reserved"
                      type="number"
                      min="0"
                      value={stockReserved}
                      onChange={(e) => setStockReserved(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="stock-min">{t("minStock")}</Label>
                    <Input
                      id="stock-min"
                      type="number"
                      min="0"
                      value={stockMinStock}
                      onChange={(e) => setStockMinStock(e.target.value)}
                    />
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setShowAddStock(false)}
                >
                  {t("cancel")}
                </Button>
                <Button
                  onClick={handleAddStock}
                  disabled={!selectedProductId || upsertStock.isPending}
                >
                  {upsertStock.isPending ? t("saving") : t("save")}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent>
          {stockLoading ? (
            <LoadingSkeleton />
          ) : stocks.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              {t("brakStanowMagazynowychDodajProduktAbyRozpoczac")}
              {t("zarzadzanieStanem")}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("product")}</TableHead>
                  <TableHead className="text-right">{t("quantity")}</TableHead>
                  <TableHead className="text-right">{t("reserved")}</TableHead>
                  <TableHead className="text-right">{t("dostepne")}</TableHead>
                  <TableHead className="text-right">{t("minStock")}</TableHead>
                  <TableHead>{t("status")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stocks.map((stock) => {
                  const available = stock.quantity - stock.reserved;
                  const isLow = available <= stock.min_stock && stock.min_stock > 0;
                  return (
                    <TableRow key={stock.id}>
                      <TableCell className="font-medium max-w-[250px] truncate">
                        {getProductName(stock.product_id)}
                      </TableCell>
                      <TableCell className="text-right">
                        {stock.quantity}
                      </TableCell>
                      <TableCell className="text-right">
                        {stock.reserved}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {available}
                      </TableCell>
                      <TableCell className="text-right">
                        {stock.min_stock}
                      </TableCell>
                      <TableCell>
                        {isLow ? (
                          <Badge
                            variant="outline"
                            className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                          >
                            {t("lowStock")}
                          </Badge>
                        ) : (
                          <Badge
                            variant="outline"
                            className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                          >
                            OK
                          </Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
