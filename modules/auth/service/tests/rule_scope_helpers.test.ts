// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { assertExclusiveScope } from '@/auth/service/models/_rule_scope_helpers';

test('rule scope helpers: method accepts service/model/application/global shapes', () => {
  const service = { IrServiceId: 'svc1', IrModelId: null, IrApplicationId: null };
  assertExclusiveScope(service, 'create', 'method');
  expect(service.IrServiceId).toBe('svc1');
  expect(service.IrModelId).toBe(null);
  expect(service.IrApplicationId).toBe(null);

  const model = { IrServiceId: null, IrModelId: 'm1', IrApplicationId: null };
  assertExclusiveScope(model, 'create', 'method');
  expect(model.IrModelId).toBe('m1');

  const app = { IrServiceId: null, IrModelId: null, IrApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'method');
  expect(app.IrApplicationId).toBe('a1');

  const global = { IrServiceId: null, IrModelId: null, IrApplicationId: null };
  assertExclusiveScope(global, 'create', 'method');
  expect(global.IrServiceId).toBe(null);
  expect(global.IrModelId).toBe(null);
  expect(global.IrApplicationId).toBe(null);
});

test('rule scope helpers: record accepts model/application/global shapes', () => {
  const model = { IrModelId: 'm1', IrApplicationId: null };
  assertExclusiveScope(model, 'create', 'record');
  expect(model.IrModelId).toBe('m1');
  expect(model.IrApplicationId).toBe(null);

  const app = { IrModelId: null, IrApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'record');
  expect(app.IrApplicationId).toBe('a1');

  const global = { IrModelId: null, IrApplicationId: null };
  assertExclusiveScope(global, 'create', 'record');
  expect(global.IrModelId).toBe(null);
  expect(global.IrApplicationId).toBe(null);
});

test('rule scope helpers: field accepts field/model/application/global shapes', () => {
  const field = { IrFieldId: 'f1', IrModelId: 'm1', IrApplicationId: null };
  assertExclusiveScope(field, 'create', 'field');
  expect(field.IrFieldId).toBe('f1');
  expect(field.IrModelId).toBe('m1');
  expect(field.IrApplicationId).toBe(null);

  const model = { IrFieldId: null, IrModelId: 'm1', IrApplicationId: null };
  assertExclusiveScope(model, 'create', 'field');
  expect(model.IrModelId).toBe('m1');

  const app = { IrFieldId: null, IrModelId: null, IrApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'field');
  expect(app.IrApplicationId).toBe('a1');

  const global = { IrFieldId: null, IrModelId: null, IrApplicationId: null };
  assertExclusiveScope(global, 'create', 'field');
  expect(global.IrFieldId).toBe(null);
});

test('rule scope helpers: ui accepts resource/application/global shapes', () => {
  const resource = { IrUiResourceId: 'u1', IrApplicationId: null };
  assertExclusiveScope(resource, 'create', 'ui');
  expect(resource.IrUiResourceId).toBe('u1');

  const app = { IrUiResourceId: null, IrApplicationId: 'a1' };
  assertExclusiveScope(app, 'create', 'ui');
  expect(app.IrApplicationId).toBe('a1');

  const global = { IrUiResourceId: null, IrApplicationId: null };
  assertExclusiveScope(global, 'create', 'ui');
  expect(global.IrUiResourceId).toBe(null);
  expect(global.IrApplicationId).toBe(null);
});

test('rule scope helpers: mixed scope throws golden error messages', () => {
  expect(() => assertExclusiveScope({ IrServiceId: 's1', IrModelId: 'm1', IrApplicationId: null }, 'create', 'method')).toThrow(
    'invalid RoleMethodAccess scope: must be exactly one of service/model/application/global'
  );
  expect(() => assertExclusiveScope({ IrModelId: 'm1', IrApplicationId: 'a1' }, 'create', 'record')).toThrow(
    'invalid RoleRecordRule scope: must be exactly one of model/application/global'
  );
  expect(() => assertExclusiveScope({ IrFieldId: 'f1', IrModelId: null, IrApplicationId: null }, 'create', 'field')).toThrow(
    'invalid RoleFieldRule scope: must be exactly one of field/model/application/global'
  );
  expect(() => assertExclusiveScope({ IrUiResourceId: 'u1', IrApplicationId: 'a1' }, 'create', 'ui')).toThrow(
    'invalid RoleUiResource scope: must be exactly one of resource/application/global'
  );
});

test('rule scope helpers: update missing sibling fields throws golden messages', () => {
  expect(() => assertExclusiveScope({ IrModelId: 'm1' }, 'update', 'method')).toThrow(
    'invalid RoleMethodAccess scope update: must provide IrServiceId/IrModelId/IrApplicationId together'
  );
  expect(() => assertExclusiveScope({ IrModelId: 'm1' }, 'update', 'record')).toThrow(
    'invalid RoleRecordRule scope update: must provide IrModelId/IrApplicationId together'
  );
  expect(() => assertExclusiveScope({ IrFieldId: 'f1' }, 'update', 'field')).toThrow(
    'invalid RoleFieldRule scope update: must provide IrFieldId/IrModelId/IrApplicationId together'
  );
  expect(() => assertExclusiveScope({ IrUiResourceId: 'u1' }, 'update', 'ui')).toThrow(
    'invalid RoleUiResource scope update: must provide IrUiResourceId/IrApplicationId together'
  );
});

test('rule scope helpers: field empty create normalizes to global', () => {
  const values: Record<string, any> = {};
  assertExclusiveScope(values, 'create', 'field');
  expect(values.IrFieldId).toBe(null);
  expect(values.IrModelId).toBe(null);
  expect(values.IrApplicationId).toBe(null);
});

test('rule scope helpers: method/record/ui empty create is a no-op', () => {
  const method: Record<string, any> = {};
  assertExclusiveScope(method, 'create', 'method');
  expect(Object.prototype.hasOwnProperty.call(method, 'IrServiceId')).toBe(false);

  const record: Record<string, any> = {};
  assertExclusiveScope(record, 'create', 'record');
  expect(Object.prototype.hasOwnProperty.call(record, 'IrModelId')).toBe(false);

  const ui: Record<string, any> = {};
  assertExclusiveScope(ui, 'create', 'ui');
  expect(Object.prototype.hasOwnProperty.call(ui, 'IrUiResourceId')).toBe(false);
});

test('rule scope helpers: normalizes object/string refs and blank strings', () => {
  const values: Record<string, any> = {
    IrModelId: { Id: '  model-1  ' },
    IrApplicationId: '   ',
  };
  assertExclusiveScope(values, 'create', 'record');
  expect(values.IrModelId).toBe('model-1');
  expect(values.IrApplicationId).toBe(null);

  const method: Record<string, any> = {
    IrServiceId: { id: 'svc-x' },
    IrModelId: null,
    IrApplicationId: undefined,
  };
  assertExclusiveScope(method, 'create', 'method');
  expect(method.IrServiceId).toBe('svc-x');
  expect(method.IrModelId).toBe(null);
  expect(method.IrApplicationId).toBe(null);
});

test('rule scope helpers: update without scope keys is a no-op', () => {
  const values: Record<string, any> = { Mode: 'allow' };
  assertExclusiveScope(values, 'update', 'method');
  expect(values.Mode).toBe('allow');
  expect(Object.prototype.hasOwnProperty.call(values, 'IrServiceId')).toBe(false);

  const fieldOnlyPerm: Record<string, any> = { PermRead: 'allow' };
  assertExclusiveScope(fieldOnlyPerm, 'update', 'field');
  expect(fieldOnlyPerm.PermRead).toBe('allow');
  expect(Object.prototype.hasOwnProperty.call(fieldOnlyPerm, 'IrFieldId')).toBe(false);
});
