// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { IStoreScopeManager, ScopedStore } from './types';

export abstract class BaseScopeManager implements IStoreScopeManager {
  readonly type: string;
  protected scopes = new Map<string, Map<string, () => void>>();
  protected persistent = new Map<string, () => void>();

  constructor(type: string) {
    this.type = type;
  }

  register(store: ScopedStore, opts: { scopeId?: string; persistent?: boolean } = {}) {
    if (opts.persistent) {
      this.persistent.set(store.storeId, store.destroy);
      return;
    }
    const sid = opts.scopeId || this.getCurrentScopeId();
    if (!sid) return;
    let bucket = this.scopes.get(sid);
    if (!bucket) {
      bucket = new Map();
      this.scopes.set(sid, bucket);
    }
    bucket.set(store.storeId, store.destroy);
  }

  unregister(storeId: string) {
    for (const bucket of this.scopes.values()) {
      bucket.delete(storeId);
    }
    this.persistent.delete(storeId);
  }

  destroyScope(scopeId: string) {
    const bucket = this.scopes.get(scopeId);
    if (!bucket) return;
    for (const fn of bucket.values()) {
      try {
        fn();
      } catch {}
    }
    this.scopes.delete(scopeId);
  }

  destroyAll() {
    for (const sid of Array.from(this.scopes.keys())) {
      this.destroyScope(sid);
    }
    for (const fn of this.persistent.values()) {
      try {
        fn();
      } catch {}
    }
    this.persistent.clear();
  }

  // Implemented by subclasses to return the default scopeId when none is passed.
  protected abstract getCurrentScopeId(): string | undefined;
}
