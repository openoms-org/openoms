"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/auth";
import { API_URL, apiFetch } from "@/lib/api-client";
import type { WSEvent } from "@/types/api";

// Convert http(s) to ws(s) with ticket-based auth
function getWSUrl(ticket: string): string {
  const base = API_URL.replace(/^http/, "ws");
  return `${base}/v1/ws?ticket=${encodeURIComponent(ticket)}`;
}

// Map event types to React Query cache keys to invalidate
const EVENT_INVALIDATION_MAP: Record<string, string[][]> = {
  "order.created": [["orders"]],
  "order.updated": [["orders"]],
  "order.deleted": [["orders"]],
  "order.status_changed": [["orders"], ["stats"]],
  "shipment.created": [["shipments"]],
  "shipment.updated": [["shipments"]],
  "shipment.deleted": [["shipments"]],
  "shipment.status_changed": [["shipments"]],
  "product.created": [["products"]],
  "product.updated": [["products"], ["product-stock"]],
  "product.deleted": [["products"]],
  "return.created": [["returns"]],
  "return.updated": [["returns"]],
  "return.deleted": [["returns"]],
  "return.status_changed": [["returns"]],
  "stock.changed": [["warehouse-stock"], ["products"]],
  "warehouse_document.created": [["warehouse-documents"]],
  "warehouse_document.confirmed": [["warehouse-documents"], ["warehouse-stock"], ["product-stock"]],
  "warehouse_document.cancelled": [["warehouse-documents"]],
  "customer.created": [["customers"]],
  "customer.updated": [["customers"]],
  "customer.deleted": [["customers"]],
  "supplier.created": [["suppliers"]],
  "supplier.updated": [["suppliers"]],
  "supplier.deleted": [["suppliers"]],
  "recurring_order.created": [["recurring-orders"]],
  "recurring_order.updated": [["recurring-orders"]],
  "recurring_order.deleted": [["recurring-orders"]],
  "purchase_order.created": [["purchase-orders"]],
  "purchase_order.updated": [["purchase-orders"]],
  "purchase_order.deleted": [["purchase-orders"]],
  "purchase_order.sent": [["purchase-orders"]],
  "purchase_order.cancelled": [["purchase-orders"]],
  "purchase_order.items_received": [["purchase-orders"], ["warehouse-stock"]],
  "dropship_order.created": [["dropship-orders"]],
  "dropship_order.status_updated": [["dropship-orders"]],
  "dropship_order.cancelled": [["dropship-orders"]],
  "stocktake.completed": [["stocktakes"], ["warehouse-stock"]],
};

interface UseWebSocketReturn {
  isConnected: boolean;
  lastEvent: WSEvent | null;
}

export function useWebSocket(): UseWebSocketReturn {
  const [isConnected, setIsConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<WSEvent | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptRef = useRef(0);
  const connectRef = useRef<() => void>(() => {});
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.token);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const connect = useCallback(async () => {
    // Re-read token from store to avoid stale closure value
    const freshToken = useAuthStore.getState().token;
    if (!freshToken || !isAuthenticated) return;

    // Clean up any existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    try {
      // Ticket-based auth (keeps JWT out of URLs/logs)
      let wsUrl: string;
      try {
        const resp = await apiFetch("/v1/auth/ws-ticket", { method: "POST" });
        const { ticket } = await resp.json();
        wsUrl = getWSUrl(ticket);
      } catch {
        const attempt = reconnectAttemptRef.current;
        const delay = Math.min(1000 * Math.pow(2, attempt), 30000);
        reconnectAttemptRef.current = attempt + 1;
        reconnectTimeoutRef.current = setTimeout(() => { connectRef.current(); }, delay);
        return;
      }

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        if (wsRef.current !== ws) return;
        setIsConnected(true);
        reconnectAttemptRef.current = 0;
      };

      ws.onmessage = (event) => {
        if (wsRef.current !== ws) return;
        try {
          const data: WSEvent = JSON.parse(event.data);
          setLastEvent(data);

          // Invalidate relevant React Query caches
          const keysToInvalidate = EVENT_INVALIDATION_MAP[data.type];
          if (keysToInvalidate) {
            for (const queryKey of keysToInvalidate) {
              queryClient.invalidateQueries({ queryKey });
            }
          }
        } catch {
          // Ignore malformed messages
        }
      };

      ws.onclose = () => {
        if (wsRef.current !== ws) return;
        setIsConnected(false);
        wsRef.current = null;

        // Auto-reconnect with exponential backoff
        if (useAuthStore.getState().isAuthenticated) {
          const attempt = reconnectAttemptRef.current;
          const delay = Math.min(1000 * Math.pow(2, attempt), 30000); // max 30s
          reconnectAttemptRef.current = attempt + 1;

          reconnectTimeoutRef.current = setTimeout(() => {
            connectRef.current();
          }, delay);
        }
      };

      ws.onerror = () => {
        if (wsRef.current !== ws) return;
        // onclose will fire after onerror
      };
    } catch {
      // Failed to create WebSocket, will retry via onclose
    }
  }, [token, isAuthenticated, queryClient]);

  // Keep ref in sync so internal timeouts can call the latest version
  // eslint-disable-next-line react-hooks/refs
  connectRef.current = connect;

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  return { isConnected, lastEvent };
}
