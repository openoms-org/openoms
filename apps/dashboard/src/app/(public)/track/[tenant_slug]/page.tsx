"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { formatCurrency, formatDate } from "@/lib/utils";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// --- Types ---

interface TrackingItem {
  name: string;
  quantity: number;
  price: number;
}

interface TrackingShipment {
  tracking_number: string;
  carrier: string;
  status: string;
}

interface TrackingEvent {
  status: string;
  timestamp: string;
}

interface TrackingResponse {
  order_number: string;
  status: string;
  status_label: string;
  customer_name: string;
  created_at: string;
  updated_at: string;
  total_amount: number;
  currency: string;
  items: TrackingItem[];
  shipments: TrackingShipment[];
  timeline: TrackingEvent[];
  company_name?: string;
  company_logo?: string;
}

// --- Status color map ---

const statusColors: Record<string, string> = {
  new: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  confirmed:
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300",
  processing:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
  ready_to_ship:
    "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
  shipped:
    "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  in_transit:
    "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  out_for_delivery:
    "bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300",
  delivered:
    "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  completed:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
  on_hold:
    "bg-gray-100 text-gray-800 dark:bg-gray-700/30 dark:text-gray-300",
  cancelled:
    "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
  refunded:
    "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
};

function getStatusColor(status: string): string {
  return (
    statusColors[status] ||
    "bg-gray-100 text-gray-800 dark:bg-gray-700/30 dark:text-gray-300"
  );
}

// --- Page Component ---

export default function TrackingPage() {
  const params = useParams();
  const tenantSlug = params.tenant_slug as string;

  const [orderId, setOrderId] = useState("");
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<TrackingResponse | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setResult(null);

    if (!orderId.trim() || !email.trim()) {
      setError("Podaj numer zamowienia i adres email.");
      return;
    }

    setLoading(true);
    try {
      const res = await fetch(
        `${API_URL}/v1/tracking/${encodeURIComponent(tenantSlug)}/${encodeURIComponent(orderId.trim())}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email: email.trim() }),
        }
      );
      if (res.status === 404) {
        setError("Nie znaleziono zamowienia. Sprawdz numer zamowienia i email.");
        return;
      }
      if (res.status === 403) {
        setError("Podany email nie pasuje do zamowienia.");
        return;
      }
      if (res.status === 429) {
        setError("Zbyt wiele prob. Sprobuj ponownie za minute.");
        return;
      }
      if (!res.ok) {
        setError("Wystapil blad. Sprobuj ponownie pozniej.");
        return;
      }
      const data: TrackingResponse = await res.json();
      setResult(data);
    } catch {
      setError("Nie udalo sie polaczyc z serwerem. Sprobuj ponownie.");
    } finally {
      setLoading(false);
    }
  }

  function handleReset() {
    setResult(null);
    setError(null);
    setOrderId("");
    setEmail("");
  }

  return (
    <div className="min-h-screen flex flex-col items-center px-4 py-8 sm:py-16">
      {/* Header / Branding */}
      <div className="w-full max-w-2xl mb-8 text-center">
        {result?.company_logo ? (
          <img
            src={result.company_logo}
            alt={result.company_name || "Logo"}
            className="mx-auto h-12 w-auto mb-4"
          />
        ) : null}
        <h1 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-slate-100">
          {result?.company_name || "Sledz zamowienie"}
        </h1>
        {!result && (
          <p className="mt-2 text-slate-600 dark:text-slate-400">
            Wpisz numer zamowienia i adres email, aby sprawdzic status.
          </p>
        )}
      </div>

      {/* Search Form */}
      {!result && (
        <form
          onSubmit={handleSubmit}
          className="w-full max-w-md bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6 space-y-4"
        >
          <div>
            <label
              htmlFor="orderId"
              className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
            >
              Numer zamowienia (ID)
            </label>
            <input
              id="orderId"
              type="text"
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
              placeholder="np. a1b2c3d4-..."
              className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-4 py-2.5 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoComplete="off"
            />
          </div>
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
            >
              Adres email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="twoj@email.pl"
              className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-4 py-2.5 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoComplete="email"
            />
          </div>

          {error && (
            <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3 text-sm text-red-700 dark:text-red-300">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2.5 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            {loading ? "Szukam..." : "Sprawdz status"}
          </button>
        </form>
      )}

      {/* Results */}
      {result && (
        <div className="w-full max-w-2xl space-y-6">
          {/* Order Summary Card */}
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
              <div>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  Zamowienie
                </p>
                <p className="text-lg font-semibold text-slate-900 dark:text-slate-100 font-mono">
                  #{result.order_number.substring(0, 8).toUpperCase()}
                </p>
              </div>
              <span
                className={`inline-flex items-center self-start rounded-full px-3 py-1 text-sm font-medium ${getStatusColor(result.status)}`}
              >
                {result.status_label}
              </span>
            </div>

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-slate-500 dark:text-slate-400">Klient</p>
                <p className="font-medium text-slate-900 dark:text-slate-100">
                  {result.customer_name}
                </p>
              </div>
              <div>
                <p className="text-slate-500 dark:text-slate-400">Kwota</p>
                <p className="font-medium text-slate-900 dark:text-slate-100">
                  {formatCurrency(result.total_amount, result.currency)}
                </p>
              </div>
              <div>
                <p className="text-slate-500 dark:text-slate-400">
                  Data zlozenia
                </p>
                <p className="font-medium text-slate-900 dark:text-slate-100">
                  {formatDate(result.created_at)}
                </p>
              </div>
              <div>
                <p className="text-slate-500 dark:text-slate-400">
                  Ostatnia aktualizacja
                </p>
                <p className="font-medium text-slate-900 dark:text-slate-100">
                  {formatDate(result.updated_at)}
                </p>
              </div>
            </div>
          </div>

          {/* Order Items */}
          {result.items.length > 0 && (
            <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
                Produkty
              </h2>
              <div className="divide-y divide-slate-200 dark:divide-slate-700">
                {result.items.map((item, i) => (
                  <div
                    key={i}
                    className="flex items-center justify-between py-3 first:pt-0 last:pb-0"
                  >
                    <div>
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        {item.name}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        Ilosc: {item.quantity}
                      </p>
                    </div>
                    <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                      {formatCurrency(item.price, result.currency)}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Shipments */}
          {result.shipments.length > 0 && (
            <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
                Przesylki
              </h2>
              <div className="space-y-3">
                {result.shipments.map((shipment, i) => (
                  <div
                    key={i}
                    className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 rounded-lg border border-slate-200 dark:border-slate-700 p-4"
                  >
                    <div>
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        {shipment.carrier}
                      </p>
                      {shipment.tracking_number && (
                        <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                          {shipment.tracking_number}
                        </p>
                      )}
                    </div>
                    <span
                      className={`inline-flex items-center self-start rounded-full px-2.5 py-0.5 text-xs font-medium ${getStatusColor(shipment.status)}`}
                    >
                      {shipment.status}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Timeline */}
          {result.timeline.length > 0 && (
            <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
                Historia statusow
              </h2>
              <div className="relative">
                <div className="absolute left-3.5 top-0 bottom-0 w-px bg-slate-200 dark:bg-slate-700" />
                <div className="space-y-4">
                  {result.timeline.map((event, i) => (
                    <div key={i} className="relative flex items-start gap-4">
                      <div
                        className={`relative z-10 h-7 w-7 flex-shrink-0 rounded-full border-2 border-white dark:border-slate-800 ${
                          i === 0
                            ? "bg-blue-500"
                            : "bg-slate-300 dark:bg-slate-600"
                        }`}
                      />
                      <div className="pt-0.5">
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                          {event.status}
                        </p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                          {formatDate(event.timestamp)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Back button */}
          <div className="text-center">
            <button
              onClick={handleReset}
              className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              Sprawdz inne zamowienie
            </button>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="mt-auto pt-12 pb-6 text-center text-xs text-slate-400 dark:text-slate-500">
        Powered by OpenOMS
      </footer>
    </div>
  );
}
