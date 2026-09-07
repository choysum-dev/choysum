// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as trackingApi from './index';

test('core/web tracking entrypoint export surface: keeps the tracking facade limited to draft tracking primitives', () => {
  expect(Object.keys(trackingApi).sort()).toEqual(['TrackedModel', 'track']);
});
