// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import BaseModel from './model';
import { buildNameSearchCondition, mergeNameSearchOptions, nameSearchModels } from './model_namesearch';

@Model('NameSearchWidget', { application: 'demo' })
class NameSearchWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;
}

@Model('NameSearchOverrideWidget', { application: 'demo' })
class NameSearchOverrideWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;

  static override async NameSearch<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    name: string,
    condition: any = [],
    options?: any
  ): Promise<T[]> {
    const kw = String(name ?? '').trim();
    const codeCond = kw ? (['Code', 'like', `%${kw}%`] as any) : [];
    const parts = [];
    if (kw) parts.push(codeCond);
    if (condition && !(Array.isArray(condition) && condition.length === 0)) parts.push(condition);
    const merged = parts.length === 0 ? [] : parts.length === 1 ? parts[0] : { And: parts };
    return (await (this as any).Search(merged, options)) as T[];
  }
}

test('buildNameSearchCondition adds DisplayName like for non-empty name', () => {
  expect(buildNameSearchCondition('  alice  ', [])).toEqual(['DisplayName', 'like', '%alice%']);
});

test('buildNameSearchCondition empty name yields empty or domain only', () => {
  expect(buildNameSearchCondition('', [])).toEqual([]);
  expect(buildNameSearchCondition('   ', [])).toEqual([]);
  expect(buildNameSearchCondition('', ['Code', '=', 'X'] as any)).toEqual(['Code', '=', 'X']);
});

test('buildNameSearchCondition Ands keyword with domain', () => {
  expect(buildNameSearchCondition('bob', ['Code', '=', 'B'] as any)).toEqual({
    And: [
      ['DisplayName', 'like', '%bob%'],
      ['Code', '=', 'B'],
    ],
  });
});

test('mergeNameSearchOptions defaults fields to Id+DisplayName', () => {
  expect(mergeNameSearchOptions()).toEqual({ fields: ['Id', 'DisplayName'] });
  expect(mergeNameSearchOptions({ limit: 10 })).toEqual({ limit: 10, fields: ['Id', 'DisplayName'] });
  expect(mergeNameSearchOptions({ fields: ['Id', 'Code'] as any, limit: 5 })).toEqual({
    fields: ['Id', 'Code'],
    limit: 5,
  });
});

test('nameSearchModels NameSearch path calls Search with merged condition and default fields', async () => {
  const calls: unknown[] = [];
  const Fake = {
    Search: async (condition: unknown, options: unknown) => {
      calls.push({ condition, options });
      return [{ Id: '1', DisplayName: 'Alice' }];
    },
  };

  const rows = await nameSearchModels(Fake as any, 'Ali', ['Code', '=', 'A'] as any, { limit: 7 });
  expect(rows).toEqual([{ Id: '1', DisplayName: 'Alice' }]);
  expect(calls).toEqual([
    {
      condition: {
        And: [
          ['DisplayName', 'like', '%Ali%'],
          ['Code', '=', 'A'],
        ],
      },
      options: { limit: 7, fields: ['Id', 'DisplayName'] },
    },
  ]);
});

test('BaseModel.NameSearch delegates to nameSearchModels via Search', async () => {
  const original = NameSearchWidget.Search;
  const seen: unknown[] = [];
  NameSearchWidget.Search = (async (condition: any, options: any) => {
    seen.push({ condition, options });
    return [{ Id: 'w1', DisplayName: 'Widget' }] as any;
  }) as any;

  try {
    const rows = await NameSearchWidget.NameSearch('Wid', [], { limit: 3 });
    expect(rows).toEqual([{ Id: 'w1', DisplayName: 'Widget' }]);
    expect(seen).toEqual([
      {
        condition: ['DisplayName', 'like', '%Wid%'],
        options: { limit: 3, fields: ['Id', 'DisplayName'] },
      },
    ]);
  } finally {
    NameSearchWidget.Search = original;
  }
});

test('static override NameSearch replaces default DisplayName matching', async () => {
  const original = NameSearchOverrideWidget.Search;
  const seen: unknown[] = [];
  NameSearchOverrideWidget.Search = (async (condition: any, options: any) => {
    seen.push({ condition, options });
    return [{ Id: 'o1', Code: 'ABC' }] as any;
  }) as any;

  try {
    const rows = await NameSearchOverrideWidget.NameSearch('AB', [], { fields: ['Id', 'Code'] as any });
    expect(rows).toEqual([{ Id: 'o1', Code: 'ABC' }]);
    expect(seen).toEqual([
      {
        condition: ['Code', 'like', '%AB%'],
        options: { fields: ['Id', 'Code'] },
      },
    ]);
  } finally {
    NameSearchOverrideWidget.Search = original;
  }
});
