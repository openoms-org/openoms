import {
  describe,
  it,
  expect,
  beforeAll,
  afterAll,
  afterEach,
  vi,
} from "vitest";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";

vi.mock("@/lib/download", () => ({ downloadBlob: vi.fn() }));
import { downloadBlob } from "@/lib/download";
import { downloadShipmentLabel } from "@/hooks/use-shipments";

const API_BASE = "*/v1";
const SHIPMENT_ID = "a8326d95-1111-2222-3333-444444444444";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  vi.mocked(downloadBlob).mockClear();
});
afterAll(() => server.close());

describe("downloadShipmentLabel", () => {
  it("GETs /v1/shipments/{id}/label and downloads the PDF blob", async () => {
    let method = "";
    let path = "";
    const pdfBytes = new Uint8Array([0x25, 0x50, 0x44, 0x46]); // %PDF

    server.use(
      http.get(`${API_BASE}/shipments/${SHIPMENT_ID}/label`, ({ request }) => {
        method = request.method;
        path = new URL(request.url).pathname;
        return new HttpResponse(pdfBytes, {
          headers: { "Content-Type": "application/pdf" },
        });
      })
    );

    await downloadShipmentLabel(SHIPMENT_ID);

    expect(method).toBe("GET");
    expect(path).toBe(`/v1/shipments/${SHIPMENT_ID}/label`);
    expect(downloadBlob).toHaveBeenCalledTimes(1);
    const [blobArg, nameArg] = vi.mocked(downloadBlob).mock.calls[0];
    expect(blobArg).toBeInstanceOf(Blob);
    expect(blobArg.type).toBe("application/pdf");
    expect(String(nameArg)).toMatch(/\.pdf$/);
    expect(String(nameArg)).toContain("a8326d95");
  });

  it("does not fetch the raw uploads URL", async () => {
    let fetchedUploads = false;
    server.use(
      http.get("https://api.openoms.org/uploads/*", () => {
        fetchedUploads = true;
        return new HttpResponse("should not be called", { status: 401 });
      }),
      http.get(`${API_BASE}/shipments/${SHIPMENT_ID}/label`, () => {
        return new HttpResponse(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
          headers: { "Content-Type": "application/pdf" },
        });
      })
    );

    await downloadShipmentLabel(SHIPMENT_ID);

    expect(fetchedUploads).toBe(false);
    expect(downloadBlob).toHaveBeenCalledTimes(1);
  });
});
