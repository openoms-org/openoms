"use client";

import { Suspense, useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { formatCurrency, formatDate } from "@/lib/utils";
import { useTranslations } from "next-intl";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// --- Types ---

interface SupplierPortalPO {
  id: string;
  po_number: string;
  status: string;
  notes?: string;
  expected_delivery_date?: string;
  total_amount: number;
  currency: string;
  items?: POItem[];
  created_at: string;
  updated_at: string;
}

interface POItem {
  id: string;
  sku: string;
  product_name: string;
  quantity_ordered: number;
  quantity_received: number;
  unit_cost: number;
  total_cost: number;
  notes?: string;
}

interface SupplierMessage {
  id: string;
  message: string;
  is_from_supplier: boolean;
  created_at: string;
}

interface PortalData {
  supplier: { id: string; name: string };
  orders: SupplierPortalPO[];
}

// --- Status helpers ---

const poStatusLabels: Record<string, string> = {
  sent: "Wyslane",
  confirmed: "Potwierdzone",
  shipped: "Wyslane (dostawca)",
  partially_received: "Czesciowo odebrane",
  received: "Odebrane",
  cancelled: "Anulowane",
};

const poStatusColors: Record<string, string> = {
  sent: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  confirmed: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300",
  shipped: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  partially_received: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
  received: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  cancelled: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
};

function getStatusColor(status: string): string {
  return poStatusColors[status] || "bg-gray-100 text-gray-800 dark:bg-gray-700/30 dark:text-gray-300";
}

function getStatusLabel(status: string): string {
  return poStatusLabels[status] || status;
}

// --- API helpers ---

async function portalFetch<T>(path: string, token: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

// --- Components ---

function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getStatusColor(status)}`}>
      {getStatusLabel(status)}
    </span>
  );
}

function OrderList({
  orders,
  onSelect,
}: {
  orders: SupplierPortalPO[];
  onSelect: (id: string) => void;
}) {
  if (orders.length === 0) {
    return (
      <div className="text-center py-12 text-slate-500 dark:text-slate-400">
        Brak zamowien zakupowych.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">Nr PO</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">Status</th>
            <th className="text-right py-3 px-4 font-medium text-slate-600 dark:text-slate-400">Kwota</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">Oczekiwana data</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">Data utworzenia</th>
            <th className="py-3 px-4"></th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
          {orders.map((order) => (
            <tr
              key={order.id}
              className="hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
              onClick={() => onSelect(order.id)}
            >
              <td className="py-3 px-4 font-mono font-medium text-slate-900 dark:text-slate-100">
                {order.po_number}
              </td>
              <td className="py-3 px-4">
                <StatusBadge status={order.status} />
              </td>
              <td className="py-3 px-4 text-right text-slate-900 dark:text-slate-100">
                {formatCurrency(order.total_amount, order.currency)}
              </td>
              <td className="py-3 px-4 text-slate-600 dark:text-slate-400">
                {order.expected_delivery_date ? formatDate(order.expected_delivery_date) : "---"}
              </td>
              <td className="py-3 px-4 text-slate-600 dark:text-slate-400">
                {formatDate(order.created_at)}
              </td>
              <td className="py-3 px-4 text-right">
                <span className="text-blue-600 dark:text-blue-400 text-xs">Szczegoly &rarr;</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function OrderDetail({
  token,
  orderId,
  onBack,
}: {
  token: string;
  orderId: string;
  onBack: () => void;
}) {
  const [order, setOrder] = useState<SupplierPortalPO | null>(null);
  const [messages, setMessages] = useState<SupplierMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newMessage, setNewMessage] = useState("");
  const [sendingMessage, setSendingMessage] = useState(false);

  // Confirm dialog
  const [showConfirm, setShowConfirm] = useState(false);
  const [confirmRef, setConfirmRef] = useState("");
  const [confirming, setConfirming] = useState(false);

  // Ship dialog
  const [showShip, setShowShip] = useState(false);
  const [trackingNumber, setTrackingNumber] = useState("");
  const [carrier, setCarrier] = useState("");
  const [shipping, setShipping] = useState(false);

  const loadOrder = useCallback(async () => {
    try {
      const [po, msgs] = await Promise.all([
        portalFetch<SupplierPortalPO>(`/v1/supplier-portal/orders/${orderId}`, token),
        portalFetch<SupplierMessage[]>(`/v1/supplier-portal/orders/${orderId}/messages`, token),
      ]);
      setOrder(po);
      setMessages(msgs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Wystapil blad");
    } finally {
      setLoading(false);
    }
  }, [orderId, token]);

  useEffect(() => {
    loadOrder();
  }, [loadOrder]);

  const handleConfirm = async () => {
    setConfirming(true);
    try {
      await portalFetch(`/v1/supplier-portal/orders/${orderId}/confirm`, token, {
        method: "POST",
        body: JSON.stringify({ reference: confirmRef }),
      });
      setShowConfirm(false);
      setConfirmRef("");
      await loadOrder();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nie udalo sie potwierdzic");
    } finally {
      setConfirming(false);
    }
  };

  const handleShip = async () => {
    if (!trackingNumber.trim() || !carrier.trim()) return;
    setShipping(true);
    try {
      await portalFetch(`/v1/supplier-portal/orders/${orderId}/ship`, token, {
        method: "POST",
        body: JSON.stringify({ tracking_number: trackingNumber.trim(), carrier: carrier.trim() }),
      });
      setShowShip(false);
      setTrackingNumber("");
      setCarrier("");
      await loadOrder();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nie udalo sie oznaczyc wysylki");
    } finally {
      setShipping(false);
    }
  };

  const handleSendMessage = async () => {
    if (!newMessage.trim()) return;
    setSendingMessage(true);
    try {
      await portalFetch(`/v1/supplier-portal/orders/${orderId}/messages`, token, {
        method: "POST",
        body: JSON.stringify({ message: newMessage.trim() }),
      });
      setNewMessage("");
      const msgs = await portalFetch<SupplierMessage[]>(`/v1/supplier-portal/orders/${orderId}/messages`, token);
      setMessages(msgs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nie udalo sie wyslac wiadomosci");
    } finally {
      setSendingMessage(false);
    }
  };

  if (loading) {
    return <div className="text-center py-12 text-slate-500">Ladowanie...</div>;
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4 text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
        <button onClick={onBack} className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
          &larr; Powrot do listy
        </button>
      </div>
    );
  }

  if (!order) return null;

  const canConfirm = order.status === "sent";
  const canShip = order.status === "sent" || order.status === "confirmed";

  return (
    <div className="space-y-6">
      {/* Back button + header */}
      <div className="flex items-center gap-4">
        <button
          onClick={onBack}
          className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
        >
          &larr; Powrot
        </button>
        <div className="flex-1">
          <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100 font-mono">
            {order.po_number}
          </h2>
        </div>
        <StatusBadge status={order.status} />
      </div>

      {/* Order details */}
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm mb-6">
          <div>
            <p className="text-slate-500 dark:text-slate-400">Kwota</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatCurrency(order.total_amount, order.currency)}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">Oczekiwana data</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {order.expected_delivery_date ? formatDate(order.expected_delivery_date) : "---"}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">Utworzono</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatDate(order.created_at)}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">Aktualizacja</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatDate(order.updated_at)}
            </p>
          </div>
        </div>

        {order.notes && (
          <div className="rounded-lg bg-slate-50 dark:bg-slate-700/50 p-3 mb-6">
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Notatki</p>
            <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">{order.notes}</p>
          </div>
        )}

        {/* Actions */}
        {(canConfirm || canShip) && (
          <div className="flex gap-3 mb-6">
            {canConfirm && (
              <button
                onClick={() => setShowConfirm(true)}
                className="rounded-lg bg-blue-600 hover:bg-blue-700 px-4 py-2 text-sm font-medium text-white transition-colors"
              >
                Potwierdz zamowienie
              </button>
            )}
            {canShip && (
              <button
                onClick={() => setShowShip(true)}
                className="rounded-lg bg-purple-600 hover:bg-purple-700 px-4 py-2 text-sm font-medium text-white transition-colors"
              >
                Oznacz jako wyslane
              </button>
            )}
          </div>
        )}

        {/* Items table */}
        {order.items && order.items.length > 0 && (
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Pozycje</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left py-2 px-3 font-medium text-slate-600 dark:text-slate-400">Produkt</th>
                    <th className="text-left py-2 px-3 font-medium text-slate-600 dark:text-slate-400">SKU</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">Zamowiono</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">Odebrano</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">Cena jedn.</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">Razem</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {order.items.map((item) => (
                    <tr key={item.id}>
                      <td className="py-2 px-3 text-slate-900 dark:text-slate-100 max-w-[200px] truncate">
                        {item.product_name}
                      </td>
                      <td className="py-2 px-3 text-slate-600 dark:text-slate-400 font-mono">
                        {item.sku || "---"}
                      </td>
                      <td className="py-2 px-3 text-right text-slate-900 dark:text-slate-100">
                        {item.quantity_ordered}
                      </td>
                      <td className="py-2 px-3 text-right text-slate-900 dark:text-slate-100">
                        {item.quantity_received}
                      </td>
                      <td className="py-2 px-3 text-right text-slate-900 dark:text-slate-100">
                        {formatCurrency(item.unit_cost, order.currency)}
                      </td>
                      <td className="py-2 px-3 text-right font-medium text-slate-900 dark:text-slate-100">
                        {formatCurrency(item.unit_cost * item.quantity_ordered, order.currency)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Messages */}
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-4">Wiadomosci</h3>
        <div className="space-y-3 max-h-64 overflow-y-auto mb-4">
          {messages.length === 0 ? (
            <p className="text-sm text-slate-500 dark:text-slate-400 text-center py-4">
              Brak wiadomosci. Napisz pierwsza wiadomosc.
            </p>
          ) : (
            messages.map((msg) => (
              <div
                key={msg.id}
                className={`rounded-lg p-3 text-sm ${
                  msg.is_from_supplier
                    ? "bg-blue-50 dark:bg-blue-900/20 ml-8"
                    : "bg-slate-50 dark:bg-slate-700/50 mr-8"
                }`}
              >
                <div className="flex justify-between items-center mb-1">
                  <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
                    {msg.is_from_supplier ? "Dostawca" : "Kupujacy"}
                  </span>
                  <span className="text-xs text-slate-400 dark:text-slate-500">
                    {formatDate(msg.created_at)}
                  </span>
                </div>
                <p className="text-slate-700 dark:text-slate-300 whitespace-pre-wrap">{msg.message}</p>
              </div>
            ))
          )}
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={newMessage}
            onChange={(e) => setNewMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSendMessage();
              }
            }}
            placeholder="Napisz wiadomosc..."
            className="flex-1 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={handleSendMessage}
            disabled={sendingMessage || !newMessage.trim()}
            className="rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            {sendingMessage ? "..." : "Wyslij"}
          </button>
        </div>
      </div>

      {/* Confirm Dialog */}
      {showConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-6 w-full max-w-md mx-4">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
              Potwierdz zamowienie
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Numer referencyjny (opcjonalnie)
                </label>
                <input
                  type="text"
                  value={confirmRef}
                  onChange={(e) => setConfirmRef(e.target.value)}
                  placeholder="np. numer potwierdzenia dostawcy"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => { setShowConfirm(false); setConfirmRef(""); }}
                  className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  Anuluj
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={confirming}
                  className="rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2 text-sm font-medium text-white transition-colors"
                >
                  {confirming ? "Potwierdzanie..." : "Potwierdz"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Ship Dialog */}
      {showShip && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-6 w-full max-w-md mx-4">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
              Oznacz jako wyslane
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Numer sledzenia *
                </label>
                <input
                  type="text"
                  value={trackingNumber}
                  onChange={(e) => setTrackingNumber(e.target.value)}
                  placeholder="np. 1Z999AA10123456784"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Przewoznik *
                </label>
                <input
                  type="text"
                  value={carrier}
                  onChange={(e) => setCarrier(e.target.value)}
                  placeholder="np. DHL, InPost, DPD"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => { setShowShip(false); setTrackingNumber(""); setCarrier(""); }}
                  className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  Anuluj
                </button>
                <button
                  onClick={handleShip}
                  disabled={shipping || !trackingNumber.trim() || !carrier.trim()}
                  className="rounded-lg bg-purple-600 hover:bg-purple-700 disabled:bg-purple-400 px-4 py-2 text-sm font-medium text-white transition-colors"
                >
                  {shipping ? "Zapisywanie..." : "Oznacz wyslane"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// --- Main Page ---

export default function SupplierPortalPage() {
  const t = useTranslations("suppliers");
  return (
    <Suspense fallback={<div className="flex min-h-screen items-center justify-center"><p className="text-muted-foreground">{t("loading")}</p></div>}>
      <SupplierPortalContent />
    </Suspense>
  );
}

function SupplierPortalContent() {
  const searchParams = useSearchParams();
  const [token, setToken] = useState("");
  const [tokenReady, setTokenReady] = useState(false);

  const [portalData, setPortalData] = useState<PortalData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);

  // One-time token hand-off: read from URL, store in sessionStorage, clean URL
  // so the token doesn't leak via proxy logs, browser history, or Referer (OPE-169).
  useEffect(() => {
    const urlToken = searchParams.get("token");
    const stored = typeof window !== "undefined" ? sessionStorage.getItem("supplier_portal_token") : null;
    if (urlToken) {
      sessionStorage.setItem("supplier_portal_token", urlToken);
      setToken(urlToken);
      // Clean token from URL to prevent leak via history/Referer
      window.history.replaceState({}, "", window.location.pathname);
    } else if (stored) {
      setToken(stored);
    }
    setTokenReady(true);
  }, [searchParams]);

  useEffect(() => {
    if (!tokenReady) return;
    if (!token) {
      setError("Brak tokenu dostepu. Sprawdz poprawnosc linku.");
      setLoading(false);
      return;
    }

    async function loadOrders() {
      try {
        const data = await portalFetch<PortalData>("/v1/supplier-portal/orders", token);
        setPortalData(data);
      } catch (err) {
        if (err instanceof Error) {
          if (err.message.includes("expired")) {
            setError("Token dostepu wygasl. Poprosi o nowy link.");
          } else if (err.message.includes("invalid") || err.message.includes("disabled")) {
            setError("Nieprawidlowy token lub dostep zostal wylaczony.");
          } else {
            setError(err.message);
          }
        } else {
          setError("Wystapil blad. Sprobuj ponownie pozniej.");
        }
      } finally {
        setLoading(false);
      }
    }

    loadOrders();
  }, [token, tokenReady]);

  return (
    <div className="min-h-screen flex flex-col items-center px-4 py-8 sm:py-16">
      <div className="w-full max-w-4xl">
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-slate-100">
            Portal Dostawcy
          </h1>
          {portalData && (
            <p className="mt-2 text-slate-600 dark:text-slate-400">
              Witaj, <span className="font-medium">{portalData.supplier.name}</span>
            </p>
          )}
        </div>

        {/* Loading */}
        {loading && (
          <div className="text-center py-12 text-slate-500 dark:text-slate-400">
            Ladowanie...
          </div>
        )}

        {/* Error */}
        {error && !loading && (
          <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-6 text-center">
            <p className="text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}

        {/* Content */}
        {!loading && !error && portalData && (
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg">
            {selectedOrderId ? (
              <div className="p-6">
                <OrderDetail
                  token={token}
                  orderId={selectedOrderId}
                  onBack={() => setSelectedOrderId(null)}
                />
              </div>
            ) : (
              <div>
                <div className="p-6 border-b border-slate-200 dark:border-slate-700">
                  <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                    Zamowienia zakupowe ({portalData.orders.length})
                  </h2>
                </div>
                <OrderList
                  orders={portalData.orders}
                  onSelect={(id) => setSelectedOrderId(id)}
                />
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      <footer className="mt-auto pt-12 pb-6 text-center text-xs text-slate-400 dark:text-slate-500">
        Powered by OpenOMS
      </footer>
    </div>
  );
}
