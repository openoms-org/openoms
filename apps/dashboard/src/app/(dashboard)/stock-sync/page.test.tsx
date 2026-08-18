import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import StockSyncPage from "./page";

const mutateAsyncPushAll = vi.fn();
const toastSuccess = vi.fn();
const toastError = vi.fn();

vi.mock("@/components/shared/admin-guard", () => ({
  AdminGuard: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}));

vi.mock("@/hooks/use-stock-sync", () => ({
  useStockSyncChannels: () => ({ isLoading: false }),
  useStockSyncDashboard: () => ({
    isLoading: false,
    data: {
      total_products: 0,
      active_channels: 0,
      recent_errors: 0,
      channel_summaries: [],
    },
    dataUpdatedAt: 0,
  }),
  useCreateStockSyncChannel: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateStockSyncChannel: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteStockSyncChannel: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePushAllStock: () => ({
    mutateAsync: mutateAsyncPushAll,
    isPending: false,
  }),
  usePushChannelStock: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

describe("StockSyncPage toasts", () => {
  beforeEach(() => {
    mutateAsyncPushAll.mockReset();
    toastSuccess.mockReset();
    toastError.mockReset();
  });

  it("toasts after a successful sync-all", async () => {
    mutateAsyncPushAll.mockResolvedValue({ channels_synced: 2, message: "ok" });
    const user = userEvent.setup();

    render(<StockSyncPage />);
    await user.click(screen.getByRole("button", { name: "syncAll" }));

    expect(mutateAsyncPushAll).toHaveBeenCalledTimes(1);
    expect(toastSuccess).toHaveBeenCalledWith("allChannelsSynced");
    expect(toastError).not.toHaveBeenCalled();
  });

  it("toasts an error when sync-all fails", async () => {
    mutateAsyncPushAll.mockRejectedValue(new Error("push failed"));
    const user = userEvent.setup();

    render(<StockSyncPage />);
    await user.click(screen.getByRole("button", { name: "syncAll" }));

    expect(toastError).toHaveBeenCalled();
    expect(toastSuccess).not.toHaveBeenCalled();
  });
});
