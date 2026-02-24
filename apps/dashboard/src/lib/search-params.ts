/**
 * Build URLSearchParams from an object, skipping null/undefined/empty values.
 */
export function buildSearchParams(
  params: Record<string, string | number | boolean | null | undefined> | object
): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value != null && value !== "") {
      sp.set(key, String(value));
    }
  }
  return sp;
}
