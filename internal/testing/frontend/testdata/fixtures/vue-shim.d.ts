// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** Ambient *.vue module for FE unit fixtures under this tsconfig. */
declare module '*.vue' {
  // Fixtures only need a mountable default export for IDE checking.
  const component: any;
  export default component;
}
