// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '../../../orm/metadata/storage';
import { RepositoryFactory } from '../../../orm/repository/repository_factory';
import { PathPlanExecutor } from './executor';
import { executeCollectionsPrefetch, executeFirstHopPrefetch, executeMultiHopM2OPrefetch, type PathPlanPreloadContext } from './preload';
import type { PathPrefetchPlan, PrefetchBatchStat, PrefetchExecStats } from '../types';

@Model('test.PlanPreloadCompany')
class PlanPreloadCompany extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.PlanPreloadPartner')
class PlanPreloadPartner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanPreloadCompany } })
  CompanyId?: PlanPreloadCompany;
}

@Model('test.PlanPreloadProduct')
class PlanPreloadProduct extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.PlanPreloadOrder')
class PlanPreloadOrder extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanPreloadPartner } })
  PartnerId?: PlanPreloadPartner;

  @Field({ type: 'OneToMany', relation: { targetModel: () => PlanPreloadLine, inverseField: 'OrderId' } })
  Lines?: PlanPreloadLine[];
}

@Model('test.PlanPreloadLine')
class PlanPreloadLine extends BaseModel {
  @Field({ type: 'decimal', precision: 10, scale: 2 })
  Qty?: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanPreloadProduct } })
  ProductId?: PlanPreloadProduct;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanPreloadOrder } })
  OrderId?: PlanPreloadOrder;
}

@Model('test.PlanPreloadBrokenNext')
class PlanPreloadBrokenNext extends BaseModel {
  @Field({ type: 'OneToMany', relation: { targetModel: () => PlanPreloadBrokenItem, inverseField: 'ParentId' } })
  Items?: PlanPreloadBrokenItem[];
}

@Model('test.PlanPreloadBrokenItem')
class PlanPreloadBrokenItem extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => undefined as any } })
  BadRef?: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanPreloadBrokenNext } })
  ParentId?: PlanPreloadBrokenNext;
}

function makePlan(): PathPrefetchPlan {
  return {
    rootManyToOne: new Map(),
    m2oChains: new Map(),
    collections: new Map(),
  };
}

function recordStat(stats: PrefetchExecStats, entry: Omit<PrefetchBatchStat, 'batchCount' | 'rowCount'> & { batchCount?: number; rowCount?: number }) {
  const batchCount = entry.batchCount ?? 1;
  const rowCount = entry.rowCount ?? 0;
  stats.totalBatches += batchCount;
  stats.totalRows += rowCount;
  stats.batches.push({
    phase: entry.phase,
    level: entry.level,
    model: entry.model,
    fields: entry.fields.slice(),
    batchCount,
    rowCount,
    idsSample: entry.idsSample?.slice(),
  });
}

test('plan preload executes first-hop, multi-hop, and collection prefetch within one draft flow', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name', 'CompanyId']));
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);
  plan.collections.set('Lines', { chains: [['Qty'], ['ProductId'], ['ProductId', 'Name']] });

  const searchCalls: Array<{ model: string; ids: string[]; fields: string[] }> = [];
  const collectionCalls: Array<{ condition: any; fields: string[] }> = [];

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async (condition: any, options: any) => {
        collectionCalls.push({ condition, fields: [...(options?.fields || [])] });
        return [
          { Id: 'L1', Qty: '2.00', ProductId: 'PR1' },
          { Id: 'L2', Qty: '5.00', ProductId: 'PR2' },
        ];
      },
    } as any
  );

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (ctor: any, ids: string | string[], fields: string[]) => {
      const idsArr = Array.isArray(ids) ? ids.map(String) : [String(ids)];
      searchCalls.push({ model: ctor.name, ids: idsArr, fields: [...fields] });

      if (ctor === PlanPreloadPartner) {
        return [{ Id: 'P1', Name: 'Partner A', CompanyId: 'C1' }];
      }
      if (ctor === PlanPreloadCompany) {
        return [{ Id: 'C1', Name: 'Company A' }];
      }
      if (ctor === PlanPreloadProduct) {
        return [
          { Id: 'PR1', Name: 'Product A' },
          { Id: 'PR2', Name: 'Product B' },
        ].filter(row => idsArr.includes(row.Id));
      }

      return [];
    },
    getNestedAt: (draft, root, prefix) => nesting.getNestedAt(draft, root, prefix),
    upsertNestedAt: (draft, root, prefix, id) => nesting.upsertNestedAt(draft, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = {
    Id: 'SO1',
    PartnerId: 'P1',
  };

  await executeFirstHopPrefetch(context, meta, draft, stats);
  await executeMultiHopM2OPrefetch(context, meta, draft, stats);
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(draft).toEqual({
    Id: 'SO1',
    PartnerId: {
      Id: 'P1',
      Name: 'Partner A',
      CompanyId: {
        Id: 'C1',
        Name: 'Company A',
      },
    },
    Lines: [
      {
        Id: 'L1',
        Qty: '2.00',
        ProductId: {
          Id: 'PR1',
          Name: 'Product A',
        },
      },
      {
        Id: 'L2',
        Qty: '5.00',
        ProductId: {
          Id: 'PR2',
          Name: 'Product B',
        },
      },
    ],
  });

  expect(searchCalls).toEqual([
    { model: 'PlanPreloadPartner', ids: ['P1'], fields: ['Id', 'Name', 'CompanyId'] },
    { model: 'PlanPreloadCompany', ids: ['C1'], fields: ['Id', 'Name'] },
    { model: 'PlanPreloadProduct', ids: ['PR1', 'PR2'], fields: ['Id', 'Name'] },
  ]);
  expect(collectionCalls).toEqual([{ condition: ['OrderId', '=', 'SO1'], fields: ['Id', 'Qty', 'ProductId'] }]);
  expect(stats.totalBatches).toBe(4);
  expect(stats.totalRows).toBe(6);
  expect(stats.batches.map(batch => `${batch.phase}:${batch.level}:${batch.model}`)).toEqual([
    'm2o:1:PlanPreloadPartner',
    'm2o:2:PlanPreloadCompany',
    'collection:1:PlanPreloadLine',
    'collection:2:PlanPreloadProduct',
  ]);
});

test('plan preload tolerates search/cache failures and keeps draft unchanged', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name']));
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);
  plan.collections.set('Lines', { chains: [['ProductId', 'Name']] });

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => {
        throw new Error('collection repo fail');
      },
    } as any
  );

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      throw new Error('cache fail');
    },
    getNestedAt: (draft, root, prefix) => nesting.getNestedAt(draft, root, prefix),
    upsertNestedAt: (draft, root, prefix, id) => nesting.upsertNestedAt(draft, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = { Id: 'SO-E1', PartnerId: 'P-E1' };

  await executeFirstHopPrefetch(context, meta, draft as any, stats);
  await executeMultiHopM2OPrefetch(context, meta, draft as any, stats);
  await executeCollectionsPrefetch(context, meta, draft as any, stats);

  expect((draft as any).Id).toBe('SO-E1');
  expect((draft as any).PartnerId?.Id || (draft as any).PartnerId).toBe('P-E1');
  expect((draft as any).Lines).toBe(undefined);
});

test('plan preload skips collection prefetch when relation already loaded or parent id missing', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.collections.set('Lines', { chains: [['Qty']] });

  const calls: any[] = [];
  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async (condition: any) => {
        calls.push(condition);
        return [];
      },
    } as any
  );

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => [],
    getNestedAt: (draft, root, prefix) => nesting.getNestedAt(draft, root, prefix),
    upsertNestedAt: (draft, root, prefix, id) => nesting.upsertNestedAt(draft, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  await executeCollectionsPrefetch(context, meta, { Id: 'SO-SKIP-1', Lines: [] } as any, stats);
  await executeCollectionsPrefetch(context, meta, { PartnerId: 'P1' } as any, stats);

  expect(calls.length).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload first-hop records empty rows and merges into existing object payload', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name']));

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = {
    Id: 'SO-FH-1',
    PartnerId: { Id: 'P-FH-1', Name: 'Local Name', Extra: 'keep' },
  } as any;

  const calls: any[] = [];
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (_ctor: any, _ids: string | string[], _fields: string[]) => {
      calls.push('called');
      return calls.length === 1 ? [] : [{ Id: 'P-FH-1', Name: 'Remote Name' }];
    },
    getNestedAt: () => undefined,
    upsertNestedAt: () => ({}),
  };

  await executeFirstHopPrefetch(context, meta, draft, stats);
  expect(draft.PartnerId).toEqual({ Id: 'P-FH-1', Name: 'Local Name', Extra: 'keep' });
  expect(stats.batches[0]?.rowCount).toBe(0);

  await executeFirstHopPrefetch(context, meta, draft, stats);
  expect(draft.PartnerId).toEqual({ Id: 'P-FH-1', Name: 'Local Name', Extra: 'keep' });
  expect(stats.batches[1]?.rowCount).toBe(1);
});

test('plan preload multi-hop skips invalid chains and hydrates object-id branch', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.m2oChains.set('PartnerId', [
    ['CompanyId', 'Name'],
    ['Name', 'Code'],
  ]);

  const draft = {
    Id: 'SO-MH-1',
    PartnerId: {
      Id: 'P-MH-1',
      CompanyId: { Id: 'C-MH-1', Name: 'Local Company' },
    },
  } as any;

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const searchCalls: any[] = [];

  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (ctor: any, ids: string | string[], fields: string[]) => {
      searchCalls.push({ ctor: ctor.name, ids: Array.isArray(ids) ? ids : [ids], fields });
      if (ctor === PlanPreloadCompany) {
        return [{ Id: 'C-MH-1', Name: 'Remote Company' }];
      }
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  await executeMultiHopM2OPrefetch(context, meta, draft, stats);

  expect(searchCalls).toEqual([{ ctor: 'PlanPreloadCompany', ids: ['C-MH-1'], fields: ['Id', 'Name'] }]);
  expect(draft.PartnerId.CompanyId).toEqual({ Id: 'C-MH-1', Name: 'Local Company' });
  expect(stats.totalBatches).toBe(1);
  expect(stats.totalRows).toBe(1);
});

test('plan preload collections handles empty rows and skips second-hop when ids are empty', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);

  const planEmptyRows = makePlan();
  planEmptyRows.collections.set('Lines', { chains: [['Qty'], [] as any] });
  const stats1: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => [],
    } as any
  );

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const contextEmptyRows: PathPlanPreloadContext = {
    plan: planEmptyRows,
    recordStat,
    searchWithCache: async () => [],
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const draft1 = { Id: 'SO-COL-1' } as any;
  await executeCollectionsPrefetch(contextEmptyRows, meta, draft1, stats1);

  expect(draft1.Lines).toEqual([]);
  expect(stats1.totalBatches).toBe(1);
  expect(stats1.totalRows).toBe(0);

  const planNoSecondHop = makePlan();
  planNoSecondHop.collections.set('Lines', { chains: [['ProductId', 'Name']] });
  const stats2: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => [{ Id: 'L-EMPTY-1', ProductId: '' }],
    } as any
  );

  let secondHopCalls = 0;
  const contextNoSecondHop: PathPlanPreloadContext = {
    plan: planNoSecondHop,
    recordStat,
    searchWithCache: async () => {
      secondHopCalls += 1;
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const draft2 = { Id: 'SO-COL-2' } as any;
  await executeCollectionsPrefetch(contextNoSecondHop, meta, draft2, stats2);

  expect(secondHopCalls).toBe(0);
  expect(stats2.totalBatches).toBe(1);
  expect(stats2.totalRows).toBe(1);
});

@Model('test.PlanPreloadBrokenParent')
class PlanPreloadBrokenParent extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => undefined as any } })
  BrokenOwner?: any;

  @Field({ type: 'OneToMany', relation: { targetModel: () => PlanPreloadLine } } as any)
  BrokenLines?: PlanPreloadLine[];
}

test('plan preload skips invalid metadata branches without issuing repository calls', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadBrokenParent as any);
  const plan = makePlan();
  plan.rootManyToOne.set('BrokenOwner', new Set(['Name']));
  plan.rootManyToOne.set('BrokenLines', new Set(['Qty']));
  plan.collections.set('BrokenLines', { chains: [['Qty']] });

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  let searchCalls = 0;

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: (draft, root, prefix) => nesting.getNestedAt(draft, root, prefix),
    upsertNestedAt: (draft, root, prefix, id) => nesting.upsertNestedAt(draft, root, prefix, id),
  };

  const draft = {
    Id: 'BROKEN-1',
    BrokenOwner: { Id: 'ANY-1' },
  } as any;

  await executeFirstHopPrefetch(context, meta, draft, stats);
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
  expect(draft.BrokenOwner).toEqual({ Id: 'ANY-1' });
  expect(draft.BrokenLines).toBe(undefined);
});

test('plan preload multi-hop skips when next relation id cannot be resolved', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);

  const draft = {
    Id: 'SO-MH-NOID',
    PartnerId: { Id: 'P-NOID' },
  } as any;

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  let searchCalls = 0;

  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  await executeMultiHopM2OPrefetch(context, meta, draft, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload first-hop ignores empty fields, non-many2one fields, and missing ids', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set<string>());
  plan.rootManyToOne.set('Lines', new Set<string>(['Qty']));
  plan.rootManyToOne.set('UnknownField', new Set<string>(['Name']));

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  let searchCalls = 0;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: () => undefined,
    upsertNestedAt: () => ({}),
  };

  await executeFirstHopPrefetch(context, meta, { PartnerId: {} } as any, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload multi-hop records batches even when searched ids return no matching rows', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);

  const draft = {
    Id: 'SO-MH-NOROW',
    PartnerId: { Id: 'P-NOROW', CompanyId: 'C-NOROW' },
  } as any;

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const searchCalls: any[] = [];
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (ctor: any, ids: string | string[], fields: string[]) => {
      searchCalls.push({ ctor: ctor.name, ids: Array.isArray(ids) ? ids : [ids], fields });
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  await executeMultiHopM2OPrefetch(context, meta, draft, stats);

  expect(searchCalls).toEqual([{ ctor: 'PlanPreloadCompany', ids: ['C-NOROW'], fields: ['Id', 'Name'] }]);
  expect(draft.PartnerId.CompanyId).toEqual({ Id: 'C-NOROW' });
  expect(stats.totalBatches).toBe(1);
  expect(stats.totalRows).toBe(0);
});

test('plan preload first-hop supports numeric foreign key ids', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name']));

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const calls: any[] = [];
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (_ctor: any, ids: string | string[], fields: string[]) => {
      calls.push({ ids: Array.isArray(ids) ? ids : [ids], fields });
      return [{ Id: '9', Name: 'Partner 9' }];
    },
    getNestedAt: () => undefined,
    upsertNestedAt: () => ({}),
  };

  const draft = { Id: 'SO-NUM', PartnerId: 9 } as any;
  await executeFirstHopPrefetch(context, meta, draft, stats);

  expect(calls).toEqual([{ ids: ['9'], fields: ['Id', 'Name'] }]);
  expect(draft.PartnerId).toEqual({ Id: '9', Name: 'Partner 9' });
  expect(stats.totalBatches).toBe(1);
  expect(stats.totalRows).toBe(1);
});

test('plan preload collections issues second-hop query with empty-string id when relation object id is empty', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.collections.set('Lines', { chains: [['ProductId', 'Name']] });

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => [{ Id: 'L-EMPTY-OBJ', ProductId: { Id: '' } }],
    } as any
  );

  const secondHopCalls: any[] = [];
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (_ctor: any, ids: string | string[], fields: string[]) => {
      secondHopCalls.push({ ids: Array.isArray(ids) ? ids : [ids], fields });
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = { Id: 'SO-COL-OBJ' } as any;
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(Array.isArray(draft.Lines)).toBe(true);
  expect(secondHopCalls).toEqual([{ ids: [''], fields: ['Id', 'Name'] }]);
  expect(stats.totalBatches).toBe(2);
  expect(stats.totalRows).toBe(1);
});

test('plan preload first-hop skips when foreign key id is blank after normalization', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name']));

  let searchCalls = 0;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: () => undefined,
    upsertNestedAt: () => ({}),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  await executeFirstHopPrefetch(context, meta, { PartnerId: { Id: '' } } as any, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload multi-hop skips chains when prefix points to non-many2one field', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.m2oChains.set('PartnerId', [['Name', 'Code', 'Value']]);

  const draft = {
    Id: 'SO-MH-PREFIX',
    PartnerId: { Id: 'P-1', Name: 'Partner' },
  } as any;

  let searchCalls = 0;
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  await executeMultiHopM2OPrefetch(context, meta, draft, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload collections skips second-hop when child many2one target model is missing', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadBrokenNext as any);
  const plan = makePlan();
  plan.collections.set('Items', { chains: [['BadRef', 'Name']] as any });

  RepositoryFactory.setRepository(
    PlanPreloadBrokenItem as any,
    {
      search: async () => [{ Id: 'I-1', BadRef: 'R-1' }],
    } as any
  );

  let secondHopCalls = 0;
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      secondHopCalls += 1;
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = { Id: 'BROKEN-NEXT-1' } as any;
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(Array.isArray(draft.Items)).toBe(true);
  expect(secondHopCalls).toBe(0);
  expect(stats.totalBatches).toBe(1);
  expect(stats.totalRows).toBe(1);
});

test('plan preload multi-hop skips invalid roots, empty chains, and too-short chains without querying', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.m2oChains.set('UnknownRoot', [['CompanyId', 'Name']]);
  plan.m2oChains.set('Lines', [['ProductId', 'Name']]);
  plan.m2oChains.set('PartnerId', [] as any);
  plan.m2oChains.set('PartnerIdShort', [['CompanyId']] as any);

  const draft = {
    Id: 'SO-MH-SKIP-MIX',
    PartnerId: 'P-1',
  } as any;

  let searchCalls = 0;
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async () => {
      searchCalls += 1;
      return [];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  await executeMultiHopM2OPrefetch(context, meta, draft, stats);

  expect(searchCalls).toBe(0);
  expect(stats.totalBatches).toBe(0);
  expect(stats.totalRows).toBe(0);
});

test('plan preload collections second-hop wraps scalar many2one values into object before merge', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.collections.set('Lines', { chains: [['ProductId', 'Name']] });

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => [{ Id: 'L-SCALAR-1', ProductId: 'PR-S1' }],
    } as any
  );

  const secondHopCalls: any[] = [];
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (_ctor: any, ids: string | string[], fields: string[]) => {
      secondHopCalls.push({ ids: Array.isArray(ids) ? ids : [ids], fields });
      return [{ Id: 'PR-S1', Name: 'Product Scalar' }];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = { Id: 'SO-COL-SCALAR' } as any;
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(secondHopCalls).toEqual([{ ids: ['PR-S1'], fields: ['Id', 'Name'] }]);
  expect(draft.Lines?.[0]?.ProductId).toEqual({ Id: 'PR-S1', Name: 'Product Scalar' });
  expect(stats.totalBatches).toBe(2);
  expect(stats.totalRows).toBe(2);
});

test('plan preload records UnknownModel label when ctor name is empty', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name', 'CompanyId']));
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);
  plan.collections.set('Lines', { chains: [['ProductId', 'Name']] });

  const originalNames = [PlanPreloadPartner.name, PlanPreloadCompany.name, PlanPreloadLine.name, PlanPreloadProduct.name];

  Object.defineProperty(PlanPreloadPartner, 'name', { value: '', configurable: true });
  Object.defineProperty(PlanPreloadCompany, 'name', { value: '', configurable: true });
  Object.defineProperty(PlanPreloadLine, 'name', { value: '', configurable: true });
  Object.defineProperty(PlanPreloadProduct, 'name', { value: '', configurable: true });

  try {
    RepositoryFactory.setRepository(
      PlanPreloadLine as any,
      {
        search: async () => [{ Id: 'L-U1', ProductId: 'PR-U1' }],
      } as any
    );

    const nesting = new PathPlanExecutor(makePlan()) as any;
    const context: PathPlanPreloadContext = {
      plan,
      recordStat,
      searchWithCache: async (ctor: any, _ids: string | string[], _fields: string[]) => {
        if (ctor === PlanPreloadPartner) return [{ Id: 'P-U1', Name: 'Partner U', CompanyId: 'C-U1' }];
        if (ctor === PlanPreloadCompany) return [{ Id: 'C-U1', Name: 'Company U' }];
        if (ctor === PlanPreloadProduct) return [{ Id: 'PR-U1', Name: 'Product U' }];
        return [];
      },
      getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
      upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
    };

    const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
    const draft = { Id: 'SO-U1', PartnerId: 'P-U1' } as any;

    await executeFirstHopPrefetch(context, meta, draft, stats);
    await executeMultiHopM2OPrefetch(context, meta, draft, stats);
    await executeCollectionsPrefetch(context, meta, draft, stats);

    expect(stats.batches.every(item => item.model === 'UnknownModel')).toBe(true);
  } finally {
    Object.defineProperty(PlanPreloadPartner, 'name', { value: originalNames[0], configurable: true });
    Object.defineProperty(PlanPreloadCompany, 'name', { value: originalNames[1], configurable: true });
    Object.defineProperty(PlanPreloadLine, 'name', { value: originalNames[2], configurable: true });
    Object.defineProperty(PlanPreloadProduct, 'name', { value: originalNames[3], configurable: true });
  }
});

test('plan preload multi-hop deep-prefix branches skip non-many2one and missing target model paths', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const partnerMeta = MetadataStorage.instance.getModelMetadata(PlanPreloadPartner as any);
  const patchedPartnerFields = new Map(partnerMeta.fields);
  patchedPartnerFields.set('BrokenRef', {
    type: 'ManyToOne',
    relation: { targetModel: () => undefined as any },
    column: {},
  } as any);

  MetadataStorage.instance.setModelMetadata(
    PlanPreloadPartner as any,
    {
      ...partnerMeta,
      fields: patchedPartnerFields,
    } as any
  );

  try {
    const plan = makePlan();
    plan.m2oChains.set('PartnerId', [
      ['Name', 'Code', 'Value'],
      ['BrokenRef', 'Code', 'Value'],
      ['UnknownPrev', 'Name'],
      ['BrokenRef', 'Name'],
    ] as any);

    let searchCalls = 0;
    const nesting = new PathPlanExecutor(makePlan()) as any;
    const context: PathPlanPreloadContext = {
      plan,
      recordStat,
      searchWithCache: async () => {
        searchCalls += 1;
        return [];
      },
      getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
      upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
    };

    const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
    const draft = {
      Id: 'SO-MH-DEEP-SKIP',
      PartnerId: { Id: 'P-DEEP-1', BrokenRef: 'BR-DEEP-1' },
    } as any;

    await executeMultiHopM2OPrefetch(context, meta, draft, stats);

    expect(searchCalls).toBe(0);
    expect(stats.totalBatches).toBe(0);
    expect(stats.totalRows).toBe(0);
  } finally {
    MetadataStorage.instance.setModelMetadata(PlanPreloadPartner as any, partnerMeta as any);
  }
});

test('plan preload collections second-hop keeps object relation payload and merges fetched fields', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(PlanPreloadOrder as any);
  const plan = makePlan();
  plan.collections.set('Lines', { chains: [['ProductId', 'Name']] });

  RepositoryFactory.setRepository(
    PlanPreloadLine as any,
    {
      search: async () => [
        {
          Id: 'L-OBJ-KEEP-1',
          ProductId: { Id: 'PR-OBJ-1', Existing: 'keep' },
        },
      ],
    } as any
  );

  const secondHopCalls: any[] = [];
  const nesting = new PathPlanExecutor(makePlan()) as any;
  const context: PathPlanPreloadContext = {
    plan,
    recordStat,
    searchWithCache: async (_ctor: any, ids: string | string[], fields: string[]) => {
      secondHopCalls.push({ ids: Array.isArray(ids) ? ids : [ids], fields });
      return [{ Id: 'PR-OBJ-1', Name: 'Product Obj' }];
    },
    getNestedAt: (d, root, prefix) => nesting.getNestedAt(d, root, prefix),
    upsertNestedAt: (d, root, prefix, id) => nesting.upsertNestedAt(d, root, prefix, id),
  };

  const stats: PrefetchExecStats = { batches: [], totalBatches: 0, totalRows: 0 };
  const draft = { Id: 'SO-COL-OBJ-MERGE' } as any;
  await executeCollectionsPrefetch(context, meta, draft, stats);

  expect(secondHopCalls).toEqual([{ ids: ['PR-OBJ-1'], fields: ['Id', 'Name'] }]);
  expect(draft.Lines?.[0]?.ProductId).toEqual({
    Id: 'PR-OBJ-1',
    Existing: 'keep',
    Name: 'Product Obj',
  });
  expect(stats.totalBatches).toBe(2);
  expect(stats.totalRows).toBe(2);
});
