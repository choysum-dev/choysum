// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import moment from 'moment-timezone';

/**
 * Reports whether a timezone string is a valid IANA timezone.
 */
export function isIanaTimezone(tz?: string): boolean {
  if (!tz) return false;
  return Boolean(moment.tz.zone(tz));
}

/**
 * Parses a fixed UTC offset string into minutes.
 *
 * Accepts forms like `+08:00`, `+8`, `-05:00`, `UTC`, `GMT`, `Z`.
 * Returns undefined for unrecognised or out-of-range inputs.
 */
export function parseTimezoneOffsetMinutes(tz?: string): number | undefined {
  if (!tz) return undefined;
  const raw = tz.trim();
  if (!raw) return undefined;
  if (raw === 'UTC' || raw === 'GMT' || raw === 'Z') return 0;

  const match = raw.match(/^([+-])(\d{1,2})(?::?(\d{2}))?$/);
  if (!match) return undefined;
  const sign = match[1] === '-' ? -1 : 1;
  const hours = Number(match[2]);
  const minutes = Number(match[3] ?? '0');
  if (Number.isNaN(hours) || Number.isNaN(minutes)) return undefined;
  if (hours > 14 || minutes >= 60) return undefined;
  return sign * (hours * 60 + minutes);
}
