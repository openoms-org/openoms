"use client";

import { useState, useCallback, useRef } from "react";
import { toast } from "sonner";
import { Eraser, Download, Upload, Loader2, ImageIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useBGRemovalStatus, useRemoveBackground } from "@/hooks/use-bg-removal";
import { downloadBlob } from "@/lib/download";
import { useTranslations } from "next-intl";

export default function BGRemovalPage() {
  const t = useTranslations("tools");
  const { data: bgStatus, isLoading: isLoadingStatus } = useBGRemovalStatus();
  const removeBackground = useRemoveBackground();

  const [originalFile, setOriginalFile] = useState<File | null>(null);
  const [originalPreview, setOriginalPreview] = useState<string | null>(null);
  const [resultUrl, setResultUrl] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback(
    (file: File) => {
      const allowedTypes = ["image/jpeg", "image/png", "image/webp"];
      if (!allowedTypes.includes(file.type)) {
        toast.error(t("nieobsługiwanyTypPlikuDozwoloneJpegPngWebp"));
        return;
      }
      if (file.size > 10 * 1024 * 1024) {
        toast.error(t("plikJestZaDuzyMaksymalnyRozmiar10Mb"));
        return;
      }

      setOriginalFile(file);
      setResultUrl(null);

      const reader = new FileReader();
      reader.onload = (e) => {
        setOriginalPreview(e.target?.result as string);
      };
      reader.readAsDataURL(file);
    },
    []
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile]
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleProcess = () => {
    if (!originalFile) return;
    removeBackground.mutate(originalFile, {
      onSuccess: (data) => {
        setResultUrl(data.url);
        toast.success(t("tłoZostałoUsuniete"));
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : t("bładUsuwaniaTła"));
      },
    });
  };

  const handleDownload = async () => {
    if (!resultUrl) return;
    const res = await fetch(resultUrl);
    const blob = await res.blob();
    downloadBlob(blob, `${originalFile?.name?.replace(/\.[^.]+$/, "") || "image"}-nobg.png`);
  };

  const handleReset = () => {
    setOriginalFile(null);
    setOriginalPreview(null);
    setResultUrl(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  if (isLoadingStatus) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!bgStatus?.configured) {
    return (
      <div className="mx-auto max-w-2xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">{t("bgRemovalTitle")}</h1>
          <p className="text-muted-foreground">
            {t("automatyczneUsuwanieTłaZeZdjecProduktowych")}
          </p>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Eraser className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <h2 className="text-lg font-semibold mb-2">
              {t("usuwanieTłaNieJestSkonfigurowane")}
            </h2>
            <p className="text-sm text-muted-foreground max-w-md">
              {t("abyKorzystacZTejFunkcjiAdministratorMusi")}{" "}
              {t("envVarConfig")}{" "}
              <code className="rounded bg-muted px-1 py-0.5 text-xs font-mono">REMOVEBG_API_KEY</code>{" "}
              {t("inServerConfig")}{" "}{t("apiKeyAvailableAt")}{" "}
              <a
                href="https://www.remove.bg/api"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary underline"
              >
                remove.bg
              </a>
              .
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("bgRemovalTitle")}</h1>
        <p className="text-muted-foreground">
          {t("automatyczneUsuwanieTłaZeZdjecProduktowychPowered")}
        </p>
      </div>

      {/* Upload area */}
      {!originalPreview && (
        <Card>
          <CardContent className="p-0">
            <div
              className={`flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-16 transition-colors cursor-pointer ${
                isDragging
                  ? "border-primary bg-primary/5"
                  : "border-muted-foreground/25 hover:border-primary/50"
              }`}
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="h-10 w-10 text-muted-foreground/50 mb-4" />
              <p className="text-sm font-medium mb-1">
                {t("przeciagnijIUpuscZdjecie")}
              </p>
              <p className="text-xs text-muted-foreground mb-4">
                {t("orClickToSelect")}
              </p>
              <Button variant="outline" size="sm" type="button">
                <ImageIcon className="mr-2 h-4 w-4" />
                {t("selectFile")}
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) handleFile(file);
                }}
              />
            </div>
          </CardContent>
        </Card>
      )}

      {/* Before/After comparison */}
      {originalPreview && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base">{t("oryginał")}</CardTitle>
                <CardDescription>
                  {originalFile?.name} ({((originalFile?.size || 0) / 1024).toFixed(0)} KB)
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="relative flex items-center justify-center rounded-lg border bg-muted/30 p-2">
                  <img
                    src={originalPreview}
                    alt={t("oryginalneZdjecie")}
                    className="max-h-80 rounded object-contain"
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base">{t("bezTła")}</CardTitle>
                <CardDescription>
                  {resultUrl ? t("gotowe") : t("kliknijUsunTloAbyPrzetworzyc")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div
                  className="relative flex items-center justify-center rounded-lg border p-2"
                  style={{
                    background: resultUrl
                      ? "repeating-conic-gradient(#e5e5e5 0% 25%, transparent 0% 50%) 50% / 16px 16px"
                      : undefined,
                  }}
                >
                  {resultUrl ? (
                    <img
                      src={resultUrl}
                      alt={t("zdjecieBezTła")}
                      className="max-h-80 rounded object-contain"
                    />
                  ) : (
                    <div className="flex h-80 flex-col items-center justify-center text-muted-foreground">
                      <Eraser className="h-10 w-10 mb-2 opacity-30" />
                      <p className="text-sm">{t("wynikPojawiSieTutaj")}</p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Action buttons */}
          <div className="flex items-center justify-center gap-3">
            <Button variant="outline" onClick={handleReset}>
              {t("noweZdjecie")}
            </Button>
            {!resultUrl ? (
              <Button
                onClick={handleProcess}
                disabled={removeBackground.isPending}
              >
                {removeBackground.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Eraser className="mr-2 h-4 w-4" />
                )}
                {t("usunTło1")}
              </Button>
            ) : (
              <Button onClick={handleDownload}>
                <Download className="mr-2 h-4 w-4" />
                {t("download")}
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
