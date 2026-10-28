// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  TRANSLATED_BASE_LANG,
  assertTranslatedLangKey,
  isTranslatedLangMap,
  unwrapTranslatedValue,
} from '../repository/projection/translated_field_codec';
import type { PropertyItemDefinition, PropertyLabel } from './properties_types';

function hasOwn(map: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(map, key);
}

/** Parse a stored property label into a lang map (plain string → `{ en_US: s }`). */
export function parsePropertyLabelMap(value: unknown): Record<string, string> | null {
  if (value == null) return null;
  if (typeof value === 'string') {
    return { [TRANSLATED_BASE_LANG]: value };
  }
  if (!isTranslatedLangMap(value)) return null;
  const out: Record<string, string> = {};
  for (const [rawKey, rawVal] of Object.entries(value)) {
    const lang = String(rawKey || '').trim();
    if (!lang || lang.includes('-')) continue;
    if (rawVal == null) {
      out[lang] = '';
      continue;
    }
    if (typeof rawVal === 'string' || typeof rawVal === 'number' || typeof rawVal === 'boolean') {
      out[lang] = String(rawVal);
    }
  }
  return Object.keys(out).length ? out : null;
}

/**
 * Normalize a label for Definition write storage.
 * Plain strings are kept; lang maps are cleaned (string values only).
 * Returns `undefined` when the label should be omitted.
 */
export function normalizePropertyLabelForStorage(value: unknown, fieldHint = 'string'): PropertyLabel | undefined {
  if (value == null) return undefined;
  if (typeof value === 'string') {
    return value;
  }
  if (!isTranslatedLangMap(value) || Array.isArray(value)) {
    throw new Error(`PropertyDefinition label "${fieldHint}" must be a string or lang map`);
  }
  const out: Record<string, string> = {};
  for (const [rawKey, rawVal] of Object.entries(value)) {
    const lang = assertTranslatedLangKey(rawKey, fieldHint);
    if (rawVal == null) {
      out[lang] = '';
      continue;
    }
    if (typeof rawVal !== 'string' && typeof rawVal !== 'number' && typeof rawVal !== 'boolean') {
      throw new Error(`PropertyDefinition label "${fieldHint}" lang map value for "${lang}" must be a string`);
    }
    out[lang] = String(rawVal);
  }
  if (!Object.keys(out).length) return undefined;
  // Single en_US-only map collapses to plain string for compact storage.
  const keys = Object.keys(out);
  if (keys.length === 1 && keys[0] === TRANSLATED_BASE_LANG) {
    return out[TRANSLATED_BASE_LANG];
  }
  return out;
}

/** Unwrap a stored label for a request language (missing → en_US → ''). */
export function unwrapPropertyLabel(label: unknown, lang: string): string {
  const map = parsePropertyLabelMap(label);
  if (!map) return '';
  const unwrapped = unwrapTranslatedValue(map, lang);
  return unwrapped == null ? '' : unwrapped;
}

/**
 * Merge a current-language label write into a previous PropertyLabel.
 * Preserves other language keys; seeds en_US on first write when missing.
 */
export function mergePropertyLabelWrite(
  previous: unknown,
  currentLangValue: string,
  lang: string,
): PropertyLabel {
  const key = assertTranslatedLangKey(lang, 'string');
  const current = String(currentLangValue ?? '');
  const prevMap = parsePropertyLabelMap(previous) || {};
  const next: Record<string, string> = { ...prevMap, [key]: current };
  if (!hasOwn(next, TRANSLATED_BASE_LANG)) {
    next[TRANSLATED_BASE_LANG] = current;
  }
  return normalizePropertyLabelForStorage(next, 'string') ?? current;
}

function selectionOptionValue(opt: unknown): string | null {
  if (Array.isArray(opt) && typeof opt[0] === 'string') return opt[0];
  if (opt && typeof opt === 'object' && !Array.isArray(opt) && typeof (opt as { value?: unknown }).value === 'string') {
    return String((opt as { value: string }).value);
  }
  return null;
}

function selectionOptionLabel(opt: unknown): unknown {
  if (Array.isArray(opt)) return opt[1] ?? opt[0];
  if (opt && typeof opt === 'object' && !Array.isArray(opt)) {
    return (opt as { label?: unknown }).label ?? (opt as { value?: unknown }).value;
  }
  return undefined;
}

/** Normalize selection options; labels may be string or lang map. */
export function normalizeSelectionLabelsForStorage(
  selection: unknown,
  itemName: string
): PropertyItemDefinition['selection'] {
  if (!Array.isArray(selection)) {
    throw new Error(`PropertyDefinition.Definition item "${itemName}" selection options are invalid`);
  }
  const out: NonNullable<PropertyItemDefinition['selection']> = [];
  for (let i = 0; i < selection.length; i++) {
    const opt = selection[i];
    const value = selectionOptionValue(opt);
    if (value == null) {
      throw new Error(`PropertyDefinition.Definition item "${itemName}" selection options are invalid`);
    }
    const rawLabel = selectionOptionLabel(opt);
    const label =
      rawLabel === undefined || rawLabel === null
        ? value
        : (normalizePropertyLabelForStorage(rawLabel, `${itemName}.selection.${value}`) ?? value);
    if (Array.isArray(opt)) {
      out.push([value, label] as [string, PropertyLabel]);
    } else {
      out.push({ value, label });
    }
  }
  return out;
}

/**
 * Merge current-lang selection labels into previous options (by value).
 * Used by the definition editor so saving does not wipe other languages.
 */
export function mergeSelectionLabelsWrite(
  previous: PropertyItemDefinition['selection'] | undefined,
  incoming: Array<{ value: string; label: string }>,
  lang: string
): PropertyItemDefinition['selection'] {
  const prevByValue = new Map<string, unknown>();
  for (const opt of previous || []) {
    const value = selectionOptionValue(opt);
    if (value == null) continue;
    prevByValue.set(value, selectionOptionLabel(opt));
  }
  return incoming.map(opt => {
    const prevLabel = prevByValue.get(opt.value);
    const merged = mergePropertyLabelWrite(prevLabel, opt.label, lang);
    return [opt.value, merged] as [string, PropertyLabel];
  });
}

/**
 * Expand item `string` / `help` / selection labels to the request language.
 * Returns a new array; does not mutate input. Selection values are unchanged.
 */
export function translatePropertyDefinitionItems(
  items: PropertyItemDefinition[],
  lang: string
): PropertyItemDefinition[] {
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  return (items || []).map(item => {
    const next: PropertyItemDefinition = { ...item };
    if (Object.prototype.hasOwnProperty.call(item, 'string')) {
      next.string = unwrapPropertyLabel(item.string, code);
    }
    if (Object.prototype.hasOwnProperty.call(item, 'help')) {
      next.help = unwrapPropertyLabel(item.help, code);
    }
    if (Array.isArray(item.selection)) {
      next.selection = item.selection.map(opt => {
        if (Array.isArray(opt) && typeof opt[0] === 'string') {
          const value = opt[0];
          const label = unwrapPropertyLabel(opt[1] ?? value, code) || value;
          return [value, label] as [string, string];
        }
        if (opt && typeof opt === 'object' && !Array.isArray(opt) && typeof (opt as { value?: unknown }).value === 'string') {
          const value = String((opt as { value: string }).value);
          const label = unwrapPropertyLabel((opt as { label?: unknown }).label ?? value, code) || value;
          return { value, label };
        }
        return opt;
      }) as PropertyItemDefinition['selection'];
    }
    return next;
  });
}
