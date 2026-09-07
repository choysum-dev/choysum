// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as applicationApi from './index';

test('core/web application entrypoint export surface: keeps the runtime facade limited to app container primitives', () => {
  expect(Object.keys(applicationApi).sort()).toEqual(['createApp', 'getPlugins']);
});
