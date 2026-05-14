import { ApiClientError } from "@/lib/api-client";

export function isExpectedClientError(error: Error): boolean {
  return error instanceof ApiClientError && [401, 403, 429].includes(error.status);
}
