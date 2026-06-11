"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAllSuppliers } from "@/hooks/use-suppliers";
import { useCreateDropshipOrder } from "@/hooks/use-dropship-orders";
import { getErrorMessage } from "@/lib/api-client";
import type { Order } from "@/types/api";

interface DropshipItemRow {
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_cost: number;
}

/**
 * Manual counterpart to useAutoRouteDropship: forwards an order's items to a
 * chosen supplier (POST /v1/dropship-orders). Mounted only while open, so the
 * lazy initializer seeds the editable item rows from the order each time.
 */
export function OrderDropshipManualDialog({
  order,
  onClose,
}: {
  order: Order;
  onClose: () => void;
}) {
  const tf = useTranslations("feHooks");
  const tc = useTranslations("common");
  const { data: suppliersData } = useAllSuppliers();
  const createDropship = useCreateDropshipOrder();
  const suppliers = suppliersData?.items ?? [];

  const [supplierId, setSupplierId] = useState("");
  const [items, setItems] = useState<DropshipItemRow[]>(() =>
    (order.items ?? []).map((it) => ({
      product_id: it.product_id,
      sku: it.sku ?? "",
      product_name: it.name,
      quantity: it.quantity,
      unit_cost: it.price,
    }))
  );

  const updateItem = (index: number, field: "quantity" | "unit_cost", value: number) => {
    setItems((rows) =>
      rows.map((row, i) => (i === index ? { ...row, [field]: value } : row))
    );
  };

  const handleSubmit = () => {
    if (!supplierId) {
      toast.error(tf("dropshipNoSupplier"));
      return;
    }
    if (items.length === 0) {
      toast.error(tf("dropshipNoItems"));
      return;
    }
    createDropship.mutate(
      {
        order_id: order.id,
        supplier_id: supplierId,
        currency: order.currency,
        items,
      },
      {
        onSuccess: () => {
          toast.success(tf("dropshipCreated"));
          onClose();
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  return (
    <Dialog
      open
      onOpenChange={(o) => {
        if (!o) onClose();
      }}
    >
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{tf("dropshipManualTitle")}</DialogTitle>
          <DialogDescription>{tf("dropshipManualDescription")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label>{tf("dropshipSupplier")}</Label>
            <Select value={supplierId} onValueChange={setSupplierId}>
              <SelectTrigger>
                <SelectValue placeholder={tf("dropshipSelectSupplier")} />
              </SelectTrigger>
              <SelectContent>
                {suppliers.map((s) => (
                  <SelectItem key={s.id} value={s.id}>
                    {s.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{tc("name")}</TableHead>
                <TableHead className="w-[90px]">{tc("quantity")}</TableHead>
                <TableHead className="w-[120px] text-right">
                  {tf("dropshipUnitCost")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item, i) => (
                <TableRow key={`${item.sku}-${i}`}>
                  <TableCell className="font-medium">{item.product_name}</TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      min={1}
                      value={item.quantity}
                      onChange={(e) =>
                        updateItem(i, "quantity", parseInt(e.target.value) || 1)
                      }
                      className="h-8"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      min={0}
                      step="0.01"
                      value={item.unit_cost}
                      onChange={(e) =>
                        updateItem(i, "unit_cost", parseFloat(e.target.value) || 0)
                      }
                      className="h-8 text-right"
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {tc("cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={createDropship.isPending}>
            {createDropship.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            {tf("dropshipCreate")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
