// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Store-level FieldsGet cache helpers (design D7).
 *
 * Cache key: lang + sorted(fields) + sorted(attributes). No draft (D9).
 *
 * Concurrent ensureFieldsGet calls that share the same lang+attributes are
 * coalesced into one RPC (union of field names) so a form with N ChoyFieldBase
 * mounts does not fire N FieldsGet round-trips.
 */

import { shallowRef } from 'vue';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';

/**
 * Shared attribute list for field-component ensure calls (selection + ACL overlay).
 * Keep identical across ChoyFieldBase / SelectionField so cache keys dedupe (P5).
 */
export const FIELD_PRESENTATION_FIELDS_GET_ATTRS = [
  'type',
  'string',
  'stringText',
  'help',
  'helpText',
  'selection',
  'selectionKind',
  'isReadonly',
  'notNull',
  'translate',
  'companyDependent',
  'size',
  'maxUploadBytes',
  'maxWidth',
  'maxHeight',
] as const;

export type FieldsGetRpc = (
  fields?: string[],
  attributes?: string[]
) => Promise<Record<string, WebFieldMetadata>>;

/** Stub shape for tests: sync or async return. */
export type FieldsGetRpcLike = (
  fields?: string[],
  attributes?: string[]
) => Promise<Record<string, WebFieldMetadata>> | Record<string, WebFieldMetadata>;

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
  getFieldsGetTranslatedHelp: (name: string) => string | undefined;
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

function normalizeFieldNames(fields?: string[]): string[] {
  if (!fields || fields.length === 0) return [];
  return [...new Set(fields.map(v => String(v || '').trim()).filter(Boolean))];
}

/** True when caller asked for the whole model (empty / omitted fields list). */
function wantsAllFields(fields?: string[]): boolean {
  return !fields || fields.length === 0 || normalizeFieldNames(fields).length === 0;
}

function pickFieldSlice(
  slice: Record<string, WebFieldMetadata>,
  names: string[]
): Record<string, WebFieldMetadata> {
  const out: Record<string, WebFieldMetadata> = {};
  for (const name of names) {
    if (Object.prototype.hasOwnProperty.call(slice, name)) {
      out[name] = slice[name]!;
    }
  }
  return out;
}

type CoalesceWaiter = {
  resolve: (value: Record<string, WebFieldMetadata>) => void;
  reject: (reason?: unknown) => void;
  /** Normalized field names; empty means whole-model request. */
  names: string[];
  wantAll: boolean;
  cacheKey: string;
};

type CoalesceBucket = {
  attributes: string[] | undefined;
  fieldSet: Set<string>;
  wantAll: boolean;
  waiters: CoalesceWaiter[];
  /** Shared promise for callers that join after flush started. */
  inflight?: Promise<Record<string, WebFieldMetadata>>;
};

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
  /** lang|attrs → pending microtask-coalesced ensures. */
  const coalesceByLangAttrs = new Map<string, CoalesceBucket>();

  const bumpOverlay = () => {
    overlayVersion.value += 1;
  };

  const clearFieldsGetCache = () => {
    responseCache.clear();
    inflight.clear();
    overlayByLang.clear();
    coalesceByLangAttrs.clear();
    bumpOverlay();
  };

  const mergeOverlay = (lang: string, slice: Record<string, WebFieldMetadata>) => {
    const prev = overlayByLang.get(lang) || {};
    overlayByLang.set(lang, { ...prev, ...slice });
    bumpOverlay();
  };

  const overlayHasFields = (lang: string, names: string[]): boolean => {
    const overlay = overlayByLang.get(lang) || {};
    return names.every(name => Object.prototype.hasOwnProperty.call(overlay, name));
  };

  const sliceFromOverlay = (lang: string, names: string[]): Record<string, WebFieldMetadata> => {
    return pickFieldSlice(overlayByLang.get(lang) || {}, names);
  };

  const flushCoalesceBucket = (bucketKey: string, lang: string, bucket: CoalesceBucket) => {
    coalesceByLangAttrs.delete(bucketKey);

    const rpcFields = bucket.wantAll ? undefined : [...bucket.fieldSet].sort();
    const rpcKey = buildCacheKey(lang, rpcFields, bucket.attributes);

    const resolveWaiters = (slice: Record<string, WebFieldMetadata>) => {
      for (const waiter of bucket.waiters) {
        let value = responseCache.get(waiter.cacheKey);
        if (!value) {
          value = waiter.wantAll ? slice : pickFieldSlice(slice, waiter.names);
          responseCache.set(waiter.cacheKey, value);
        }
        waiter.resolve(value);
      }
    };

    const cached = responseCache.get(rpcKey);
    if (cached) {
      resolveWaiters(cached);
      return;
    }

    const pendingExact = inflight.get(rpcKey);
    const request =
      pendingExact ||
      (async () => {
        const slice = (await host.FieldsGet(rpcFields, bucket.attributes)) || {};
        responseCache.set(rpcKey, slice);
        mergeOverlay(lang, slice);
        inflight.delete(rpcKey);
        return slice;
      })().catch(err => {
        inflight.delete(rpcKey);
        throw err;
      });

    if (!pendingExact) {
      inflight.set(rpcKey, request);
    }
    bucket.inflight = request;

    void request.then(resolveWaiters).catch(err => {
      for (const waiter of bucket.waiters) {
        waiter.reject(err);
      }
    });
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

    const wantAll = wantsAllFields(fields);
    const names = wantAll ? [] : normalizeFieldNames(fields);

    // Specific-field ensure: reuse overlay from a prior coalesced / broader fetch.
    if (!wantAll && names.length > 0 && overlayHasFields(lang, names)) {
      const fromOverlay = sliceFromOverlay(lang, names);
      responseCache.set(key, fromOverlay);
      return fromOverlay;
    }

    const pending = inflight.get(key);
    if (pending) {
      return pending;
    }

    // Coalesce concurrent ensures that share lang + attributes into one RPC.
    const attrsPart = sortedKeyPart(attributes);
    const bucketKey = `${lang}|${attrsPart}`;
    let bucket = coalesceByLangAttrs.get(bucketKey);
    if (!bucket) {
      bucket = {
        attributes: attributes ? [...attributes] : undefined,
        fieldSet: new Set<string>(),
        wantAll: false,
        waiters: [],
      };
      coalesceByLangAttrs.set(bucketKey, bucket);
      queueMicrotask(() => {
        const current = coalesceByLangAttrs.get(bucketKey);
        if (!current) return;
        flushCoalesceBucket(bucketKey, lang, current);
      });
    }

    if (wantAll) {
      bucket.wantAll = true;
      bucket.fieldSet.clear();
    } else if (!bucket.wantAll) {
      for (const name of names) bucket.fieldSet.add(name);
    }

    return new Promise<Record<string, WebFieldMetadata>>((resolve, reject) => {
      bucket!.waiters.push({
        resolve,
        reject,
        names,
        wantAll,
        cacheKey: key,
      });
    });
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

  const getFieldsGetTranslatedHelp = (name: string): string | undefined => {
    void overlayVersion.value;
    const fieldName = String(name || '').trim();
    if (!fieldName) return undefined;
    const lang = resolveLang(options?.getLang);
    const overlay = overlayByLang.get(lang)?.[fieldName];
    const translated = typeof overlay?.help === 'string' ? overlay.help.trim() : '';
    return translated || undefined;
  };

  return {
    ensureFieldsGet,
    getFieldMeta,
    getFieldsGetTranslatedString,
    getFieldsGetTranslatedHelp,
    clearFieldsGetCache,
  };
}
