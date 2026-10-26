// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@codemirror/state`. */

export const EditorState = {
  /**
   * @param {{ doc?: string, extensions?: unknown[] }} config
   */
  create: function (config) {
    var doc = config && config.doc != null ? String(config.doc) : '';
    return {
      doc: {
        toString: function () {
          return doc;
        },
        length: doc.length,
      },
      __feStubDoc: doc,
      extensions: (config && config.extensions) || [],
    };
  },
};
