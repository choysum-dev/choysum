// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ParseJSONResultsPlugin, KyselyPlugin, PluginTransformResultArgs, QueryResult } from 'kysely';
import { REL_ALIAS_PREFIX } from '../constants';
import type { UnknownRow } from '../types';
import { isObjectRecord } from '@/core/utils/object';

// Only parse $rel$ columns: strings attempt JSON.parse, while objects and arrays recurse as-is.
export class ChoysumParseJSONResultsPlugin extends ParseJSONResultsPlugin implements KyselyPlugin {
  async transformResult(args: PluginTransformResultArgs): Promise<QueryResult<UnknownRow>> {
    const rows = args.result.rows as UnknownRow[] | undefined;
    if (!rows || rows.length === 0) {
      return args.result;
    }

    // Fast sampling path: if the first 1 to 3 rows are already normalized, skip the whole transform.
    const sampleN = Math.min(3, rows.length);
    let canFastSkip = true;
    for (let i = 0; i < sampleN; i++) {
      const row = rows[i];
      if (!row) continue;
      const relKeys = Object.keys(row).filter(k => k.startsWith(REL_ALIAS_PREFIX));
      if (relKeys.length === 0) continue;
      for (const k of relKeys) {
        const v = row[k];
        const suffix = k.slice(REL_ALIAS_PREFIX.length);
        const looksPascal = !!suffix && suffix[0] >= 'A' && suffix[0] <= 'Z' && suffix.indexOf('_') === -1;
        if (!looksPascal || typeof v === 'string') {
          canFastSkip = false;
          break;
        }
      }
      if (!canFastSkip) break;
    }
    if (canFastSkip) {
      return args.result;
    }

    // Regular path: only process $rel$* keys and parse or rename them when needed.
    for (let i = 0; i < rows.length; i++) {
      const row = rows[i];
      if (!row) continue;

      // Collect keys first so renaming does not mutate the object during iteration.
      const relKeys = Object.keys(row).filter(k => k.startsWith(REL_ALIAS_PREFIX));

      for (const k of relKeys) {
        const rawVal = row[k];

        // Fast path: skip when the key is already $rel$PascalCase and the value is not a string.
        const suffix = k.slice(REL_ALIAS_PREFIX.length);
        const looksPascal = !!suffix && suffix[0] >= 'A' && suffix[0] <= 'Z' && suffix.indexOf('_') === -1;
        if (looksPascal && typeof rawVal !== 'string') {
          continue;
        }

        const parsed = parseRelValue(rawVal);
        const camelized = camelizeDeepInPlace(parsed);

        // Normalize aliases as $rel$ + PascalCase after trimming leading underscores from snake_case suffixes.
        const normSuffix = suffix.replace(/^_+/, '');
        const pascal = camelCaseFast(normSuffix);
        const newKey = `${REL_ALIAS_PREFIX}${pascal}`;

        // Write back, migrating the key name when normalization changed it.
        if (newKey !== k) {
          if (row[newKey] === undefined) {
            row[newKey] = camelized;
          }
          delete row[k];
        } else {
          row[k] = camelized;
        }
      }
    }
    return args.result;
  }
}

// Lightweight helpers used only for $rel$ values.

// Cheap JSON-likeness check based on the first character.
function maybeJsonFast(str: string): boolean {
  if (!str) return false;
  const c = str.charCodeAt(0);
  return c === 123 /* '{' */ || c === 91 /* '[' */;
}

function parseRelValue(v: unknown): unknown {
  if (typeof v === 'string') {
    if (!maybeJsonFast(v)) return v;
    try {
      return JSON.parse(v);
    } catch {
      return v;
    }
  }
  return v;
}

// PascalCase conversion aligned with the CamelCase plugin, including the Id special case and a small cache.
const KEY_CACHE = new Map<string, string>();
function camelCaseFast(str: string): string {
  if (!str) return str;
  const cached = KEY_CACHE.get(str);
  if (cached) return cached;

  if (str === 'id') {
    KEY_CACHE.set(str, 'Id');
    return 'Id';
  }
  if (str[0] >= 'A' && str[0] <= 'Z' && str.indexOf('_') === -1) {
    KEY_CACHE.set(str, str);
    return str;
  }
  if (str.indexOf('_') === -1) {
    const r = str[0].toUpperCase() + str.slice(1);
    KEY_CACHE.set(str, r);
    return r;
  }
  let out = '';
  let upperNext = true;
  for (let i = 0; i < str.length; i++) {
    const ch = str[i];
    if (ch === '_') {
      upperNext = true;
      continue;
    }
    out += upperNext ? ch.toUpperCase() : ch;
    upperNext = false;
  }
  KEY_CACHE.set(str, out);
  return out;
}

// Recurse only through $rel$ values, converting object keys to PascalCase and descending into arrays.
function camelizeDeepInPlace(v: unknown): unknown {
  if (Array.isArray(v)) {
    for (let i = 0; i < v.length; i++) {
      const item = v[i];
      if (isObjectRecord(item) || Array.isArray(item)) {
        v[i] = camelizeDeepInPlace(item);
      }
    }
    return v;
  }
  if (isObjectRecord(v)) {
    const keys = Object.keys(v);
    for (const k of keys) {
      const val = v[k];
      const nk = camelCaseFast(k);
      const nv = isObjectRecord(val) || Array.isArray(val) ? camelizeDeepInPlace(val) : val;
      if (nk !== k) delete v[k];
      v[nk] = nv;
    }
    return v;
  }
  return v;
}
