import { describe, expect, it } from "vitest";
import type { NextRequest } from "next/server";
import { config, middleware } from "@/middleware";

/** Mirrors how Next.js applies `config.matcher`: middleware only runs on a matching path. */
function matcherRunsOn(pathname: string): boolean {
  return config.matcher.some((pattern) => new RegExp(`^${pattern}$`).test(pathname));
}

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

  it.each(["/sw.js", "/icon-192.png", "/icon-512.png", "/manifest.webmanifest"])(
    "serves %s to anonymous visitors instead of redirecting to /login",
    (pathname) => {
      expect(matcherRunsOn(pathname)).toBe(false);

      // Defence in depth: even if the matcher lets it through, the handler must not redirect.
      const response = middleware(requestFor(pathname));
      expect(response.headers.get("location")).toBeNull();
      expect(response.headers.get("x-middleware-next")).toBe("1");
    },
  );

  it.each(["/", "/carriers", "/settings/warehouse-documents", "/swXjs", "/sw.js.map", "/icon"])(
    "keeps %s behind the middleware gate",
    (pathname) => {
      expect(matcherRunsOn(pathname)).toBe(true);
      expect(middleware(requestFor(pathname)).headers.get("location")).toBe(
        "https://app.openoms.org/login",
      );
    },
  );
});
