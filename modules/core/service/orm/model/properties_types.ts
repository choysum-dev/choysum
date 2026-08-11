// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** V1 property item types (PP7). */
export const PROPERTIES_V1_TYPES = new Set([
  'boolean',
  'integer',
  'float',
  'char',
  'text',
  'date',
  'datetime',
  'selection',
]);

export type PropertyItemDefinition = {
  name: string;
  type: string;
  string?: string;
  default?: unknown;
  readonly?: boolean;
  selection?: Array<{ value: string; label?: string } | [string, string]>;
  [key: string]: unknown;
};

/** Resolved item for Form (schema ⊕ value). */
export type ResolvedPropertyItem = PropertyItemDefinition & {
  value?: unknown;
};

/** True only for plain Object maps (rejects Date / Map / arrays / class instances). */
export function isPlainPropertiesMap(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && Object.prototype.toString.call(value) === '[object Object]';
}

export function normalizePropertiesMap(value: unknown): Record<string, unknown> {
  if (value == null) return {};
  if (isPlainPropertiesMap(value)) return { ...value };
  return {};
}

function selectionOptionValues(selection: unknown): string[] {
  if (!Array.isArray(selection)) return [];
  const out: string[] = [];
  for (const opt of selection) {
    if (Array.isArray(opt) && typeof opt[0] === 'string') {
      out.push(opt[0]);
      continue;
    }
    if (opt && typeof opt === 'object' && !Array.isArray(opt) && typeof (opt as { value?: unknown }).value === 'string') {
      out.push(String((opt as { value: string }).value));
      continue;
    }
  }
  return out;
}

/**
 * Coarse type check for property values / Definition defaults (PP7).
 * `null` / `undefined` are always allowed (cleared / unset).
 */
export function propertyValueMatchesType(item: Pick<PropertyItemDefinition, 'type' | 'selection'>, value: unknown): boolean {
  if (value === null || value === undefined) return true;
  switch (item.type) {
    case 'boolean':
      return typeof value === 'boolean';
    case 'integer':
      return typeof value === 'number' && Number.isFinite(value) && Number.isInteger(value);
    case 'float':
      return typeof value === 'number' && Number.isFinite(value);
    case 'char':
    case 'text':
    case 'date':
    case 'datetime':
      return typeof value === 'string';
    case 'selection': {
      if (typeof value !== 'string') return false;
      const allowed = selectionOptionValues(item.selection);
      if (allowed.length === 0) return true;
      return allowed.includes(value);
    }
    default:
      return true;
  }
}

export function parsePropertyDefinitionItems(raw: unknown): PropertyItemDefinition[] {
  if (!Array.isArray(raw)) return [];
  const out: PropertyItemDefinition[] = [];
  for (const item of raw) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) continue;
    const rec = item as Record<string, unknown>;
    const name = typeof rec.name === 'string' ? rec.name.trim() : '';
    if (!name) continue;
    const type = typeof rec.type === 'string' ? rec.type.trim() : '';
    out.push({ ...rec, name, type } as PropertyItemDefinition);
  }
  return out;
}

/**
 * Validate Definition JSON for write (PP7): unknown types fail.
 * Returns normalized items (does not mutate input array objects in place beyond copy).
 */
export function assertValidPropertyDefinitionItems(raw: unknown): PropertyItemDefinition[] {
  if (raw == null) return [];
  if (!Array.isArray(raw)) {
    throw new Error('PropertyDefinition.Definition must be an array');
  }
  const seen = new Set<string>();
  const out: PropertyItemDefinition[] = [];
  for (const item of raw) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error('PropertyDefinition.Definition items must be objects');
    }
    const rec = item as Record<string, unknown>;
    const name = typeof rec.name === 'string' ? rec.name.trim() : '';
    if (!name) {
      throw new Error('PropertyDefinition.Definition item requires a non-empty name');
    }
    if (seen.has(name)) {
      throw new Error(`PropertyDefinition.Definition duplicate name "${name}"`);
    }
    seen.add(name);
    const type = typeof rec.type === 'string' ? rec.type.trim() : '';
    if (!type) {
      throw new Error(`PropertyDefinition.Definition item "${name}" requires type`);
    }
    if (!PROPERTIES_V1_TYPES.has(type)) {
      throw new Error(
        `PropertyDefinition.Definition item "${name}" has unsupported type "${type}" (V1 allows: ${[...PROPERTIES_V1_TYPES].join(', ')})`
      );
    }
    const normalized = { ...rec, name, type } as PropertyItemDefinition;
    if (type === 'selection') {
      const values = selectionOptionValues(rec.selection);
      if (values.length === 0) {
        throw new Error(`PropertyDefinition.Definition item "${name}" of type selection requires a non-empty selection`);
      }
      if (!Array.isArray(rec.selection) || values.length !== rec.selection.length) {
        throw new Error(`PropertyDefinition.Definition item "${name}" selection options are invalid`);
      }
      normalized.selection = rec.selection as PropertyItemDefinition['selection'];
    }
    if (Object.prototype.hasOwnProperty.call(rec, 'default') && !propertyValueMatchesType(normalized, rec.default)) {
      throw new Error(`PropertyDefinition.Definition item "${name}" default does not match type "${type}"`);
    }
    out.push(normalized);
  }
  return out;
}

/** Skip unknown types on read (dirty data); warn via callback. */
export function filterReadablePropertyDefinitionItems(
  raw: unknown,
  onSkip?: (item: PropertyItemDefinition) => void
): PropertyItemDefinition[] {
  const items = parsePropertyDefinitionItems(raw);
  const out: PropertyItemDefinition[] = [];
  for (const item of items) {
    if (!PROPERTIES_V1_TYPES.has(item.type)) {
      onSkip?.(item);
      continue;
    }
    out.push(item);
  }
  return out;
}
