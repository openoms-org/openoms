// Graceful fallback for enum-coded i18n labels (OPE-519 defense-in-depth).
//
// Backend enums (fulfillment blocker codes, provider operations) can grow
// ahead of the dashboard's translation catalogs. Rendering t(`...${code}`)
// for an unknown code leaks the raw dotted key ("fulfillment.blockerCode.x")
// to operators. These helpers check key existence via next-intl's t.has()
// and fall back to a humanized form of the code (snake_case -> sentence) so
// new codes degrade readably until a real label lands in the catalogs.

/** Minimal surface of a next-intl translator needed for fallback lookups. */
export interface EnumTranslator {
  (key: string): string;
  has(key: string): boolean;
}

/**
 * Turns a snake_case enum code into readable words
 * ("supplier_order_rejected" -> "Supplier order rejected"). Returns the input
 * unchanged when no word characters remain after splitting.
 */
export function humanizeEnumCode(code: string): string {
  const words = code.trim().split(/[_\s]+/).filter(Boolean).join(" ");
  if (!words) {
    return code;
  }
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/**
 * Translates `${prefix}.${code}`, falling back to the humanized code when the
 * catalog has no entry for it.
 */
export function enumLabel(
  t: EnumTranslator,
  prefix: string,
  code: string,
): string {
  const key = `${prefix}.${code}`;
  return t.has(key) ? t(key) : humanizeEnumCode(code);
}
