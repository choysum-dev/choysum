// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';

const getByPath = (obj: any, path: string) =>
  path
    .split('.')
    .filter(Boolean)
    .reduce((a, k) => (a == null ? a : a[k]), obj);

const setByPath = (obj: any, path: string, v: any) => {
  const segs = path.split('.').filter(Boolean);
  let cur = obj;
  for (let i = 0; i < segs.length - 1; i++) {
    const k = segs[i];
    if (!cur[k] || typeof cur[k] !== 'object') cur[k] = {};
    cur = cur[k];
  }
  cur[segs[segs.length - 1]] = v;
};

/**
 * Provides array mutation helpers for list results or to-many form fields.
 */
export function useArrayMutations<T extends BaseModel>(opts: {
  store: WebModelStore<T>;

  /** Optional to-many field path; when omitted the list result array is mutated. */
  prop?: string;

  /** Optional external row provider that overrides the default list source. */
  listProvider?: { get: () => any[]; set?: (items: any[]) => void };
  /** Optional form or draft root accessor used instead of store.state._draftRecord/record. */
  recordGetter?: () => any;
  /** Optional immutable-style whole-record setter. */
  recordSetter?: (next: any) => void;
}) {
  const { store } = opts;
  const prop = (opts.prop ?? '').trim();
  const listProvider = opts.listProvider;

  // Read the to-many array from the current record.
  function recordArray(): any[] {
    const root = opts.recordGetter ? opts.recordGetter() : undefined;
    if (!prop) return [];
    if (!root) return [];
    let arr = getByPath(root, prop);
    if (!Array.isArray(arr)) {
      setByPath(root, prop, []);
      arr = getByPath(root, prop);
    }
    return arr as any[];
  }

  // Read the flat list result when no record path is provided.
  function listArray(): any[] {
    if (listProvider) {
      const arr = listProvider.get?.() as any;
      return Array.isArray(arr) ? arr : [];
    }
    const res: any = store.state.result as any;
    if (!res) return [];
    if (res.kind && res.kind !== 'record') {
      try {
        console.warn(
          '[useArrayMutations] Result shape is',
          res.kind,
          '; default list mutations only support kind="record". Provide leaf rows through listProvider for grouped data.'
        );
      } catch {}
      return [];
    }
    const rs = res.rows as any[];
    return Array.isArray(rs) ? rs : [];
  }

  // Resolve the active mutation target.
  function targetArray(): any[] {
    return prop ? recordArray() : listArray();
  }

  function withAssign(mutate: (arr: any[]) => void) {
    if (!prop && listProvider?.set) {
      const src = listArray();
      const next = Array.isArray(src) ? src.slice() : [];
      mutate(next);
      listProvider.set(next);
      return;
    }
    const arr = targetArray();
    mutate(arr);
    if (prop && opts.recordSetter && opts.recordGetter) {
      const root = opts.recordGetter();
      opts.recordSetter(root);
    }
  }

  function add(item: any, index?: number) {
    withAssign(arr => {
      if (index == null || index < 0 || index > arr.length) arr.push(item);
      else arr.splice(index, 0, item);
    });
  }

  function remove(index: number) {
    withAssign(arr => {
      if (index >= 0 && index < arr.length) arr.splice(index, 1);
    });
  }

  function move(from: number, to: number) {
    withAssign(arr => {
      if (from === to) return;
      if (from < 0 || from >= arr.length || to < 0 || to >= arr.length) return;
      const [it] = arr.splice(from, 1);
      arr.splice(to, 0, it);
    });
  }

  function clear() {
    withAssign(arr => arr.splice(0, arr.length));
  }

  // Replace the entire array contents in one mutation.
  function replaceAll(items: any[]) {
    withAssign(arr => {
      arr.splice(0, arr.length, ...items);
    });
  }

  // Return a shallow copy so callers do not mutate the source array directly.
  function getAll(): any[] {
    const arr = targetArray();
    return arr.slice();
  }

  return { add, remove, move, clear, replaceAll, getAll };
}
