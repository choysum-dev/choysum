// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../../../metadata/storage';
import { REL_ALIAS_PREFIX } from '../../../relation/relation_alias';
import { buildHiddenScaleAlias } from '../../hidden_scale_alias';
import { decodeRowWithTree } from '..';
import Decimal from 'decimal.js';

function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => T): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

test('repository tree decode normalizes selected scalar columns and removes hidden scale aliases', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    fields: new Map([
      ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', precision: 6, scaleField: 'AmountScale' } }],
      ['AmountScale', { type: 'int', column: { name: 'AmountScale' } }],
      ['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }],
    ]),
  } as any;

  const hiddenScaleAlias = buildHiddenScaleAlias('Amount');
  const row = {
    Payload: '{"ok":true}',
    Amount: { $bigdecimal: '7.891' },
    [hiddenScaleAlias]: 1,
    TagIds: '[1,"a"]',
  } as any;

  const decoded = decodeRowWithTree(meta, { columns: new Set(['Payload', 'Amount', 'TagIds']), relations: new Map() } as any, row) as any;

  expect(decoded.Payload).toEqual({ ok: true });
  expect(decoded.Amount.toString()).toBe('7.9');
  expect(decoded.TagIds).toEqual(['1', 'a']);
  expect(hiddenScaleAlias in decoded).toBe(false);
});

test('repository tree decode normalizes monetary fields from currency digits and removes hidden scale aliases', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    fields: new Map([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
    ]),
  } as any;

  const hiddenScaleAlias = buildHiddenScaleAlias('Amount');
  const row = {
    Amount: { $bigdecimal: '7.891' },
    CurrencyId: { Id: 'C1', DecimalDigits: 1 },
    [hiddenScaleAlias]: 1,
  } as any;

  const decoded = decodeRowWithTree(meta, { columns: new Set(['Amount']), relations: new Map() } as any, row) as any;

  expect(decoded.Amount.toString()).toBe('7.9');
  expect(hiddenScaleAlias in decoded).toBe(false);
});

test('repository tree decode recursively normalizes many2one alias payloads and to-many relation rows', () => {
  class DemoModel {}
  class OwnerModel {}
  class TaskModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map([
      ['Amount', { type: 'decimal', column: { name: 'Amount', precision: 6, scaleField: 'AmountScale' } }],
      ['AmountScale', { type: 'int', column: { name: 'AmountScale' } }],
    ]),
  } as any;

  const taskMeta = {
    type: TaskModel,
    fields: new Map([['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Tasks', { type: 'OneToMany', relation: { targetModel: () => TaskModel } }],
    ]),
  } as any;

  const ownerHiddenScaleAlias = buildHiddenScaleAlias('Amount');
  const row = {
    Owner: { Amount: { $bigdecimal: '99.99' } },
    [`${REL_ALIAS_PREFIX}Owner`]: { Amount: { $bigdecimal: '1.25' }, [ownerHiddenScaleAlias]: 1 },
    Tasks: [{ TagIds: '[1,2]' }, { TagIds: ['3', 4] }],
  } as any;

  const node = {
    columns: new Set<string>(),
    relations: new Map([
      ['Owner', { fieldType: 'ManyToOne', relation: { targetModel: () => OwnerModel }, node: { columns: new Set(['Amount']), relations: new Map() } }],
      ['Tasks', { fieldType: 'OneToMany', relation: { targetModel: () => TaskModel }, node: { columns: new Set(['TagIds']), relations: new Map() } }],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
      [TaskModel, taskMeta],
    ]),
    () => {
      const decoded = decodeRowWithTree(demoMeta, node, row) as any;
      expect(decoded[`${REL_ALIAS_PREFIX}Owner`].Amount.toString()).toBe('1.3');
      expect(ownerHiddenScaleAlias in decoded[`${REL_ALIAS_PREFIX}Owner`]).toBe(false);
      expect(decoded.Owner.Amount.$bigdecimal).toBe('99.99');
      expect(decoded.Tasks).toEqual([{ TagIds: ['1', '2'] }, { TagIds: ['3', '4'] }]);
    }
  );
});

test('repository tree decode handles non-object row and many2manyref fallback variants', () => {
  expect(decodeRowWithTree({ fields: new Map() } as any, { columns: new Set(), relations: new Map() } as any, null as any)).toBeNull();
  expect(decodeRowWithTree({ fields: new Map() } as any, { columns: new Set(), relations: new Map() } as any, 'x' as any)).toBe('x');

  class DemoModel {}
  const meta = {
    type: DemoModel,
    fields: new Map([
      ['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }],
      ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
    ]),
  } as any;

  const row = {
    TagIds: '{"x":1}',
    Payload: 'bad-json',
  } as any;
  const decoded = decodeRowWithTree(meta, { columns: new Set(['TagIds', 'Payload']), relations: new Map() } as any, row) as any;
  expect(decoded.TagIds).toEqual(['{"x":1}']);
  expect(decoded.Payload).toBe('bad-json');

  const row2 = { TagIds: 7 } as any;
  const decoded2 = decodeRowWithTree(meta, { columns: new Set(['TagIds']), relations: new Map() } as any, row2) as any;
  expect(decoded2.TagIds).toEqual(['7']);
});

test('repository tree decode relation traversal skips non-object/non-array and missing target branches', () => {
  class DemoModel {}
  class OwnerModel {}
  class TaskModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;
  const taskMeta = {
    type: TaskModel,
    fields: new Map([['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }]]),
  } as any;
  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Tasks', { type: 'OneToMany', relation: { targetModel: () => TaskModel } }],
      ['Links', { type: 'ManyToMany', relation: { targetModel: () => TaskModel } }],
    ]),
  } as any;

  const node = {
    columns: new Set<string>(),
    relations: new Map([
      ['Owner', { fieldType: 'ManyToOne', relation: { targetModel: () => undefined }, node: { columns: new Set(['Name']), relations: new Map() } }],
      ['Tasks', { fieldType: 'OneToMany', relation: { targetModel: () => TaskModel }, node: { columns: new Set(['TagIds']), relations: new Map() } }],
      ['Links', { fieldType: 'ManyToMany', relation: { targetModel: () => undefined }, node: { columns: new Set(['TagIds']), relations: new Map() } }],
    ]),
  } as any;

  const row = {
    Owner: 'not-object',
    Tasks: 'not-array',
    Links: [{ TagIds: [1] }],
    [REL_ALIAS_PREFIX + 'Tasks']: [{ TagIds: ['1', 2] }, null],
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
      [TaskModel, taskMeta],
    ]),
    () => {
      const decoded = decodeRowWithTree(demoMeta, node, row) as any;
      expect(decoded.Owner).toBe('not-object');
      expect(decoded.Tasks).toBe('not-array');
      expect(decoded[REL_ALIAS_PREFIX + 'Tasks']).toEqual([{ TagIds: ['1', '2'] }, null]);
      expect(decoded.Links).toEqual([{ TagIds: [1] }]);
    }
  );
});

test('repository tree decode covers many2manyref null and decimal select/no-spec/isDecimal/non-array relation branches', () => {
  class DemoModel {}
  class ChildModel {}

  const childMeta = {
    type: ChildModel,
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }],
      ['AmountSelect', { type: 'decimal', column: { scale: 1 } }],
      ['AmountRaw', { type: 'decimal' }],
      ['AmountDecimal', { type: 'decimal', column: { scale: 2 } }],
      ['AmountLarge', { type: 'decimal', column: { precision: 2 } }],
      ['Children', { type: 'OneToMany', relation: { targetModel: () => ChildModel } }],
    ]),
  } as any;

  const row = {
    TagIds: null,
    AmountSelect: null,
    AmountRaw: '12.34',
    AmountDecimal: new Decimal('1.239'),
    AmountLarge: new Decimal('1234'),
    Children: 'not-array',
  } as any;

  const node = {
    columns: new Set(['TagIds', 'AmountSelect', 'AmountRaw', 'AmountDecimal', 'AmountLarge']),
    relations: new Map([
      ['Children', { fieldType: 'OneToMany', relation: { targetModel: () => ChildModel }, node: { columns: new Set(['Name']), relations: new Map() } }],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [ChildModel, childMeta],
    ]),
    () => {
      const decoded = decodeRowWithTree(demoMeta, node, row) as any;

      expect(decoded.TagIds).toEqual([]);
      expect(decoded.AmountSelect).toBeNull();
      expect(decoded.AmountRaw.toString()).toBe('12.34');
      expect(decoded.AmountDecimal.toString()).toBe('1.24');
      // With precision=2, 1234 is out of range, so normalize returns undefined and the original value stays intact.
      expect(decoded.AmountLarge.toString()).toBe('1234');
      expect(decoded.Children).toBe('not-array');
    }
  );
});
