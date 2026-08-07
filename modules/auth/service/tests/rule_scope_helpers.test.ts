// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { assertExclusiveScope } from '@/auth/service/models/_rule_scope_helpers';

test('rule scope helpers: method accepts service/model/application/logical/global shapes', () => {
  const service = { MetaServiceId: 'svc1', MetaModelId: null, MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(service, 'create', 'method');
  expect(service.MetaServiceId).toBe('svc1');
  expect(service.MetaModelId).toBe(null);
  expect(service.MetaApplicationId).toBe(null);
  expect(service.LogicalModelName).toBe(null);

  const model = { MetaServiceId: null, MetaModelId: 'm1', MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(model, 'create', 'method');
  expect(model.MetaModelId).toBe('m1');

  const app = { MetaServiceId: null, MetaModelId: null, MetaApplicationId: 'a1', LogicalModelName: null };
  assertExclusiveScope(app, 'create', 'method');
  expect(app.MetaApplicationId).toBe('a1');

  const logical = { MetaServiceId: null, MetaModelId: null, MetaApplicationId: null, LogicalModelName: 'TranslationTerm' };
  assertExclusiveScope(logical, 'create', 'method');
  expect(logical.LogicalModelName).toBe('TranslationTerm');

  const global = { MetaServiceId: null, MetaModelId: null, MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(global, 'create', 'method');
  expect(global.MetaServiceId).toBe(null);
  expect(global.MetaModelId).toBe(null);
  expect(global.MetaApplicationId).toBe(null);
  expect(global.LogicalModelName).toBe(null);
});

test('rule scope helpers: record accepts model/application/global shapes', () => {
  const model = { MetaModelId: 'm1', MetaApplicationId: null };
  assertExclusiveScope(model, 'create', 'record');
  expect(model.MetaModelId).toBe('m1');
  expect(model.MetaApplicationId).toBe(null);

  const app = { MetaModelId: null, MetaApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'record');
  expect(app.MetaApplicationId).toBe('a1');

  const global = { MetaModelId: null, MetaApplicationId: null };
  assertExclusiveScope(global, 'create', 'record');
  expect(global.MetaModelId).toBe(null);
  expect(global.MetaApplicationId).toBe(null);
});

test('rule scope helpers: field accepts field/model/application/logical/global shapes', () => {
  const field = { MetaFieldId: 'f1', MetaModelId: 'm1', MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(field, 'create', 'field');
  expect(field.MetaFieldId).toBe('f1');
  expect(field.MetaModelId).toBe('m1');
  expect(field.MetaApplicationId).toBe(null);
  expect(field.LogicalModelName).toBe(null);

  const model = { MetaFieldId: null, MetaModelId: 'm1', MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(model, 'create', 'field');
  expect(model.MetaModelId).toBe('m1');

  const app = { MetaFieldId: null, MetaModelId: null, MetaApplicationId: 'a1', LogicalModelName: null };
  assertExclusiveScope(app, 'create', 'field');
  expect(app.MetaApplicationId).toBe('a1');

  const logical = { MetaFieldId: null, MetaModelId: null, MetaApplicationId: null, LogicalModelName: 'AppSetting' };
  assertExclusiveScope(logical, 'create', 'field');
  expect(logical.LogicalModelName).toBe('AppSetting');

  const global = { MetaFieldId: null, MetaModelId: null, MetaApplicationId: null, LogicalModelName: null };
  assertExclusiveScope(global, 'create', 'field');
  expect(global.MetaFieldId).toBe(null);
  expect(global.LogicalModelName).toBe(null);
});

test('rule scope helpers: ui accepts resource/application/global shapes', () => {
  const resource = { MetaUiResourceId: 'u1', MetaApplicationId: null };
  assertExclusiveScope(resource, 'create', 'ui');
  expect(resource.MetaUiResourceId).toBe('u1');

  const app = { MetaUiResourceId: null, MetaApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'ui');
  expect(app.MetaApplicationId).toBe('a1');

  const global = { MetaUiResourceId: null, MetaApplicationId: null };
  assertExclusiveScope(global, 'create', 'ui');
  expect(global.MetaUiResourceId).toBe(null);
  expect(global.MetaApplicationId).toBe(null);
});

test('rule scope helpers: mixed scope throws golden error messages', () => {
  expect(() =>
    assertExclusiveScope({ MetaServiceId: 's1', MetaModelId: 'm1', MetaApplicationId: null, LogicalModelName: null }, 'create', 'method')
  ).toThrow('invalid RoleMethodAccess scope: must be exactly one of service/model/application/logical_model/global');
  expect(() =>
    assertExclusiveScope(
      { MetaServiceId: null, MetaModelId: null, MetaApplicationId: null, LogicalModelName: 'Partner' },
      'create',
      'method'
    )
  ).toThrow(/not a registered logical model/);
  expect(() => assertExclusiveScope({ MetaModelId: 'm1', MetaApplicationId: 'a1' }, 'create', 'record')).toThrow(
    'invalid RoleRecordRule scope: must be exactly one of model/application/global'
  );
  expect(() =>
    assertExclusiveScope({ MetaFieldId: 'f1', MetaModelId: null, MetaApplicationId: null, LogicalModelName: null }, 'create', 'field')
  ).toThrow('invalid RoleFieldRule scope: must be exactly one of field/model/application/logical_model/global');
  expect(() => assertExclusiveScope({ MetaUiResourceId: 'u1', MetaApplicationId: 'a1' }, 'create', 'ui')).toThrow(
    'invalid RoleUiResource scope: must be exactly one of resource/application/global'
  );
});

test('rule scope helpers: update missing sibling fields throws golden messages', () => {
  expect(() => assertExclusiveScope({ MetaModelId: 'm1' }, 'update', 'method')).toThrow(
    'invalid RoleMethodAccess scope update: must provide MetaServiceId/MetaModelId/MetaApplicationId/LogicalModelName together'
  );
  expect(() => assertExclusiveScope({ MetaModelId: 'm1' }, 'update', 'record')).toThrow(
    'invalid RoleRecordRule scope update: must provide MetaModelId/MetaApplicationId together'
  );
  expect(() => assertExclusiveScope({ MetaFieldId: 'f1' }, 'update', 'field')).toThrow(
    'invalid RoleFieldRule scope update: must provide MetaFieldId/MetaModelId/MetaApplicationId/LogicalModelName together'
  );
  expect(() => assertExclusiveScope({ MetaUiResourceId: 'u1' }, 'update', 'ui')).toThrow(
    'invalid RoleUiResource scope update: must provide MetaUiResourceId/MetaApplicationId together'
  );
});

test('rule scope helpers: field empty create normalizes to global', () => {
  const values: Record<string, any> = {};
  assertExclusiveScope(values, 'create', 'field');
  expect(values.MetaFieldId).toBe(null);
  expect(values.MetaModelId).toBe(null);
  expect(values.MetaApplicationId).toBe(null);
  expect(values.LogicalModelName).toBe(null);
});

test('rule scope helpers: method/record/ui empty create is a no-op', () => {
  const method: Record<string, any> = {};
  assertExclusiveScope(method, 'create', 'method');
  expect(Object.prototype.hasOwnProperty.call(method, 'MetaServiceId')).toBe(false);

  const record: Record<string, any> = {};
  assertExclusiveScope(record, 'create', 'record');
  expect(Object.prototype.hasOwnProperty.call(record, 'MetaModelId')).toBe(false);

  const ui: Record<string, any> = {};
  assertExclusiveScope(ui, 'create', 'ui');
  expect(Object.prototype.hasOwnProperty.call(ui, 'MetaUiResourceId')).toBe(false);
});

test('rule scope helpers: normalizes object/string refs and blank strings', () => {
  const values: Record<string, any> = {
    MetaModelId: { Id: '  model-1  ' },
    MetaApplicationId: '   ',
  };
  assertExclusiveScope(values, 'create', 'record');
  expect(values.MetaModelId).toBe('model-1');
  expect(values.MetaApplicationId).toBe(null);

  const method: Record<string, any> = {
    MetaServiceId: { id: 'svc-x' },
    MetaModelId: null,
    MetaApplicationId: undefined,
    LogicalModelName: null,
  };
  assertExclusiveScope(method, 'create', 'method');
  expect(method.MetaServiceId).toBe('svc-x');
  expect(method.MetaModelId).toBe(null);
  expect(method.MetaApplicationId).toBe(null);
  expect(method.LogicalModelName).toBe(null);
});

test('rule scope helpers: update without scope keys is a no-op', () => {
  const values: Record<string, any> = { Mode: 'allow' };
  assertExclusiveScope(values, 'update', 'method');
  expect(values.Mode).toBe('allow');
  expect(Object.prototype.hasOwnProperty.call(values, 'MetaServiceId')).toBe(false);

  const fieldOnlyPerm: Record<string, any> = { PermRead: 'allow' };
  assertExclusiveScope(fieldOnlyPerm, 'update', 'field');
  expect(fieldOnlyPerm.PermRead).toBe('allow');
  expect(Object.prototype.hasOwnProperty.call(fieldOnlyPerm, 'MetaFieldId')).toBe(false);
});

test('rule scope helpers: unknown profile throws', () => {
  expect(() => assertExclusiveScope({}, 'create', 'bogus' as any)).toThrow('unknown rule scope profile: bogus');
});

test('rule scope helpers: update with all sibling fields succeeds', () => {
  const record: Record<string, any> = { MetaModelId: 'm1', MetaApplicationId: null };
  assertExclusiveScope(record, 'update', 'record');
  expect(record.MetaModelId).toBe('m1');
  expect(record.MetaApplicationId).toBe(null);

  const method: Record<string, any> = {
    MetaServiceId: null,
    MetaModelId: null,
    MetaApplicationId: 'a1',
    LogicalModelName: null,
  };
  assertExclusiveScope(method, 'update', 'method');
  expect(method.MetaApplicationId).toBe('a1');
  expect(method.LogicalModelName).toBe(null);
});
