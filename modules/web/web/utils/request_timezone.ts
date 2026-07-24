// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** True when `tz` is accepted by Intl as an IANA time zone id. */
export function isIanaTimezone(tz?: string | null): boolean {
  const value = String(tz || '').trim();
  if (!value) return false;
  try {
    Intl.DateTimeFormat('en-US', { timeZone: value });
    return true;
  } catch {
    return false;
  }
}

/**
 * Resolve the timezone value sent in RPC baggage (`ctx.tz`).
 * Prefer saved User.Timezone when it is a valid IANA id; otherwise browser IANA.
 * Invalid non-empty saved values must not block the browser fallback (server D7).
 */
export function resolveRequestTimezone(
  userTimezone?: string | null,
  browserTimezone?: string | null
): string {
  const saved = String(userTimezone || '').trim();
  if (saved && isIanaTimezone(saved)) return saved;
  const browser = String(browserTimezone || '').trim();
  if (browser && isIanaTimezone(browser)) return browser;
  return '';
}

export function detectBrowserTimezone(): string {
  try {
    return String(Intl.DateTimeFormat().resolvedOptions().timeZone || '').trim();
  } catch {
    return '';
  }
}
