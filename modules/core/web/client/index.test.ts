// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as clientApi from './index';

test('core/web client entrypoint export surface: keeps the client facade limited to stable service helpers', () => {
  expect(Object.keys(clientApi).sort()).toEqual(['webApiService']);
});
