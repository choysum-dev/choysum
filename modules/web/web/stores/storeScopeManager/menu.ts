// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseScopeManager } from './base';
import { effectScope, watch } from 'vue';
import { useMenuStore } from '@/web/web/stores/menuStore';

export class MenuScopeManager extends BaseScopeManager {
  private static _inst: MenuScopeManager | null = null;
  private static _scope = effectScope(true);

  static instance() {
    return (this._inst ??= new MenuScopeManager());
  }

  private constructor() {
    super('menu');

    // Run the watcher in a dedicated effectScope to avoid mixing with component scopes.
    MenuScopeManager._scope.run(() => {
      const menuStore = useMenuStore();

      watch(
        () => menuStore.activeMenuId,
        (nextId, prevId) => {
          if (prevId && prevId !== nextId) {
            this.destroyScope(prevId);
          }
        },
        { immediate: true }
      );
    });
  }

  protected getCurrentScopeId(): string | undefined {
    return useMenuStore().activeMenuId ?? undefined;
  }
}
