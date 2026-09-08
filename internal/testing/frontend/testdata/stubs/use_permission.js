// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@/auth/web/composables/usePermission`. */
export function usePermission() {
  return {
    hasAction: function () {
      return true;
    },
    canRoute: function () {
      return true;
    },
    canMenu: function () {
      return true;
    },
  };
}

export default { usePermission: usePermission };
