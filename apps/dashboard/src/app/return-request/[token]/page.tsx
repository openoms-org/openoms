"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { RotateCcw, Clock, CheckCircle2, XCircle, Package, ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { PublicReturnStatus } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: React.ComponentType<{ className?: string }> }> = {
  requested: { label: "Zgloszone", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200", icon: Clock },
  approved: { label: "Zatwierdzone", color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200", icon: CheckCircle2 },
  received: { label: "Odebrane", color: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200", icon: Package },
  refunded: { label: "Zwrocone", color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200", icon: CheckCircle2 },
  rejected: { label: "Odrzucone", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200", icon: XCircle },
  cancelled: { label: "Anulowane", color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200", icon: XCircle },
};

const STATUS_ORDER = ["requested", "approved", "received", "refunded"];

export default function PublicReturnStatusPage() {
  const params = useParams();
  const token = params.token as string;
  const [data, setData] = useState<PublicReturnStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!token) return;

    const fetchStatus = async () => {
      try {
        const res = await fetch(`${API_URL}/v1/public/returns/${token}/status`);
        if (!res.ok) {
          if (res.status === 404) {
            setError("Nie znaleziono zwrotu o podanym tokenie.");
          } else {
            setError("Wystapil blad podczas ladowania statusu zwrotu.");
          }
          return;
        }
        const statusData: PublicReturnStatus = await res.json();
        setData(statusData);
      } catch {
        setError("Nie udalo sie polaczyc z serwerem.");
      } finally {
        setIsLoading(false);
      }
    };

    fetchStatus();
  }, [token]);

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("pl-PL", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const currentStatusIndex = data ? STATUS_ORDER.indexOf(data.status) : -1;
  const isTerminal = data?.status === "rejected" || data?.status === "cancelled";

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-2">
            <RotateCcw className="h-8 w-8 text-primary" />
            <span className="text-2xl font-bold">OpenOMS</span>
          </div>
          <p className="text-muted-foreground">Status zwrotu</p>
        </div>

        {isLoading && (
          <Card>
            <CardContent className="py-12">
              <div className="flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
              </div>
              <p className="text-center mt-4 text-muted-foreground">
                Ladowanie...
              </p>
            </CardContent>
          </Card>
        )}

        {error && (
          <Card>
            <CardContent className="py-12">
              <div className="text-center">
                <XCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
                <p className="text-destructive">{error}</p>
                <Button variant="outline" className="mt-4" asChild>
                  <Link href="/return-request">
                    <ArrowLeft className="h-4 w-4 mr-2" />
                    Powrot do formularza
                  </Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {data && !error && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Zwrot #{data.id.slice(0, 8)}</CardTitle>
                {(() => {
                  const config = STATUS_CONFIG[data.status] || STATUS_CONFIG.requested;
                  return (
                    <Badge variant="outline" className={config.color}>
                      {config.label}
                    </Badge>
                  );
                })()}
              </div>
              <CardDescription>
                Zgloszono: {formatDate(data.created_at)}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Status timeline */}
              {!isTerminal && (
                <div className="space-y-3">
                  <h3 className="text-sm font-medium text-muted-foreground">
                    Postep
                  </h3>
                  <div className="flex items-center gap-1">
                    {STATUS_ORDER.map((status, index) => {
                      const isActive = index <= currentStatusIndex;
                      const config = STATUS_CONFIG[status];
                      return (
                        <div key={status} className="flex-1 flex flex-col items-center">
                          <div
                            className={`w-full h-2 rounded-full ${
                              isActive
                                ? "bg-primary"
                                : "bg-gray-200 dark:bg-gray-700"
                            }`}
                          />
                          <span
                            className={`text-xs mt-1 ${
                              isActive
                                ? "text-primary font-medium"
                                : "text-muted-foreground"
                            }`}
                          >
                            {config.label}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {isTerminal && (
                <div className="rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    {data.status === "rejected"
                      ? "Twoje zgloszenie zwrotu zostalo odrzucone."
                      : "Zwrot zostal anulowany."}
                  </p>
                </div>
              )}

              {/* Reason */}
              <div>
                <h3 className="text-sm font-medium text-muted-foreground mb-1">
                  Powod zwrotu
                </h3>
                <p className="text-sm">{data.reason}</p>
              </div>

              {/* Items */}
              {data.items && data.items.length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-muted-foreground mb-2">
                    Produkty do zwrotu
                  </h3>
                  <div className="space-y-2">
                    {data.items.map((item, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between text-sm border rounded-md p-2"
                      >
                        <span>{item.name}</span>
                        <span className="text-muted-foreground">
                          x{item.quantity}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <div className="text-xs text-muted-foreground">
                Ostatnia aktualizacja: {formatDate(data.updated_at)}
              </div>

              <Button variant="outline" className="w-full" asChild>
                <Link href="/return-request">
                  <ArrowLeft className="h-4 w-4 mr-2" />
                  Zglos kolejny zwrot
                </Link>
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
