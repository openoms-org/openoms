"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { BadgePercent, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  usePriceLists,
  useDeletePriceList,
  useCreatePriceList,
} from "@/hooks/use-price-lists";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { QueryError } from "@/components/shared/query-error";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
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

export default function PriceListsPage() {
  const t = useTranslations("settings");
  const tp = useTranslations("settings.priceLists");
  const tc = useTranslations("common");
  const router = useRouter();
  const { data, isLoading, isError, refetch } = usePriceLists();
  const deletePriceList = useDeletePriceList();
  const createPriceList = useCreatePriceList();

  const discountTypeLabels: Record<string, string> = {
    percentage: tp("percentage"),
    fixed: tp("fixed"),
    override: tp("override"),
  };

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [newDiscountType, setNewDiscountType] = useState("percentage");
  const [newCurrency, setNewCurrency] = useState("PLN");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const priceLists = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deletePriceList.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("priceListDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    if (!newName.trim()) return;
    createPriceList.mutate(
      {
        name: newName,
        description: newDescription || undefined,
        discount_type: newDiscountType as "percentage" | "fixed" | "override",
        currency: newCurrency,
      },
      {
        onSuccess: () => {
          toast.success(t("priceListCreated"));
          setShowCreate(false);
          setNewName("");
          setNewDescription("");
          setNewDiscountType("percentage");
          setNewCurrency("PLN");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{tp("title")}</h1>
          <p className="text-muted-foreground">
            {t("zarzadzajCennikamiIRabatamiDlaKlientowBiznesowych")}
          </p>
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              {tp("newPriceList")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{tp("newPriceListDialog")}</DialogTitle>
              <DialogDescription>
                {t("utworzNowyCennikZRabatamiDlaKlientow")}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-name">{tp("nameLabel")}</Label>
                <Input
                  id="new-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder={tp("namePlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-desc">{tp("descriptionLabel")}</Label>
                <Input
                  id="new-desc"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder={tp("descriptionPlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label>{tp("discountType")}</Label>
                <Select
                  value={newDiscountType}
                  onValueChange={setNewDiscountType}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="percentage">{tp("percentage")}</SelectItem>
                    <SelectItem value="fixed">{tp("fixed")}</SelectItem>
                    <SelectItem value="override">{tp("override")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-currency">{tp("currency")}</Label>
                <Input
                  id="new-currency"
                  value={newCurrency}
                  onChange={(e) => setNewCurrency(e.target.value)}
                  placeholder="PLN"
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowCreate(false)}
              >
                {tp("cancelButton")}
              </Button>
              <Button
                onClick={handleCreate}
                disabled={!newName.trim() || createPriceList.isPending}
              >
                {createPriceList.isPending ? tp("creating") : tc("create")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isError && (
        <QueryError onRetry={() => refetch()} />
      )}

      {priceLists.length === 0 ? (
        <EmptyState
          icon={BadgePercent}
          title={t("brakCennikow")}
          description={t("utworzPierwszyCennikAbyOferowacIndywidualneCenyDla")}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{tp("columns.name")}</TableHead>
                <TableHead>{tp("columns.discountType")}</TableHead>
                <TableHead>{tp("columns.currency")}</TableHead>
                <TableHead>{tc("default")}</TableHead>
                <TableHead>{tp("columns.active")}</TableHead>
                <TableHead>{tp("columns.createdAt")}</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {priceLists.map((pl) => (
                <TableRow
                  key={pl.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() =>
                    router.push(`/settings/price-lists/${pl.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    <div>
                      <p>{pl.name}</p>
                      {pl.description && (
                        <p className="text-xs text-muted-foreground">
                          {pl.description}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {discountTypeLabels[pl.discount_type] || pl.discount_type}
                  </TableCell>
                  <TableCell>{pl.currency}</TableCell>
                  <TableCell>
                    {pl.is_default ? (
                      <Badge variant="default">{tp("yes")}</Badge>
                    ) : (
                      <span className="text-muted-foreground">{tp("no")}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {pl.active ? (
                      <Badge
                        variant="outline"
                        className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      >
                        {tp("active")}
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                      >
                        {tp("inactive")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{formatDate(pl.created_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(pl.id);
                      }}
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

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("usunCennik")}
        description={t("czyNaPewnoChceszUsunacTenCennikWszystkiePrzypisane")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deletePriceList.isPending}
      />
    </AdminGuard>
  );
}
