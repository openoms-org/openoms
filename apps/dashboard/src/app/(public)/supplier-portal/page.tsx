"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { StatusBadge } from "@/components/shared/status-badge";
import { API_URL } from "@/lib/api-client";
import { PURCHASE_ORDER_STATUSES } from "@/lib/constants";
import { formatCurrency, formatDate } from "@/lib/utils";
import {
  getSupplierPortalTokenHandoff,
  SUPPLIER_PORTAL_TOKEN_STORAGE_KEY,
} from "@/lib/supplier-portal-token";

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

function OrderList({
  orders,
  onSelect,
}: {
  orders: SupplierPortalPO[];
  onSelect: (id: string) => void;
}) {
  const t = useTranslations("supplierPortal.orders");

  if (orders.length === 0) {
    return (
      <div className="text-center py-12 text-slate-500 dark:text-slate-400">
        {t("empty")}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">{t("columns.poNumber")}</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">{t("columns.status")}</th>
            <th className="text-right py-3 px-4 font-medium text-slate-600 dark:text-slate-400">{t("columns.amount")}</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">{t("columns.expectedDate")}</th>
            <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-400">{t("columns.createdAt")}</th>
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
                <StatusBadge status={order.status} statusMap={PURCHASE_ORDER_STATUSES} translationPrefix="purchaseOrder" />
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
                <span className="text-blue-600 dark:text-blue-400 text-xs">{t("details")} &rarr;</span>
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
  const t = useTranslations("supplierPortal.detail");
  const genericError = t("errors.generic");
  const confirmError = t("errors.confirm");
  const shipError = t("errors.ship");
  const messageError = t("errors.message");
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
      setError(err instanceof Error ? err.message : genericError);
    } finally {
      setLoading(false);
    }
  }, [genericError, orderId, token]);

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
      setError(err instanceof Error ? err.message : confirmError);
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
      setError(err instanceof Error ? err.message : shipError);
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
      setError(err instanceof Error ? err.message : messageError);
    } finally {
      setSendingMessage(false);
    }
  };

  if (loading) {
    return <div className="text-center py-12 text-slate-500">{t("loading")}</div>;
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4 text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
        <button onClick={onBack} className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
          &larr; {t("backToList")}
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
          &larr; {t("back")}
        </button>
        <div className="flex-1">
          <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100 font-mono">
            {order.po_number}
          </h2>
        </div>
        <StatusBadge status={order.status} statusMap={PURCHASE_ORDER_STATUSES} translationPrefix="purchaseOrder" />
      </div>

      {/* Order details */}
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm mb-6">
          <div>
            <p className="text-slate-500 dark:text-slate-400">{t("summary.amount")}</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatCurrency(order.total_amount, order.currency)}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">{t("summary.expectedDate")}</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {order.expected_delivery_date ? formatDate(order.expected_delivery_date) : "---"}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">{t("summary.createdAt")}</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatDate(order.created_at)}
            </p>
          </div>
          <div>
            <p className="text-slate-500 dark:text-slate-400">{t("summary.updatedAt")}</p>
            <p className="font-medium text-slate-900 dark:text-slate-100">
              {formatDate(order.updated_at)}
            </p>
          </div>
        </div>

        {order.notes && (
          <div className="rounded-lg bg-slate-50 dark:bg-slate-700/50 p-3 mb-6">
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">{t("summary.notes")}</p>
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
                {t("actions.confirmOrder")}
              </button>
            )}
            {canShip && (
              <button
                onClick={() => setShowShip(true)}
                className="rounded-lg bg-purple-600 hover:bg-purple-700 px-4 py-2 text-sm font-medium text-white transition-colors"
              >
                {t("actions.markShipped")}
              </button>
            )}
          </div>
        )}

        {/* Items table */}
        {order.items && order.items.length > 0 && (
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">{t("items.title")}</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.product")}</th>
                    <th className="text-left py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.sku")}</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.ordered")}</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.received")}</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.unitPrice")}</th>
                    <th className="text-right py-2 px-3 font-medium text-slate-600 dark:text-slate-400">{t("items.columns.total")}</th>
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
        <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-4">{t("messages.title")}</h3>
        <div className="space-y-3 max-h-64 overflow-y-auto mb-4">
          {messages.length === 0 ? (
            <p className="text-sm text-slate-500 dark:text-slate-400 text-center py-4">
              {t("messages.empty")}
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
                    {msg.is_from_supplier ? t("messages.fromSupplier") : t("messages.fromBuyer")}
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
            placeholder={t("messages.placeholder")}
            className="flex-1 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={handleSendMessage}
            disabled={sendingMessage || !newMessage.trim()}
            className="rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            {sendingMessage ? t("messages.sending") : t("messages.send")}
          </button>
        </div>
      </div>

      {/* Confirm Dialog */}
      {showConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-6 w-full max-w-md mx-4">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
              {t("confirmDialog.title")}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t("confirmDialog.referenceLabel")}
                </label>
                <input
                  type="text"
                  value={confirmRef}
                  onChange={(e) => setConfirmRef(e.target.value)}
                  placeholder={t("confirmDialog.referencePlaceholder")}
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => { setShowConfirm(false); setConfirmRef(""); }}
                  className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  {t("confirmDialog.cancel")}
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={confirming}
                  className="rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2 text-sm font-medium text-white transition-colors"
                >
                  {confirming ? t("confirmDialog.confirming") : t("confirmDialog.confirm")}
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
              {t("shipDialog.title")}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t("shipDialog.trackingLabel")}
                </label>
                <input
                  type="text"
                  value={trackingNumber}
                  onChange={(e) => setTrackingNumber(e.target.value)}
                  placeholder={t("shipDialog.trackingPlaceholder")}
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t("shipDialog.carrierLabel")}
                </label>
                <input
                  type="text"
                  value={carrier}
                  onChange={(e) => setCarrier(e.target.value)}
                  placeholder={t("shipDialog.carrierPlaceholder")}
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => { setShowShip(false); setTrackingNumber(""); setCarrier(""); }}
                  className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  {t("shipDialog.cancel")}
                </button>
                <button
                  onClick={handleShip}
                  disabled={shipping || !trackingNumber.trim() || !carrier.trim()}
                  className="rounded-lg bg-purple-600 hover:bg-purple-700 disabled:bg-purple-400 px-4 py-2 text-sm font-medium text-white transition-colors"
                >
                  {shipping ? t("shipDialog.saving") : t("shipDialog.confirm")}
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
  return <SupplierPortalContent />;
}

function SupplierPortalContent() {
  const t = useTranslations("supplierPortal");
  const missingTokenError = t("errors.missingToken");
  const expiredTokenError = t("errors.expiredToken");
  const invalidTokenError = t("errors.invalidToken");
  const genericError = t("errors.generic");
  const [token, setToken] = useState("");
  const [tokenReady, setTokenReady] = useState(false);

  const [portalData, setPortalData] = useState<PortalData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);

  // One-time token hand-off: read from the URL fragment, store in sessionStorage,
  // and scrub token material from browser history before API calls.
  useEffect(() => {
    const { token: fragmentToken, cleanURL } = getSupplierPortalTokenHandoff(window.location.href);
    const stored = sessionStorage.getItem(SUPPLIER_PORTAL_TOKEN_STORAGE_KEY);
    if (fragmentToken) {
      sessionStorage.setItem(SUPPLIER_PORTAL_TOKEN_STORAGE_KEY, fragmentToken);
      setToken(fragmentToken);
    } else if (stored) {
      setToken(stored);
    }
    if (cleanURL) {
      window.history.replaceState({}, "", cleanURL);
    }
    setTokenReady(true);
  }, []);

  useEffect(() => {
    if (!tokenReady) return;
    if (!token) {
      setError(missingTokenError);
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
            setError(expiredTokenError);
          } else if (err.message.includes("invalid") || err.message.includes("disabled")) {
            setError(invalidTokenError);
          } else {
            setError(err.message);
          }
        } else {
          setError(genericError);
        }
      } finally {
        setLoading(false);
      }
    }

    loadOrders();
  }, [
    expiredTokenError,
    genericError,
    invalidTokenError,
    missingTokenError,
    token,
    tokenReady,
  ]);

  return (
    <div className="min-h-screen flex flex-col items-center px-4 py-8 sm:py-16">
      <div className="w-full max-w-4xl">
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-slate-100">
            {t("title")}
          </h1>
          {portalData && (
            <p className="mt-2 text-slate-600 dark:text-slate-400">
              {t("welcome")} <span className="font-medium">{portalData.supplier.name}</span>
            </p>
          )}
        </div>

        {/* Loading */}
        {loading && (
          <div className="text-center py-12 text-slate-500 dark:text-slate-400">
            {t("loading")}
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
                    {t("orders.title", { count: portalData.orders.length })}
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
