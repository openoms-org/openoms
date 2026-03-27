"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Plus, Trash2, Users } from "lucide-react";
import { toast } from "sonner";
import {
  useSegment,
  useUpdateSegment,
  useSegmentMembers,
  useAddSegmentMember,
  useRemoveSegmentMember,
  useDeleteSegment,
} from "@/hooks/use-segments";
import { useCustomers } from "@/hooks/use-customers";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, formatCurrency } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTranslations } from "next-intl";

// SEGMENT_TYPE_LABELS moved inside component to use translations

export default function SegmentDetailPage() {
  const t = useTranslations("customers");
  const tc = useTranslations("common");
  const SEGMENT_TYPE_LABELS: Record<string, string> = {
    manual: t("segmentTypeManual"),
    rfm_auto: t("segmentTypeRfm"),
    rule_based: t("segmentTypeRuleBased"),
  };
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [memberPagination, setMemberPagination] = useState({
    limit: 20,
    offset: 0,
  });

  const { data: segment, isLoading } = useSegment(params.id);
  const updateSegment = useUpdateSegment(params.id);
  const deleteSegment = useDeleteSegment();
  const { data: membersData, isLoading: isLoadingMembers } =
    useSegmentMembers(params.id, memberPagination);
  const addMember = useAddSegmentMember(params.id);
  const removeMember = useRemoveSegmentMember(params.id);

  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showAddMemberDialog, setShowAddMemberDialog] = useState(false);
  const [removeCustomerId, setRemoveCustomerId] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editColor, setEditColor] = useState("");

  // Customer search for adding members
  const [customerSearch, setCustomerSearch] = useState("");
  const [selectedCustomerId, setSelectedCustomerId] = useState("");
  const { data: customersData } = useCustomers({
    search: customerSearch,
    limit: 10,
  });
  const customers = customersData?.items ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (!segment) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <p className="text-muted-foreground">{t("segmentNotFound")}</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => router.push("/customers/segments")}
        >
          {t("detail.backToList")}
        </Button>
      </div>
    );
  }

  const startEditing = () => {
    setEditName(segment.name);
    setEditDescription(segment.description || "");
    setEditColor(segment.color);
    setIsEditing(true);
  };

  const handleUpdate = async () => {
    try {
      await updateSegment.mutateAsync({
        name: editName,
        description: editDescription,
        color: editColor,
      });
      toast.success(t("segmentUpdated"));
      setIsEditing(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteSegment.mutateAsync(params.id);
      toast.success(t("segmentDeleted"));
      router.push("/customers/segments");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleAddMember = async () => {
    if (!selectedCustomerId) return;
    try {
      await addMember.mutateAsync({ customer_id: selectedCustomerId });
      toast.success(t("customerAddedToSegment"));
      setShowAddMemberDialog(false);
      setSelectedCustomerId("");
      setCustomerSearch("");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleRemoveMember = () => {
    if (!removeCustomerId) return;
    removeMember.mutate(removeCustomerId, {
      onSuccess: () => {
        toast.success(t("customerRemovedFromSegment"));
        setRemoveCustomerId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const members = membersData?.items ?? [];

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push("/customers/segments")}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex items-center gap-2 flex-1">
          <div
            className="h-4 w-4 rounded-full"
            style={{ backgroundColor: segment.color }}
          />
          <h1 className="text-2xl font-bold tracking-tight">{segment.name}</h1>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={startEditing}>
            {tc("edit")}
          </Button>
          <Button
            variant="destructive"
            onClick={() => setShowDeleteDialog(true)}
          >
            {t("delete")}
          </Button>
        </div>
      </div>

      {isEditing && (
        <Card>
          <CardHeader>
            <CardTitle>{t("editSegment")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4 max-w-md">
              <div className="space-y-2">
                <Label>{tc("name")}</Label>
                <Input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>{tc("description")}</Label>
                <Input
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("segmentColor")}</Label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={editColor}
                    onChange={(e) => setEditColor(e.target.value)}
                    className="h-9 w-9 rounded border cursor-pointer"
                  />
                  <Input
                    value={editColor}
                    onChange={(e) => setEditColor(e.target.value)}
                    className="flex-1"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={handleUpdate}
                  disabled={updateSegment.isPending}
                >
                  {updateSegment.isPending ? tc("saving") : tc("save")}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setIsEditing(false)}
                >
                  {tc("cancel")}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>{tc("details")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {segment.description && (
              <div>
                <p className="text-sm text-muted-foreground">{tc("description")}</p>
                <p className="mt-1 text-sm">{segment.description}</p>
              </div>
            )}
            <div>
              <p className="text-sm text-muted-foreground">{tc("type")}</p>
              <p className="mt-1 text-sm font-medium">
                {SEGMENT_TYPE_LABELS[segment.segment_type] ||
                  segment.segment_type}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("klientow")}</p>
              <p className="mt-1 text-2xl font-bold">
                {segment.customer_count}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{tc("createdAt")}</p>
              <p className="mt-1 text-sm">{formatDate(segment.created_at)}</p>
            </div>
          </CardContent>
        </Card>

        {segment.rules && (
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>{t("rules1")}</CardTitle>
            </CardHeader>
            <CardContent>
              <pre className="rounded bg-muted p-3 text-sm font-mono overflow-auto">
                {JSON.stringify(segment.rules, null, 2)}
              </pre>
            </CardContent>
          </Card>
        )}
      </div>

      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle>{t("members1")}</CardTitle>
          {segment.segment_type === "manual" && (
            <Button
              size="sm"
              onClick={() => setShowAddMemberDialog(true)}
            >
              <Plus className="mr-1 h-4 w-4" />
              {t("addCustomer")}
            </Button>
          )}
        </CardHeader>
        <CardContent>
          {isLoadingMembers ? (
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
            </div>
          ) : members.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <Users className="h-8 w-8 text-muted-foreground/50 mb-2" />
              <p className="text-sm text-muted-foreground">
                {t("brakKlientowWTymSegmencie")}
              </p>
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("form.fullName")}</TableHead>
                    <TableHead>{t("form.email")}</TableHead>
                    <TableHead className="text-right">{t("zamowien")}</TableHead>
                    <TableHead className="text-right">{t("wydanoŁacznie")}</TableHead>
                    <TableHead>{t("addedAt")}</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.map((member) => (
                    <TableRow
                      key={member.customer_id}
                      className="cursor-pointer"
                      onClick={() =>
                        router.push(`/customers/${member.customer_id}`)
                      }
                    >
                      <TableCell className="font-medium">
                        {member.customer_name}
                      </TableCell>
                      <TableCell>
                        {member.customer_email || "---"}
                      </TableCell>
                      <TableCell className="text-right">
                        {member.total_orders}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatCurrency(member.total_spent)}
                      </TableCell>
                      <TableCell>{formatDate(member.added_at)}</TableCell>
                      <TableCell>
                        {segment.segment_type === "manual" && (
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            onClick={(e) => {
                              e.stopPropagation();
                              setRemoveCustomerId(member.customer_id);
                            }}
                          >
                            <Trash2 className="h-3.5 w-3.5 text-destructive" />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {membersData && (
                <DataTablePagination
                  total={membersData.total}
                  limit={membersData.limit}
                  offset={membersData.offset}
                  onPageChange={(offset) =>
                    setMemberPagination((prev) => ({ ...prev, offset }))
                  }
                  onPageSizeChange={(limit) =>
                    setMemberPagination({ limit, offset: 0 })
                  }
                />
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* Add member dialog */}
      <Dialog open={showAddMemberDialog} onOpenChange={setShowAddMemberDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("addCustomerToSegment")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("searchCustomer")}</Label>
              <Input
                value={customerSearch}
                onChange={(e) => setCustomerSearch(e.target.value)}
                placeholder={t("wpiszImieLubEmail")}
              />
            </div>
            {customers.length > 0 && (
              <Select
                value={selectedCustomerId}
                onValueChange={setSelectedCustomerId}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("selectCustomer")} />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} {c.email ? `(${c.email})` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowAddMemberDialog(false)}
              >
                {tc("cancel")}
              </Button>
              <Button
                onClick={handleAddMember}
                disabled={addMember.isPending || !selectedCustomerId}
              >
                {addMember.isPending ? t("adding") : tc("add")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("deleteSegmentTitle")}
        description={t("czyNaPewnoChceszUsunacTenSegment")}
        confirmLabel={t("usunSegment")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteSegment.isPending}
      />

      <ConfirmDialog
        open={!!removeCustomerId}
        onOpenChange={(open) => !open && setRemoveCustomerId(null)}
        title={t("usunZSegmentu")}
        description={t("czyNaPewnoChceszUsunacTegoKlientaZSegmentu")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleRemoveMember}
        isLoading={removeMember.isPending}
      />
    </div>
  );
}
