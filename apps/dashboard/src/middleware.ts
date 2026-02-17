import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login", "/register", "/return-request", "/track", "/supplier-portal"];

function isPublicPath(pathname: string): boolean {
  if (pathname === "/") return true;
  return publicPaths.some((p) => pathname.startsWith(p));
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has("has_session");

  // Authenticated user trying to access auth pages -> redirect to dashboard
  if (hasSession && ["/login", "/register"].some((p) => pathname.startsWith(p))) {
    return NextResponse.redirect(new URL("/orders", request.url));
  }

  // Unauthenticated user trying to access protected pages -> redirect to login
  if (!hasSession && !isPublicPath(pathname)) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
