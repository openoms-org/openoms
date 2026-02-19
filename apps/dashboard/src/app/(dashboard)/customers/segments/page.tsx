"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Users, Trash2, BarChart3 } from "lucide-react";
import { toast } from "sonner";
import {
  useSegments,
  useCreateSegment,
  useDeleteSegment,
  useRunRFMAnalysis,
} from "@/hooks/use-segments";
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
import type { CreateSegmentRequest, CustomerRFM } from "@/types/api";

const DEFAULT_LIMIT = 20;

const SEGMENT_TYPE_LABELS: Record<string, string> = {
  manual: "Ręczny",
  rfm_auto: "RFM (auto)",
  rule_based: "Regułowy",
};

export default function SegmentsPage() {
  const router = useRouter();
  const [pagination, setPagination] = useState({
    limit: DEFAULT_LIMIT,
    offset: 0,
  });
  const { data, isLoading, refetch } = useSegments(pagination);
  const createSegment = useCreateSegment();
  const deleteSegment = useDeleteSegment();
  const runRFM = useRunRFMAnalysis();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [rfmResults, setRfmResults] = useState<CustomerRFM[] | null>(null);
  const [rfmOpen, setRfmOpen] = useState(false);

  const [formData, setFormData] = useState<CreateSegmentRequest>({
    name: "",
    description: "",
    color: "#6366f1",
    segment_type: "manual",
  });

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const segments = data?.items ?? [];

  const handleCreate = async () => {
    try {
      await createSegment.mutateAsync(formData);
      toast.success("Segment został utworzony");
      setCreateOpen(false);
      setFormData({ name: "", description: "", color: "#6366f1", segment_type: "manual" });
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteSegment.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Segment został usunięty");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleRunRFM = async () => {
    try {
      const results = await runRFM.mutateAsync();
      setRfmResults(results);
      setRfmOpen(true);
      toast.success("Analiza RFM zakończona");
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  // Count customers per RFM segment
  const rfmSummary = rfmResults
    ? rfmResults.reduce(
        (acc, r) => {
          acc[r.segment_label] = (acc[r.segment_label] || 0) + 1;
          return acc;
        },
        {} as Record<string, number>
      )
    : {};

  const rfmColors: Record<string, string> = {
    Champions: "#10b981",
    "Lojalni klienci": "#3b82f6",
    "Potencjalni lojalni": "#8b5cf6",
    "Zagrożeni": "#f59e0b",
    Utraceni: "#ef4444",
    Pozostali: "#6b7280",
  };

  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Segmenty klientów</h1>
          <p className="text-muted-foreground">
            Grupuj klientów według zachowań zakupowych i analizy RFM
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleRunRFM}
            disabled={runRFM.isPending}
          >
            <BarChart3 className="mr-2 h-4 w-4" />
            {runRFM.isPending ? "Analiza..." : "Analiza RFM"}
          </Button>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Users className="mr-2 h-4 w-4" />
                Nowy segment
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Utwórz segment</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Nazwa *</Label>
                  <Input
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="np. VIP klienci"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Opis</Label>
                  <Textarea
                    value={formData.description || ""}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                    placeholder="Opis segmentu..."
                    rows={2}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Typ</Label>
                    <Select
                      value={formData.segment_type}
                      onValueChange={(val) =>
                        setFormData({
                          ...formData,
                          segment_type: val as CreateSegmentRequest["segment_type"],
                        })
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="manual">Ręczny</SelectItem>
                        <SelectItem value="rule_based">Regułowy</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Kolor</Label>
                    <div className="flex items-center gap-2">
                      <input
                        type="color"
                        value={formData.color}
                        onChange={(e) =>
                          setFormData({ ...formData, color: e.target.value })
                        }
                        className="h-9 w-9 rounded border cursor-pointer"
                      />
                      <Input
                        value={formData.color}
                        onChange={(e) =>
                          setFormData({ ...formData, color: e.target.value })
                        }
                        className="flex-1"
                      />
                    </div>
                  </div>
                </div>
                {formData.segment_type === "rule_based" && (
                  <div className="space-y-2">
                    <Label>Reguły (JSON)</Label>
                    <Textarea
                      value={
                        formData.rules
                          ? JSON.stringify(formData.rules, null, 2)
                          : '{\n  "min_orders": 5,\n  "min_total_spent": 1000\n}'
                      }
                      onChange={(e) => {
                        try {
                          const rules = JSON.parse(e.target.value);
                          setFormData({ ...formData, rules });
                        } catch {
                          // Allow invalid JSON while typing
                        }
                      }}
                      rows={4}
                      className="font-mono text-sm"
                    />
                  </div>
                )}
                <div className="flex justify-end gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setCreateOpen(false)}
                  >
                    Anuluj
                  </Button>
                  <Button
                    onClick={handleCreate}
                    disabled={createSegment.isPending || !formData.name.trim()}
                  >
                    {createSegment.isPending ? "Tworzenie..." : "Utwórz"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {segments.length === 0 ? (
        <EmptyState
          icon={Users}
          title="Brak segmentów"
          description="Utwórz pierwszy segment klientów lub uruchom analizę RFM."
        />
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {segments.map((segment) => (
              <div
                key={segment.id}
                className="rounded-lg border bg-card p-4 cursor-pointer hover:shadow-md transition-shadow"
                onClick={() =>
                  router.push(`/customers/segments/${segment.id}`)
                }
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <div
                      className="h-3 w-3 rounded-full"
                      style={{ backgroundColor: segment.color }}
                    />
                    <h3 className="font-semibold">{segment.name}</h3>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    onClick={(e) => {
                      e.stopPropagation();
                      setDeleteId(segment.id);
                    }}
                  >
                    <Trash2 className="h-3.5 w-3.5 text-destructive" />
                  </Button>
                </div>
                {segment.description && (
                  <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
                    {segment.description}
                  </p>
                )}
                <div className="mt-3 flex items-center gap-3 text-sm">
                  <span className="flex items-center gap-1">
                    <Users className="h-3.5 w-3.5 text-muted-foreground" />
                    {segment.customer_count} klientów
                  </span>
                  <span
                    className="rounded-full px-2 py-0.5 text-xs font-medium"
                    style={{
                      backgroundColor: segment.color + "20",
                      color: segment.color,
                    }}
                  >
                    {SEGMENT_TYPE_LABELS[segment.segment_type] ||
                      segment.segment_type}
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  Utworzono {formatDate(segment.created_at)}
                </p>
              </div>
            ))}
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

      {/* RFM Results Dialog */}
      <Dialog open={rfmOpen} onOpenChange={setRfmOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Wyniki analizy RFM</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Klienci zostali automatycznie przypisani do segmentów RFM na
              podstawie aktualności (Recency), częstotliwości (Frequency) i
              wartości zakupów (Monetary).
            </p>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {Object.entries(rfmSummary).map(([label, count]) => (
                <div
                  key={label}
                  className="rounded-lg border p-3 text-center"
                  style={{
                    borderColor: rfmColors[label] || "#6b7280",
                  }}
                >
                  <div
                    className="text-2xl font-bold"
                    style={{ color: rfmColors[label] || "#6b7280" }}
                  >
                    {count}
                  </div>
                  <div className="text-sm font-medium mt-1">{label}</div>
                </div>
              ))}
            </div>
            {rfmResults && rfmResults.length > 0 && (
              <p className="text-xs text-muted-foreground">
                Przeanalizowano {rfmResults.length} klientów. Segmenty zostały
                automatycznie zaktualizowane.
              </p>
            )}
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usuń segment"
        description="Czy na pewno chcesz usunąć ten segment? Klienci nie zostaną usunięci."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteSegment.isPending}
      />
    </>
  );
}
