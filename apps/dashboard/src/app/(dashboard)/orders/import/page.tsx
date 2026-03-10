"use client";

import { useState, useCallback } from "react";
import { toast } from "sonner";
import { Upload, FileSpreadsheet, ArrowRight, ArrowLeft, Check, AlertCircle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
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
import { Badge } from "@/components/ui/badge";
import { useImportPreview, useImportOrders } from "@/hooks/use-order-import";
import { getErrorMessage } from "@/lib/api-client";
import { useTranslations } from "next-intl";
import type { ImportColumnMapping, ImportPreviewResponse, ImportResult } from "@/types/api";

const ORDER_FIELD_KEYS = [
  { value: "", labelKey: "fields.skip" },
  { value: "customer_name", labelKey: "fields.customerName" },
  { value: "customer_email", labelKey: "fields.customerEmail" },
  { value: "customer_phone", labelKey: "fields.customerPhone" },
  { value: "total_amount", labelKey: "fields.amount" },
  { value: "currency", labelKey: "fields.currency" },
  { value: "source", labelKey: "fields.source" },
  { value: "external_id", labelKey: "fields.externalId" },
  { value: "notes", labelKey: "fields.notes" },
  { value: "status", labelKey: "fields.status" },
  { value: "ordered_at", labelKey: "fields.orderDate" },
  { value: "payment_status", labelKey: "fields.paymentStatus" },
  { value: "payment_method", labelKey: "fields.paymentMethod" },
  { value: "items", labelKey: "fields.itemsJson" },
  { value: "tags", labelKey: "fields.tagsComma" },
] as const;

type Step = 1 | 2 | 3 | 4;

export default function ImportOrdersPage() {
  const t = useTranslations("import");
  const [step, setStep] = useState<Step>(1);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreviewResponse | null>(null);
  const [mappings, setMappings] = useState<ImportColumnMapping[]>([]);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragOver, setDragOver] = useState(false);

  const previewMutation = useImportPreview();
  const importMutation = useImportOrders();

  // Step 1: File upload
  const handleFile = useCallback(
    async (selectedFile: File) => {
      if (!selectedFile.name.endsWith(".csv")) {
        toast.error(t("selectCsvFile"));
        return;
      }
      setFile(selectedFile);
      try {
        const data = await previewMutation.mutateAsync(selectedFile);
        setPreview(data);
        // Initialize mappings from auto-detected or empty
        const initialMappings: ImportColumnMapping[] = data.headers.map((h) => {
          const detected = data.mappings?.find((m) => m.csv_column === h);
          return {
            csv_column: h,
            order_field: detected?.order_field || "",
          };
        });
        setMappings(initialMappings);
        setStep(2);
      } catch (error) {
        toast.error(getErrorMessage(error));
      }
    },
    [previewMutation]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      const droppedFile = e.dataTransfer.files[0];
      if (droppedFile) handleFile(droppedFile);
    },
    [handleFile]
  );

  const handleFileInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFile = e.target.files?.[0];
      if (selectedFile) handleFile(selectedFile);
    },
    [handleFile]
  );

  // Step 2: Mapping
  const updateMapping = (csvColumn: string, orderField: string) => {
    setMappings((prev) =>
      prev.map((m) =>
        m.csv_column === csvColumn ? { ...m, order_field: orderField } : m
      )
    );
  };

  // Step 3: Preview with mapped data
  const activeMappings = mappings.filter((m) => m.order_field !== "");

  const getMappedPreviewData = () => {
    if (!preview) return [];
    return preview.sample_rows.map((row) => {
      const mapped: Record<string, unknown> = {};
      for (const m of activeMappings) {
        mapped[m.order_field] = row.data[m.csv_column] ?? "";
      }
      return { row: row.row, data: mapped, errors: row.errors };
    });
  };

  // Step 4: Import
  const handleImport = async () => {
    if (!file) return;
    try {
      const importResult = await importMutation.mutateAsync({
        file,
        mappings: activeMappings,
      });
      setResult(importResult);
      setStep(4);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const stepLabels = [
    t("steps.selectFile"),
    t("steps.columnMapping"),
    t("steps.dataPreview"),
    t("steps.importResult"),
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="text-muted-foreground mt-1">
          {t("subtitle")}
        </p>
      </div>

      {/* Step indicator */}
      <div className="flex items-center gap-2">
        {stepLabels.map((label, i) => {
          const stepNum = (i + 1) as Step;
          const isActive = step === stepNum;
          const isDone = step > stepNum;
          return (
            <div key={i} className="flex items-center gap-2">
              {i > 0 && (
                <div
                  className={`h-px w-8 ${isDone ? "bg-primary" : "bg-border"}`}
                />
              )}
              <div className="flex items-center gap-2">
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : isDone
                        ? "bg-primary/20 text-primary"
                        : "bg-muted text-muted-foreground"
                  }`}
                >
                  {isDone ? <Check className="size-4" /> : stepNum}
                </div>
                <span
                  className={`text-sm hidden sm:inline ${
                    isActive ? "font-medium" : "text-muted-foreground"
                  }`}
                >
                  {label}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Step 1: Upload */}
      {step === 1 && (
        <Card>
          <CardHeader>
            <CardTitle>{t("steps.selectFile")}</CardTitle>
            <CardDescription>
              {t("dragOrClick")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div
              onDragOver={(e) => {
                e.preventDefault();
                setDragOver(true);
              }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              className={`flex flex-col items-center justify-center gap-4 rounded-lg border-2 border-dashed p-12 transition-colors ${
                dragOver
                  ? "border-primary bg-primary/5"
                  : "border-border hover:border-primary/50"
              }`}
            >
              {previewMutation.isPending ? (
                <Loader2 className="size-12 text-muted-foreground animate-spin" />
              ) : (
                <Upload className="size-12 text-muted-foreground" />
              )}
              <div className="text-center">
                <p className="text-sm font-medium">
                  {t("dragOrClick")}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {t("maxSize")}
                </p>
              </div>
              <label>
                <input
                  type="file"
                  accept=".csv"
                  className="hidden"
                  onChange={handleFileInput}
                  disabled={previewMutation.isPending}
                />
                <Button
                  variant="outline"
                  asChild
                  disabled={previewMutation.isPending}
                >
                  <span>
                    <FileSpreadsheet className="size-4" />
                    {t("steps.selectFile")}
                  </span>
                </Button>
              </label>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 2: Column Mapping */}
      {step === 2 && preview && (
        <Card>
          <CardHeader>
            <CardTitle>{t("steps.columnMapping")}</CardTitle>
            <CardDescription>
              {t("mapColumnsDescription", { rows: preview.total_rows })}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {preview.headers.map((header) => {
                const mapping = mappings.find((m) => m.csv_column === header);
                return (
                  <div
                    key={header}
                    className="flex items-center gap-4"
                  >
                    <div className="w-1/3">
                      <Badge variant="outline">{header}</Badge>
                    </div>
                    <ArrowRight className="size-4 text-muted-foreground shrink-0" />
                    <div className="w-1/3">
                      <Select
                        value={mapping?.order_field || ""}
                        onValueChange={(val) => updateMapping(header, val)}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder={t("fields.skip")} />
                        </SelectTrigger>
                        <SelectContent>
                          {ORDER_FIELD_KEYS.map((f) => (
                            <SelectItem
                              key={f.value || "__skip__"}
                              value={f.value || "__skip__"}
                            >
                              {t(f.labelKey)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="mt-6 flex gap-3">
              <Button
                variant="outline"
                onClick={() => {
                  setStep(1);
                  setFile(null);
                  setPreview(null);
                }}
              >
                <ArrowLeft className="size-4" />
                {t("back")}
              </Button>
              <Button
                onClick={() => setStep(3)}
                disabled={activeMappings.length === 0}
              >
                {t("next")}
                <ArrowRight className="size-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 3: Preview */}
      {step === 3 && preview && (
        <Card>
          <CardHeader>
            <CardTitle>{t("steps.dataPreview")}</CardTitle>
            <CardDescription>
              {t("previewDescription", { sample: preview.sample_rows.length, total: preview.total_rows })}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">#</TableHead>
                    {activeMappings.map((m) => {
                      const field = ORDER_FIELD_KEYS.find(
                        (f) => f.value === m.order_field
                      );
                      return (
                        <TableHead key={m.order_field}>
                          {field ? t(field.labelKey) : m.order_field}
                        </TableHead>
                      );
                    })}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {getMappedPreviewData().map((row) => (
                    <TableRow key={row.row}>
                      <TableCell className="text-muted-foreground">
                        {row.row}
                      </TableCell>
                      {activeMappings.map((m) => (
                        <TableCell key={m.order_field}>
                          {String(row.data[m.order_field] ?? "")}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="mt-6 flex gap-3">
              <Button variant="outline" onClick={() => setStep(2)}>
                <ArrowLeft className="size-4" />
                {t("back")}
              </Button>
              <Button
                onClick={handleImport}
                disabled={importMutation.isPending}
              >
                {importMutation.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Check className="size-4" />
                )}
                {t("confirmImport")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 4: Results */}
      {step === 4 && result && (
        <Card>
          <CardHeader>
            <CardTitle>{t("steps.importResult")}</CardTitle>
            <CardDescription>
              {t("resultSummary", { imported: result.imported, skipped: result.skipped, errors: result.errors.length })}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="rounded-lg border p-4 text-center">
                <div className="text-2xl font-bold text-green-600">
                  {result.imported}
                </div>
                <div className="text-sm text-muted-foreground">
                  {t("imported")}
                </div>
              </div>
              <div className="rounded-lg border p-4 text-center">
                <div className="text-2xl font-bold text-warning">
                  {result.skipped}
                </div>
                <div className="text-sm text-muted-foreground">
                  {t("skipped")}
                </div>
              </div>
              <div className="rounded-lg border p-4 text-center">
                <div className="text-2xl font-bold text-destructive">
                  {result.errors.length}
                </div>
                <div className="text-sm text-muted-foreground">
                  {t("errorsLabel")}
                </div>
              </div>
            </div>

            {result.errors.length > 0 && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium flex items-center gap-2">
                  <AlertCircle className="size-4 text-red-500" />
                  {t("errorDetails")}
                </h3>
                <div className="max-h-64 overflow-y-auto rounded-lg border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-16">{t("row")}</TableHead>
                        <TableHead className="w-32">{t("field")}</TableHead>
                        <TableHead>{t("error")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.errors.map((err, i) => (
                        <TableRow key={i}>
                          <TableCell>{err.row}</TableCell>
                          <TableCell>
                            {err.field ? (
                              <Badge variant="outline">{err.field}</Badge>
                            ) : (
                              "-"
                            )}
                          </TableCell>
                          <TableCell className="text-red-600">
                            {err.message}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}

            <div className="mt-6 flex gap-3">
              <Button
                variant="outline"
                onClick={() => {
                  setStep(1);
                  setFile(null);
                  setPreview(null);
                  setMappings([]);
                  setResult(null);
                }}
              >
                {t("importAnotherFile")}
              </Button>
              <Button
                onClick={() => (window.location.href = "/orders")}
              >
                {t("goToOrders")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
