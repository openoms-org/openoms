"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/auth";
import { API_URL } from "@/lib/api-client";
import type { WSEvent } from "@/types/api";

// Convert http(s) to ws(s) with ticket-based or token-based auth
function getWSUrl(param: string, isTicket: boolean): string {
  const base = API_URL.replace(/^http/, "ws");
  const key = isTicket ? "ticket" : "token";
  return `${base}/v1/ws?${key}=${encodeURIComponent(param)}`;
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
      // Try ticket-based auth first (keeps JWT out of URLs/logs)
      let wsUrl: string;
      try {
        const resp = await fetch(`${API_URL}/v1/auth/ws-ticket`, {
          method: "POST",
          headers: { Authorization: `Bearer ${freshToken}` },
          credentials: "include",
        });
        if (resp.ok) {
          const { ticket } = await resp.json();
          wsUrl = getWSUrl(ticket, true);
        } else {
          wsUrl = getWSUrl(freshToken, false);
        }
      } catch {
        wsUrl = getWSUrl(freshToken, false);
      }

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        reconnectAttemptRef.current = 0;
      };

      ws.onmessage = (event) => {
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
        setIsConnected(false);
        wsRef.current = null;

        // Auto-reconnect with exponential backoff
        if (useAuthStore.getState().isAuthenticated) {
          const attempt = reconnectAttemptRef.current;
          const delay = Math.min(1000 * Math.pow(2, attempt), 30000); // max 30s
          reconnectAttemptRef.current = attempt + 1;

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };

      ws.onerror = () => {
        // onclose will fire after onerror
      };
    } catch {
      // Failed to create WebSocket, will retry via onclose
    }
  }, [token, isAuthenticated, queryClient]);

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
