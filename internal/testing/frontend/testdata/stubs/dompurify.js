// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `dompurify`.
 * Keeps markup as-is so OHtmlField display/edit wiring can be asserted without a real sanitizer.
 */
var hooks = [];

var DOMPurify = {
  addHook: function (_name, fn) {
    hooks.push(fn);
  },
  sanitize: function (html) {
    return html == null ? '' : String(html);
  },
};

export default DOMPurify;
