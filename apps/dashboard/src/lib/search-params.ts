type SearchParamValue =
  | string
  | number
  | boolean
  | readonly (string | number | boolean)[]
  | null
  | undefined;

/**
 * Build URLSearchParams from an object, skipping null/undefined/empty values.
 */
export function buildSearchParams(
  params: Record<string, SearchParamValue> | object,
): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value == null || value === "") {
      continue;
    }

    if (Array.isArray(value)) {
      if (value.length === 0) {
        continue;
      }

      sp.set(key, value.join(","));
    } else {
      sp.set(key, String(value));
    }
  }
  return sp;
}
