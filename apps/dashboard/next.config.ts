import type { NextConfig } from "next";
import { withSentryConfig } from "@sentry/nextjs";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

const configuredAPIURL = process.env.NEXT_PUBLIC_API_URL?.trim();
const apiUrl = configuredAPIURL ? configuredAPIURL.replace(/\/$/, "") : "";

const isDev = process.env.NODE_ENV === "development";
const apiProxyUrl = process.env.DASHBOARD_API_PROXY_URL || (isDev ? apiUrl || "http://localhost:8080" : "");
const apiConnectSrc = apiUrl ? ` ${apiUrl}` : "";
const wsConnectSrc = (() => {
  if (!apiUrl) return " wss: ws:";
  try {
    const url = new URL(apiUrl);
    return ` ${url.protocol === "https:" ? "wss:" : "ws:"}//${url.host}`;
  } catch {
    return " wss: ws:";
  }
})();

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  productionBrowserSourceMaps: false,
  rewrites: async () => apiProxyUrl ? [
    { source: "/v1/:path*", destination: `${apiProxyUrl}/v1/:path*` },
  ] : [],
  redirects: async () => [
    // Marketplace provider pages moved from /integrations/ to /marketplaces/
    { source: "/integrations/allegro", destination: "/marketplaces/allegro", permanent: true },
    { source: "/integrations/allegro/:path*", destination: "/marketplaces/allegro/:path*", permanent: true },
    { source: "/integrations/amazon", destination: "/marketplaces/amazon", permanent: true },
    { source: "/integrations/shoper", destination: "/marketplaces/shoper", permanent: true },
    { source: "/integrations/prestashop", destination: "/marketplaces/prestashop", permanent: true },
    { source: "/integrations/shopify", destination: "/marketplaces/shopify", permanent: true },
    // Invoicing settings moved to dedicated section
    { source: "/settings/invoicing", destination: "/invoicing", permanent: true },
  ],
  headers: async () => [
    {
      source: "/(.*)",
      headers: [
        { key: "X-Frame-Options", value: "DENY" },
        { key: "X-Content-Type-Options", value: "nosniff" },
        { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
        { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(self)" },
        {
          key: "Content-Security-Policy",
          // unsafe-inline required: Next.js injects inline scripts for __NEXT_DATA__ and chunk loading.
          // Do NOT add strict-dynamic — it causes browsers to ignore unsafe-inline.
          // unsafe-eval needed in dev mode: React 19 uses eval() for dev-time debugging (callstack reconstruction).
          value: `default-src 'self'; script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""} https://geowidget.inpost.pl https://static.cloudflareinsights.com https://js.stripe.com; style-src 'self' 'unsafe-inline' https://geowidget.inpost.pl; img-src 'self' data: https: blob:; connect-src 'self'${apiConnectSrc}${wsConnectSrc} https://*.inpost.pl https://cloudflareinsights.com https://api.stripe.com https://*.sentry.io; font-src 'self' data:; frame-src https://js.stripe.com https://hooks.stripe.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self';`,
        },
      ],
    },
  ],
};

export default withNextIntl(withSentryConfig(nextConfig, {
  // Upload source maps only when SENTRY_AUTH_TOKEN is set (CI only)
  silent: !process.env.CI,
  org: process.env.SENTRY_ORG,
  project: process.env.SENTRY_PROJECT,
  sourcemaps: {
    disable: !process.env.SENTRY_AUTH_TOKEN,
    deleteSourcemapsAfterUpload: true,
  },
  telemetry: false,
}));
