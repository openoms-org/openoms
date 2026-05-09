import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login", "/register", "/return-request", "/track", "/supplier-portal"];

function isPublicPath(pathname: string): boolean {
  return publicPaths.some((p) => pathname.startsWith(p));
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
  matcher: ["/((?!v1|api|_next/static|_next/image|favicon.ico).*)"],
};
