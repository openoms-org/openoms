"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Award, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useLoyaltyPrograms,
  useCreateLoyaltyProgram,
  useDeleteLoyaltyProgram,
} from "@/hooks/use-loyalty";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
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
import type { CreateLoyaltyProgramRequest } from "@/types/api";
import { useTranslations } from "next-intl";

const DEFAULT_LIMIT = 20;

const DEFAULT_CONFIGS: Record<string, Record<string, unknown>> = {
  points: { points_per_pln: 1, redemption_rate: 100, redemption_value: 5 },
  tier: {
    tiers: [
      { name: "Bronze", min_spent: 0, discount_pct: 0 },
      { name: "Silver", min_spent: 500, discount_pct: 5 },
      { name: "Gold", min_spent: 2000, discount_pct: 10 },
    ],
  },
  discount_after_n: { orders_required: 5, discount_pct: 10 },
};

export default function LoyaltyPage() {
  const t = useTranslations("loyalty");
  const tc = useTranslations("common");

  const PROGRAM_TYPE_LABELS: Record<string, string> = {
    points: t("type.points"),
    tier: t("type.tier"),
    discount_after_n: t("type.discountAfterN"),
  };

  const STATUS_LABELS: Record<string, { label: string; className: string }> = {
    active: {
      label: t("statusActive"),
      className: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    },
    paused: {
      label: t("statusPaused"),
      className: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    },
    ended: {
      label: t("zakonczony"),
      className: "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200",
    },
  };
  const router = useRouter();
  const [pagination, setPagination] = useState({
    limit: DEFAULT_LIMIT,
    offset: 0,
  });
  const { data, isLoading } = useLoyaltyPrograms(pagination);
  const createProgram = useCreateLoyaltyProgram();
  const deleteProgram = useDeleteLoyaltyProgram();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [formData, setFormData] = useState<CreateLoyaltyProgramRequest>({
    name: "",
    program_type: "points",
    config: DEFAULT_CONFIGS.points,
  });
  const [configText, setConfigText] = useState(
    JSON.stringify(DEFAULT_CONFIGS.points, null, 2)
  );

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const programs = data?.items ?? [];

  const handleTypeChange = (type: string) => {
    const config = DEFAULT_CONFIGS[type] || {};
    setFormData({
      ...formData,
      program_type: type as CreateLoyaltyProgramRequest["program_type"],
      config,
    });
    setConfigText(JSON.stringify(config, null, 2));
  };

  const handleCreate = async () => {
    try {
      let config;
      try {
        config = JSON.parse(configText);
      } catch {
        toast.error(t("nieprawidłowyformatjsonwkonfiguracji"));
        return;
      }

      await createProgram.mutateAsync({ ...formData, config });
      toast.success(t("programlojalnosciowyzostałutworzony"));
      setCreateOpen(false);
      setFormData({
        name: "",
        program_type: "points",
        config: DEFAULT_CONFIGS.points,
      });
      setConfigText(JSON.stringify(DEFAULT_CONFIGS.points, null, 2));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteProgram.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("programzostałusuniety"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("programyLojalnosciowe")}
          </h1>
          <p className="text-muted-foreground">
            {t("zarzadzajProgramamiPunktowymiPoziomamiIRabatami")}
          </p>
        </div>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              {t("newProgram")}
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>{t("nowyProgramLojalnosciowy")}</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>{t("nameRequired")}</Label>
                <Input
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  placeholder={t("namePlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("programType")}</Label>
                <Select
                  value={formData.program_type}
                  onValueChange={handleTypeChange}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="points">
                      {t("type.pointsDesc")}
                    </SelectItem>
                    <SelectItem value="tier">
                      {t("type.tierDesc")}
                    </SelectItem>
                    <SelectItem value="discount_after_n">
                      {t("rabatPoNZamowieniach")}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("configurationJson")}</Label>
                <Textarea
                  value={configText}
                  onChange={(e) => setConfigText(e.target.value)}
                  rows={6}
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  {formData.program_type === "points" &&
                    t("points_per_plnIlePunktowZa1PlnRedemption_rateIlePu")}
                  {formData.program_type === "tier" &&
                    t("tiersListaPoziomowZProgamiMin_spentIRabatamiDiscou")}
                  {formData.program_type === "discount_after_n" &&
                    t("orders_requiredIleZamowienPotrzebaDiscount_pctProc")}
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => setCreateOpen(false)}
                >
                  {tc("cancel")}
                </Button>
                <Button
                  onClick={handleCreate}
                  disabled={
                    createProgram.isPending || !formData.name.trim()
                  }
                >
                  {createProgram.isPending ? t("creating") : tc("create")}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {programs.length === 0 ? (
        <EmptyState
          icon={Award}
          title={t("brakProgramowLojalnosciowych")}
          description={t("utworzPierwszyProgramAbyNagradzacKlientowZaZakupy")}
        />
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {programs.map((program) => {
              const statusInfo = STATUS_LABELS[program.status] ||
                STATUS_LABELS.active;

              return (
                <div
                  key={program.id}
                  className="rounded-lg border bg-card p-4 cursor-pointer hover:shadow-md transition-shadow"
                  onClick={() => router.push(`/loyalty/${program.id}`)}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="font-semibold">{program.name}</h3>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-sm text-muted-foreground">
                          {PROGRAM_TYPE_LABELS[program.program_type] ||
                            program.program_type}
                        </span>
                        <span
                          className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusInfo.className}`}
                        >
                          {statusInfo.label}
                        </span>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(program.id);
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-destructive" />
                    </Button>
                  </div>
                  <div className="mt-3">
                    <Award className="h-8 w-8 text-muted-foreground/30" />
                  </div>
                  <p className="mt-2 text-xs text-muted-foreground">
                    {t("createdOn", { date: formatDate(program.created_at) })}
                  </p>
                </div>
              );
            })}
          </div>

          {data && (
            <DataTablePagination
              total={data.total}
              limit={data.limit}
              offset={data.offset}
              onPageChange={(offset) =>
                setPagination((prev) => ({ ...prev, offset }))
              }
              onPageSizeChange={(limit) =>
                setPagination({ limit, offset: 0 })
              }
            />
          )}
        </>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("usunProgram")}
        description={t("czyNaPewnoChceszUsunacTenProgramLojalnosciowyWszys1")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteProgram.isPending}
      />
    </>
  );
}
