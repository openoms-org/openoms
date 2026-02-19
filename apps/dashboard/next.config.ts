import type { NextConfig } from "next";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  productionBrowserSourceMaps: false,
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
          value: `default-src 'self'; script-src 'self' 'unsafe-inline' https://geowidget.inpost.pl; style-src 'self' 'unsafe-inline' https://geowidget.inpost.pl; img-src 'self' data: https: blob:; connect-src 'self' ${apiUrl} https://*.inpost.pl wss: ws:; font-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';`,
        },
      ],
    },
  ],
};

export default nextConfig;
