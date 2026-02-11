"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ArrowLeft, Pencil } from "lucide-react";
import { useCustomer, useUpdateCustomer, useDeleteCustomer, useCustomerOrders } from "@/hooks/use-customers";
import { usePriceLists } from "@/hooks/use-price-lists";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import { StatusBadge } from "@/components/shared/status-badge";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
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

export default function CustomerDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { data: customer, isLoading } = useCustomer(params.id);
  const updateCustomer = useUpdateCustomer(params.id);
  const deleteCustomer = useDeleteCustomer();
  const { data: ordersData } = useCustomerOrders(params.id);

  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;
  const { data: priceListsData } = usePriceLists({ limit: 100, active: true });
  const priceLists = priceListsData?.items ?? [];

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
      toast.error("Imię i nazwisko jest wymagane");
      return;
    }
    try {
      await updateCustomer.mutateAsync(formData);
      toast.success("Dane klienta zostaly zaktualizowane");
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteCustomer.mutateAsync(params.id);
      toast.success("Klient zostal usuniety");
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
        <p className="text-muted-foreground">Nie znaleziono klienta</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/customers")}>
          Wroc do listy
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.push("/customers")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{customer.name}</h1>
          <p className="text-muted-foreground mt-1">
            Klient od {formatDate(customer.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={startEditing}>
            <Pencil className="mr-2 h-4 w-4" />
            Edytuj
          </Button>
          <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
            Usun
          </Button>
        </div>
      </div>

      {isEditing && (
        <Card>
          <CardHeader>
            <CardTitle>Edycja danych klienta</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4 max-w-xl">
              <div className="space-y-2">
                <Label htmlFor="edit-name">Imie i nazwisko *</Label>
                <Input
                  id="edit-name"
                  value={formData.name || ""}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-email">E-mail</Label>
                  <Input
                    id="edit-email"
                    type="email"
                    value={formData.email || ""}
                    onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-phone">Telefon</Label>
                  <Input
                    id="edit-phone"
                    value={formData.phone || ""}
                    onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="edit-company">Firma</Label>
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
                <Label htmlFor="edit-notes">Notatki</Label>
                <Textarea
                  id="edit-notes"
                  value={formData.notes || ""}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  rows={3}
                />
              </div>
              <div className="space-y-2">
                <Label>Cennik</Label>
                <Select
                  value={formData.price_list_id || "none"}
                  onValueChange={(val) =>
                    setFormData({ ...formData, price_list_id: val === "none" ? undefined : val })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Brak cennika" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Brak cennika</SelectItem>
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
                  {updateCustomer.isPending ? "Zapisywanie..." : "Zapisz"}
                </Button>
                <Button variant="outline" onClick={() => setIsEditing(false)}>
                  Anuluj
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
              <CardTitle>Dane klienta</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Imie i nazwisko</p>
                  <p className="mt-1 font-medium">{customer.name}</p>
                </div>
                {customer.email && (
                  <div>
                    <p className="text-sm text-muted-foreground">E-mail</p>
                    <p className="mt-1 text-sm">{customer.email}</p>
                  </div>
                )}
                {customer.phone && (
                  <div>
                    <p className="text-sm text-muted-foreground">Telefon</p>
                    <p className="mt-1 text-sm">{customer.phone}</p>
                  </div>
                )}
                {customer.company_name && (
                  <div>
                    <p className="text-sm text-muted-foreground">Firma</p>
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
                    <p className="text-sm text-muted-foreground">Cennik</p>
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
                    <p className="text-sm text-muted-foreground">Tagi</p>
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
                    <p className="text-sm text-muted-foreground">Notatki</p>
                    <p className="mt-1 text-sm whitespace-pre-wrap">{customer.notes}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Historia zamowien</CardTitle>
            </CardHeader>
            <CardContent>
              {ordersData?.items && ordersData.items.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Zrodlo</TableHead>
                      <TableHead className="text-right">Kwota</TableHead>
                      <TableHead>Data</TableHead>
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
                <p className="text-sm text-muted-foreground">
                  Brak zamowien dla tego klienta.
                </p>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Podsumowanie</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm text-muted-foreground">Zamowien</p>
                <p className="mt-1 text-2xl font-bold">{customer.total_orders}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Wydano lacznie</p>
                <p className="mt-1 text-2xl font-bold">
                  {formatCurrency(customer.total_spent)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Klient od</p>
                <p className="mt-1 font-medium">{formatDate(customer.created_at)}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Usuwanie klienta"
        description="Czy na pewno chcesz usunac tego klienta? Ta operacja jest nieodwracalna."
        confirmLabel="Usun klienta"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteCustomer.isPending}
      />
    </div>
  );
}
