// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as serviceApi from './index';

test('core/service entrypoint export surface stays limited to modeling DSL primitives', () => {
  expect(Object.keys(serviceApi).sort()).toEqual(['BaseModel', 'Decimal', 'Field', 'Model']);
});

test('core/service entrypoint supports safe require replay without cache mutation', () => {
  const replay = require('./index');

  expect(replay.BaseModel).toBe(serviceApi.BaseModel);
  expect(replay.Field).toBe(serviceApi.Field);
  expect(replay.Model).toBe(serviceApi.Model);
  expect(replay.Decimal).toBe(serviceApi.Decimal);
});
