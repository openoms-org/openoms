import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login", "/register", "/return-request", "/track", "/supplier-portal"];
const publicAssetPrefixes = ["/logos/"];
// Root-served PWA assets: the browser fetches them before any session exists, and a 307 to
// /login breaks service-worker registration and manifest parsing. Exact matches only — this
// is an authorization gate, so no prefix or extension wildcards here.
const publicAssetFiles = ["/sw.js", "/icon-192.png", "/icon-512.png", "/manifest.webmanifest"];

function isPublicPath(pathname: string): boolean {
  return (
    publicPaths.some((p) => pathname.startsWith(p)) ||
    publicAssetPrefixes.some((p) => pathname.startsWith(p)) ||
    publicAssetFiles.includes(pathname)
  );
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has("has_session");

  // Root "/" -> redirect to login if not logged in (Pulpit page handles logged-in users)
  if (pathname === "/" && !hasSession) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // Authenticated user trying to access auth pages -> redirect to dashboard
  if (hasSession && ["/login", "/register"].some((p) => pathname.startsWith(p))) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  // Unauthenticated user trying to access protected pages -> redirect to login
  if (!hasSession && !isPublicPath(pathname)) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Directory prefixes stay prefix-matched; the root files are anchored with `$` so that only
    // the exact asset skips the gate (`/sw.js.map` and friends stay behind it).
    "/((?!v1|api|_next/static|_next/image|logos|favicon\\.ico$|sw\\.js$|icon-192\\.png$|icon-512\\.png$|manifest\\.webmanifest$).*)",
  ],
};
