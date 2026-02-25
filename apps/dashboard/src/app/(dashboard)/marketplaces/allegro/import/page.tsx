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

export default function AllegroImportPage() {
  const importMutation = useImportAllegroOffers();
  const result = importMutation.data;

  const handleImport = () => {
    importMutation.mutate(undefined, {
      onSuccess: (data: AllegroImportResult) => {
        toast.success(
          `Import zakończony: ${data.created} nowych, ${data.linked} powiązanych`
        );
      },
      onError: (error: Error) => {
        toast.error(`Błąd importu: ${error.message}`);
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
          <h1 className="text-2xl font-bold">Importuj oferty z Allegro</h1>
          <p className="text-muted-foreground">
            Pobierz swoje oferty z Allegro i utwórz produkty w systemie
          </p>
        </div>
      </div>

      {/* Import action card */}
      {!result && (
        <Card>
          <CardHeader>
            <CardTitle>Import ofert</CardTitle>
            <CardDescription>
              Import pobiera wszystkie Twoje aktywne oferty z Allegro, dopasowuje
              je po SKU do istniejących produktów lub tworzy nowe. Oferty już
              powiązane zostaną pominięte.
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
                  Importowanie ofert...
                </>
              ) : (
                <>
                  <Download className="mr-2 h-4 w-4" />
                  Rozpocznij import
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
              title="Utworzone"
              value={result.created}
              variant="success"
              icon={<CheckCircle2 className="h-4 w-4" />}
            />
            <SummaryCard
              title="Powiązane"
              value={result.linked}
              variant="info"
              icon={<Link2 className="h-4 w-4" />}
            />
            <SummaryCard
              title="Pominięte"
              value={result.skipped}
              variant="warning"
              icon={<SkipForward className="h-4 w-4" />}
            />
            <SummaryCard
              title="Błędy"
              value={result.errors}
              variant="error"
              icon={<AlertCircle className="h-4 w-4" />}
            />
          </div>

          {/* Details table */}
          {result.details && result.details.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Szczegóły importu</CardTitle>
                <CardDescription>
                  Pokazano {result.details.length} z {result.total_offers} ofert
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Oferta</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Produkt</TableHead>
                      <TableHead>Błąd</TableHead>
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
              Importuj ponownie
            </Button>
            <Button asChild>
              <Link href="/products">Przejdź do produktów</Link>
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
    created: "Utworzony",
    linked: "Powiązany",
    skipped: "Pominięty",
    error: "Błąd",
  };
  return (
    <Badge variant={variants[action] || "outline"}>
      {labels[action] || action}
    </Badge>
  );
}
