export type IntegrationMaturity = "verified" | "in_development";

export const INTEGRATION_STATUS: Record<string, IntegrationMaturity> = {
  // Verified — production-tested
  allegro: "verified",
  inpost: "verified",
  // In Development — implemented, not yet production-verified
  amazon: "in_development",
  woocommerce: "in_development",
  ebay: "in_development",
  kaufland: "in_development",
  olx: "verified",
  mirakl: "in_development",
  erli: "in_development",
  empik: "in_development",
  dhl: "in_development",
  dpd: "in_development",
  gls: "in_development",
  ups: "in_development",
  poczta_polska: "in_development",
  orlen_paczka: "in_development",
  fedex: "in_development",
  fakturownia: "in_development",
  ksef: "in_development",
  smsapi: "in_development",
  mailchimp: "in_development",
  freshdesk: "in_development",
  shoper: "in_development",
  prestashop: "in_development",
  shopify: "in_development",
};

export function isInDevelopment(provider: string): boolean {
  return INTEGRATION_STATUS[provider] === "in_development";
}
