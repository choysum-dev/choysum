// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import * as resourceApi from './index';

describe('core/web resource entrypoint export surface', () => {
  it('keeps the runtime facade limited to resource declaration helpers', () => {
    expect(Object.keys(resourceApi).sort()).toEqual([
      'clearResourceDeclarations',
      'defineAction',
      'defineMenu',
      'defineModelActions',
      'defineRoute',
      'getResourceDeclaration',
      'getResourceDeclarationFromMeta',
      'listResourceDeclarations',
    ]);
  });
});
