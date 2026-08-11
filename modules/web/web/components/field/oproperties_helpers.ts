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

/** Convert stored UTC datetime to a picker Date (user wall-clock carrier). */
export function propertyDatetimeToPicker(
  raw: unknown,
  tz: string = getUserTimeZone()
): Date | null {
  if (raw == null || raw === '') return null;
  if (raw instanceof Date) return utcToUserWallDate(raw, tz);
  if (typeof raw === 'string' || typeof raw === 'number') return utcToUserWallDate(raw, tz);
  return null;
}

/** Convert picker Date/value to stored UTC ISO string. */
export function propertyDatetimeFromPicker(
  value: unknown,
  tz: string = getUserTimeZone()
): string | null {
  if (value == null || value === '') return null;
  const wall = value instanceof Date ? value : new Date(value as any);
  const utc = userWallDateToUtc(wall, tz);
  return utc ? formatUtcIso(utc, PROPERTY_DATETIME_STORAGE_FORMAT) : null;
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

/** Count keys present in both schema names and the value map (List V1 summary). */
export function countSchemaMapIntersection(schemaNames: string[], map: unknown): number {
  const valueMap = normalizePropertiesMap(map);
  if (!schemaNames.length) return 0;
  let n = 0;
  for (const name of schemaNames) {
    if (Object.prototype.hasOwnProperty.call(valueMap, name)) n += 1;
  }
  return n;
}

export function buildFullPropertiesMap(items: ResolvedPropertyItem[], previous: unknown): PropertiesMap {
  const prev = normalizePropertiesMap(previous);
  const next: PropertiesMap = {};
  for (const item of items) {
    if (!item?.name) continue;
    if (!PROPERTIES_V1_TYPES.has(String(item.type))) continue;
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

/** Write one key into a full replace map (schema keys only). */
export function writePropertyValue(
  items: ResolvedPropertyItem[],
  previous: unknown,
  name: string,
  value: unknown
): PropertiesMap {
  const next = buildFullPropertiesMap(items, previous);
  if (!items.some(i => i.name === name && PROPERTIES_V1_TYPES.has(String(i.type)))) {
    return next;
  }
  next[name] = value;
  return next;
}
