// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CamelCasePlugin, KyselyPlugin, PluginTransformResultArgs, QueryResult } from 'kysely';
import type { UnknownRow } from '../types';

export type { UnknownRow };

const KEY_CACHE = new Map<string, string>();

export class ChoysumCamelCasePlugin extends CamelCasePlugin implements KyselyPlugin {
  async transformResult(args: PluginTransformResultArgs): Promise<QueryResult<UnknownRow>> {
    const rows = args.result.rows as UnknownRow[] | undefined;
    if (!rows || rows.length === 0) {
      return args.result;
    }

    // Fast path: skip when the first row is already PascalCase or starts with $ like $rel$....
    const first = rows[0]!;
    const keys = Object.keys(first);
    const alreadyPascal = keys.every(k => k.startsWith('$') || (k[0] >= 'A' && k[0] <= 'Z' && k.indexOf('_') === -1));
    if (alreadyPascal) {
      return args.result;
    }

    const out = new Array(rows.length);
    for (let i = 0; i < rows.length; i++) {
      const row = rows[i]!;
      const obj: UnknownRow = {};
      for (const k of Object.keys(row)) {
        // 1) Preserve top-level keys that start with $, such as $rel$....
        if (k.startsWith('$')) {
          obj[k] = row[k];
          continue;
        }
        // 2) Preserve internal keys that start with "__", such as __count or __level.
        if (k.startsWith('__')) {
          obj[k] = row[k];
          continue;
        }
        // 3) For aggregate suffixes, PascalCase only the base and keep the "__agg" suffix unchanged.
        const idx = k.indexOf('__');
        if (idx >= 0) {
          const base = k.slice(0, idx); // For example: total_amount.
          const suffix = k.slice(idx); // For example: __sum or __max.
          const nk = camelCaseFast(base) + suffix;
          obj[nk] = row[k];
          continue;
        }
        // 4) Convert ordinary keys to PascalCase.
        const nk = this.camelCase(k);
        obj[nk] = row[k];
      }
      out[i] = obj;
    }

    const res = { ...args.result, rows: out };
    return res;
  }

  protected override camelCase(str: string): string {
    return camelCaseFast(str);
  }
}

// High-performance PascalCase conversion with the Id special case, caching, and fast paths.
function camelCaseFast(str: string): string {
  if (str.length === 0) return str;
  // Cache hit.
  const cached = KEY_CACHE.get(str);
  if (cached) return cached;

  // Already PascalCase without underscores, so reuse it directly.
  if (str[0] >= 'A' && str[0] <= 'Z' && str.indexOf('_') === -1) {
    KEY_CACHE.set(str, str);
    return str;
  }

  if (str === 'id') {
    KEY_CACHE.set(str, 'Id');
    return 'Id';
  }

  // No underscores: only uppercase the first letter.
  if (str.indexOf('_') === -1) {
    const r = str[0].toUpperCase() + str.slice(1);
    KEY_CACHE.set(str, r);
    return r;
  }

  // With underscores: build the result character by character.
  let out = '';
  let upperNext = true; // The first character should be uppercase.
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
