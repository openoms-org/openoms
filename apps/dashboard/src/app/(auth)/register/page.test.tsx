import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, waitFor } from "@/test/test-utils";
import RegisterPage from "./page";

const replaceMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: replaceMock }),
}));

describe("RegisterPage", () => {
  beforeEach(() => {
    replaceMock.mockReset();
    vi.restoreAllMocks();
  });

  it("does not redirect to invite registration before public config has loaded", async () => {
    let resolveConfig: (value: Response) => void = () => {};
    vi.stubGlobal("fetch", vi.fn((url: string) => {
      if (url.endsWith("/v1/config/public")) {
        return new Promise<Response>((resolve) => {
          resolveConfig = resolve;
        });
      }
      return Promise.resolve(jsonResponse([]));
    }));

    render(<RegisterPage />);

    await act(async () => {
      await Promise.resolve();
    });

    expect(replaceMock).not.toHaveBeenCalledWith("/register/invite");

    await act(async () => {
      resolveConfig(jsonResponse({
        registration_mode: "invite",
        license_enabled: true,
        billing_enabled: true,
      }));
    });

    await waitFor(() => expect(fetch).toHaveBeenCalledWith(
      "/v1/billing/plans",
      { credentials: "include" }
    ));
  });
});

function jsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    json: async () => body,
  } as Response;
}
