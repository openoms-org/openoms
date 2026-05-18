"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ArrowLeft, Pencil, ShoppingBag, Users, Award } from "lucide-react";
import { useCustomer, useUpdateCustomer, useDeleteCustomer, useCustomerOrders } from "@/hooks/use-customers";
import { useCustomerSegments } from "@/hooks/use-segments";
import { useCustomerLoyaltyStatus } from "@/hooks/use-loyalty";
import { useAllPriceLists } from "@/hooks/use-price-lists";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ORDER_STATUSES } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import {
  Card,
  CardContent,
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatDate, formatCurrency, shortId } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { UpdateCustomerRequest } from "@/types/api";
import { useTranslations } from "next-intl";

export default function CustomerDetailPage() {
  const t = useTranslations("customers");
  const tc = useTranslations("common");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { data: customer, isLoading } = useCustomer(params.id);
  const updateCustomer = useUpdateCustomer(params.id);
  const deleteCustomer = useDeleteCustomer();
  const { data: ordersData, isLoading: isLoadingOrders } = useCustomerOrders(params.id);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const { data: priceListsData } = useAllPriceLists({ active: true });
  const priceLists = priceListsData?.items ?? [];
  const { data: customerSegments } = useCustomerSegments(params.id);
  const { data: loyaltyStatus } = useCustomerLoyaltyStatus(params.id);

  const [formData, setFormData] = useState<UpdateCustomerRequest>({});

  const startEditing = () => {
    if (!customer) return;
    setFormData({
      name: customer.name,
      email: customer.email || "",
      phone: customer.phone || "",
      company_name: customer.company_name || "",
      nip: customer.nip || "",
      notes: customer.notes || "",
      price_list_id: customer.price_list_id || undefined,
    });
    setIsEditing(true);
  };

  const handleUpdate = async () => {
    if (!formData.name?.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }
    try {
      await updateCustomer.mutateAsync(formData);
      toast.success(t("customerDataUpdated"));
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteCustomer.mutateAsync(params.id);
      toast.success(t("customerDeleted"));
      router.push("/customers");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (!customer) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">{t("notFound")}</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/customers")}>
          {t("detail.backToList")}
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/customers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{customer.name}</h1>
          <p className="text-muted-foreground mt-1">
            {t("customerSince", { date: formatDate(customer.created_at) })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={startEditing}>
            <Pencil className="mr-2 h-4 w-4" />
            {tc("edit")}
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            {t("delete")}
          </Button>
        </div>
      </div>

      {isEditing && (
        <Card>
          <CardHeader>
            <CardTitle>{t("editCustomerData")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4 max-w-xl">
              <div className="space-y-2">
                <Label htmlFor="edit-name">{t("imieINazwisko")}</Label>
                <Input
                  id="edit-name"
                  value={formData.name || ""}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-email">{t("form.email")}</Label>
                  <Input
                    id="edit-email"
                    type="email"
                    value={formData.email || ""}
                    onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-phone">{t("form.phone")}</Label>
                  <Input
                    id="edit-phone"
                    value={formData.phone || ""}
                    onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-company">{t("form.company")}</Label>
                  <Input
                    id="edit-company"
                    value={formData.company_name || ""}
                    onChange={(e) => setFormData({ ...formData, company_name: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-nip">NIP</Label>
                  <Input
                    id="edit-nip"
                    value={formData.nip || ""}
                    onChange={(e) => setFormData({ ...formData, nip: e.target.value })}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-notes">{t("form.notes")}</Label>
                <Textarea
                  id="edit-notes"
                  value={formData.notes || ""}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  rows={3}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("priceList")}</Label>
                <Select
                  value={formData.price_list_id || "none"}
                  onValueChange={(val) =>
                    setFormData({ ...formData, price_list_id: val === "none" ? undefined : val })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t("noPriceList")} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">{t("noPriceList")}</SelectItem>
                    {priceLists.map((pl) => (
                      <SelectItem key={pl.id} value={pl.id}>
                        {pl.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex gap-2 pt-2">
                <Button onClick={handleUpdate} disabled={updateCustomer.isPending}>
                  {updateCustomer.isPending ? tc("saving") : tc("save")}
                </Button>
                <Button variant="outline" onClick={() => setIsEditing(false)}>
                  {tc("cancel")}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{t("customerData")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">{t("form.fullName")}</p>
                  <p className="mt-1 font-medium">{customer.name}</p>
                </div>
                {customer.email && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("form.email")}</p>
                    <p className="mt-1 text-sm">{customer.email}</p>
                  </div>
                )}
                {customer.phone && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("form.phone")}</p>
                    <p className="mt-1 text-sm">{customer.phone}</p>
                  </div>
                )}
                {customer.company_name && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("form.company")}</p>
                    <p className="mt-1 font-medium">{customer.company_name}</p>
                  </div>
                )}
                {customer.nip && (
                  <div>
                    <p className="text-sm text-muted-foreground">NIP</p>
                    <p className="mt-1 font-mono text-sm">{customer.nip}</p>
                  </div>
                )}
                {customer.price_list_id && (
                  <div>
                    <p className="text-sm text-muted-foreground">{t("priceList")}</p>
                    <p className="mt-1 text-sm">
                      {priceLists.find((pl) => pl.id === customer.price_list_id)?.name ?? customer.price_list_id.slice(0, 8)}
                    </p>
                  </div>
                )}
              </div>

              {customer.tags && customer.tags.length > 0 && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">{tc("tags")}</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {customer.tags.map((tag) => (
                        <span
                          key={tag}
                          className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                </>
              )}

              {customer.notes && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">{t("form.notes")}</p>
                    <p className="mt-1 text-sm whitespace-pre-wrap">{customer.notes}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("historiaZamowien")}</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoadingOrders ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                </div>
              ) : ordersData?.items && ordersData.items.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>{t("columns.source")}</TableHead>
                      <TableHead className="text-right">{tc("amount")}</TableHead>
                      <TableHead>{tc("date")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {ordersData.items.map((order) => (
                      <TableRow key={order.id}>
                        <TableCell>
                          <Link
                            href={`/orders/${order.id}`}
                            className="font-medium text-primary hover:underline"
                          >
                            {shortId(order.id)}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={order.status} statusMap={orderStatuses} />
                        </TableCell>
                        <TableCell>{order.source}</TableCell>
                        <TableCell className="text-right">
                          {formatCurrency(order.total_amount, order.currency)}
                        </TableCell>
                        <TableCell>{formatDate(order.created_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <ShoppingBag className="h-8 w-8 text-muted-foreground/50 mb-2" />
                  <p className="text-sm text-muted-foreground">{t("brakZamowienDlaTegoKlienta")}</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{t("summary")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm text-muted-foreground">{t("zamowien")}</p>
                <p className="mt-1 text-2xl font-bold">{customer.total_orders}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("totalSpent")}</p>
                <p className="mt-1 text-2xl font-bold">
                  {formatCurrency(customer.total_spent)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{t("customerSinceLabel")}</p>
                <p className="mt-1 font-medium">{formatDate(customer.created_at)}</p>
              </div>
            </CardContent>
          </Card>

          {customerSegments && customerSegments.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Users className="h-4 w-4" />
                  {t("segments")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-1.5">
                  {customerSegments.map((seg) => (
                    <Link
                      key={seg.id}
                      href={`/customers/segments/${seg.id}`}
                      className="rounded-full px-2.5 py-0.5 text-xs font-medium hover:opacity-80 transition-opacity"
                      style={{
                        backgroundColor: seg.color + "20",
                        color: seg.color,
                      }}
                    >
                      {seg.name}
                    </Link>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {loyaltyStatus && loyaltyStatus.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Award className="h-4 w-4" />
                  {t("programyLojalnosciowe")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {loyaltyStatus.map((ls) => (
                  <Link
                    key={ls.program_id}
                    href={`/loyalty/${ls.program_id}`}
                    className="block rounded-md border p-2 hover:bg-muted/50 transition-colors"
                  >
                    <p className="text-sm font-medium">{ls.program_name}</p>
                    <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                      {ls.program_type === "points" && (
                        <span>{ls.points_balance.toLocaleString("pl-PL")} pkt</span>
                      )}
                      {ls.current_tier && (
                        <span className="rounded-full bg-primary/10 px-1.5 py-0.5 text-primary font-medium">
                          {ls.current_tier}
                        </span>
                      )}
                      <span>{ls.order_count} {t("zamowien")}</span>
                    </div>
                  </Link>
                ))}
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteCustomerTitle")}
        description={t("czyNaPewnoChceszUsunacTegoKlientaTaOperacjaJestNie")}
        confirmLabel={t("usunKlienta")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteCustomer.isPending}
      />
    </div>
  );
}
