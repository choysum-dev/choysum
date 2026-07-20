// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Store-level FieldsGet cache helpers (design D7).
 *
 * Cache key: lang + sorted(fields) + sorted(attributes). No draft (D9).
 */

import { shallowRef } from 'vue';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';

/**
 * Shared attribute list for field-component ensure calls (selection + ACL overlay).
 * Keep identical across OFieldBase / OSelectionField so cache keys dedupe (P5).
 */
export const FIELD_PRESENTATION_FIELDS_GET_ATTRS = [
  'type',
  'string',
  'stringText',
  'selection',
  'selectionKind',
  'isReadonly',
  'notNull',
] as const;

export type FieldsGetRpc = (
  fields?: string[],
  attributes?: string[]
) => Promise<Record<string, WebFieldMetadata>>;

export type FieldsGetHost = {
  fieldsMetadata: Record<string, WebFieldMetadata>;
  FieldsGet: FieldsGetRpc;
};

export type CreateFieldsGetHelpersOptions = {
  /** Active terminology lang (defaults to en_US when unset). */
  getLang?: () => string;
};

export type FieldsGetHelpers = {
  ensureFieldsGet: (
    fields?: string[],
    attributes?: string[]
  ) => Promise<Record<string, WebFieldMetadata>>;
  getFieldMeta: (name: string) => WebFieldMetadata | undefined;
  getFieldsGetTranslatedString: (name: string) => string | undefined;
  clearFieldsGetCache: () => void;
};

function sortedKeyPart(values: string[] | undefined): string {
  if (!values || values.length === 0) return '*';
  return [...new Set(values.map(v => String(v || '').trim()).filter(Boolean))].sort().join(',');
}

function buildCacheKey(lang: string, fields?: string[], attributes?: string[]): string {
  return `${lang}|${sortedKeyPart(fields)}|${sortedKeyPart(attributes)}`;
}

function resolveLang(getLang?: () => string): string {
  try {
    const lang = String(getLang?.() || '').trim();
    return lang || 'en_US';
  } catch {
    return 'en_US';
  }
}

/**
 * Attach ensureFieldsGet / getFieldMeta helpers to a model store host.
 */
export function createFieldsGetHelpers(
  host: FieldsGetHost,
  options?: CreateFieldsGetHelpersOptions
): FieldsGetHelpers {
  const responseCache = new Map<string, Record<string, WebFieldMetadata>>();
  const inflight = new Map<string, Promise<Record<string, WebFieldMetadata>>>();
  /** Per-lang overlay written after successful ensures (for getFieldMeta). */
  const overlayByLang = new Map<string, Record<string, WebFieldMetadata>>();
  /** Bumped so Vue computeds that call getFieldMeta / getFieldsGetTranslatedString re-run. */
  const overlayVersion = shallowRef(0);

  const bumpOverlay = () => {
    overlayVersion.value += 1;
  };

  const clearFieldsGetCache = () => {
    responseCache.clear();
    inflight.clear();
    overlayByLang.clear();
    bumpOverlay();
  };

  const mergeOverlay = (lang: string, slice: Record<string, WebFieldMetadata>) => {
    const prev = overlayByLang.get(lang) || {};
    overlayByLang.set(lang, { ...prev, ...slice });
    bumpOverlay();
  };

  const ensureFieldsGet = async (
    fields?: string[],
    attributes?: string[]
  ): Promise<Record<string, WebFieldMetadata>> => {
    const lang = resolveLang(options?.getLang);
    const key = buildCacheKey(lang, fields, attributes);
    const cached = responseCache.get(key);
    if (cached) {
      return cached;
    }
    const pending = inflight.get(key);
    if (pending) {
      return pending;
    }

    const request = (async () => {
      const slice = (await host.FieldsGet(fields, attributes)) || {};
      responseCache.set(key, slice);
      mergeOverlay(lang, slice);
      inflight.delete(key);
      return slice;
    })().catch(err => {
      inflight.delete(key);
      throw err;
    });

    inflight.set(key, request);
    return request;
  };

  const getFieldMeta = (name: string): WebFieldMetadata | undefined => {
    void overlayVersion.value;
    const fieldName = String(name || '').trim();
    if (!fieldName) return undefined;
    const staticMeta = host.fieldsMetadata?.[fieldName];
    const lang = resolveLang(options?.getLang);
    const overlay = overlayByLang.get(lang)?.[fieldName];
    if (!staticMeta && !overlay) return undefined;
    if (!overlay) return staticMeta;
    if (!staticMeta) return { ...overlay };
    return { ...staticMeta, ...overlay };
  };

  const getFieldsGetTranslatedString = (name: string): string | undefined => {
    void overlayVersion.value;
    const fieldName = String(name || '').trim();
    if (!fieldName) return undefined;
    const lang = resolveLang(options?.getLang);
    const overlay = overlayByLang.get(lang)?.[fieldName];
    const translated = typeof overlay?.string === 'string' ? overlay.string.trim() : '';
    return translated || undefined;
  };

  return {
    ensureFieldsGet,
    getFieldMeta,
    getFieldsGetTranslatedString,
    clearFieldsGetCache,
  };
}
