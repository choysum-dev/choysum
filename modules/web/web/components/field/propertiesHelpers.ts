// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  PROPERTIES_V1_TYPES,
  normalizePropertiesMap,
  type ResolvedPropertyItem,
} from '@/core/service/orm/model/properties_types';

import {
  formatUtcIso,
  getUserTimeZone,
  userWallDateToUtc,
  utcToUserWallDate,
} from '@/web/web/utils/datetime';

export type PropertiesMap = Record<string, unknown>;

export type PropertySelectionOption = { value: string; label: string };

export const PROPERTY_DATETIME_STORAGE_FORMAT = 'YYYY-MM-DD[T]HH:mm:ss.SSSZ';

/** Resolve the properties field name for RPC / DOM ids (empty fallback when resolving). */
export function propertiesFieldKey(
  bindingProp: unknown,
  propsProp: unknown,
  fallback: string = 'properties',
): string {
  return String(bindingProp || propsProp || fallback);
}

/** Convert stored date (ISO midnight or YYYY-MM-DD) to native date input value. */
export function propertyDateToInput(raw: unknown): string {
  if (raw == null || raw === '') return '';
  if (raw instanceof Date) {
    if (Number.isNaN(raw.getTime())) return '';
    const y = raw.getUTCFullYear();
    const m = String(raw.getUTCMonth() + 1).padStart(2, '0');
    const d = String(raw.getUTCDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
  }
  const s = String(raw).trim();
  const m = /^(\d{4}-\d{2}-\d{2})/.exec(s);
  return m ? m[1]! : '';
}

/** Convert native date input (YYYY-MM-DD) to stored UTC midnight ISO. */
export function propertyDateFromInput(value: unknown): string | null {
  if (value == null || value === '') return null;
  const s = String(value).trim();
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return null;
  return `${s}T00:00:00Z`;
}

/** Display date as `YYYY-MM-DD` (same as native date input). */
export function propertyDateToDisplay(raw: unknown): string {
  return propertyDateToInput(raw);
}

/** Convert stored UTC datetime to a picker Date (user wall-clock carrier). */
export function propertyDatetimeToPicker(
  raw: unknown,
  tz: string = getUserTimeZone(),
): Date | null {
  if (raw == null || raw === '') return null;
  if (raw instanceof Date) return utcToUserWallDate(raw, tz);
  if (typeof raw === 'string' || typeof raw === 'number') return utcToUserWallDate(raw, tz);
  return null;
}

/** Convert picker Date/value to stored UTC ISO string. */
export function propertyDatetimeFromPicker(
  value: unknown,
  tz: string = getUserTimeZone(),
): string | null {
  if (value == null || value === '') return null;
  const wall = value instanceof Date ? value : new Date(value as any);
  const utc = userWallDateToUtc(wall, tz);
  return utc ? formatUtcIso(utc, PROPERTY_DATETIME_STORAGE_FORMAT) : null;
}

/** Wall-clock `YYYY-MM-DDTHH:mm:ss` for DateTimePicker from stored UTC. */
export function propertyDatetimeToPickerModel(
  raw: unknown,
  tz: string = getUserTimeZone(),
): string | null {
  const wall = propertyDatetimeToPicker(raw, tz);
  if (!wall || Number.isNaN(wall.getTime())) return null;
  const y = wall.getFullYear();
  const m = String(wall.getMonth() + 1).padStart(2, '0');
  const d = String(wall.getDate()).padStart(2, '0');
  const hh = String(wall.getHours()).padStart(2, '0');
  const mm = String(wall.getMinutes()).padStart(2, '0');
  const ss = String(wall.getSeconds()).padStart(2, '0');
  return `${y}-${m}-${d}T${hh}:${mm}:${ss}`;
}

/** DateTimePicker model → stored UTC ISO (null when cleared / invalid). */
export function propertyDatetimeFromPickerModel(
  value: unknown,
  tz: string = getUserTimeZone(),
): string | null {
  if (value == null || value === '') return null;
  if (value instanceof Date) return propertyDatetimeFromPicker(value, tz);
  const s = String(value).trim();
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?/.exec(s);
  if (!m) return propertyDatetimeFromPicker(s, tz);
  const wall = new Date(
    Number(m[1]),
    Number(m[2]) - 1,
    Number(m[3]),
    Number(m[4]),
    Number(m[5]),
    Number(m[6] || 0),
  );
  return propertyDatetimeFromPicker(wall, tz);
}

/** Display datetime as wall-clock `YYYY-MM-DD HH:mm:ss`. */
export function propertyDatetimeToDisplay(
  raw: unknown,
  tz: string = getUserTimeZone(),
): string {
  const model = propertyDatetimeToPickerModel(raw, tz);
  return model ? model.replace('T', ' ') : '';
}

/** Parse integer property input; rejects empty / non-integer strings. */
export function propertyIntegerFromInput(raw: unknown): number | null {
  if (raw == null || raw === '') return null;
  const s = String(raw).trim();
  if (!/^-?\d+$/.test(s)) return null;
  const n = Number.parseInt(s, 10);
  return Number.isSafeInteger(n) ? n : null;
}

/** Parse float property input; rejects empty / non-finite. */
export function propertyFloatFromInput(raw: unknown): number | null {
  if (raw == null || raw === '') return null;
  const n = Number(String(raw).trim());
  return Number.isFinite(n) ? n : null;
}

/** Allow typing `-` before digits (do not commit yet). */
export function isPropertyIntegerIntermediate(raw: string): boolean {
  return raw === '' || raw === '-';
}

/** Allow typing trailing `.` / bare `-` before a complete float commits. */
export function isPropertyFloatIntermediate(raw: string): boolean {
  if (raw === '' || raw === '-' || raw === '.' || raw === '-.') return true;
  return /^-?\d+\.$/.test(raw);
}

/** Char filter for integer property drafts (matches ChoyIntField). */
export function isPropertyIntegerRawAllowed(raw: string): boolean {
  return /^-?\d*$/.test(raw);
}

/** Char filter for float property drafts (matches ChoyNumberField). */
export function isPropertyFloatRawAllowed(raw: string): boolean {
  return /^-?\d*(\.\d*)?$/.test(raw);
}

/**
 * Compact table/inline summary: up to `maxItems` formatted pairs, then a count remainder.
 * Empty when nothing to show.
 */
export function formatPropertiesSummary(params: {
  items: ResolvedPropertyItem[];
  map: unknown;
  formatPair: (item: ResolvedPropertyItem, displayValue: string) => string;
  formatValue: (item: ResolvedPropertyItem, raw: unknown) => string;
  moreLabel: (remaining: number) => string;
  maxItems?: number;
}): string {
  const maxItems = params.maxItems ?? 2;
  const valueMap = normalizePropertiesMap(params.map);
  const pairs: string[] = [];
  for (const item of params.items) {
    if (!item?.name || !PROPERTIES_V1_TYPES.has(String(item.type))) continue;
    if (!Object.prototype.hasOwnProperty.call(valueMap, item.name)) continue;
    const display = params.formatValue(item, valueMap[item.name]);
    if (display === '') continue;
    pairs.push(params.formatPair(item, display));
  }
  if (!pairs.length) return '';
  if (pairs.length <= maxItems) return pairs.join(' · ');
  const head = pairs.slice(0, maxItems).join(' · ');
  return `${head} · ${params.moreLabel(pairs.length - maxItems)}`;
}

/** Fingerprint scalar / Many2One-ish record fields for ResolveProperties watch (excludes `excludeKey`). */
export function propertiesRecordWatchFingerprint(
  record: unknown,
  excludeKey: string,
): string {
  if (record == null || typeof record !== 'object' || Array.isArray(record)) return '';
  const rec = record as Record<string, unknown>;
  const parts: string[] = [];
  for (const key of Object.keys(rec)) {
    if (key === excludeKey) continue;
    const v = rec[key];
    if (v == null || typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') {
      parts.push(`${key}=${String(v)}`);
      continue;
    }
    if (typeof v === 'object' && !Array.isArray(v) && typeof (v as { Id?: unknown }).Id === 'string') {
      parts.push(`${key}=${String((v as { Id: string }).Id)}`);
    }
  }
  return parts.join('|');
}

/** Keep only V1-renderable items; unknown types are skipped (caller may warn). */
export function filterRenderablePropertyItems(items: ResolvedPropertyItem[]): {
  renderable: ResolvedPropertyItem[];
  skipped: ResolvedPropertyItem[];
} {
  const renderable: ResolvedPropertyItem[] = [];
  const skipped: ResolvedPropertyItem[] = [];
  for (const item of items || []) {
    if (!item || typeof item.name !== 'string' || !item.name) continue;
    if (PROPERTIES_V1_TYPES.has(String(item.type))) renderable.push(item);
    else skipped.push(item);
  }
  return { renderable, skipped };
}

export function normalizeSelectionOptions(selection: unknown): PropertySelectionOption[] {
  if (!Array.isArray(selection)) return [];
  const out: PropertySelectionOption[] = [];
  for (const opt of selection) {
    if (Array.isArray(opt) && typeof opt[0] === 'string') {
      out.push({ value: opt[0], label: String(opt[1] ?? opt[0]) });
      continue;
    }
    if (opt && typeof opt === 'object' && !Array.isArray(opt) && typeof (opt as any).value === 'string') {
      const value = String((opt as any).value);
      out.push({ value, label: String((opt as any).label ?? value) });
    }
  }
  return out;
}

/** Count keys present in both schema names and the value map. */
export function countSchemaMapIntersection(schemaNames: string[], map: unknown): number {
  const valueMap = normalizePropertiesMap(map);
  if (!schemaNames.length) return 0;
  let n = 0;
  for (const name of schemaNames) {
    if (Object.prototype.hasOwnProperty.call(valueMap, name)) n += 1;
  }
  return n;
}

/**
 * Build a full replace map for schema keys.
 * Non-V1 schema items keep previous values so a UI that cannot render them does not drop them.
 */
export function buildFullPropertiesMap(items: ResolvedPropertyItem[], previous: unknown): PropertiesMap {
  const prev = normalizePropertiesMap(previous);
  // null-prototype so schema names like "__proto__" become own data keys (matches BE write).
  const next: PropertiesMap = Object.create(null);
  for (const item of items) {
    if (!item?.name) continue;
    if (!PROPERTIES_V1_TYPES.has(String(item.type))) {
      if (Object.prototype.hasOwnProperty.call(prev, item.name)) {
        next[item.name] = prev[item.name];
      }
      continue;
    }
    if (Object.prototype.hasOwnProperty.call(prev, item.name)) {
      next[item.name] = prev[item.name];
    } else if (Object.prototype.hasOwnProperty.call(item, 'value')) {
      next[item.name] = item.value;
    } else if (Object.prototype.hasOwnProperty.call(item, 'default')) {
      next[item.name] = item.default;
    }
  }
  return next;
}

/** Write one key into a full replace map (V1 schema keys only). */
export function writePropertyValue(
  items: ResolvedPropertyItem[],
  previous: unknown,
  name: string,
  value: unknown,
): PropertiesMap {
  const next = buildFullPropertiesMap(items, previous);
  if (!items.some(i => i.name === name && PROPERTIES_V1_TYPES.has(String(i.type)))) {
    return next;
  }
  next[name] = value;
  return next;
}

export { PROPERTIES_V1_TYPES, normalizePropertiesMap };
export type { ResolvedPropertyItem };
