// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Local wall clock with seconds: HH:mm:ss */
export type TimePickerParts = {
  hour: number;
  minute: number;
  second: number;
};

const TIME_RE = /^(\d{2}):(\d{2})(?::(\d{2}))?$/;

/** Formats clock parts as HH:mm:ss. */
export function formatTimePickerValue(parts: TimePickerParts): string {
  const { hour, minute, second } = parts;
  if (!Number.isInteger(hour) || hour < 0 || hour > 23) return '';
  if (!Number.isInteger(minute) || minute < 0 || minute > 59) return '';
  if (!Number.isInteger(second) || second < 0 || second > 59) return '';
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}:${String(second).padStart(2, '0')}`;
}

/**
 * Parses HH:mm:ss or HH:mm (seconds default 0).
 * Returns null for empty / whitespace / invalid input.
 */
export function parseTimePickerValue(value: string | null | undefined): TimePickerParts | null {
  const text = String(value ?? '').trim();
  if (!text) return null;
  const match = text.match(TIME_RE);
  if (!match) return null;
  const hour = Number(match[1]);
  const minute = Number(match[2]);
  const second = match[3] != null ? Number(match[3]) : 0;
  if (!Number.isInteger(hour) || hour < 0 || hour > 23) return null;
  if (!Number.isInteger(minute) || minute < 0 || minute > 59) return null;
  if (!Number.isInteger(second) || second < 0 || second > 59) return null;
  return { hour, minute, second };
}

export function formatTimePickerDisplay(value: string | null | undefined): string {
  const parts = parseTimePickerValue(value);
  if (!parts) return String(value ?? '').trim();
  return formatTimePickerValue(parts);
}

/** Now in local clock (seconds precision). */
export function nowTimePickerValue(date: Date = new Date()): string {
  return formatTimePickerValue({
    hour: date.getHours(),
    minute: date.getMinutes(),
    second: date.getSeconds(),
  });
}

/** Replace hour / minute / second on an existing value (missing parts default to 0). */
export function withTimePickerParts(
  current: string | null | undefined,
  patch: Partial<TimePickerParts>,
): string {
  const prev = parseTimePickerValue(current) ?? { hour: 0, minute: 0, second: 0 };
  return formatTimePickerValue({
    hour: patch.hour ?? prev.hour,
    minute: patch.minute ?? prev.minute,
    second: patch.second ?? prev.second,
  });
}

export function clearTimePickerValue(): null {
  return null;
}
