"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { ScanBarcode, Check, X, Package, Search } from "lucide-react";
import { toast } from "sonner";
import { apiClient, getErrorMessage } from "@/lib/api-client";
import { useOrders } from "@/hooks/use-orders";
import { PageHeader } from "@/components/shared/page-header";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type {
  Order,
  OrderItem,
  BarcodeLookupResponse,
  ScannedItem,
} from "@/types/api";

interface ScannedEntry {
  sku: string;
  name: string;
  quantity: number;
}

export default function PackingPage() {
  const [orderSearch, setOrderSearch] = useState("");
  const [orderSearchQuery, setOrderSearchQuery] = useState("");
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [scannedItems, setScannedItems] = useState<ScannedEntry[]>([]);
  const [barcodeInput, setBarcodeInput] = useState("");
  const [isPacking, setIsPacking] = useState(false);

  const barcodeRef = useRef<HTMLInputElement>(null);

  const { data: ordersData, isLoading: ordersLoading } = useOrders({
    search: orderSearchQuery,
    limit: 10,
  });

  // Auto-focus barcode input
  useEffect(() => {
    if (selectedOrder && barcodeRef.current) {
      barcodeRef.current.focus();
    }
  }, [selectedOrder]);

  const orderItems: OrderItem[] = selectedOrder?.items ?? [];

  const getExpectedQuantity = useCallback(
    (sku: string): number => {
      return orderItems
        .filter((item) => item.sku === sku)
        .reduce((sum, item) => sum + item.quantity, 0);
    },
    [orderItems]
  );

  const getScannedQuantity = useCallback(
    (sku: string): number => {
      const entry = scannedItems.find((s) => s.sku === sku);
      return entry?.quantity ?? 0;
    },
    [scannedItems]
  );

  const allItemsScanned = orderItems.length > 0 && orderItems.every((item) => {
    if (!item.sku) return true;
    return getScannedQuantity(item.sku) >= getExpectedQuantity(item.sku);
  });

  const handleOrderSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setOrderSearchQuery(orderSearch);
  };

  const handleSelectOrder = (order: Order) => {
    setSelectedOrder(order);
    setScannedItems([]);
    setBarcodeInput("");
  };

  const handleBarcodeScan = async (e: React.FormEvent) => {
    e.preventDefault();
    const code = barcodeInput.trim();
    if (!code) return;

    try {
      const resp = await apiClient<BarcodeLookupResponse>(
        `/v1/barcode/${encodeURIComponent(code)}`
      );

      let sku = "";
      let name = "";

      if (resp.product) {
        sku = resp.product.sku ?? "";
        name = resp.product.name;
      }
      if (resp.variants && resp.variants.length > 0) {
        const v = resp.variants[0];
        sku = v.sku ?? sku;
        name = v.name || name;
      }

      if (!sku) {
        toast.error("Produkt nie ma SKU, nie mozna dodac do pakowania");
        setBarcodeInput("");
        return;
      }

      // Check if this SKU is expected in the order
      const expected = getExpectedQuantity(sku);
      const alreadyScanned = getScannedQuantity(sku);

      if (expected === 0) {
        toast.error(`Produkt ${sku} nie znajduje sie w tym zamowieniu`);
        setBarcodeInput("");
        return;
      }

      if (alreadyScanned >= expected) {
        toast.warning(
          `Produkt ${sku} juz w pelni zeskanowany (${expected}/${expected})`
        );
        setBarcodeInput("");
        return;
      }

      setScannedItems((prev) => {
        const existing = prev.find((s) => s.sku === sku);
        if (existing) {
          return prev.map((s) =>
            s.sku === sku ? { ...s, quantity: s.quantity + 1 } : s
          );
        }
        return [...prev, { sku, name, quantity: 1 }];
      });

      toast.success(`Zeskanowano: ${name} (${sku})`);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }

    setBarcodeInput("");
    barcodeRef.current?.focus();
  };

  const handleConfirmPacking = async () => {
    if (!selectedOrder) return;
    setIsPacking(true);

    const packItems: ScannedItem[] = scannedItems.map((s) => ({
      sku: s.sku,
      quantity: s.quantity,
    }));

    try {
      await apiClient(`/v1/orders/${selectedOrder.id}/pack`, {
        method: "POST",
        body: JSON.stringify({ scanned_items: packItems }),
      });
      toast.success("Zamowienie zostalo spakowane!");
      setSelectedOrder(null);
      setScannedItems([]);
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setIsPacking(false);
    }
  };

  const handleReset = () => {
    setSelectedOrder(null);
    setScannedItems([]);
    setBarcodeInput("");
    setOrderSearch("");
    setOrderSearchQuery("");
  };

  return (
    <>
      <PageHeader
        title="Stacja pakowania"
        description="Skanuj kody kreskowe, aby potwierdzic zawartosc zamowienia"
      />

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left panel: Order selector */}
        <div className="lg:col-span-3">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Wybierz zamowienie</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleOrderSearch} className="flex gap-2 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="ID lub klient..."
                    value={orderSearch}
                    onChange={(e) => setOrderSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
                <Button type="submit" variant="outline" size="sm">
                  Szukaj
                </Button>
              </form>

              {ordersLoading && <LoadingSkeleton />}

              {selectedOrder && (
                <div className="p-3 rounded-md border border-primary bg-primary/5 mb-3">
                  <p className="text-sm font-medium">
                    {selectedOrder.customer_name}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    #{selectedOrder.id.slice(0, 8)}
                  </p>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="mt-1"
                    onClick={handleReset}
                  >
                    Zmien zamowienie
                  </Button>
                </div>
              )}

              {!selectedOrder && ordersData?.items && (
                <div className="space-y-1 max-h-[400px] overflow-y-auto">
                  {ordersData.items.map((order) => (
                    <button
                      key={order.id}
                      className="w-full text-left p-2 rounded-md hover:bg-muted/50 transition-colors text-sm"
                      onClick={() => handleSelectOrder(order)}
                    >
                      <p className="font-medium truncate">
                        {order.customer_name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        #{order.id.slice(0, 8)} &middot; {order.status}
                      </p>
                    </button>
                  ))}
                  {ordersData.items.length === 0 && (
                    <p className="text-sm text-muted-foreground text-center py-4">
                      Brak wynikow
                    </p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Center panel: Order items & scanning */}
        <div className="lg:col-span-6">
          {!selectedOrder ? (
            <Card>
              <CardContent className="py-12 text-center">
                <ScanBarcode className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-muted-foreground">
                  Wybierz zamowienie z panelu po lewej stronie, aby rozpoczac
                  pakowanie
                </p>
              </CardContent>
            </Card>
          ) : (
            <>
              {/* Barcode input */}
              <Card className="mb-4">
                <CardContent className="pt-6">
                  <form onSubmit={handleBarcodeScan} className="flex gap-2">
                    <div className="relative flex-1">
                      <ScanBarcode className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                      <Input
                        ref={barcodeRef}
                        placeholder="Skanuj kod kreskowy lub wpisz SKU/EAN..."
                        value={barcodeInput}
                        onChange={(e) => setBarcodeInput(e.target.value)}
                        className="pl-11 text-lg h-12"
                        autoFocus
                      />
                    </div>
                    <Button type="submit" size="lg">
                      Skanuj
                    </Button>
                  </form>
                </CardContent>
              </Card>

              {/* Items list */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    Pozycje zamowienia
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {orderItems.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      Brak pozycji w zamowieniu
                    </p>
                  ) : (
                    <div className="space-y-2">
                      {orderItems.map((item, idx) => {
                        const expected = item.sku
                          ? getExpectedQuantity(item.sku)
                          : item.quantity;
                        const scanned = item.sku
                          ? getScannedQuantity(item.sku)
                          : 0;
                        const isComplete = scanned >= expected;

                        return (
                          <div
                            key={`${item.sku}-${idx}`}
                            className={`flex items-center justify-between p-3 rounded-md border ${
                              isComplete
                                ? "border-green-300 bg-green-50 dark:border-green-800 dark:bg-green-950"
                                : "border-border"
                            }`}
                          >
                            <div className="flex items-center gap-3">
                              {isComplete ? (
                                <Check className="h-5 w-5 text-green-600" />
                              ) : (
                                <Package className="h-5 w-5 text-muted-foreground" />
                              )}
                              <div>
                                <p className="font-medium text-sm">
                                  {item.name}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  SKU: {item.sku || "---"}
                                </p>
                              </div>
                            </div>
                            <Badge
                              variant={isComplete ? "default" : "outline"}
                              className={
                                isComplete
                                  ? "bg-green-600"
                                  : ""
                              }
                            >
                              {scanned} / {expected}
                            </Badge>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </CardContent>
              </Card>
            </>
          )}
        </div>

        {/* Right panel: Tally & confirm */}
        <div className="lg:col-span-3">
          {selectedOrder && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Podsumowanie</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  {scannedItems.map((entry) => (
                    <div
                      key={entry.sku}
                      className="flex justify-between text-sm"
                    >
                      <span className="truncate mr-2">{entry.name}</span>
                      <span className="font-medium whitespace-nowrap">
                        {entry.quantity}x
                      </span>
                    </div>
                  ))}
                  {scannedItems.length === 0 && (
                    <p className="text-sm text-muted-foreground">
                      Brak zeskanowanych produktow
                    </p>
                  )}
                </div>

                <div className="pt-2 border-t">
                  <div className="flex justify-between text-sm mb-1">
                    <span>Zeskanowano pozycji:</span>
                    <span className="font-medium">
                      {scannedItems.reduce((s, e) => s + e.quantity, 0)}
                    </span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span>Oczekiwano pozycji:</span>
                    <span className="font-medium">
                      {orderItems.reduce((s, item) => s + item.quantity, 0)}
                    </span>
                  </div>
                </div>

                {allItemsScanned && (
                  <div className="p-3 rounded-md bg-green-50 border border-green-200 dark:bg-green-950 dark:border-green-800">
                    <p className="text-sm text-green-700 dark:text-green-300 font-medium flex items-center gap-2">
                      <Check className="h-4 w-4" />
                      Wszystkie pozycje zeskanowane
                    </p>
                  </div>
                )}

                <Button
                  className="w-full"
                  size="lg"
                  disabled={!allItemsScanned || isPacking}
                  onClick={handleConfirmPacking}
                >
                  {isPacking ? "Pakowanie..." : "Potwierdz pakowanie"}
                </Button>

                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => setScannedItems([])}
                >
                  <X className="h-4 w-4 mr-2" />
                  Wyczysc skanowanie
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </>
  );
}
