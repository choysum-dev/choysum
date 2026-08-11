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

export function isPlainPropertiesMap(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

export function normalizePropertiesMap(value: unknown): Record<string, unknown> {
  if (value == null) return {};
  if (isPlainPropertiesMap(value)) return { ...value };
  return {};
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
    out.push({ ...rec, name, type } as PropertyItemDefinition);
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
