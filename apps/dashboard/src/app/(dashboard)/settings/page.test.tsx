import { describe, expect, it, vi } from "vitest";

const { redirect } = vi.hoisted(() => ({
  redirect: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  redirect,
}));

import SettingsIndexPage from "./page";

describe("SettingsIndexPage", () => {
  it("redirects the settings root to security settings", () => {
    SettingsIndexPage();

    expect(redirect).toHaveBeenCalledWith("/settings/security");
  });
});
