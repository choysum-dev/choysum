// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as coreWeb from './index';

describe('core/web root export surface', () => {
  it('keeps the root facade limited to component primitives', () => {
    expect(Object.keys(coreWeb).sort()).toEqual(['Xpath']);
  });
});
