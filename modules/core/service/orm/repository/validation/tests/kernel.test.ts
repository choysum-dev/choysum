// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { KernelValidationError, validateFields } from '..';

function expectKernelError(fn: () => void, code: string, field: string) {
  let error: unknown;
  try {
    fn();
  } catch (caught) {
    error = caught;
  }

  expect(error instanceof KernelValidationError).toBe(true);
  expect((error as KernelValidationError).code).toBe(code as any);
  expect((error as KernelValidationError).field).toBe(field);
}

test('repository kernel validation enforces int and selection rules with stable issue metadata', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Count', { type: 'int', column: { name: 'Count' } }],
      ['State', { type: 'selection', selection: [{ value: 'draft' }, { value: 'done' }] }],
    ]),
  } as any;

  expectKernelError(() => validateFields(meta, { Count: 1.5 }, { rules: ['int'] }), 'kernel_int_invalid', 'Count');
  expectKernelError(() => validateFields(meta, { State: 'blocked' }, { rules: ['selection'] }), 'kernel_selection_invalid', 'State');
});

test('repository kernel validation enforces create required fields and decimal format', () => {
  const meta = {
    fields: new Map<string, any>([
      ['RequiredName', { type: 'varchar', column: { name: 'RequiredName', notNull: true } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', precision: 5, scale: 2 } }],
    ]),
  } as any;

  expectKernelError(() => validateFields(meta, {}, { mode: 'create', rules: ['required'] }), 'kernel_required_missing', 'RequiredName');
  expectKernelError(() => validateFields(meta, { Amount: 'abc' }, { rules: ['decimal'] }), 'kernel_decimal_invalid', 'Amount');
});

test('repository kernel validation quantizes monetary using stamped currency digits', () => {
  const meta = {
    fields: new Map<string, any>([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
    ]),
  } as any;

  expect(() =>
    validateFields(
      meta,
      { Amount: '1.23', CurrencyId: { Id: 'C1', DecimalDigits: 2 } },
      { rules: ['decimal'] }
    )
  ).not.toThrow();

  expectKernelError(
    () => validateFields(meta, { Amount: 'not-a-decimal', CurrencyId: { Id: 'C1', DecimalDigits: 2 } }, { rules: ['decimal'] }),
    'kernel_decimal_invalid',
    'Amount'
  );

  // Empty / missing monetary values are skipped.
  expect(() => validateFields(meta, { Amount: '' }, { rules: ['decimal'] })).not.toThrow();
});

test('repository kernel validation enforces many2one and many2one-ref reference shapes', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }],
      ['CompanyId', { type: 'ManyToOneRef', column: { name: 'CompanyId' } }],
    ]),
  } as any;

  expectKernelError(() => validateFields(meta, { Owner: { foo: 'bar' } }, { rules: ['relationShape'] }), 'kernel_relation_shape_invalid', 'Owner');
  expectKernelError(() => validateFields(meta, { CompanyId: { foo: 'bar' } }, { rules: ['relationShape'] }), 'kernel_relation_shape_invalid', 'CompanyId');
});

test('repository kernel validation allows valid relation reference scalar and object shapes', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }],
      ['CompanyId', { type: 'ManyToOneRef', column: { name: 'CompanyId' } }],
    ]),
  } as any;

  expect(() => validateFields(meta, { Owner: null }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Owner: undefined }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Owner: 'U-1' }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Owner: 1 }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Owner: 1n }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Owner: { Id: 'U-2' } }, { rules: ['relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { CompanyId: { id: 3 } }, { rules: ['relationShape'] })).not.toThrow();
});

test('repository kernel validation rejects malformed relation objects with empty and null ids', () => {
  const meta = {
    fields: new Map<string, any>([['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }]]),
  } as any;

  expectKernelError(() => validateFields(meta, { Owner: { Id: '' } }, { rules: ['relationShape'] }), 'kernel_relation_shape_invalid', 'Owner');
  expectKernelError(() => validateFields(meta, { Owner: { id: null } }, { rules: ['relationShape'] }), 'kernel_relation_shape_invalid', 'Owner');
});

test('repository kernel validation create mode handles required null, default, and primary key branches', () => {
  const meta = {
    fields: new Map<string, any>([
      ['RequiredName', { type: 'varchar', column: { notNull: true } }],
      ['HasDefault', { type: 'varchar', column: { notNull: true, default: 'x' } }],
      ['Id', { type: 'char', column: { notNull: true, primaryKey: true } }],
    ]),
  } as any;

  expectKernelError(() => validateFields(meta, { RequiredName: null }, { mode: 'create', rules: ['required'] }), 'kernel_required_null', 'RequiredName');
  expect(() => validateFields(meta, { RequiredName: 'ok' }, { mode: 'create', rules: ['required'] })).not.toThrow();

  const metaWithDefaultAndPkOnly = {
    fields: new Map<string, any>([
      ['HasDefault', { type: 'varchar', column: { notNull: true, default: 'x' } }],
      ['Id', { type: 'char', column: { notNull: true, primaryKey: true } }],
    ]),
  } as any;
  expect(() => validateFields(metaWithDefaultAndPkOnly, {}, { mode: 'create', rules: ['required'] })).not.toThrow();
});

test('repository kernel validation update mode only blocks explicit nullish updates', () => {
  const meta = {
    fields: new Map<string, any>([['Name', { type: 'varchar', column: { notNull: true } }]]),
  } as any;

  expectKernelError(() => validateFields(meta, { Name: undefined }, { mode: 'update', rules: ['required'] }), 'kernel_required_null', 'Name');
  expect(() => validateFields(meta, {}, { mode: 'update', rules: ['required'] })).not.toThrow();
  expect(() => validateFields(meta, { Name: 'ok' }, { mode: 'update', rules: ['required'] })).not.toThrow();
});

test('repository kernel validation skips non-object payloads and allows empty decimal inputs', () => {
  const meta = {
    fields: new Map<string, any>([
      ['State', { type: 'selection', selection: [{ value: 'draft' }] }],
      ['Amount', { type: 'decimal', column: { precision: 6, scale: 2 } }],
      ['Owner', { type: 'ManyToOne', column: {} }],
    ]),
  } as any;

  expect(() => validateFields(meta, null as any, { rules: ['selection', 'required', 'decimal', 'relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, 1 as any, { rules: ['selection', 'required', 'decimal', 'relationShape'] })).not.toThrow();
  expect(() => validateFields(meta, { Amount: '' }, { rules: ['decimal'] })).not.toThrow();
  expect(() => validateFields(meta, { Amount: undefined }, { rules: ['decimal'] })).not.toThrow();
});
