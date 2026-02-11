"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/auth";
import type { WSEvent } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// Convert http(s) to ws(s)
function getWSUrl(token: string): string {
  const base = API_URL.replace(/^http/, "ws");
  return `${base}/v1/ws?token=${encodeURIComponent(token)}`;
}

// Map event types to React Query cache keys to invalidate
const EVENT_INVALIDATION_MAP: Record<string, string[][]> = {
  "order.created": [["orders"]],
  "order.updated": [["orders"]],
  "order.status_changed": [["orders"], ["stats"]],
  "shipment.created": [["shipments"]],
  "shipment.updated": [["shipments"]],
  "product.updated": [["products"], ["product-stock"]],
  "return.created": [["returns"]],
  "return.updated": [["returns"]],
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

  const connect = useCallback(() => {
    if (!token || !isAuthenticated) return;

    // Clean up any existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    try {
      const ws = new WebSocket(getWSUrl(token));
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
        if (isAuthenticated) {
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
