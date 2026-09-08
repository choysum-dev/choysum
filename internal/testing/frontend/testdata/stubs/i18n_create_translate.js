// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@/web/web/i18n` createTranslate used by pages under QJS. */
export function createTranslate(_app, _opts) {
  return {
    _t: function (msg) {
      return String(msg == null ? '' : msg);
    },
  };
}

export default { createTranslate: createTranslate };
