// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '../../../orm/metadata/storage';
import { RepositoryFactory } from '../../../orm/repository/repository_factory';
import type { PathPrefetchPlan } from '../types';
import { OnchangeCacheManager } from '../cache';
import { PathPlanExecutor } from './executor';

@Model('test.PlanExecutorModel')
class PlanExecutorModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.PlanExecutorCompany')
class PlanExecutorCompany extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.PlanExecutorPartner')
class PlanExecutorPartner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanExecutorCompany } })
  CompanyId?: PlanExecutorCompany;
}

@Model('test.PlanExecutorProduct')
class PlanExecutorProduct extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.PlanExecutorOrder')
class PlanExecutorOrder extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanExecutorPartner } })
  PartnerId?: PlanExecutorPartner;

  @Field({ type: 'OneToMany', relation: { targetModel: () => PlanExecutorLine, inverseField: 'OrderId' } })
  Lines?: PlanExecutorLine[];
}

@Model('test.PlanExecutorLine')
class PlanExecutorLine extends BaseModel {
  @Field({ type: 'decimal', precision: 10, scale: 2 })
  Qty?: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanExecutorProduct } })
  ProductId?: PlanExecutorProduct;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => PlanExecutorOrder } })
  OrderId?: PlanExecutorOrder;
}

function makePlan(): PathPrefetchPlan {
  return {
    rootManyToOne: new Map(),
    m2oChains: new Map(),
    collections: new Map(),
  };
}

function makeCacheKey(modelCtor: any, fields: string[]): string {
  const meta = MetadataStorage.instance.getModelMetadata(modelCtor);
  const modelKey = (meta.fullModelName || meta.modelName || modelCtor.name || 'UnknownModel') as string;
  const fieldsSig = Array.from(new Set(['Id', ...fields]))
    .sort()
    .join(',');
  return `${modelKey}#${fieldsSig}`;
}

test('path plan executor reuses request cache first and then lru cache across instances', async () => {
  OnchangeCacheManager.clear();

  const searchCalls: Array<{ condition: any; options: any }> = [];
  RepositoryFactory.setRepository(
    PlanExecutorModel as any,
    {
      search: async (condition: any, options: any) => {
        searchCalls.push({ condition, options });
        const ids = condition[1] === 'in' ? condition[2] : [condition[2]];
        return ids.map((id: string) => ({ Id: String(id), Name: `Name-${id}` }));
      },
    } as any
  );

  const firstExecutor = new PathPlanExecutor(makePlan()) as any;
  const first = await firstExecutor.searchWithCache(PlanExecutorModel as any, ['1', '2'], ['Name']);
  const second = await firstExecutor.searchWithCache(PlanExecutorModel as any, ['2'], ['Name']);
  const thirdExecutor = new PathPlanExecutor(makePlan()) as any;
  const third = await thirdExecutor.searchWithCache(PlanExecutorModel as any, ['1'], ['Name']);

  expect(searchCalls.length).toBe(1);
  expect(searchCalls[0]?.condition).toEqual(['Id', 'in', ['1', '2']]);
  expect(searchCalls[0]?.options?.fields).toEqual(['Id', 'Name']);
  expect(first.map((row: any) => row.Id).sort()).toEqual(['1', '2']);
  expect(second.map((row: any) => row.Id)).toEqual(['2']);
  expect(third.map((row: any) => row.Id)).toEqual(['1']);

  OnchangeCacheManager.clear();
});

test('path plan executor upgrades primitive relation nodes into nested draft objects', () => {
  const executor = new PathPlanExecutor(makePlan()) as any;
  const draft: Record<string, any> = {
    PartnerId: 'P1',
  };

  const company = executor.getNestedAt(draft, 'PartnerId', ['CompanyId']);
  executor.upsertNestedAt(draft, 'PartnerId', ['CompanyId'], 'C1');

  expect(company).toBe(draft.PartnerId.CompanyId);
  expect(draft).toEqual({
    PartnerId: {
      Id: 'P1',
      CompanyId: {
        Id: 'C1',
      },
    },
  });
});

test('path plan executor keeps batch stats stable under mixed cache hits and repo misses', async () => {
  OnchangeCacheManager.clear();

  const orderMeta = MetadataStorage.instance.getModelMetadata(PlanExecutorOrder as any);
  const plan = makePlan();
  plan.rootManyToOne.set('PartnerId', new Set(['Name', 'CompanyId']));
  plan.m2oChains.set('PartnerId', [['CompanyId', 'Name']]);
  plan.collections.set('Lines', { chains: [['Qty'], ['ProductId'], ['ProductId', 'Name']] });

  const repoCalls: Array<{ model: string; condition: any; fields: string[] }> = [];

  RepositoryFactory.setRepository(
    PlanExecutorPartner as any,
    {
      search: async (condition: any, options: any) => {
        repoCalls.push({ model: 'PlanExecutorPartner', condition, fields: [...(options?.fields || [])] });
        return [{ Id: 'P1', Name: 'Partner A', CompanyId: 'C1' }];
      },
    } as any
  );

  RepositoryFactory.setRepository(
    PlanExecutorCompany as any,
    {
      search: async (condition: any, options: any) => {
        repoCalls.push({ model: 'PlanExecutorCompany', condition, fields: [...(options?.fields || [])] });
        return [{ Id: 'C1', Name: 'Company A' }];
      },
    } as any
  );

  RepositoryFactory.setRepository(
    PlanExecutorLine as any,
    {
      search: async (condition: any, options: any) => {
        repoCalls.push({ model: 'PlanExecutorLine', condition, fields: [...(options?.fields || [])] });
        return [
          { Id: 'L1', Qty: '2.00', ProductId: 'PR1' },
          { Id: 'L2', Qty: '5.00', ProductId: 'PR2' },
        ];
      },
    } as any
  );

  RepositoryFactory.setRepository(
    PlanExecutorProduct as any,
    {
      search: async (condition: any, options: any) => {
        repoCalls.push({ model: 'PlanExecutorProduct', condition, fields: [...(options?.fields || [])] });
        return [{ Id: 'PR2', Name: 'Product B' }];
      },
    } as any
  );

  OnchangeCacheManager.set(
    makeCacheKey(PlanExecutorPartner as any, ['Name', 'CompanyId']),
    new Map([['P1', { Id: 'P1', Name: 'Partner A', CompanyId: 'C1' }]])
  );
  OnchangeCacheManager.set(makeCacheKey(PlanExecutorProduct as any, ['Name']), new Map([['PR1', { Id: 'PR1', Name: 'Product A' }]]));

  const executor = new PathPlanExecutor(plan);
  const draft = { Id: 'SO1', PartnerId: 'P1' };

  const stats = await executor.execute(orderMeta as any, draft as any);

  expect(repoCalls).toEqual([
    { model: 'PlanExecutorCompany', condition: ['Id', '=', 'C1'], fields: ['Id', 'Name'] },
    { model: 'PlanExecutorLine', condition: ['OrderId', '=', 'SO1'], fields: ['Id', 'Qty', 'ProductId'] },
    { model: 'PlanExecutorProduct', condition: ['Id', '=', 'PR2'], fields: ['Id', 'Name'] },
  ]);

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

  expect(stats.totalBatches).toBe(4);
  expect(stats.totalRows).toBe(6);
  expect(stats.batches).toEqual([
    {
      phase: 'm2o',
      level: 1,
      model: 'PlanExecutorPartner',
      fields: ['Id', 'Name', 'CompanyId'],
      batchCount: 1,
      rowCount: 1,
      idsSample: ['P1'],
    },
    {
      phase: 'm2o',
      level: 2,
      model: 'PlanExecutorCompany',
      fields: ['Id', 'Name'],
      batchCount: 1,
      rowCount: 1,
      idsSample: ['C1'],
    },
    {
      phase: 'collection',
      level: 1,
      model: 'PlanExecutorLine',
      fields: ['Id', 'Qty', 'ProductId'],
      batchCount: 1,
      rowCount: 2,
      idsSample: ['SO1'],
    },
    {
      phase: 'collection',
      level: 2,
      model: 'PlanExecutorProduct',
      fields: ['Id', 'Name'],
      batchCount: 1,
      rowCount: 2,
      idsSample: ['PR1', 'PR2'],
    },
  ]);

  OnchangeCacheManager.clear();
});

test('path plan executor searchWithCache short-circuits empty ids without repository calls', async () => {
  const calls: any[] = [];
  RepositoryFactory.setRepository(
    PlanExecutorModel as any,
    {
      search: async () => {
        calls.push('called');
        return [];
      },
    } as any
  );

  const executor = new PathPlanExecutor(makePlan()) as any;
  const rows = await executor.searchWithCache(PlanExecutorModel as any, [], ['Name']);

  expect(rows).toEqual([]);
  expect(calls.length).toBe(0);
});

test('path plan executor upsertNestedAt injects id when node exists without Id field', () => {
  const executor = new PathPlanExecutor(makePlan()) as any;
  const draft: Record<string, any> = {
    PartnerId: {
      CompanyId: {},
    },
  };

  const node = executor.upsertNestedAt(draft, 'PartnerId', ['CompanyId'], 'C9');

  expect(node.Id).toBe('C9');
  expect(draft).toEqual({
    PartnerId: {
      CompanyId: {
        Id: 'C9',
      },
    },
  });
});
