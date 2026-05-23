// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export interface ScopedStore {
  storeId: string;
  destroy: () => void;
}

export interface IStoreScopeManager {
  readonly type: string; // 'global' | 'component' | 'menu' | 'tab' | 'custom'
  register(store: ScopedStore, opts?: { scopeId?: string; persistent?: boolean }): void;
  unregister(storeId: string): void;
  destroyScope(scopeId: string): void;
  destroyAll(): void;
}
