// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@/web/web/stores/storeScopeManager` (+ `/component`). */

export class ComponentScopeManager {
  constructor() {
    this.type = 'component';
  }
  register() {}
  unregister() {}
  destroyScope() {}
  destroyAll() {}
}

export function useScopeManager() {
  return {
    menuScopeManager: {
      getScope: function () {
        return {};
      },
    },
    globalScopeManager: {
      register: function () {},
      unregister: function () {},
      destroyScope: function () {},
      destroyAll: function () {},
    },
    componentScopeManager: function () {
      return new ComponentScopeManager();
    },
  };
}

export default { useScopeManager: useScopeManager, ComponentScopeManager: ComponentScopeManager };
