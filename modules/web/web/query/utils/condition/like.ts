// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import utc from 'dayjs/plugin/utc';

import {
  resolveActiveDateDisplayFormat,
  resolveActiveDateTimeDisplayFormat,
  resolveActiveTimeDisplayFormat,
} from '@/web/web/components/internal/calendarLocale';
import { formatUtcInTimeZone, getUserTimeZone } from '@/web/web/utils/datetime';

dayjs.extend(customParseFormat);
dayjs.extend(utc);

/**
 * Reports whether an operator belongs to the LIKE family.
 */
export function isLikeOperator(op?: string): boolean {
  if (!op) return false;
  const o = op.toLowerCase();
  return o === 'like' || o === 'not like' || o === 'contains';
}

/**
 * Wraps a raw value in SQL LIKE wildcards.
 */
export function toLikePattern(raw: any): string {
  const s = raw == null ? '' : String(raw);
  return `%${s}%`;
}

/**
 * Normalizes a value for LIKE-based operators.
 */
export function normalizeLikeValue(val: any): string {
  return toLikePattern(val);
}

function looksLikeUtcDatetime(value: unknown): boolean {
  if (value instanceof Date) return !Number.isNaN(value.getTime());
  if (typeof value !== 'string') return false;
  const raw = value.trim();
  // ISO instants (with time); calendar dates stay literal.
  return /^\d{4}-\d{2}-\d{2}T/.test(raw);
}

export type ValuePreviewOptions = {
  /** Field metadata type (`datetime` | `date` | …). */
  fieldType?: string;
  /** Override display timezone (defaults to getUserTimeZone()). */
  timeZone?: string;
  /** Display format for datetime preview. */
  displayFormat?: string;
};

function localIsoDate(value: Date): string {
  // Literal calendar day — use local Y/M/D, not toISOString (east-of-UTC shift).
  const y = value.getFullYear();
  const m = String(value.getMonth() + 1).padStart(2, '0');
  const d = String(value.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

function formatDatePreview(value: unknown, displayFormat?: string | null): string {
  const fmt = resolveActiveDateDisplayFormat(displayFormat);
  if (value instanceof Date) {
    const iso = localIsoDate(value);
    const m = dayjs.utc(iso, 'YYYY-MM-DD', true);
    return m.isValid() ? m.format(fmt) : iso;
  }
  const raw = String(value ?? '').trim();
  if (!raw) return '';
  // Stored date conditions may be bare YYYY-MM-DD or YYYY-MM-DDTHH:mm:ss[Z].
  const day = raw.match(/^(\d{4}-\d{2}-\d{2})(?:$|[T\s])/)?.[1];
  if (!day) return raw;
  const m = dayjs.utc(day, 'YYYY-MM-DD', true);
  return m.isValid() ? m.format(fmt) : raw;
}

function formatTimePreview(value: unknown, displayFormat?: string | null): string {
  const fmt = resolveActiveTimeDisplayFormat(displayFormat);
  if (value instanceof Date) {
    const canonical = dayjs(value).format('HH:mm:ss');
    const m = dayjs.utc(`1970-01-01T${canonical}`, 'YYYY-MM-DDTHH:mm:ss', true);
    return m.isValid() ? m.format(fmt) : canonical;
  }
  const raw = String(value ?? '').trim();
  if (!raw) return '';
  const parts = parseTimePreviewParts(raw);
  if (!parts) return raw;
  const canonical = `${String(parts.hour).padStart(2, '0')}:${String(parts.minute).padStart(2, '0')}:${String(parts.second).padStart(2, '0')}`;
  const m = dayjs.utc(`1970-01-01T${canonical}`, 'YYYY-MM-DDTHH:mm:ss', true);
  return m.isValid() ? m.format(fmt) : raw;
}

function parseTimePreviewParts(
  raw: string,
): { hour: number; minute: number; second: number } | null {
  const match = raw.match(/^(\d{2}):(\d{2})(?::(\d{2}))?$/);
  if (!match) return null;
  const hour = Number(match[1]);
  const minute = Number(match[2]);
  const second = match[3] != null ? Number(match[3]) : 0;
  if (!Number.isInteger(hour) || hour < 0 || hour > 23) return null;
  if (!Number.isInteger(minute) || minute < 0 || minute > 59) return null;
  if (!Number.isInteger(second) || second < 0 || second > 59) return null;
  return { hour, minute, second };
}

/**
 * Converts a value into the preview string shown for an operator.
 * Date/time/datetime previews follow Language overlays (same defaults as ChoyDate*).
 */
export function valueToPreview(op: string, value: any, options?: ValuePreviewOptions): string {
  if (isLikeOperator(op)) return toLikePattern(value);
  if (Array.isArray(value)) {
    return `(${value.map(v => valueToPreview('=', v, options)).join(', ')})`;
  }

  const fieldType = String(options?.fieldType || '').toLowerCase();
  if (fieldType === 'date') {
    return formatDatePreview(value, options?.displayFormat);
  }
  if (fieldType === 'time') {
    return formatTimePreview(value, options?.displayFormat);
  }

  if (fieldType === 'datetime' || (!fieldType && looksLikeUtcDatetime(value))) {
    const tz = options?.timeZone || getUserTimeZone();
    const format = resolveActiveDateTimeDisplayFormat(options?.displayFormat);
    const wall = formatUtcInTimeZone(value, format, tz);
    if (wall) return wall;
  }

  if (value instanceof Date) return value.toISOString();
  return String(value ?? '');
}
