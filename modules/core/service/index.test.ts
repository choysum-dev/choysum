// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as serviceApi from './index';

test('core/service entrypoint export surface stays limited to modeling DSL primitives', () => {
  expect(Object.keys(serviceApi).sort()).toEqual([
    'AppSettingBaseModel',
    'BaseModel',
    'Compute',
    'Decimal',
    'Field',
    'Inverse',
    'Model',
    'Search',
    'SqlCompute',
    'dial',
    'pool',
  ]);
});

test('core/service entrypoint supports safe require replay without cache mutation', () => {
  const replay = require('./index');

  expect(replay.BaseModel).toBe(serviceApi.BaseModel);
  expect(replay.AppSettingBaseModel).toBe(serviceApi.AppSettingBaseModel);
  expect(replay.pool).toBe(serviceApi.pool);
  expect(replay.dial).toBe(serviceApi.dial);
  expect(replay.Compute).toBe(serviceApi.Compute);
  expect(replay.Field).toBe(serviceApi.Field);
  expect(replay.Inverse).toBe(serviceApi.Inverse);
  expect(replay.Model).toBe(serviceApi.Model);
  expect(replay.Search).toBe(serviceApi.Search);
  expect(replay.SqlCompute).toBe(serviceApi.SqlCompute);
  expect(replay.Decimal).toBe(serviceApi.Decimal);
});
