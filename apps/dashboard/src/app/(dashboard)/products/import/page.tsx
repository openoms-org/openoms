"use client";

import { useState, useCallback } from "react";
import { Upload, FileSpreadsheet, CheckCircle, AlertCircle, ArrowLeft } from "lucide-react";
import Link from "next/link";
import { toast } from "sonner";
import { useProductImportPreview, useProductImport } from "@/hooks/use-product-import";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
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
import { Badge } from "@/components/ui/badge";
import type { ProductImportPreview, ProductImportResult } from "@/types/api";

export default function ProductImportPage() {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ProductImportPreview | null>(null);
  const [result, setResult] = useState<ProductImportResult | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  const previewMutation = useProductImportPreview();
  const importMutation = useProductImport();

  const handleFile = useCallback(
    (f: File) => {
      setFile(f);
      setResult(null);
      previewMutation.mutate(f, {
        onSuccess: (data) => {
          setPreview(data);
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
          setPreview(null);
        },
      });
    },
    [previewMutation]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      const droppedFile = e.dataTransfer.files[0];
      if (droppedFile && droppedFile.name.endsWith(".csv")) {
        handleFile(droppedFile);
      } else {
        toast.error("Wybierz plik CSV");
      }
    },
    [handleFile]
  );

  const handleImport = () => {
    if (!file) return;
    importMutation.mutate(file, {
      onSuccess: (data) => {
        setResult(data);
        toast.success(
          `Import zakończony: ${data.created} utworzonych, ${data.updated} zaktualizowanych`
        );
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const resetState = () => {
    setFile(null);
    setPreview(null);
    setResult(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/products">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Import produktów CSV
          </h1>
          <p className="text-muted-foreground">
            Importuj produkty z pliku CSV. Istniejące produkty (dopasowane po SKU) zostaną zaktualizowane.
          </p>
        </div>
      </div>

      {/* Result view */}
      {result && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-600" />
              Import zakończony
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div className="rounded-lg border p-4 text-center">
                <p className="text-2xl font-bold text-green-600">
                  {result.created}
                </p>
                <p className="text-sm text-muted-foreground">
                  Nowe produkty
                </p>
              </div>
              <div className="rounded-lg border p-4 text-center">
                <p className="text-2xl font-bold text-blue-600">
                  {result.updated}
                </p>
                <p className="text-sm text-muted-foreground">
                  Zaktualizowane
                </p>
              </div>
              <div className="rounded-lg border p-4 text-center">
                <p className="text-2xl font-bold text-red-600">
                  {result.errors.length}
                </p>
                <p className="text-sm text-muted-foreground">
                  Błędy
                </p>
              </div>
            </div>

            {result.errors.length > 0 && (
              <div className="space-y-2">
                <h3 className="font-medium text-destructive">Błędy importu:</h3>
                <div className="max-h-60 overflow-y-auto rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Wiersz</TableHead>
                        <TableHead>Pole</TableHead>
                        <TableHead>Komunikat</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.errors.map((err, i) => (
                        <TableRow key={i}>
                          <TableCell>{err.row}</TableCell>
                          <TableCell>{err.field || "-"}</TableCell>
                          <TableCell>{err.message}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}

            <div className="flex gap-2">
              <Button variant="outline" onClick={resetState}>
                Importuj kolejny plik
              </Button>
              <Button asChild>
                <Link href="/products">Wróć do produktów</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Upload zone (shown when no result) */}
      {!result && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Plik CSV</CardTitle>
              <CardDescription>
                Przeciągnij plik CSV lub kliknij, aby wybrać. Wymagane kolumny: name. Opcjonalne: sku, ean, price, stock_quantity, category, tags, weight, width, height, length, short_description.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div
                className={`relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-12 transition-colors ${
                  isDragging
                    ? "border-primary bg-primary/5"
                    : "border-muted-foreground/25 hover:border-primary/50"
                }`}
                onDragOver={(e) => {
                  e.preventDefault();
                  setIsDragging(true);
                }}
                onDragLeave={() => setIsDragging(false)}
                onDrop={handleDrop}
              >
                <input
                  type="file"
                  accept=".csv"
                  className="absolute inset-0 cursor-pointer opacity-0"
                  onChange={(e) => {
                    const f = e.target.files?.[0];
                    if (f) handleFile(f);
                  }}
                />
                {file ? (
                  <div className="flex items-center gap-3">
                    <FileSpreadsheet className="h-8 w-8 text-green-600" />
                    <div>
                      <p className="font-medium">{file.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {(file.size / 1024).toFixed(1)} KB
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        resetState();
                      }}
                    >
                      Zmień
                    </Button>
                  </div>
                ) : (
                  <>
                    <Upload className="mb-4 h-10 w-10 text-muted-foreground" />
                    <p className="text-sm font-medium">
                      Przeciągnij plik CSV tutaj
                    </p>
                    <p className="text-xs text-muted-foreground">
                      lub kliknij, aby wybrać plik
                    </p>
                  </>
                )}
              </div>
              {previewMutation.isPending && (
                <p className="mt-2 text-sm text-muted-foreground">
                  Analizowanie pliku...
                </p>
              )}
            </CardContent>
          </Card>

          {/* Preview */}
          {preview && (
            <Card>
              <CardHeader>
                <CardTitle>Podgląd importu</CardTitle>
                <CardDescription>
                  Pierwsze 10 wierszy z pliku CSV
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex gap-4">
                  <Badge variant="outline" className="text-sm px-3 py-1">
                    Łącznie: {preview.total_rows}
                  </Badge>
                  <Badge
                    variant="outline"
                    className="text-sm px-3 py-1 bg-success/15 text-success"
                  >
                    Nowe produkty: {preview.new_count}
                  </Badge>
                  <Badge
                    variant="outline"
                    className="text-sm px-3 py-1 bg-info/15 text-info"
                  >
                    Aktualizacje: {preview.update_count}
                  </Badge>
                </div>

                <div className="overflow-x-auto rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>#</TableHead>
                        {preview.headers.map((h) => (
                          <TableHead key={h}>{h}</TableHead>
                        ))}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {preview.sample_rows.map((row) => (
                        <TableRow key={row.row}>
                          <TableCell className="text-muted-foreground">
                            {row.row}
                          </TableCell>
                          {preview.headers.map((h) => (
                            <TableCell key={h} className="max-w-[200px] truncate">
                              {String(row.data[h] ?? "")}
                            </TableCell>
                          ))}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                <div className="flex justify-end gap-2">
                  <Button variant="outline" onClick={resetState}>
                    Anuluj
                  </Button>
                  <Button
                    onClick={handleImport}
                    disabled={importMutation.isPending}
                  >
                    {importMutation.isPending
                      ? "Importowanie..."
                      : `Importuj ${preview.total_rows} produktów`}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
