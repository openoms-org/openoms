export function allegroSalesCenterCreateShipmentURL(input: {
  checkoutFormId?: string;
  sellerId?: string;
  sandbox: boolean;
}): string {
  const checkoutFormId = input.checkoutFormId?.trim() ?? "";
  const sellerId = input.sellerId?.trim() ?? "";
  if (!checkoutFormId || !sellerId) {
    return "";
  }
  const host = input.sandbox
    ? "https://salescenter.allegro.com.allegrosandbox.pl"
    : "https://salescenter.allegro.com";
  return `${host}/ship-with-allegro/swa/create-shipment/${encodeURIComponent(checkoutFormId)}?sellerId=${encodeURIComponent(sellerId)}`;
}
