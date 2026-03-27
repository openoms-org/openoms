"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  usePriceList,
  useUpdatePriceList,
  usePriceListItems,
  useCreatePriceListItem,
  useDeletePriceListItem,
} from "@/hooks/use-price-lists";
import { useProducts } from "@/hooks/use-products";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTranslations } from "next-intl";

export default function PriceListDetailPage() {
  const t = useTranslations("settings");
  const tp = useTranslations("settings.priceLists");
  const tc = useTranslations("common");
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const discountTypeLabels: Record<string, string> = {
    percentage: tp("percentage"),
    fixed: tp("fixed"),
    override: tp("override"),
  };

  const { data: priceList, isLoading } = usePriceList(id);
  const updatePriceList = useUpdatePriceList(id);
  const { data: itemsData, isLoading: itemsLoading } = usePriceListItems(id, {
    limit: 100,
  });
  const createItem = useCreatePriceListItem(id);
  const deleteItem = useDeletePriceListItem(id);
  const { data: productsData } = useProducts({ limit: 100 });

  const [showAddProduct, setShowAddProduct] = useState(false);
  const [selectedProductId, setSelectedProductId] = useState("");
  const [newDiscount, setNewDiscount] = useState("");
  const [newPrice, setNewPrice] = useState("");
  const [newMinQty, setNewMinQty] = useState("1");
  const [deleteItemId, setDeleteItemId] = useState<string | null>(null);

  const [editName, setEditName] = useState<string | null>(null);
  const [editDescription, setEditDescription] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (!priceList) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">{tp("notFound")}</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => router.push("/settings/price-lists")}
        >
          {tp("backToList")}
        </Button>
      </div>
    );
  }

  const items = itemsData?.items ?? [];
  const products = productsData?.items ?? [];

  const handleToggleActive = () => {
    updatePriceList.mutate(
      { active: !priceList.active },
      {
        onSuccess: () => {
          toast.success(
            priceList.active ? tp("deactivated") : tp("activated")
          );
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleSaveName = () => {
    if (editName === null || editName === priceList.name) {
      setEditName(null);
      return;
    }
    updatePriceList.mutate(
      { name: editName },
      {
        onSuccess: () => {
          toast.success(tp("nameUpdated"));
          setEditName(null);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleSaveDescription = () => {
    if (editDescription === null) return;
    updatePriceList.mutate(
      { description: editDescription },
      {
        onSuccess: () => {
          toast.success(tp("descriptionUpdated"));
          setEditDescription(null);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
      }
    );
  };

  const handleAddProduct = () => {
    if (!selectedProductId) return;
    const data: {
      product_id: string;
      discount?: number;
      price?: number;
      min_quantity?: number;
    } = {
      product_id: selectedProductId,
    };
    if (newDiscount) data.discount = parseFloat(newDiscount);
    if (newPrice) data.price = parseFloat(newPrice);
    if (newMinQty) data.min_quantity = parseInt(newMinQty, 10);

    createItem.mutate(data, {
      onSuccess: () => {
        toast.success(tp("productAdded"));
        setShowAddProduct(false);
        setSelectedProductId("");
        setNewDiscount("");
        setNewPrice("");
        setNewMinQty("1");
      },
      onError: (err) => toast.error(getErrorMessage(err)),
    });
  };

  const handleDeleteItem = () => {
    if (!deleteItemId) return;
    deleteItem.mutate(deleteItemId, {
      onSuccess: () => {
        toast.success(t("pozycjaUsunietaZCennika"));
        setDeleteItemId(null);
      },
      onError: (err) => toast.error(getErrorMessage(err)),
    });
  };

  const getProductName = (productId: string) => {
    const product = products.find((p) => p.id === productId);
    return product?.name ?? productId.slice(0, 8);
  };

  return (
    <>
      <div className="flex items-center gap-4 mb-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.push("/settings/price-lists")}
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          {t("powrot")}
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t("priceListDetails")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>{tc("name")}</Label>
              {editName !== null ? (
                <div className="flex gap-2">
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                  />
                  <Button size="sm" onClick={handleSaveName}>
                    {tp("saveButton")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditName(null)}
                  >
                    {tp("cancelButton")}
                  </Button>
                </div>
              ) : (
                <p
                  className="text-sm cursor-pointer hover:text-primary"
                  onClick={() => setEditName(priceList.name)}
                >
                  {priceList.name}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label>{tc("description")}</Label>
              {editDescription !== null ? (
                <div className="flex gap-2">
                  <Input
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                  />
                  <Button size="sm" onClick={handleSaveDescription}>
                    {tp("saveButton")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditDescription(null)}
                  >
                    {tp("cancelButton")}
                  </Button>
                </div>
              ) : (
                <p
                  className="text-sm cursor-pointer hover:text-primary"
                  onClick={() =>
                    setEditDescription(priceList.description ?? "")
                  }
                >
                  {priceList.description || "---"}
                </p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>{tp("discountType")}</Label>
                <p className="text-sm mt-1">
                  {discountTypeLabels[priceList.discount_type] ??
                    priceList.discount_type}
                </p>
              </div>
              <div>
                <Label>{tp("currency")}</Label>
                <p className="text-sm mt-1">{priceList.currency}</p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>{tc("createdAt")}</Label>
                <p className="text-sm mt-1">
                  {formatDate(priceList.created_at)}
                </p>
              </div>
              <div>
                <Label>{tc("updatedAt")}</Label>
                <p className="text-sm mt-1">
                  {formatDate(priceList.updated_at)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{tc("status")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <Label>{tc("active")}</Label>
              <Switch
                checked={priceList.active}
                onCheckedChange={handleToggleActive}
              />
            </div>
            {priceList.is_default && (
              <Badge variant="default">{t("domyslnyCennik")}</Badge>
            )}
            {priceList.valid_from && (
              <div>
                <Label>{t("waznyOd")}</Label>
                <p className="text-sm">{formatDate(priceList.valid_from)}</p>
              </div>
            )}
            {priceList.valid_to && (
              <div>
                <Label>{t("waznyDo")}</Label>
                <p className="text-sm">{formatDate(priceList.valid_to)}</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Items */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{tp("items")}</CardTitle>
          <Dialog open={showAddProduct} onOpenChange={setShowAddProduct}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-2" />
                {tp("addProduct")}
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{tp("addProductDialog")}</DialogTitle>
                <DialogDescription>
                  {t("wybierzProduktIUstawRabatLubCene")}
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>{tp("productLabel")}</Label>
                  <Select
                    value={selectedProductId}
                    onValueChange={setSelectedProductId}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder={tp("selectProduct")} />
                    </SelectTrigger>
                    <SelectContent>
                      {products.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.name} {p.sku ? `(${p.sku})` : ""}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {priceList.discount_type === "override" ? (
                  <div className="space-y-2">
                    <Label>{tp("priceLabel", { currency: priceList.currency })}</Label>
                    <Input
                      type="number"
                      step="0.01"
                      value={newPrice}
                      onChange={(e) => setNewPrice(e.target.value)}
                      placeholder="0.00"
                    />
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Label>
                      {tp("discountLabel", {
                        unit: priceList.discount_type === "percentage"
                          ? "%"
                          : priceList.currency,
                      })}
                    </Label>
                    <Input
                      type="number"
                      step="0.01"
                      value={newDiscount}
                      onChange={(e) => setNewDiscount(e.target.value)}
                      placeholder="0"
                    />
                  </div>
                )}

                <div className="space-y-2">
                  <Label>{t("minimalnaIlosc")}</Label>
                  <Input
                    type="number"
                    min="1"
                    value={newMinQty}
                    onChange={(e) => setNewMinQty(e.target.value)}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setShowAddProduct(false)}
                >
                  {tp("cancelButton")}
                </Button>
                <Button
                  onClick={handleAddProduct}
                  disabled={!selectedProductId || createItem.isPending}
                >
                  {createItem.isPending ? tp("addingButton") : tp("addButton")}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent>
          {itemsLoading ? (
            <LoadingSkeleton />
          ) : items.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">
              {t("brakPozycjiWCennikuDodajProduktyAby")}
            </p>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{tp("columns.product")}</TableHead>
                    <TableHead>
                      {priceList.discount_type === "override"
                        ? tp("columns.priceOrDiscount")
                        : tp("columns.discount")}
                    </TableHead>
                    <TableHead>{t("minIlosc")}</TableHead>
                    <TableHead>{tp("columns.addedAt")}</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium">
                        {getProductName(item.product_id)}
                      </TableCell>
                      <TableCell>
                        {priceList.discount_type === "override"
                          ? item.price != null
                            ? `${item.price.toFixed(2)} ${priceList.currency}`
                            : "---"
                          : item.discount != null
                          ? priceList.discount_type === "percentage"
                            ? `${item.discount}%`
                            : `${item.discount.toFixed(2)} ${priceList.currency}`
                          : "---"}
                      </TableCell>
                      <TableCell>{item.min_quantity}</TableCell>
                      <TableCell>{formatDate(item.created_at)}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => setDeleteItemId(item.id)}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!deleteItemId}
        onOpenChange={(open) => !open && setDeleteItemId(null)}
        title={t("usunPozycje")}
        description={t("czyNaPewnoChceszUsunacTePozycjeZCennika")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDeleteItem}
        isLoading={deleteItem.isPending}
      />
    </>
  );
}
