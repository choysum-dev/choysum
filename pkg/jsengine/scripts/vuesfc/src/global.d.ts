// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import type { CompilerFs } from './compiler-fs';

declare global {
  interface GlobalThis {
    compilerFs: CompilerFs;
  }
}

export {};
