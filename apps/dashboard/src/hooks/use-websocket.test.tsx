import { createElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { apiFetch } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import { useWebSocket } from "./use-websocket";
import type { Tenant, User } from "@/types/api";

vi.mock("@/lib/api-client", () => ({
  API_URL: "",
  apiFetch: vi.fn(),
}));

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(public readonly url: string) {
    FakeWebSocket.instances.push(this);
  }

  close = vi.fn();

  emitOpen() {
    this.onopen?.();
  }

  emitClose() {
    this.onclose?.();
  }
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function authenticate(token: string) {
  useAuthStore.getState().setAuth(
    token,
    { id: "user-1", email: "user@example.com", name: "User" } as User,
    { id: "tenant-1", slug: "tenant", name: "Tenant" } as Tenant,
  );
}

function mockTicketResponses(...tickets: string[]) {
  let index = 0;
  vi.mocked(apiFetch).mockImplementation(async () => {
    const ticket = tickets[Math.min(index, tickets.length - 1)];
    index += 1;
    return new Response(JSON.stringify({ ticket }), { status: 200 });
  });

  // Existing raw fetch implementation should not be used after OPE-191, but keep
  // this mock so the RED test demonstrates apiFetch bypass instead of failing on network IO.
  vi.spyOn(globalThis, "fetch").mockImplementation(async () => {
    const ticket = tickets[Math.min(index, tickets.length - 1)];
    index += 1;
    return new Response(JSON.stringify({ ticket }), { status: 200 });
  });
}

beforeEach(() => {
  vi.useRealTimers();
  window.history.pushState(null, "", "/dashboard");
  FakeWebSocket.instances = [];
  vi.stubGlobal("WebSocket", FakeWebSocket);
  vi.mocked(apiFetch).mockReset();
  useAuthStore.getState().clearAuth();
});

afterEach(() => {
  act(() => {
    useAuthStore.getState().clearAuth();
  });
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("useWebSocket", () => {
  it("fetches WebSocket tickets through apiFetch so auth refresh and CSRF handling are preserved", async () => {
    authenticate("access-token-1");
    mockTicketResponses("ticket-1");

    renderHook(() => useWebSocket(), { wrapper: createWrapper() });

    await act(async () => {
      await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    });

    expect(apiFetch).toHaveBeenCalledWith("/v1/auth/ws-ticket", { method: "POST" });
    expect(FakeWebSocket.instances[0].url).toBe("ws://localhost:3000/v1/ws?ticket=ticket-1");
  });

  it("ignores stale close events from an old socket after a reconnect creates a newer socket", async () => {
    authenticate("access-token-1");
    mockTicketResponses("ticket-1", "ticket-2");

    const { result } = renderHook(() => useWebSocket(), { wrapper: createWrapper() });

    await act(async () => {
      await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    });
    const oldSocket = FakeWebSocket.instances[0];

    act(() => {
      oldSocket.emitOpen();
    });
    await act(async () => {
      await waitFor(() => expect(result.current.isConnected).toBe(true));
    });

    act(() => {
      authenticate("access-token-2");
    });

    await act(async () => {
      await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(2));
    });
    const newSocket = FakeWebSocket.instances[1];

    act(() => {
      newSocket.emitOpen();
    });
    await act(async () => {
      await waitFor(() => expect(result.current.isConnected).toBe(true));
    });

    act(() => {
      oldSocket.emitClose();
    });

    expect(result.current.isConnected).toBe(true);
  });
});
