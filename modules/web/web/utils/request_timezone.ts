// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve the timezone value sent in RPC baggage (`ctx.tz`).
 * Prefer saved User.Timezone; otherwise browser IANA. Server still applies D7.
 */
export function resolveRequestTimezone(
  userTimezone?: string | null,
  browserTimezone?: string | null
): string {
  const saved = String(userTimezone || '').trim();
  if (saved) return saved;
  return String(browserTimezone || '').trim();
}

export function detectBrowserTimezone(): string {
  try {
    return String(Intl.DateTimeFormat().resolvedOptions().timeZone || '').trim();
  } catch {
    return '';
  }
}
