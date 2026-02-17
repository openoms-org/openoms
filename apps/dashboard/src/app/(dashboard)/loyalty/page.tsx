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
import { PageHeader } from "@/components/shared/page-header";
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

const DEFAULT_LIMIT = 20;

const PROGRAM_TYPE_LABELS: Record<string, string> = {
  points: "Punkty",
  tier: "Poziomy",
  discount_after_n: "Rabat po N zamówieniach",
};

const STATUS_LABELS: Record<string, { label: string; className: string }> = {
  active: {
    label: "Aktywny",
    className: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  },
  paused: {
    label: "Wstrzymany",
    className: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  },
  ended: {
    label: "Zakończony",
    className: "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200",
  },
};

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
      let config = formData.config;
      try {
        config = JSON.parse(configText);
      } catch {
        toast.error("Nieprawidłowy format JSON w konfiguracji");
        return;
      }

      await createProgram.mutateAsync({ ...formData, config });
      toast.success("Program lojalnościowy został utworzony");
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
        toast.success("Program został usunięty");
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
            Programy lojalnościowe
          </h1>
          <p className="text-muted-foreground">
            Zarządzaj programami punktowymi, poziomami i rabatami
          </p>
        </div>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Nowy program
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Nowy program lojalnościowy</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Nazwa *</Label>
                <Input
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  placeholder="np. Program punktowy"
                />
              </div>
              <div className="space-y-2">
                <Label>Typ programu</Label>
                <Select
                  value={formData.program_type}
                  onValueChange={handleTypeChange}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="points">
                      Punkty (zbieraj za zakupy)
                    </SelectItem>
                    <SelectItem value="tier">
                      Poziomy (Bronze/Silver/Gold)
                    </SelectItem>
                    <SelectItem value="discount_after_n">
                      Rabat po N zamówieniach
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Konfiguracja (JSON)</Label>
                <Textarea
                  value={configText}
                  onChange={(e) => setConfigText(e.target.value)}
                  rows={6}
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  {formData.program_type === "points" &&
                    "points_per_pln: ile punktów za 1 PLN; redemption_rate: ile punktów wymienić; redemption_value: wartość w PLN"}
                  {formData.program_type === "tier" &&
                    "tiers: lista poziomów z progami min_spent i rabatami discount_pct"}
                  {formData.program_type === "discount_after_n" &&
                    "orders_required: ile zamówień potrzeba; discount_pct: procent rabatu"}
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => setCreateOpen(false)}
                >
                  Anuluj
                </Button>
                <Button
                  onClick={handleCreate}
                  disabled={
                    createProgram.isPending || !formData.name.trim()
                  }
                >
                  {createProgram.isPending ? "Tworzenie..." : "Utwórz"}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {programs.length === 0 ? (
        <EmptyState
          icon={Award}
          title="Brak programów lojalnościowych"
          description="Utwórz pierwszy program, aby nagradzać klientów za zakupy."
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
                    Utworzono {formatDate(program.created_at)}
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
        title="Usuń program"
        description="Czy na pewno chcesz usunąć ten program lojalnościowy? Wszystkie dane punktowe zostaną utracone."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteProgram.isPending}
      />
    </>
  );
}
