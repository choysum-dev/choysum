// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { Field } from '../decorator';
import { ReadOperations } from './model_read';
import { browseManyModels, browseModel, countGroupedModels, countModels, readGroupedModels, searchModels } from './model_read_facade';

class ReadFacadeModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

class ReadGroupFacadeModel extends ReadFacadeModel {
  static get ctx() {
    return { timezone: 'Asia/Shanghai', tz: 'UTC' } as any;
  }
}

test('model read facade wraps browse/search results through createProxyModel without adding read semantics', async () => {
  const originalBrowse = ReadOperations.Browse;
  const originalSearch = ReadOperations.Search;

  const browseCalls: any[] = [];
  const searchCalls: any[] = [];
  const ctor = ReadFacadeModel as any;

  try {
    ReadOperations.Browse = (async (ModelCtor: any, id: string, fields?: any, options?: any) => {
      browseCalls.push({ ModelCtor, id, fields, options });
      return { Id: id, Name: 'browse' };
    }) as any;

    ReadOperations.Search = (async (ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ ModelCtor, condition, options });
      if (Array.isArray(condition) && condition[0] === 'Id' && condition[1] === 'in') {
        return [
          { Id: 'A', Name: 'many-1' },
          { Id: 'B', Name: 'many-2' },
        ];
      }
      return [{ Id: 'S', Name: 'search' }];
    }) as any;

    const browsed = await browseModel(ctor, 'B1', ['Name'] as any, { withDeleted: true } as any);
    const many = await browseManyModels(ctor, ['A', 'B'], ['Name'] as any, { onlyDeleted: true } as any);
    const searched = await searchModels(ctor, ['Name', '=', 'demo'] as any, { fields: ['Name'] as any } as any);

    expect(browsed instanceof ReadFacadeModel).toBe(true);
    expect((browsed as any).Id).toBe('B1');
    expect((browsed as any).Name).toBe('browse');
    expect((browsed as any).fields).toEqual(['Name']);

    expect(many.length).toBe(2);
    expect((many[0] as any).Id).toBe('A');
    expect((many[1] as any).Id).toBe('B');
    expect((many[0] as any).fields).toEqual(['Name']);

    expect(searched.length).toBe(1);
    expect((searched[0] as any).Id).toBe('S');
    expect((searched[0] as any).Name).toBe('search');
    expect((searched[0] as any).fields).toEqual(['Name']);

    expect(browseCalls.length).toBe(1);
    expect(browseCalls[0]?.id).toBe('B1');
    expect(browseCalls[0]?.fields).toEqual(['Name']);
    expect(browseCalls[0]?.options).toEqual({ withDeleted: true });

    expect(searchCalls.length).toBe(2);
    expect(searchCalls[0]?.condition).toEqual(['Id', 'in', ['A', 'B']]);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Name'], onlyDeleted: true });
    expect(searchCalls[1]?.condition).toEqual(['Name', '=', 'demo']);
    expect(searchCalls[1]?.options).toEqual({ fields: ['Name'] });
  } finally {
    ReadOperations.Browse = originalBrowse;
    ReadOperations.Search = originalSearch;
  }
});

test('model read facade forwards timezone and keeps readGroup results on the plain boundary', async () => {
  const originalReadGroup = ReadOperations.ReadGroup;
  const originalReadGroupCount = ReadOperations.ReadGroupCount;

  const readGroupCalls: any[] = [];
  const readGroupCountCalls: any[] = [];
  const ctor = ReadGroupFacadeModel as any;

  try {
    ReadOperations.ReadGroup = (async (ModelCtor: any, groupby: any, condition: any, options: any) => {
      readGroupCalls.push({ ModelCtor, groupby, condition, options });
      return { rows: [{ key: { Status: 'draft' }, count: 2 }] } as any;
    }) as any;

    ReadOperations.ReadGroupCount = (async (ModelCtor: any, groupby: any, condition: any, options: any) => {
      readGroupCountCalls.push({ ModelCtor, groupby, condition, options });
      return '7' as any;
    }) as any;

    const grouped = await readGroupedModels(ctor, ['Status'] as any, ['Active', '=', true] as any, {} as any);
    const total = await countGroupedModels(ctor, ['Status'] as any, ['Active', '=', true] as any, { timezone: 'Europe/Paris' } as any);
    const totalFromCtxTz = await countGroupedModels(ctor, ['Status'] as any, ['Active', '=', true] as any, {} as any);

    expect(grouped).toEqual({ rows: [{ key: { Status: 'draft' }, count: 2 }] });
    expect(total).toBe(7);
    expect(totalFromCtxTz).toBe(7);

    // Default: options.timezone unset → ctx.timezone (then ctx.tz) for day buckets only.
    expect(readGroupCalls.length).toBe(1);
    expect(readGroupCalls[0]?.options).toEqual({ timezone: 'Asia/Shanghai' });

    // Explicit options.timezone wins over ctx.
    expect(readGroupCountCalls.length).toBe(2);
    expect(readGroupCountCalls[0]?.options).toEqual({ timezone: 'Europe/Paris' });
    expect(readGroupCountCalls[1]?.options).toEqual({ timezone: 'Asia/Shanghai' });
  } finally {
    ReadOperations.ReadGroup = originalReadGroup;
    ReadOperations.ReadGroupCount = originalReadGroupCount;
  }
});

test('model read facade ReadGroup defaults to ctx.tz when timezone alias is absent', async () => {
  class TzOnlyReadGroupModel extends ReadFacadeModel {
    static get ctx() {
      return { tz: 'America/New_York' } as any;
    }
  }

  const originalReadGroup = ReadOperations.ReadGroup;
  const calls: any[] = [];
  try {
    ReadOperations.ReadGroup = (async (_ModelCtor: any, _groupby: any, _condition: any, options: any) => {
      calls.push(options);
      return [] as any;
    }) as any;

    await readGroupedModels(TzOnlyReadGroupModel as any, ['Status'] as any, [] as any, {} as any);
    expect(calls[0]).toEqual({ timezone: 'America/New_York' });
  } finally {
    ReadOperations.ReadGroup = originalReadGroup;
  }
});

test('model read facade handles empty ids, default args and falsy grouped count', async () => {
  const originalSearch = ReadOperations.Search;
  const originalCount = ReadOperations.Count;
  const originalReadGroup = ReadOperations.ReadGroup;
  const originalReadGroupCount = ReadOperations.ReadGroupCount;

  const searchCalls: any[] = [];
  const countCalls: any[] = [];
  const readGroupCalls: any[] = [];
  const readGroupCountCalls: any[] = [];
  const ctor = ReadFacadeModel as any;

  try {
    ReadOperations.Search = (async (_ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ condition, options });
      return [];
    }) as any;
    ReadOperations.Count = (async (_ModelCtor: any, condition: any, options?: any) => {
      countCalls.push({ condition, options });
      return 0;
    }) as any;
    ReadOperations.ReadGroup = (async (_ModelCtor: any, groupby: any, condition: any, options: any) => {
      readGroupCalls.push({ groupby, condition, options });
      return [] as any;
    }) as any;
    ReadOperations.ReadGroupCount = (async (_ModelCtor: any, groupby: any, condition: any, options: any) => {
      readGroupCountCalls.push({ groupby, condition, options });
      return undefined as any;
    }) as any;

    const emptyMany = await browseManyModels(ctor, [] as any, ['Name'] as any, { withDeleted: true } as any);
    expect(emptyMany).toEqual([]);
    expect(searchCalls.length).toBe(0);

    await browseManyModels(ctor, ['A'] as any, ['Name'] as any, { withDeleted: true } as any);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Name'], withDeleted: true });

    const defaultSearch = await searchModels(ctor);
    expect(defaultSearch).toEqual([]);
    expect(searchCalls[1]?.condition).toEqual([]);
    expect(searchCalls[1]?.options).toBeUndefined();

    const defaultCount = await countModels(ctor);
    expect(defaultCount).toBe(0);
    expect(countCalls[0]?.condition).toEqual([]);
    expect(countCalls[0]?.options).toBeUndefined();

    const grouped = await readGroupedModels(ctor, ['Status'] as any);
    expect(grouped).toEqual([]);
    expect(readGroupCalls[0]?.condition).toEqual([]);
    expect(readGroupCalls[0]?.options).toEqual({});

    const groupedCount = await countGroupedModels(ctor, ['Status'] as any);
    expect(groupedCount).toBe(0);
    expect(readGroupCountCalls[0]?.condition).toEqual([]);
    expect(readGroupCountCalls[0]?.options).toEqual({});
  } finally {
    ReadOperations.Search = originalSearch;
    ReadOperations.Count = originalCount;
    ReadOperations.ReadGroup = originalReadGroup;
    ReadOperations.ReadGroupCount = originalReadGroupCount;
  }
});
