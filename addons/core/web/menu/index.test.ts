// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as menuApi from './index';

describe('core/web menu entrypoint export surface', () => {
  it('keeps the menu facade limited to stable menu primitives', () => {
    expect(Object.keys(menuApi).sort()).toEqual(['MenuSymbol', 'createMenuPlugin']);
  });
});
