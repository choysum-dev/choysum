// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import * as sass from './sass';

// attention: beaware of cannot import sass module in other place
//  cause vue will using sassRequire to import sass module,
export function sassRequire(module: string) {
  if (module === 'sass') {
    return sass;
  } else {
    return undefined;
  }
}
