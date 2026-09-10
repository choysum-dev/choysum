// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as resourceApi from './index';

test('core/web resource entrypoint export surface: keeps the runtime facade limited to resource declaration helpers', () => {
  expect(Object.keys(resourceApi).sort()).toEqual([
    'clearResourceDeclarations',
    'defineAction',
    'defineMenu',
    'defineModelActions',
    'defineRoute',
    'getResourceDeclaration',
    'getResourceDeclarationFromMeta',
    'getRouteActionsFromMeta',
    'listResourceDeclarations',
  ]);
});
