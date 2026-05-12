import { describe, expect, it } from "vitest";
import type { NextRequest } from "next/server";
import { middleware } from "@/middleware";

function requestFor(pathname: string, hasSession = false): NextRequest {
  return {
    nextUrl: new URL(`https://app.openoms.org${pathname}`),
    url: `https://app.openoms.org${pathname}`,
    cookies: {
      has: (name: string) => hasSession && name === "has_session",
    },
  } as unknown as NextRequest;
}

describe("dashboard middleware", () => {
  it("allows logo assets without a dashboard session cookie", () => {
    const response = middleware(requestFor("/logos/official/inpost.svg"));

    expect(response.headers.get("location")).toBeNull();
    expect(response.headers.get("x-middleware-next")).toBe("1");
  });

  it("still redirects protected dashboard pages without a session cookie", () => {
    const response = middleware(requestFor("/carriers"));

    expect(response.headers.get("location")).toBe("https://app.openoms.org/login");
  });
});
