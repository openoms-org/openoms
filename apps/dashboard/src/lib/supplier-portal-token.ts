export const SUPPLIER_PORTAL_TOKEN_STORAGE_KEY = "supplier_portal_token";

export interface SupplierPortalTokenHandoff {
  token: string;
  cleanURL: string | null;
}

export function getSupplierPortalTokenHandoff(href: string): SupplierPortalTokenHandoff {
  const url = new URL(href);
  const fragmentParams = new URLSearchParams(url.hash.startsWith("#") ? url.hash.slice(1) : url.hash);
  const token = fragmentParams.get("token")?.trim() ?? "";
  let changed = false;

  if (fragmentParams.has("token")) {
    fragmentParams.delete("token");
    const nextFragment = fragmentParams.toString();
    url.hash = nextFragment ? `#${nextFragment}` : "";
    changed = true;
  }

  if (url.searchParams.has("token")) {
    url.searchParams.delete("token");
    changed = true;
  }

  return {
    token,
    cleanURL: changed ? `${url.pathname}${url.search}${url.hash}` : null,
  };
}
