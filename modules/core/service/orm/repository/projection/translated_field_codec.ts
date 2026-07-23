// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, ModelMetadata } from '../../metadata';
import { resolveRequestLang } from '../../../i18n/request_lang';
import { getContextLang, getCtxValue } from '../../../runtime/context/scope';
import type { Entity } from '../types';
import type { ObjectRecord, UnknownRecord } from '../../../../utils/types';

/** Base / fallback language for translated field values (data-i18n-design.md D3). */
export const TRANSLATED_BASE_LANG = 'en_US';

export type TranslatedWriteMode = 'create' | 'update';

function hasOwn(map: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(map, key);
}

function maybeJsonFast(str: string): boolean {
  if (!str) return false;
  const c = str.charCodeAt(0);
  return c === 123 || c === 91 || c === 34 || c === 45 || (c >= 48 && c <= 57) || c === 116 || c === 102 || c === 110;
}

function parseJsonObjectLike(v: unknown): unknown {
  if (v == null) return {};
  if (typeof v === 'string') {
    const s = v.trim();
    if (!s) return {};
    if (!maybeJsonFast(s)) return s;
    try {
      return JSON.parse(s);
    } catch {
      return s;
    }
  }
  if (typeof v === 'object') return v;
  return v;
}

/**
 * Resolve the language used for translated field unwrap/wrap.
 * Prefer Model/request ctx.lang; never treat UI locale (zh-CN) as lang.
 */
export function resolveTranslatedFieldLang(): string {
  const fromCtx = getContextLang();
  if (fromCtx) return fromCtx;
  return resolveRequestLang(undefined, { final: TRANSLATED_BASE_LANG });
}

/** When true, decode returns the full lang map instead of an unwrapped string. */
export function getPrefetchLangs(): boolean {
  const raw = getCtxValue('prefetch_langs') ?? getCtxValue('prefetchLangs');
  return raw === true;
}

export function assertTranslatedLangKey(lang: string, fieldName: string): string {
  const trimmed = String(lang || '').trim();
  if (!trimmed) {
    throw new Error(`Translated field "${fieldName}" requires a non-empty language code`);
  }
  // UI keys use hyphens (zh-CN); column keys must be POSIX terminology codes (zh_CN).
  if (trimmed.includes('-')) {
    throw new Error(
      `Translated field "${fieldName}" language key "${trimmed}" looks like a UI locale; use a terminology code such as zh_CN`
    );
  }
  return trimmed;
}

export function isTranslatedLangMap(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

export function parseTranslatedStoredMap(value: unknown): Record<string, string> | null {
  if (value == null) return null;
  const parsed = parseJsonObjectLike(value);
  if (!isTranslatedLangMap(parsed)) return null;
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(parsed)) {
    if (v == null) {
      out[k] = '';
      continue;
    }
    if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') {
      out[k] = String(v);
    }
  }
  return out;
}

/**
 * Unwrap a stored lang map for the request language.
 * Key present with '' → '' (no fallback). Missing key → en_US → null.
 */
export function unwrapTranslatedValue(map: Record<string, unknown> | null | undefined, lang: string): string | null {
  if (map == null || !isTranslatedLangMap(map)) return null;
  if (hasOwn(map, lang)) {
    const v = map[lang];
    return v == null ? '' : String(v);
  }
  if (hasOwn(map, TRANSLATED_BASE_LANG)) {
    const v = map[TRANSLATED_BASE_LANG];
    return v == null ? '' : String(v);
  }
  return null;
}

export function decodeTranslatedFieldValue(stored: unknown, opts?: { lang?: string; prefetchLangs?: boolean }): string | Record<string, string> | null {
  const map = parseTranslatedStoredMap(stored);
  if (map == null) return null;
  if (opts?.prefetchLangs ?? getPrefetchLangs()) {
    return { ...map };
  }
  const lang = assertTranslatedLangKey(opts?.lang ?? resolveTranslatedFieldLang(), '(translated)');
  return unwrapTranslatedValue(map, lang);
}

function assertValueSize(fieldName: string, value: string, size: number | undefined): void {
  if (size == null) return;
  if (value.length > size) {
    throw new Error(`Translated field "${fieldName}" value exceeds size=${size} (got length ${value.length})`);
  }
}

function normalizeIncomingLangMap(fieldName: string, raw: Record<string, unknown>, size: number | undefined): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(raw)) {
    const lang = assertTranslatedLangKey(k, fieldName);
    if (v === false) {
      throw new Error(
        `Translated field "${fieldName}" does not accept false in write payloads; use UpdateFieldTranslations to delete a language key`
      );
    }
    if (v == null) {
      throw new Error(`Translated field "${fieldName}" lang map value for "${lang}" must be a string (use null on the field to clear the column)`);
    }
    const s = String(v);
    assertValueSize(fieldName, s, size);
    out[lang] = s;
  }
  return out;
}

/**
 * Merge a write value into the current lang map.
 * - null → whole-column clear (caller stores null)
 * - '' / string → set current lang key
 * - object → merge keys
 * - create mode: also write en_US when missing
 */
export function mergeTranslatedWrite(args: {
  fieldName: string;
  value: unknown;
  lang: string;
  currentMap: Record<string, string> | null;
  mode: TranslatedWriteMode;
  size?: number;
}): Record<string, string> | null {
  const { fieldName, value, mode, size } = args;
  const lang = assertTranslatedLangKey(args.lang, fieldName);

  if (value === null) {
    return null;
  }

  if (isTranslatedLangMap(value)) {
    const incoming = normalizeIncomingLangMap(fieldName, value, size);
    const merged: Record<string, string> = { ...(args.currentMap || {}), ...incoming };
    if (mode === 'create' && !hasOwn(merged, TRANSLATED_BASE_LANG)) {
      // Prefer explicit en_US from incoming; else seed from current write lang value.
      const seed = hasOwn(incoming, lang) ? incoming[lang] : Object.values(incoming)[0];
      if (seed != null) merged[TRANSLATED_BASE_LANG] = seed;
    }
    return merged;
  }

  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    const s = String(value);
    assertValueSize(fieldName, s, size);
    const merged: Record<string, string> = { ...(args.currentMap || {}) };
    merged[lang] = s;
    if (mode === 'create' && !hasOwn(merged, TRANSLATED_BASE_LANG)) {
      merged[TRANSLATED_BASE_LANG] = s;
    }
    return merged;
  }

  throw new Error(`Translated field "${fieldName}" expects string, lang map object, or null`);
}

export function encodeTranslatedMapForDb(map: Record<string, string> | null): string | null {
  if (map == null) return null;
  return JSON.stringify(map);
}

export function fieldTranslateSize(fm: FieldMetadata | undefined): number | undefined {
  const size = fm?.storageHints?.size;
  return typeof size === 'number' && Number.isInteger(size) && size > 0 ? size : undefined;
}

/**
 * Rewrite translate fields on a write payload into full lang maps (objects)
 * ready for encodeForDb JSON persistence.
 */
export function applyTranslatedFieldsForWrite(
  meta: ModelMetadata,
  input: Entity,
  opts: { mode: TranslatedWriteMode; lang?: string; current?: ObjectRecord | null }
): Entity {
  if (!input || typeof input !== 'object') return input;
  const lang = opts.lang ?? resolveTranslatedFieldLang();
  const out: UnknownRecord = { ...(input as UnknownRecord) };
  let changed = false;

  meta.fields.forEach((fm, name) => {
    if (!fm?.translate) return;
    if (!Object.prototype.hasOwnProperty.call(out, name)) return;
    const raw = out[name];
    if (raw === undefined) return;

    const currentStored = opts.current ? opts.current[name] : undefined;
    const currentMap = opts.mode === 'update' ? parseTranslatedStoredMap(currentStored) : null;
    const merged = mergeTranslatedWrite({
      fieldName: name,
      value: raw,
      lang,
      currentMap,
      mode: opts.mode,
      size: fieldTranslateSize(fm),
    });
    out[name] = merged;
    changed = true;
  });

  return (changed ? out : input) as Entity;
}

export function payloadHasTranslatedFieldWrite(meta: ModelMetadata, input: Entity): boolean {
  if (!input || typeof input !== 'object') return false;
  for (const [name, fm] of meta.fields) {
    if (!fm?.translate) continue;
    if (Object.prototype.hasOwnProperty.call(input, name) && (input as UnknownRecord)[name] !== undefined) {
      return true;
    }
  }
  return false;
}
