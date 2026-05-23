// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from 'decimal.js';
import { isDecimal, isDecimalLeak } from './decimal';
import type { ObjectRecord } from './types';

type DecimalInternalWritable = Decimal & {
  s?: number;
  e?: number;
  d?: number[];
};

type DecimalLeakShape = {
  s?: number;
  e?: number;
  d?: number[];
};

export function deepClonePreserve<T>(o: T): T {
  const seen = new WeakMap<object, unknown>();

  const clone = (v: unknown): unknown => {
    if (v == null) return v;
    const t = typeof v;
    if (t !== 'object') return v;

    const objectValue = v as object;

    if (seen.has(objectValue)) return seen.get(objectValue);

    // Decimal or leaked Decimal structure.
    try {
      if (isDecimal(v)) return new Decimal(v.toString());
      if (isDecimalLeak(v)) {
        const leak = v as DecimalLeakShape;
        const reconstructed = new Decimal(0) as DecimalInternalWritable;
        if (typeof leak.s === 'number') reconstructed.s = leak.s;
        if (typeof leak.e === 'number') reconstructed.e = leak.e;
        if (Array.isArray(leak.d)) reconstructed.d = leak.d.slice();
        return reconstructed;
      }
    } catch {}

    // Date
    if (v instanceof Date) return new Date(v.getTime());

    // Array
    if (Array.isArray(v)) {
      const arr: unknown[] = [];
      seen.set(objectValue, arr);
      for (const item of v) arr.push(clone(item));
      return arr;
    }

    // Map
    if (v instanceof Map) {
      const m = new Map<unknown, unknown>();
      seen.set(objectValue, m);
      for (const [k, val] of v.entries()) m.set(k, clone(val));
      return m;
    }

    // Set
    if (v instanceof Set) {
      const s = new Set<unknown>();
      seen.set(objectValue, s);
      for (const val of v.values()) s.add(clone(val));
      return s;
    }

    // Plain object.
    const out: ObjectRecord = {};
    seen.set(objectValue, out);
    for (const [k, val] of Object.entries(v as ObjectRecord)) out[k] = clone(val);
    return out;
  };

  return clone(o) as T;
}
