"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ArrowLeft,
  Download,
  Loader2,
  CheckCircle2,
  AlertCircle,
  Link2,
  SkipForward,
} from "lucide-react";
import { toast } from "sonner";
import {
  useImportAllegroOffers,
  type AllegroImportResult,
} from "@/hooks/use-allegro-import";
import { useTranslations } from "next-intl";

export default function AllegroImportPage() {
  const t = useTranslations("marketplaces");
  const importMutation = useImportAllegroOffers();
  const result = importMutation.data;

  const handleImport = () => {
    importMutation.mutate(undefined, {
      onSuccess: (data: AllegroImportResult) => {
        toast.success(
          t("importCompleted", { created: data.created, linked: data.linked })
        );
      },
      onError: (error: Error) => {
        toast.error(t("importError", { message: error.message }));
      },
    });
  };

  return (
    <div className="space-y-6">
      {/* Header with back button */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/marketplaces/allegro">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{t("importAllegroOffers")}</h1>
          <p className="text-muted-foreground">
            {t("pobierzSwojeOfertyZAllegroIUtworz")}
          </p>
        </div>
      </div>

      {/* Import action card */}
      {!result && (
        <Card>
          <CardHeader>
            <CardTitle>{t("offerImport")}</CardTitle>
            <CardDescription>
              Import pobiera wszystkie Twoje aktywne oferty z Allegro, dopasowuje
              {t("jePoSkuDoIstniejacychProduktowLub")}
              {t("powiazaneZostanaPominiete")}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleImport}
              disabled={importMutation.isPending}
              size="lg"
            >
              {importMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {t("importingOffers")}
                </>
              ) : (
                <>
                  <Download className="mr-2 h-4 w-4" />
                  {t("startImport")}
                </>
              )}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Results */}
      {result && (
        <>
          {/* Summary cards in 2x2 grid */}
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <SummaryCard
              title={t("created")}
              value={result.created}
              variant="success"
              icon={<CheckCircle2 className="h-4 w-4" />}
            />
            <SummaryCard
              title={t("powiazane")}
              value={result.linked}
              variant="info"
              icon={<Link2 className="h-4 w-4" />}
            />
            <SummaryCard
              title={t("pominiete")}
              value={result.skipped}
              variant="warning"
              icon={<SkipForward className="h-4 w-4" />}
            />
            <SummaryCard
              title={t("błedy")}
              value={result.errors}
              variant="error"
              icon={<AlertCircle className="h-4 w-4" />}
            />
          </div>

          {/* Details table */}
          {result.details && result.details.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>{t("szczegołyImportu")}</CardTitle>
                <CardDescription>
                  {t("showingOfTotal", { shown: result.details.length, total: result.total_offers })}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("offer")}</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>{t("product")}</TableHead>
                      <TableHead>{t("invoice.error")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {result.details.map((detail) => (
                      <TableRow key={detail.offer_id}>
                        <TableCell className="font-medium">
                          {detail.offer_name}
                        </TableCell>
                        <TableCell>
                          <ActionBadge action={detail.action} />
                        </TableCell>
                        <TableCell>
                          {detail.product_id && (
                            <Link
                              href={`/products/${detail.product_id}`}
                              className="text-primary hover:underline"
                            >
                              {detail.product_id.slice(0, 8)}...
                            </Link>
                          )}
                        </TableCell>
                        <TableCell className="text-destructive text-sm">
                          {detail.error}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* Action buttons */}
          <div className="flex gap-4">
            <Button onClick={() => importMutation.reset()} variant="outline">
              {t("importAgain")}
            </Button>
            <Button asChild>
              <Link href="/products">{t("przejdzDoProduktow")}</Link>
            </Button>
          </div>
        </>
      )}
    </div>
  );
}

function SummaryCard({
  title,
  value,
  variant,
  icon,
}: {
  title: string;
  value: number;
  variant: "success" | "info" | "warning" | "error";
  icon: React.ReactNode;
}) {
  const colorMap = {
    success: "text-green-600",
    info: "text-blue-600",
    warning: "text-yellow-600",
    error: "text-red-600",
  };
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className={`text-2xl font-bold ${colorMap[variant]}`}>{value}</p>
          </div>
          <div className={colorMap[variant]}>{icon}</div>
        </div>
      </CardContent>
    </Card>
  );
}

function ActionBadge({ action }: { action: string }) {
  const t = useTranslations("marketplaces");
  const variants: Record<
    string,
    "default" | "secondary" | "destructive" | "outline"
  > = {
    created: "default",
    linked: "secondary",
    skipped: "outline",
    error: "destructive",
  };
  const labels: Record<string, string> = {
    created: t("statusCreated"),
    linked: t("powiazany"),
    skipped: t("pominiety"),
    error: t("invoice.error"),
  };
  return (
    <Badge variant={variants[action] || "outline"}>
      {labels[action] || action}
    </Badge>
  );
}
