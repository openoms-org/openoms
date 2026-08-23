import layout from '../../messages/pl/layout.json';
import track from '../../messages/pl/track.json';

/**
 * Copy read straight out of messages/pl.
 *
 * E2E assertions used to hardcode Polish UI strings, so a copy change left the
 * test asserting text that no longer existed while the feature itself was fine
 * (the login screen redesign renamed "Logowanie" to "Zaloguj się do panelu" and
 * six auth tests failed on the assertion alone). Reading the expected string
 * from the same source the app renders from means a copy change either keeps
 * the test passing or breaks it loudly at the key, never silently.
 *
 * Tests run with locale pl-PL (see playwright.config.ts), so pl is the right
 * catalogue to assert against.
 */

// The app merges every module in messages/pl into one namespace tree
// (src/i18n/request.ts). Tests only assert against a few of them; add modules
// here as specs start needing them.
const MESSAGES: Record<string, unknown> = {
  ...layout,
  ...track,
};

/**
 * Resolves a dotted message key, e.g. "auth.login.title".
 *
 * Message files mix nested objects with flat dotted keys, so at each step we
 * try the longest matching literal key before descending — mirroring the
 * convertDotKeys() normalisation the app does at runtime.
 */
export function plCopy(key: string): string {
  const parts = key.split('.');
  let node: unknown = MESSAGES;

  for (let i = 0; i < parts.length; ) {
    if (node === null || typeof node !== 'object') {
      throw new Error(`messages/pl: "${key}" resolves through a non-object at "${parts.slice(0, i).join('.')}"`);
    }
    const record = node as Record<string, unknown>;

    // Prefer the longest literal key that matches, so a flat "login.title"
    // entry resolves as readily as a nested one.
    let matched = false;
    for (let end = parts.length; end > i; end--) {
      const candidate = parts.slice(i, end).join('.');
      if (candidate in record) {
        node = record[candidate];
        i = end;
        matched = true;
        break;
      }
    }
    if (!matched) {
      throw new Error(`messages/pl: missing key "${key}" (no entry for "${parts[i]}")`);
    }
  }

  if (typeof node !== 'string') {
    throw new Error(`messages/pl: "${key}" is not a string`);
  }
  return node;
}

/** Login screen copy asserted by the auth specs. */
export const LOGIN_COPY = {
  title: plCopy('auth.login.title'),
  organization: plCopy('auth.login.organization'),
  email: plCopy('auth.login.email'),
  submit: plCopy('auth.login.submit'),
  showPassword: plCopy('auth.login.showPassword'),
  hidePassword: plCopy('auth.login.hidePassword'),
  register: plCopy('auth.login.register'),
  orgRequired: plCopy('auth.validation.orgRequired'),
  passwordRequired: plCopy('auth.validation.passwordRequired'),
} as const;

/** Public /track copy. The page is unauthenticated. */
export const TRACK_COPY = {
  title: plCopy('track.title'),
  subtitle: plCopy('track.subtitle'),
  orderId: plCopy('track.orderId'),
  email: plCopy('track.email'),
  submit: plCopy('track.submit'),
  missingFields: plCopy('track.errors.missingFields'),
} as const;
