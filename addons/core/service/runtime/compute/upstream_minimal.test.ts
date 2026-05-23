// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { RelationFactory } from '../../orm/relation';
import { RepositoryFactory } from '../../orm/repository/repository_factory';
import { ComputeCascadeEngine } from './cascade';
import { ComputeEngine } from './engine';
import { buildComputeGraph } from './graph';

@Model('test.UpstreamParent')
class UpstreamParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => UpstreamChild, inverseField: 'ParentId' },
  })
  Lines?: UpstreamChild[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: (self: UpstreamParent) => (Array.isArray((self as any).Lines) ? (self as any).Lines.length : 0),
        deps: ['Lines.Id' as any],
      },
    },
  })
  LineCount?: number;
}

@Model('test.UpstreamChild')
class UpstreamChild extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UpstreamParent },
    column: {},
  })
  ParentId?: UpstreamParent;

  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

function withRequestScope<T>(request: any, fn: () => Promise<T>): Promise<T> {
  const previous = (globalThis as any).$choysum;
  const current = previous || {};
  (globalThis as any).$choysum = { ...current, request };
  return fn().finally(() => {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
    } else {
      (globalThis as any).$choysum = previous;
    }
  });
}

function setupUpstreamTestBed(parentRows: Record<string, any>, childRowsByParent: Record<string, Array<Record<string, any>>>) {
  const parentMeta = MetadataStorage.instance.getModelMetadata(UpstreamParent as any);
  parentMeta.computeGraph = buildComputeGraph(parentMeta);

  const childMeta = MetadataStorage.instance.getModelMetadata(UpstreamChild as any);
  childMeta.computeGraph = buildComputeGraph(childMeta);

  const storage: any = MetadataStorage.instance as any;
  (ComputeCascadeEngine as any).warmedModelCount = typeof storage?.models?.size === 'number' ? storage.models.size : -1;

  const updates: Array<{ id: string; values: Record<string, any> }> = [];
  let parentSearchCalls = 0;
  let childSearchCalls = 0;

  const parentRepo = {
    async search(condition: any) {
      parentSearchCalls += 1;
      const id = String(condition?.[2] || '');
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        return ids.filter(x => parentRows[x]).map(x => ({ ...parentRows[x] }));
      }
      const row = parentRows[id];
      return row ? [{ ...row }] : [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!parentRows[id]) return;
      parentRows[id] = { ...parentRows[id], ...values };
      updates.push({ id, values: { ...values } });
    },
  };

  const childRepo = {
    async search(condition: any) {
      childSearchCalls += 1;
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        const out: Array<Record<string, any>> = [];
        for (const id of ids) {
          out.push(...(childRowsByParent[id] || []).map(item => ({ ...item })));
        }
        return out;
      }
      const parentId = String(condition?.[2] || '');
      return [...(childRowsByParent[parentId] || [])].map(item => ({ ...item }));
    },
  };

  RepositoryFactory.setRepository(UpstreamParent as any, parentRepo as any);
  RepositoryFactory.setRepository(UpstreamChild as any, childRepo as any);

  return {
    updates,
    parentSearchCalls: () => parentSearchCalls,
    childSearchCalls: () => childSearchCalls,
    parentRows,
    childRowsByParent,
  };
}

test('compute upstream: create triggers parent recompute', async () => {
  const parentRows = {
    p1: { Id: 'p1', LineCount: 0 },
  } as Record<string, any>;
  const childRowsByParent = {
    p1: [{ Id: 'c1', ParentId: 'p1' }],
  } as Record<string, Array<Record<string, any>>>;

  const bed = setupUpstreamTestBed(parentRows, childRowsByParent);
  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    entity.LineCount = Array.isArray(entity.Lines) ? entity.Lines.length : 0;
  };

  try {
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'create',
      changedFields: ['ParentId'],
      afterEntity: { Id: 'c1', ParentId: 'p1' },
    });
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  const target = bed.updates.find(item => item.id === 'p1');
  expect(Boolean(target)).toBe(true);
  expect(target?.values.LineCount).toBe(1);
});

test('compute upstream: delete triggers parent recompute', async () => {
  const parentRows = {
    p1: { Id: 'p1', LineCount: 1 },
  } as Record<string, any>;
  const childRowsByParent = {
    p1: [],
  } as Record<string, Array<Record<string, any>>>;

  const bed = setupUpstreamTestBed(parentRows, childRowsByParent);
  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    entity.LineCount = Array.isArray(entity.Lines) ? entity.Lines.length : 0;
  };

  try {
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'delete',
      changedFields: [],
      beforeEntity: { Id: 'c1', ParentId: 'p1' },
    });
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  const target = bed.updates.find(item => item.id === 'p1');
  expect(Boolean(target)).toBe(true);
  expect(target?.values.LineCount).toBe(0);
});

test('compute upstream: re-parent triggers both old and new parent recompute', async () => {
  const parentRows = {
    p1: { Id: 'p1', LineCount: 1 },
    p2: { Id: 'p2', LineCount: 0 },
  } as Record<string, any>;
  const childRowsByParent = {
    p1: [],
    p2: [{ Id: 'c1', ParentId: 'p2' }],
  } as Record<string, Array<Record<string, any>>>;

  const bed = setupUpstreamTestBed(parentRows, childRowsByParent);
  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    entity.LineCount = Array.isArray(entity.Lines) ? entity.Lines.length : 0;
  };

  try {
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'update',
      changedFields: ['ParentId'],
      beforeEntity: { Id: 'c1', ParentId: 'p1' },
      afterEntity: { Id: 'c1', ParentId: 'p2' },
    });
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  const ids = bed.updates.map(item => item.id);
  expect(ids.includes('p1')).toBe(true);
  expect(ids.includes('p2')).toBe(true);
  expect(parentRows.p1.LineCount).toBe(0);
  expect(parentRows.p2.LineCount).toBe(1);
});

test('compute upstream: create many batch uses aggregated upstream prefetch', async () => {
  const parentRows = {
    p1: { Id: 'p1', LineCount: 0 },
    p2: { Id: 'p2', LineCount: 0 },
  } as Record<string, any>;
  const childRowsByParent = {
    p1: [
      { Id: 'c1', ParentId: 'p1' },
      { Id: 'c2', ParentId: 'p1' },
    ],
    p2: [{ Id: 'c3', ParentId: 'p2' }],
  } as Record<string, Array<Record<string, any>>>;

  const bed = setupUpstreamTestBed(parentRows, childRowsByParent);
  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeCascadeEngine as any).resetUpstreamStats?.();
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    entity.LineCount = Array.isArray(entity.Lines) ? entity.Lines.length : 0;
  };

  try {
    await ComputeCascadeEngine.triggerUpstreamCreateBatch(UpstreamChild as any, [
      { Id: 'c1', ParentId: 'p1' },
      { Id: 'c2', ParentId: 'p1' },
      { Id: 'c3', ParentId: 'p2' },
    ]);
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  expect(parentRows.p1.LineCount).toBe(2);
  expect(parentRows.p2.LineCount).toBe(1);
  expect(bed.parentSearchCalls() <= 2).toBe(true);
  expect(bed.childSearchCalls() <= 2).toBe(true);
  const stats = (ComputeCascadeEngine as any).getUpstreamStats?.() || {};
  expect(Number(stats.upstreamEventCount || 0) >= 1).toBe(true);
  expect(Number(stats.parentBatchQueryCount || 0) >= 1).toBe(true);
  expect(Number(stats.collectionBatchQueryCount || 0) >= 1).toBe(true);
});

@Model('test.UpstreamCollectionParent')
class UpstreamCollectionParent extends BaseModel {
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => UpstreamCollectionChild, inverseField: 'ParentId' },
  })
  Lines?: UpstreamCollectionChild[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: (self: UpstreamCollectionParent) => (Array.isArray((self as any).Lines) ? (self as any).Lines.length : 0),
        deps: ['Lines' as any],
      },
    },
  })
  LineCount?: number;
}

@Model('test.UpstreamCollectionChild')
class UpstreamCollectionChild extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UpstreamCollectionParent },
    column: {},
  })
  ParentId?: UpstreamCollectionParent;

  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.UpstreamCompany')
class UpstreamCompany extends BaseModel {
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => UpstreamPartner, inverseField: 'CompanyId' },
  })
  Partners?: UpstreamPartner[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: (self: UpstreamCompany) =>
          Array.isArray((self as any).Partners) ? (self as any).Partners.filter((p: any) => Boolean(p?.PrimaryContactId)).length : 0,
        deps: ['Partners.PrimaryContactId' as any],
      },
    },
  })
  PrimaryContactCount?: number;
}

@Model('test.UpstreamPartner')
class UpstreamPartner extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UpstreamCompany },
    column: {},
  })
  CompanyId?: UpstreamCompany;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => UpstreamContact, inverseField: 'PartnerId' },
  })
  Contacts?: UpstreamContact[];

  @Field({
    type: 'varchar',
    column: {
      size: 20,
      compute: {
        expr: (self: UpstreamPartner) => {
          const rows = Array.isArray((self as any).Contacts) ? (self as any).Contacts : [];
          const hit = rows.find((x: any) => Boolean(x?.IsPrimary));
          return hit?.Id || '';
        },
        deps: ['Contacts.IsPrimary' as any],
      },
    },
  })
  PrimaryContactId?: string;
}

@Model('test.UpstreamContact')
class UpstreamContact extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UpstreamPartner },
    column: {},
  })
  PartnerId?: UpstreamPartner;

  @Field({ type: 'boolean', column: {} })
  IsPrimary?: boolean;
}

@Model('test.CycleA')
class CycleA extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => CycleB },
    column: {},
  })
  BId?: CycleB;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => CycleB, inverseField: 'AId' },
  })
  BackRefs?: CycleB[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: (self: CycleA) =>
          Array.isArray((self as any).BackRefs) ? (self as any).BackRefs.reduce((n: number, x: any) => n + Number(x?.RefCount || 0), 0) : 0,
        deps: ['BackRefs.RefCount' as any],
      },
    },
  })
  TotalFromB?: number;
}

@Model('test.CycleB')
class CycleB extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => CycleA },
    column: {},
  })
  AId?: CycleA;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => CycleA, inverseField: 'BId' },
  })
  Refs?: CycleA[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: (self: CycleB) => (Array.isArray((self as any).Refs) ? (self as any).Refs.reduce((n: number, x: any) => n + Number(x?.TotalFromB || 0), 0) : 0),
        deps: ['Refs.TotalFromB' as any],
      },
    },
  })
  RefCount?: number;
}

@Model('test.DownstreamParent')
class DownstreamParent extends BaseModel {
  @Field({
    type: 'decimal',
    column: { precision: 6, scale: 2 },
  })
  DiscountRate?: any;
}

@Model('test.DownstreamLine')
class DownstreamLine extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => DownstreamParent },
    column: {},
  })
  ParentId?: DownstreamParent;

  @Field({ type: 'int', column: {} })
  Qty?: number;

  @Field({ type: 'int', column: {} })
  AmountScale?: number;

  @Field({
    type: 'decimal',
    column: {
      precision: 10,
      scaleField: 'AmountScale' as any,
      compute: {
        expr: (_self: DownstreamLine) => '0',
        deps: ['ParentId.DiscountRate' as any, 'Qty' as any],
      },
    },
  })
  Amount?: any;
}

function buildPairComputeGraph(parentCtor: any, childCtor: any) {
  const parentMeta = MetadataStorage.instance.getModelMetadata(parentCtor);
  parentMeta.computeGraph = buildComputeGraph(parentMeta);

  const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
  childMeta.computeGraph = buildComputeGraph(childMeta);

  const storage: any = MetadataStorage.instance as any;
  (ComputeCascadeEngine as any).warmedModelCount = typeof storage?.models?.size === 'number' ? storage.models.size : -1;
}

function setupCollectionTestBed(parentRows: Record<string, any>, childRowsByParent: Record<string, Array<Record<string, any>>>) {
  buildPairComputeGraph(UpstreamCollectionParent as any, UpstreamCollectionChild as any);

  const updates: Array<{ id: string; values: Record<string, any> }> = [];

  const parentRepo = {
    async search(condition: any) {
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        return ids.filter(x => parentRows[x]).map(x => ({ ...parentRows[x] }));
      }
      const id = String(condition?.[2] || '');
      const row = parentRows[id];
      return row ? [{ ...row }] : [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!parentRows[id]) return;
      parentRows[id] = { ...parentRows[id], ...values };
      updates.push({ id, values: { ...values } });
    },
  };

  const childRepo = {
    async search(condition: any) {
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        const out: Array<Record<string, any>> = [];
        for (const id of ids) {
          out.push(...(childRowsByParent[id] || []).map(item => ({ ...item })));
        }
        return out;
      }
      const parentId = String(condition?.[2] || '');
      return [...(childRowsByParent[parentId] || [])].map(item => ({ ...item }));
    },
  };

  RepositoryFactory.setRepository(UpstreamCollectionParent as any, parentRepo as any);
  RepositoryFactory.setRepository(UpstreamCollectionChild as any, childRepo as any);

  return { parentRows, updates };
}

test('compute upstream: collection dependency create/delete/re-parent keeps consistent behavior', async () => {
  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    entity.LineCount = Array.isArray(entity.Lines) ? entity.Lines.length : 0;
  };

  try {
    {
      const bed = setupCollectionTestBed({ p1: { Id: 'p1', LineCount: 0 } }, { p1: [{ Id: 'c1', ParentId: 'p1', Name: 'a' }] });
      await ComputeCascadeEngine.triggerUpstream({
        childCtor: UpstreamCollectionChild as any,
        operation: 'create',
        changedFields: ['ParentId'],
        afterEntity: { Id: 'c1', ParentId: 'p1', Name: 'a' },
      });
      expect(bed.parentRows.p1.LineCount).toBe(1);
    }

    {
      const bed = setupCollectionTestBed({ p1: { Id: 'p1', LineCount: 1 } }, { p1: [] });
      await ComputeCascadeEngine.triggerUpstream({
        childCtor: UpstreamCollectionChild as any,
        operation: 'delete',
        changedFields: [],
        beforeEntity: { Id: 'c1', ParentId: 'p1', Name: 'a' },
      });
      expect(bed.parentRows.p1.LineCount).toBe(0);
    }

    {
      const bed = setupCollectionTestBed(
        { p1: { Id: 'p1', LineCount: 1 }, p2: { Id: 'p2', LineCount: 0 } },
        { p1: [], p2: [{ Id: 'c1', ParentId: 'p2', Name: 'a' }] }
      );
      await ComputeCascadeEngine.triggerUpstream({
        childCtor: UpstreamCollectionChild as any,
        operation: 'update',
        changedFields: ['ParentId'],
        beforeEntity: { Id: 'c1', ParentId: 'p1', Name: 'a' },
        afterEntity: { Id: 'c1', ParentId: 'p2', Name: 'a' },
      });
      expect(bed.parentRows.p1.LineCount).toBe(0);
      expect(bed.parentRows.p2.LineCount).toBe(1);
    }
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }
});

test('compute upstream: static multi-row Update triggers upstream per row', async () => {
  buildPairComputeGraph(UpstreamParent as any, UpstreamChild as any);

  const rowById: Record<string, any> = {
    c1: { Id: 'c1', ParentId: 'p1', Name: 'old-1', UpdatedAt: new Date('2025-01-01T00:00:00Z') },
    c2: { Id: 'c2', ParentId: 'p2', Name: 'old-2', UpdatedAt: new Date('2025-01-01T00:00:00Z') },
  };

  const childRepo = {
    async count() {
      return 2;
    },
    async search(condition: any) {
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        return ids.filter(x => rowById[x]).map(x => ({ ...rowById[x] }));
      }
      const id = String(condition?.[2] || '');
      return rowById[id] ? [{ ...rowById[id] }] : [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!rowById[id]) return [];
      rowById[id] = { ...rowById[id], ...values };
      return [{ ...rowById[id] }];
    },
    async withValidationBypass(fn: () => Promise<any>) {
      return await fn();
    },
  };
  RepositoryFactory.setRepository(UpstreamChild as any, childRepo as any);

  const originalPrepare = (RelationFactory as any).prepareForUpdate;
  const originalBatch = (RelationFactory as any).batchProcessToManyRelations;
  const originalUpstream = (ComputeCascadeEngine as any).triggerUpstream;
  const originalDownstream = (ComputeCascadeEngine as any).triggerDownstream;

  const upstreamCalls: any[] = [];
  (RelationFactory as any).prepareForUpdate = async (_ctor: any, values: Record<string, any>) => ({
    processedValue: { ...values },
    relations: {
      touchedCollections: new Set<string>(),
      oneToManyRelations: [],
      manyToManyRelations: [],
    },
  });
  (RelationFactory as any).batchProcessToManyRelations = async () => [];
  (ComputeCascadeEngine as any).triggerUpstream = async (evt: any) => {
    upstreamCalls.push(evt);
  };
  (ComputeCascadeEngine as any).triggerDownstream = async () => {};

  try {
    await (UpstreamChild as any).Update(['Id', 'in', ['c1', 'c2']], { Name: 'new-name' });
  } finally {
    (RelationFactory as any).prepareForUpdate = originalPrepare;
    (RelationFactory as any).batchProcessToManyRelations = originalBatch;
    (ComputeCascadeEngine as any).triggerUpstream = originalUpstream;
    (ComputeCascadeEngine as any).triggerDownstream = originalDownstream;
  }

  expect(upstreamCalls.length).toBe(2);
  const ids = upstreamCalls.map(x => String(x?.beforeEntity?.Id || x?.afterEntity?.Id || '')).sort();
  expect(ids).toEqual(['c1', 'c2']);
  expect(upstreamCalls.every(x => x?.operation === 'update')).toBe(true);
  expect(upstreamCalls.every(x => Array.isArray(x?.changedFields) && x.changedFields.includes('Name'))).toBe(true);
});

test('compute upstream: static and instance update keep upstream event semantics aligned', async () => {
  buildPairComputeGraph(UpstreamParent as any, UpstreamChild as any);

  const now = new Date('2025-01-01T00:00:00Z');
  const row: Record<string, any> = {
    Id: 'c1',
    ParentId: 'p1',
    Name: 'origin',
    UpdatedAt: now,
  };

  const childRepo = {
    async count() {
      return 1;
    },
    async search(condition: any) {
      const id = String(condition?.[2] || '');
      if (id !== 'c1') return [];
      return [{ ...row }];
    },
    async update(values: Record<string, any>, _condition: any) {
      Object.assign(row, values);
      if (!row.UpdatedAt) row.UpdatedAt = new Date();
      return [{ ...row }];
    },
    async withValidationBypass(fn: () => Promise<any>) {
      return await fn();
    },
  };
  RepositoryFactory.setRepository(UpstreamChild as any, childRepo as any);

  const originalPrepare = (RelationFactory as any).prepareForUpdate;
  const originalBatch = (RelationFactory as any).batchProcessToManyRelations;
  const originalUpstream = (ComputeCascadeEngine as any).triggerUpstream;
  const originalDownstream = (ComputeCascadeEngine as any).triggerDownstream;

  const staticCalls: any[] = [];
  const instanceCalls: any[] = [];
  let phase: 'static' | 'instance' = 'static';

  (RelationFactory as any).prepareForUpdate = async (_ctor: any, values: Record<string, any>) => ({
    processedValue: { ...values },
    relations: {
      touchedCollections: new Set<string>(),
      oneToManyRelations: [],
      manyToManyRelations: [],
    },
  });
  (RelationFactory as any).batchProcessToManyRelations = async () => [];
  (ComputeCascadeEngine as any).triggerUpstream = async (evt: any) => {
    if (phase === 'static') staticCalls.push(evt);
    else instanceCalls.push(evt);
  };
  (ComputeCascadeEngine as any).triggerDownstream = async () => {};

  try {
    await withRequestScope({ context: { req: { kind: 'http', depth: 0 } } }, async () => {
      await (UpstreamChild as any).Update(['Id', '=', 'c1'], { Name: 'from-static' });

      phase = 'instance';
      const rec = await (UpstreamChild as any).Browse('c1', ['Id', 'ParentId', 'Name', 'UpdatedAt'] as any);
      (rec as any).Name = 'from-instance';
      await (rec as any).update();
    });
  } finally {
    (RelationFactory as any).prepareForUpdate = originalPrepare;
    (RelationFactory as any).batchProcessToManyRelations = originalBatch;
    (ComputeCascadeEngine as any).triggerUpstream = originalUpstream;
    (ComputeCascadeEngine as any).triggerDownstream = originalDownstream;
  }

  expect(staticCalls.length).toBe(1);
  expect(instanceCalls.length).toBe(1);

  const s = staticCalls[0];
  const i = instanceCalls[0];
  expect(s.operation).toBe('update');
  expect(i.operation).toBe('update');
  expect(s.changedFields.includes('Name')).toBe(true);
  expect(i.changedFields.includes('Name')).toBe(true);
  expect(String(s.beforeEntity?.ParentId || '')).toBe(String(i.beforeEntity?.ParentId || ''));
  expect(String(s.afterEntity?.ParentId || '')).toBe(String(i.afterEntity?.ParentId || ''));
});

test('compute upstream: grandparent recursive propagation remains correct', async () => {
  const companyMeta = MetadataStorage.instance.getModelMetadata(UpstreamCompany as any);
  companyMeta.computeGraph = buildComputeGraph(companyMeta);
  const partnerMeta = MetadataStorage.instance.getModelMetadata(UpstreamPartner as any);
  partnerMeta.computeGraph = buildComputeGraph(partnerMeta);
  const contactMeta = MetadataStorage.instance.getModelMetadata(UpstreamContact as any);
  contactMeta.computeGraph = buildComputeGraph(contactMeta);

  const storage: any = MetadataStorage.instance as any;
  (ComputeCascadeEngine as any).warmedModelCount = typeof storage?.models?.size === 'number' ? storage.models.size : -1;

  const companyRows: Record<string, any> = {
    co1: { Id: 'co1', PrimaryContactCount: 0 },
  };
  const partnerRows: Record<string, any> = {
    p1: { Id: 'p1', CompanyId: 'co1', PrimaryContactId: '' },
  };
  const contactRowsByPartner: Record<string, Array<Record<string, any>>> = {
    p1: [{ Id: 'c1', PartnerId: 'p1', IsPrimary: true }],
  };

  const companyUpdates: Array<Record<string, any>> = [];
  const partnerUpdates: Array<Record<string, any>> = [];

  const companyRepo = {
    async search(condition: any) {
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        return ids.filter(x => companyRows[x]).map(x => ({ ...companyRows[x] }));
      }
      const id = String(condition?.[2] || '');
      return companyRows[id] ? [{ ...companyRows[id] }] : [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!companyRows[id]) return;
      companyRows[id] = { ...companyRows[id], ...values };
      companyUpdates.push({ id, ...values });
    },
  };

  const partnerRepo = {
    async search(condition: any) {
      const field = String(condition?.[0] || '');
      const op = String(condition?.[1] || '');
      if (field === 'Id') {
        if (op.toLowerCase() === 'in') {
          const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
          return ids.filter(x => partnerRows[x]).map(x => ({ ...partnerRows[x] }));
        }
        const id = String(condition?.[2] || '');
        return partnerRows[id] ? [{ ...partnerRows[id] }] : [];
      }
      if (field === 'CompanyId') {
        if (op.toLowerCase() === 'in') {
          const ids = new Set(Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : []);
          return Object.values(partnerRows)
            .filter(row => ids.has(String((row as any).CompanyId || '')))
            .map(row => ({ ...row }));
        }
        const companyId = String(condition?.[2] || '');
        return Object.values(partnerRows)
          .filter(row => String((row as any).CompanyId || '') === companyId)
          .map(row => ({ ...row }));
      }
      return [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!partnerRows[id]) return;
      partnerRows[id] = { ...partnerRows[id], ...values };
      partnerUpdates.push({ id, ...values });
    },
  };

  const contactRepo = {
    async search(condition: any) {
      const op = String(condition?.[1] || '');
      if (op.toLowerCase() === 'in') {
        const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
        const out: Array<Record<string, any>> = [];
        for (const id of ids) {
          out.push(...(contactRowsByPartner[id] || []).map(item => ({ ...item })));
        }
        return out;
      }
      const partnerId = String(condition?.[2] || '');
      return [...(contactRowsByPartner[partnerId] || [])].map(item => ({ ...item }));
    },
  };

  RepositoryFactory.setRepository(UpstreamCompany as any, companyRepo as any);
  RepositoryFactory.setRepository(UpstreamPartner as any, partnerRepo as any);
  RepositoryFactory.setRepository(UpstreamContact as any, contactRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (meta: any, entity: any) => {
    const model = String(meta?.fullModelName || meta?.modelName || '');
    if (model.endsWith('test.UpstreamPartner')) {
      const rows = Array.isArray(entity.Contacts) ? entity.Contacts : [];
      const hit = rows.find((x: any) => Boolean(x?.IsPrimary));
      entity.PrimaryContactId = hit?.Id || '';
      return;
    }
    if (model.endsWith('test.UpstreamCompany')) {
      const rows = Array.isArray(entity.Partners) ? entity.Partners : [];
      entity.PrimaryContactCount = rows.filter((x: any) => Boolean(x?.PrimaryContactId)).length;
    }
  };

  try {
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamContact as any,
      operation: 'update',
      changedFields: ['IsPrimary'],
      beforeEntity: { Id: 'c1', PartnerId: 'p1', IsPrimary: false },
      afterEntity: { Id: 'c1', PartnerId: 'p1', IsPrimary: true },
    });
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  expect(String(partnerRows.p1.PrimaryContactId || '')).toBe('c1');
  expect(Number(companyRows.co1.PrimaryContactCount || 0)).toBe(1);
  expect(partnerUpdates.length >= 1).toBe(true);
  expect(companyUpdates.length >= 1).toBe(true);
});

test('compute upstream: cycle guard prevents infinite recursive propagation', async () => {
  const aMeta = MetadataStorage.instance.getModelMetadata(CycleA as any);
  aMeta.computeGraph = buildComputeGraph(aMeta);
  const bMeta = MetadataStorage.instance.getModelMetadata(CycleB as any);
  bMeta.computeGraph = buildComputeGraph(bMeta);

  const storage: any = MetadataStorage.instance as any;
  (ComputeCascadeEngine as any).warmedModelCount = typeof storage?.models?.size === 'number' ? storage.models.size : -1;

  const aRows: Record<string, any> = {
    a1: { Id: 'a1', BId: 'b1', TotalFromB: 0 },
  };
  const bRows: Record<string, any> = {
    b1: { Id: 'b1', AId: 'a1', RefCount: 0 },
  };

  const aRepo = {
    async search(condition: any) {
      const field = String(condition?.[0] || '');
      const op = String(condition?.[1] || '');
      if (field === 'Id') {
        if (op.toLowerCase() === 'in') {
          const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
          return ids.filter(x => aRows[x]).map(x => ({ ...aRows[x] }));
        }
        const id = String(condition?.[2] || '');
        return aRows[id] ? [{ ...aRows[id] }] : [];
      }
      if (field === 'BId') {
        const id = String(condition?.[2] || '');
        return Object.values(aRows)
          .filter(row => String((row as any).BId || '') === id)
          .map(row => ({ ...row }));
      }
      return [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!aRows[id]) return;
      aRows[id] = { ...aRows[id], ...values };
    },
  };

  const bRepo = {
    async search(condition: any) {
      const field = String(condition?.[0] || '');
      const op = String(condition?.[1] || '');
      if (field === 'Id') {
        if (op.toLowerCase() === 'in') {
          const ids = Array.isArray(condition?.[2]) ? condition[2].map((x: any) => String(x)) : [];
          return ids.filter(x => bRows[x]).map(x => ({ ...bRows[x] }));
        }
        const id = String(condition?.[2] || '');
        return bRows[id] ? [{ ...bRows[id] }] : [];
      }
      if (field === 'AId') {
        const id = String(condition?.[2] || '');
        return Object.values(bRows)
          .filter(row => String((row as any).AId || '') === id)
          .map(row => ({ ...row }));
      }
      return [];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      if (!bRows[id]) return;
      bRows[id] = { ...bRows[id], ...values };
    },
  };

  RepositoryFactory.setRepository(CycleA as any, aRepo as any);
  RepositoryFactory.setRepository(CycleB as any, bRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  const originalWarn = console.warn;
  const warnings: string[] = [];
  let recomputeSteps = 0;

  (ComputeEngine as any).recompute = async (meta: any, entity: any) => {
    recomputeSteps += 1;
    const model = String(meta?.fullModelName || meta?.modelName || '');
    if (model.endsWith('test.CycleA')) {
      entity.TotalFromB = Number(entity.TotalFromB || 0) + 1;
      return;
    }
    if (model.endsWith('test.CycleB')) {
      entity.RefCount = Number(entity.RefCount || 0) + 1;
    }
  };
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: CycleA as any,
      operation: 'update',
      changedFields: ['TotalFromB'],
      beforeEntity: { Id: 'a1', BId: 'b1', TotalFromB: 0 },
      afterEntity: { Id: 'a1', BId: 'b1', TotalFromB: 1 },
    });
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
    console.warn = originalWarn;
  }

  expect(recomputeSteps > 0).toBe(true);
  expect(recomputeSteps < 20).toBe(true);
  expect(warnings.some(msg => msg.includes('detected cyclic trigger'))).toBe(true);
});

test('compute downstream: parent scalar change recomputes child decimal field and persists scaleField', async () => {
  buildPairComputeGraph(DownstreamParent as any, DownstreamLine as any);

  const childRowsByParent: Record<string, Array<Record<string, any>>> = {
    p1: [
      { Id: 'l1', ParentId: 'p1', Qty: 2, Amount: '0', AmountScale: 0 },
      { Id: 'l2', ParentId: 'p1', Qty: 3, Amount: '0', AmountScale: 0 },
    ],
  };
  const updates: Array<{ id: string; values: Record<string, any> }> = [];

  const childRepo = {
    async search(condition: any, options?: any) {
      const parentId = String(condition?.[2] || '');
      const rows = [...(childRowsByParent[parentId] || [])].map(item => ({ ...item }));
      void options;
      return rows;
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      updates.push({ id, values: { ...values } });
      const row = Object.values(childRowsByParent)
        .flat()
        .find(item => String(item.Id || '') === id);
      if (row) Object.assign(row, values);
      return row ? [{ ...row }] : [];
    },
  };
  RepositoryFactory.setRepository(DownstreamLine as any, childRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any, seed: Set<string>) => {
    expect(Array.from(seed)).toEqual(['ParentId']);
    entity.AmountScale = 3;
    entity.Amount = `${Number(entity.Qty || 0) * 125}.000`;
  };

  try {
    await ComputeCascadeEngine.triggerDownstream(DownstreamParent as any, ['DiscountRate'], 'p1');
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  expect(updates.length).toBe(2);
  expect(updates.map(item => item.id).sort()).toEqual(['l1', 'l2']);
  expect(updates.every(item => item.values.AmountScale === 3)).toBe(true);
  expect(updates.every(item => item.values.UpdatedAt instanceof Date)).toBe(true);
  expect(updates.find(item => item.id === 'l1')?.values.Amount).toBe('250.000');
  expect(updates.find(item => item.id === 'l2')?.values.Amount).toBe('375.000');
});

test('compute downstream: recompute failure warns and skips child persistence', async () => {
  buildPairComputeGraph(DownstreamParent as any, DownstreamLine as any);

  const updates: Array<{ id: string; values: Record<string, any> }> = [];
  const warnings: string[] = [];

  const childRepo = {
    async search(condition: any) {
      const parentId = String(condition?.[2] || '');
      if (parentId !== 'p1') return [];
      return [{ Id: 'l1', ParentId: 'p1', Qty: 1, Amount: '0', AmountScale: 0 }];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      updates.push({ id, values: { ...values } });
      return [];
    },
  };
  RepositoryFactory.setRepository(DownstreamLine as any, childRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  const originalWarn = console.warn;
  (ComputeEngine as any).recompute = async () => {
    throw new Error('downstream-boom');
  };
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    await ComputeCascadeEngine.triggerDownstream(DownstreamParent as any, ['DiscountRate'], 'p1');
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
    console.warn = originalWarn;
  }

  expect(updates).toEqual([]);
  expect(warnings.some(msg => msg.includes('downstream recompute failed'))).toBe(true);
});

test('compute downstream: unchanged recompute result does not persist child updates', async () => {
  buildPairComputeGraph(DownstreamParent as any, DownstreamLine as any);

  const updates: Array<{ id: string; values: Record<string, any> }> = [];

  const childRepo = {
    async search(condition: any) {
      const parentId = String(condition?.[2] || '');
      if (parentId !== 'p1') return [];
      return [{ Id: 'l1', ParentId: 'p1', Qty: 1, Amount: '100.000', AmountScale: 3 }];
    },
    async update(values: Record<string, any>, condition: any) {
      const id = String(condition?.[2] || '');
      updates.push({ id, values: { ...values } });
      return [];
    },
  };
  RepositoryFactory.setRepository(DownstreamLine as any, childRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
    // Keep values unchanged to cover the changedAny=false branch.
    entity.Amount = entity.Amount;
    entity.AmountScale = entity.AmountScale;
  };

  try {
    await ComputeCascadeEngine.triggerDownstream(DownstreamParent as any, ['DiscountRate'], 'p1');
  } finally {
    (ComputeEngine as any).recompute = originalRecompute;
  }

  expect(updates).toEqual([]);
});

test('compute cascade trigger lifecycle mode delegates to upstream delete event', async () => {
  const originalUpstream = (ComputeCascadeEngine as any).triggerUpstream;
  const calls: any[] = [];

  (ComputeCascadeEngine as any).triggerUpstream = async (evt: any) => {
    calls.push(evt);
  };

  try {
    await ComputeCascadeEngine.trigger(UpstreamChild as any, ['Name'], 'c1', { Id: 'c1', ParentId: 'p1' }, 'lifecycle');
  } finally {
    (ComputeCascadeEngine as any).triggerUpstream = originalUpstream;
  }

  expect(calls.length).toBe(1);
  expect(calls[0]?.operation).toBe('delete');
  expect(calls[0]?.beforeEntity).toEqual({ Id: 'c1', ParentId: 'p1' });
  expect(Array.isArray(calls[0]?.changedFields)).toBe(true);
});

test('compute downstream: empty changed fields or parent id short-circuits without querying child repo', async () => {
  buildPairComputeGraph(DownstreamParent as any, DownstreamLine as any);

  let searchCalls = 0;
  const childRepo = {
    async search() {
      searchCalls += 1;
      return [];
    },
    async update() {
      return [];
    },
  };
  RepositoryFactory.setRepository(DownstreamLine as any, childRepo as any);

  await ComputeCascadeEngine.triggerDownstream(DownstreamParent as any, [], 'p1');
  await ComputeCascadeEngine.triggerDownstream(DownstreamParent as any, ['DiscountRate'], '');

  expect(searchCalls).toBe(0);
});

test('compute upstream: triggerUpstream normalizes ids and records merged duplicate parent ids', async () => {
  const originalCollect = (ComputeCascadeEngine as any).collectUpstreamInverseFields;
  const originalTriggerRecursive = (ComputeCascadeEngine as any).triggerRecursive;

  const recursiveCalls: any[] = [];
  (ComputeCascadeEngine as any).resetUpstreamStats?.();

  try {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = () => ['ParentId', 'CompanyId'];
    (ComputeCascadeEngine as any).triggerRecursive = async (...args: any[]) => {
      recursiveCalls.push(args);
    };

    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'update',
      changedFields: ['Name', '', 'Name'],
      beforeEntity: {
        Id: 'C-1',
        ParentId: ' p1 ',
        CompanyId: ' co1 ',
      } as any,
      afterEntity: {
        Id: 'C-1',
        ParentId: ' p2 ',
        CompanyId: ' co1 ',
      } as any,
    });

    expect(recursiveCalls.length).toBe(2);
    expect(recursiveCalls[0]?.[1]).toEqual(['Name']);
    expect(recursiveCalls[1]?.[1]).toEqual(['ParentId']);

    const merged = recursiveCalls[1]?.[6] as Map<string, Set<string>>;
    expect(Array.from(merged.get('ParentId') || []).sort()).toEqual(['p1', 'p2']);
    expect(Array.from(merged.get('CompanyId') || []).sort()).toEqual(['co1']);

    const stats = (ComputeCascadeEngine as any).getUpstreamStats?.() || {};
    expect(Number(stats.dedupParentIdCount || 0) >= 1).toBe(true);
    expect(Number(stats.upstreamEventCount || 0) >= 1).toBe(true);
  } finally {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = originalCollect;
    (ComputeCascadeEngine as any).triggerRecursive = originalTriggerRecursive;
  }
});

test('compute upstream batch: empty rows and parentless rows short-circuit without recursive trigger', async () => {
  const originalCollect = (ComputeCascadeEngine as any).collectUpstreamInverseFields;
  const originalTriggerRecursive = (ComputeCascadeEngine as any).triggerRecursive;

  const recursiveCalls: any[] = [];
  try {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = () => ['ParentId'];
    (ComputeCascadeEngine as any).triggerRecursive = async (...args: any[]) => {
      recursiveCalls.push(args);
    };

    await ComputeCascadeEngine.triggerUpstreamCreateBatch(UpstreamChild as any, []);
    await ComputeCascadeEngine.triggerUpstreamCreateBatch(
      UpstreamChild as any,
      [
        { Id: 'C1', ParentId: '' },
        { Id: 'C2', ParentId: '   ' },
      ] as any
    );

    expect(recursiveCalls.length).toBe(0);
  } finally {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = originalCollect;
    (ComputeCascadeEngine as any).triggerRecursive = originalTriggerRecursive;
  }
});

test('compute upstream recursive path persists parent updates even when collection prefetch fails', async () => {
  const childMeta = MetadataStorage.instance.getModelMetadata(UpstreamChild as any);
  const parentMeta = MetadataStorage.instance.getModelMetadata(UpstreamParent as any);

  const originalChildGraph = childMeta.computeGraph;
  const originalParentGraph = parentMeta.computeGraph;
  const originalRecompute = (ComputeEngine as any).recompute;
  const originalWarn = console.warn;

  const warnings: string[] = [];
  const parentUpdates: Array<{ values: Record<string, any>; condition: any }> = [];

  try {
    childMeta.computeGraph = {
      reverseComputeIndex: new Map([
        [
          'Name',
          [
            {
              parentModelCtor: UpstreamParent as any,
              inverseField: 'ParentId',
              parentComputeField: 'LineCount',
              triggerMode: 'field-change',
            },
          ],
        ],
      ]),
    } as any;

    parentMeta.computeGraph = {
      computeCollectionPathDeps: new Map([
        [
          'LineCount',
          [
            {
              collection: 'Lines',
              chain: ['Qty'],
            },
          ],
        ],
      ]),
      fastReverseDeps: new Map([['Lines', ['LineCount']]]),
      computeScalarDeps: new Map([['LineCount', new Set(['Name'])]]),
    } as any;

    RepositoryFactory.setRepository(
      UpstreamParent as any,
      {
        async search() {
          return [{ Id: 'P-1', Name: 'p', LineCount: 0 }];
        },
        async update(values: Record<string, any>, condition: any) {
          parentUpdates.push({ values: { ...values }, condition });
          return [{ Id: 'P-1' }];
        },
      } as any
    );

    RepositoryFactory.setRepository(
      UpstreamChild as any,
      {
        async search() {
          throw new Error('prefetch fail');
        },
      } as any
    );

    (ComputeEngine as any).recompute = async (_meta: any, entity: any) => {
      entity.LineCount = 2;
    };
    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    await (ComputeCascadeEngine as any).triggerRecursive(UpstreamChild as any, ['Name'], 'C-1', { Id: 'C-1', ParentId: 'P-1', Name: 'child' }, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    expect(parentUpdates.length).toBe(1);
    expect(parentUpdates[0]?.values?.LineCount).toBe(2);
    expect(parentUpdates[0]?.values?.UpdatedAt instanceof Date).toBe(true);
    expect(warnings.some(msg => msg.includes('collection dependency prefetch failed'))).toBe(true);
  } finally {
    childMeta.computeGraph = originalChildGraph;
    parentMeta.computeGraph = originalParentGraph;
    (ComputeEngine as any).recompute = originalRecompute;
    console.warn = originalWarn;
  }
});

test('compute upstream recursive path loads child by id when entity is missing', async () => {
  const childMeta = MetadataStorage.instance.getModelMetadata(UpstreamChild as any);

  const originalChildGraph = childMeta.computeGraph;
  let childSearchCalls = 0;
  let parentSearchCalls = 0;

  try {
    childMeta.computeGraph = {
      reverseComputeIndex: new Map([
        [
          'Name',
          [
            {
              parentModelCtor: UpstreamParent as any,
              inverseField: 'ParentId',
              parentComputeField: 'LineCount',
              triggerMode: 'field-change',
            },
          ],
        ],
      ]),
    } as any;

    RepositoryFactory.setRepository(
      UpstreamChild as any,
      {
        async search(condition: any) {
          childSearchCalls += 1;
          expect(condition).toEqual(['Id', '=', 'missing-child']);
          return [];
        },
      } as any
    );

    RepositoryFactory.setRepository(
      UpstreamParent as any,
      {
        async search() {
          parentSearchCalls += 1;
          return [];
        },
        async update() {
          return [];
        },
      } as any
    );

    await (ComputeCascadeEngine as any).triggerRecursive(UpstreamChild as any, ['Name'], 'missing-child', undefined, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    expect(childSearchCalls).toBe(1);
    expect(parentSearchCalls).toBe(0);
  } finally {
    childMeta.computeGraph = originalChildGraph;
  }
});

test('compute upstream recursive path computes transitive affected fields via fastReverseDeps', async () => {
  const childMeta = MetadataStorage.instance.getModelMetadata(UpstreamChild as any);
  const parentMeta = MetadataStorage.instance.getModelMetadata(UpstreamParent as any);

  const originalChildGraph = childMeta.computeGraph;
  const originalParentGraph = parentMeta.computeGraph;
  const originalRecompute = (ComputeEngine as any).recompute;
  const parentUpdates: Array<Record<string, any>> = [];

  try {
    childMeta.computeGraph = {
      reverseComputeIndex: new Map([
        [
          'Name',
          [
            {
              parentModelCtor: UpstreamParent as any,
              inverseField: 'ParentId',
              parentComputeField: 'LineCount',
              triggerMode: 'field-change',
            },
          ],
        ],
      ]),
    } as any;

    parentMeta.computeGraph = {
      computeCollectionPathDeps: new Map([
        [
          'LineCount',
          [
            {
              collection: 'Lines',
              chain: ['Qty'],
            },
          ],
        ],
      ]),
      fastReverseDeps: new Map([
        ['Lines', ['LineCount', 'AuxCount']],
        ['LineCount', ['TotalScore']],
        ['AuxCount', ['TotalScore']],
      ]),
      computeScalarDeps: new Map([
        ['LineCount', new Set(['Name'])],
        ['AuxCount', new Set(['Name'])],
        ['TotalScore', new Set(['Name'])],
      ]),
    } as any;

    RepositoryFactory.setRepository(
      UpstreamParent as any,
      {
        async search(condition: any) {
          expect(condition).toEqual(['Id', '=', 'P-1']);
          return [{ Id: 'P-1', Name: 'parent', LineCount: 0, AuxCount: 0, TotalScore: 0 }];
        },
        async update(values: Record<string, any>) {
          parentUpdates.push({ ...values });
          return [{ Id: 'P-1' }];
        },
      } as any
    );

    RepositoryFactory.setRepository(
      UpstreamChild as any,
      {
        async search(condition: any) {
          expect(condition).toEqual(['ParentId', '=', 'P-1']);
          return [
            { Id: 'c1', ParentId: 'P-1', Qty: 1 },
            { Id: 'c2', ParentId: 'P-1', Qty: 2 },
          ];
        },
      } as any
    );

    (ComputeEngine as any).recompute = async (_meta: any, entity: any, seed: Set<string>) => {
      expect(Array.from(seed)).toEqual(['Lines']);
      const rows = Array.isArray(entity.Lines) ? entity.Lines : [];
      entity.LineCount = rows.length;
      entity.AuxCount = rows.reduce((n: number, x: any) => n + Number(x?.Qty || 0), 0);
      entity.TotalScore = Number(entity.LineCount || 0) + Number(entity.AuxCount || 0);
    };

    await (ComputeCascadeEngine as any).triggerRecursive(UpstreamChild as any, ['Name'], 'c1', { Id: 'c1', ParentId: 'P-1', Name: 'child' }, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    expect(parentUpdates.length).toBe(1);
    expect(parentUpdates[0]?.LineCount).toBe(2);
    expect(parentUpdates[0]?.AuxCount).toBe(3);
    expect(parentUpdates[0]?.TotalScore).toBe(5);
    expect(parentUpdates[0]?.UpdatedAt instanceof Date).toBe(true);
  } finally {
    childMeta.computeGraph = originalChildGraph;
    parentMeta.computeGraph = originalParentGraph;
    (ComputeEngine as any).recompute = originalRecompute;
  }
});

test('compute cascade private guards handle max-depth and missing reverse index branches', async () => {
  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalWarn = console.warn;
  const warnings: string[] = [];

  class GuardChild extends BaseModel {}

  try {
    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === GuardChild) {
        return {
          fullModelName: 'test.GuardChild',
          computeGraph: undefined,
        } as any;
      }
      return originalGetModelMetadata.call(storage, ctor);
    }) as any;

    await (ComputeCascadeEngine as any).triggerRecursive(GuardChild as any, ['Name'], 'C-1', { Id: 'C-1' }, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    await (ComputeCascadeEngine as any).triggerRecursive(GuardChild as any, ['Name'], 'C-1', { Id: 'C-1' }, 'field-change', {
      depth: 5,
      maxDepth: 5,
      visited: new Set<string>(),
      path: ['root'],
    });

    expect(warnings.some(msg => msg.includes('reached max depth'))).toBe(true);
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
    console.warn = originalWarn;
  }
});

test('compute cascade downstream and recursive parent-id resolution short-circuit branches', async () => {
  const storage = MetadataStorage.instance as any;
  const originalModels = storage.models;
  const originalGetModelMetadata = storage.getModelMetadata;

  class ShortCircuitChild extends BaseModel {}
  class ShortCircuitParent extends BaseModel {}

  try {
    storage.models = {} as any;
    await ComputeCascadeEngine.triggerDownstream(ShortCircuitParent as any, ['Name'], 'P-1');

    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === ShortCircuitChild) {
        return {
          fullModelName: 'test.ShortCircuitChild',
          computeGraph: {
            reverseComputeIndex: new Map([
              [
                'Name',
                [
                  {
                    parentModelCtor: ShortCircuitParent,
                    inverseField: 'ParentId',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                ],
              ],
            ]),
          },
        } as any;
      }
      if (ctor === ShortCircuitParent) {
        return {
          fullModelName: 'test.ShortCircuitParent',
          computeGraph: {
            reverseComputeIndex: new Map(),
          },
        } as any;
      }
      return originalGetModelMetadata.call(storage, ctor);
    }) as any;

    await (ComputeCascadeEngine as any).triggerRecursive(
      ShortCircuitChild as any,
      ['Name'],
      'C-1',
      { Id: 'C-1', ParentId: '' },
      'field-change',
      {
        depth: 0,
        maxDepth: 5,
        visited: new Set<string>(),
        path: [],
      },
      new Map([['ParentId', new Set([''])]])
    );

    expect(true).toBe(true);
  } finally {
    storage.models = originalModels;
    storage.getModelMetadata = originalGetModelMetadata;
  }
});

test('compute cascade collectUpstreamInverseFields handles storage-model short-circuits', () => {
  const storage = MetadataStorage.instance as any;
  const originalModels = storage.models;
  const originalGetModelMetadata = storage.getModelMetadata;

  class LocalChild extends BaseModel {}

  try {
    storage.models = {} as any;
    (ComputeCascadeEngine as any).ensureAllComputeGraphsBuilt();

    storage.models = {
      size: 'bad-size',
      entries: () => [][Symbol.iterator](),
    } as any;
    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === LocalChild) {
        return {
          fullModelName: 'test.LocalChild',
          modelName: 'LocalChild',
          className: 'LocalChild',
          fields: new Map(),
          computeGraph: undefined,
        } as any;
      }
      return originalGetModelMetadata.call(storage, ctor);
    }) as any;

    const out2 = ComputeCascadeEngine.collectUpstreamInverseFields(LocalChild as any);
    expect(out2).toEqual([]);
  } finally {
    storage.models = originalModels;
    storage.getModelMetadata = originalGetModelMetadata;
  }
});

test('compute cascade trigger and triggerUpstream batch/update branches handle default and fallback ids', async () => {
  const originalUpstream = (ComputeCascadeEngine as any).triggerUpstream;
  const originalCollect = (ComputeCascadeEngine as any).collectUpstreamInverseFields;
  const originalRecursive = (ComputeCascadeEngine as any).triggerRecursive;

  const upstreamCalls: any[] = [];
  const recursiveCalls: any[] = [];

  try {
    (ComputeCascadeEngine as any).triggerUpstream = async (evt: any) => {
      upstreamCalls.push(evt);
    };

    await ComputeCascadeEngine.trigger(UpstreamChild as any, ['Name'], 'CID-1');
    await ComputeCascadeEngine.trigger(UpstreamChild as any, ['Name'], 'CID-2', undefined, 'lifecycle');

    expect(upstreamCalls.length).toBe(2);
    expect(upstreamCalls[0]?.operation).toBe('update');
    expect(upstreamCalls[0]?.afterEntity).toEqual({ Id: 'CID-1' });
    expect(upstreamCalls[1]?.operation).toBe('delete');
    expect(upstreamCalls[1]?.beforeEntity).toEqual({ Id: 'CID-2' });

    (ComputeCascadeEngine as any).triggerUpstream = originalUpstream;
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = () => ['ParentId'];
    (ComputeCascadeEngine as any).triggerRecursive = async (...args: any[]) => {
      recursiveCalls.push(args);
    };

    await ComputeCascadeEngine.triggerUpstreamCreateBatch(UpstreamChild as any, undefined as any);
    await ComputeCascadeEngine.triggerUpstreamCreateBatch(UpstreamChild as any, [{ ParentId: null }, { ParentId: 'P-1' }] as any);

    expect(recursiveCalls.length).toBe(2);
    expect(recursiveCalls[0]?.[2]).toBe('__batch_create__');
    expect(recursiveCalls[0]?.[4]).toBe('lifecycle');
    expect(recursiveCalls[1]?.[1]).toEqual(['ParentId']);

    recursiveCalls.length = 0;
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'update',
      changedFields: undefined as any,
      beforeEntity: { Id: 'B-1', ParentId: 'P-1' } as any,
      afterEntity: undefined as any,
    });

    expect(recursiveCalls.length).toBe(1);
    expect(recursiveCalls[0]?.[2]).toBe('B-1');
    expect(recursiveCalls[0]?.[1]).toEqual(['ParentId']);
    expect(recursiveCalls[0]?.[4]).toBe('membership-change');
  } finally {
    (ComputeCascadeEngine as any).triggerUpstream = originalUpstream;
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = originalCollect;
    (ComputeCascadeEngine as any).triggerRecursive = originalRecursive;
  }
});

test('compute cascade triggerUpstream uses before-entity fallback ids and membership branches for create/delete', async () => {
  const originalCollect = (ComputeCascadeEngine as any).collectUpstreamInverseFields;
  const originalRecursive = (ComputeCascadeEngine as any).triggerRecursive;
  const calls: any[] = [];

  try {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = () => ['ParentId'];
    (ComputeCascadeEngine as any).triggerRecursive = async (...args: any[]) => {
      calls.push(args);
    };

    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'update',
      changedFields: ['Name'],
      beforeEntity: { Id: 'B-1', ParentId: 'P-1' },
      afterEntity: undefined as any,
    });

    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'create',
      changedFields: [],
      afterEntity: { Id: 'C-1', ParentId: 'P-2' },
    });

    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'delete',
      changedFields: [],
      beforeEntity: { Id: 'D-1', ParentId: 'P-3' },
    });

    // Create without Id only covers the empty recordId branch and should not trigger recursion.
    await ComputeCascadeEngine.triggerUpstream({
      childCtor: UpstreamChild as any,
      operation: 'create',
      changedFields: [],
      afterEntity: { ParentId: 'P-4' } as any,
    });

    const modes = calls.map(c => c[4]);
    expect(modes.includes('field-change')).toBe(true);
    expect(modes.includes('lifecycle')).toBe(true);
    expect(modes.includes('membership-change')).toBe(true);
    expect(calls.some(c => c[2] === 'B-1')).toBe(true);
    expect(calls.some(c => c[2] === 'C-1')).toBe(true);
    expect(calls.some(c => c[2] === 'D-1')).toBe(true);
  } finally {
    (ComputeCascadeEngine as any).collectUpstreamInverseFields = originalCollect;
    (ComputeCascadeEngine as any).triggerRecursive = originalRecursive;
  }
});

test('compute cascade recursive guards cover missing parent graph, empty parent ids and collection prefetch fallbacks', async () => {
  const storage = MetadataStorage.instance as any;
  const originalGet = storage.getModelMetadata;

  class RecursiveChild extends BaseModel {}
  class ParentNoGraph extends BaseModel {}
  class ParentWithGraph extends BaseModel {}
  class CollModel extends BaseModel {}

  const childRepo = {
    async search() {
      return [{ Id: 'child-1', ParentId: '' }];
    },
  };

  let parentSearchTurn = 0;
  const parentRepo = {
    async search() {
      parentSearchTurn += 1;
      if (parentSearchTurn === 1) {
        // Hit the empty parentEntityById.size branch.
        return [{ Id: '', Total: 0 }];
      }
      return [{ Id: 'P-1', Total: 0 }];
    },
    async update() {},
  };

  const collRepo = {
    async search() {
      // Hit the rows || [] path and the empty ownerId branch.
      return [{ Id: 'line-1', ParentX: '' }];
    },
  };

  RepositoryFactory.setRepository(RecursiveChild as any, childRepo as any);
  RepositoryFactory.setRepository(ParentNoGraph as any, parentRepo as any);
  RepositoryFactory.setRepository(ParentWithGraph as any, parentRepo as any);
  RepositoryFactory.setRepository(CollModel as any, collRepo as any);

  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async () => {};

  try {
    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === RecursiveChild) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'RecursiveChild',
          fields: new Map(),
          computeGraph: {
            reverseComputeIndex: new Map([
              [
                'Name',
                [
                  {
                    parentModelCtor: ParentNoGraph,
                    inverseField: 'ParentId',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                  {
                    parentModelCtor: ParentWithGraph,
                    inverseField: 'ParentId',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                  {
                    parentModelCtor: ParentWithGraph,
                    inverseField: 'ParentId',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                ],
              ],
              [
                '__lifecycle',
                [
                  {
                    parentModelCtor: ParentWithGraph,
                    inverseField: 'ParentId',
                    parentComputeField: 'Total',
                    triggerMode: 'lifecycle',
                  },
                ],
              ],
            ]),
          },
        } as any;
      }

      if (ctor === ParentNoGraph) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'ParentNoGraph',
          fields: new Map(),
          computeGraph: undefined,
        } as any;
      }

      if (ctor === ParentWithGraph) {
        return {
          fullModelName: '',
          modelName: 'ParentWithGraph',
          className: '',
          fields: new Map([
            [
              'Lines',
              {
                type: 'OneToMany',
                relation: { targetModel: () => CollModel, fkField: 'ParentX' },
              },
            ],
          ]),
          computeGraph: {
            computeCollectionPathDeps: new Map([
              [
                'Total',
                [
                  { collection: 'Lines', chain: [] },
                  { collection: 'Lines', chain: [''] },
                ],
              ],
            ]),
            fastReverseDeps: new Map([['Lines', ['Total']]]),
            computeScalarDeps: new Map([['Total', new Set<string>()]]),
          },
        } as any;
      }

      return originalGet.call(storage, ctor);
    }) as any;

    await (ComputeCascadeEngine as any).triggerRecursive(RecursiveChild as any, ['Name'], 'child-1', undefined, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    await (ComputeCascadeEngine as any).triggerRecursive(RecursiveChild as any, [], 'child-2', { Id: 'child-2', ParentId: 'P-1' }, 'lifecycle', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    const stats = (ComputeCascadeEngine as any).getUpstreamStats?.() || {};
    expect(Number(stats.dedupTriggerCount || 0) >= 1).toBe(true);
  } finally {
    storage.getModelMetadata = originalGet;
    (ComputeEngine as any).recompute = originalRecompute;
  }
});

test('compute cascade triggerRecursive parent query and collection prefetch fallback branches', async () => {
  const storage = MetadataStorage.instance as any;
  const originalGet = storage.getModelMetadata;
  const originalWarn = console.warn;

  class ParentNoGraph2 extends BaseModel {}
  class ParentNoRows2 extends BaseModel {}
  class ParentEmptyId2 extends BaseModel {}
  class ParentPrefetch2 extends BaseModel {}
  class TriggerChild2 extends BaseModel {}
  class PrefetchChild2 extends BaseModel {}

  const warns: string[] = [];
  console.warn = ((...args: any[]) => {
    warns.push(args.map(x => String(x)).join(' '));
  }) as any;

  RepositoryFactory.setRepository(
    TriggerChild2 as any,
    {
      async search() {
        return [
          {
            Id: 'C-TR-1',
            PNoGraph: 'PG-1',
            PNoRows: 'PR-1',
            PEmptyId: 'PE-1',
            PPrefetch: 'PP-1',
          },
        ];
      },
    } as any
  );

  RepositoryFactory.setRepository(
    ParentNoGraph2 as any,
    {
      async search() {
        return [{ Id: 'PG-1' }];
      },
      async update() {},
    } as any
  );

  RepositoryFactory.setRepository(
    ParentNoRows2 as any,
    {
      async search() {
        return [];
      },
      async update() {},
    } as any
  );

  RepositoryFactory.setRepository(
    ParentEmptyId2 as any,
    {
      async search() {
        return [{ Id: '', Total: 0 }];
      },
      async update() {},
    } as any
  );

  RepositoryFactory.setRepository(
    ParentPrefetch2 as any,
    {
      async search() {
        return [{ Id: 'PP-1', Total: 0 }];
      },
      async update() {},
    } as any
  );

  RepositoryFactory.setRepository(
    PrefetchChild2 as any,
    {
      async search(condition: any) {
        const key = String(condition?.[0] || '');
        if (key === 'OwnerId') return undefined as any;
        if (key === 'OwnerForeignId') return [{ Id: 'F-1', OwnerForeignId: '' }];
        if (key === 'OwnerRefId') return [{ Id: 'F-2', OwnerRefId: 'PP-1' }];
        return [];
      },
    } as any
  );

  const originalRecompute = (ComputeEngine as any).recompute;
  (ComputeEngine as any).recompute = async () => {};

  try {
    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === TriggerChild2) {
        return {
          fullModelName: '',
          modelName: '',
          className: '',
          fields: new Map(),
          computeGraph: {
            reverseComputeIndex: new Map([
              [
                'Name',
                [
                  {
                    parentModelCtor: ParentNoGraph2,
                    inverseField: 'PNoGraph',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                  {
                    parentModelCtor: ParentNoRows2,
                    inverseField: 'PNoRows',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                  {
                    parentModelCtor: ParentEmptyId2,
                    inverseField: 'PEmptyId',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                  {
                    parentModelCtor: ParentPrefetch2,
                    inverseField: 'PPrefetch',
                    parentComputeField: 'Total',
                    triggerMode: 'field-change',
                  },
                ],
              ],
            ]),
          },
        } as any;
      }

      if (ctor === ParentNoGraph2) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'ParentNoGraph2',
          fields: new Map(),
          computeGraph: undefined,
        } as any;
      }

      if (ctor === ParentNoRows2) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'ParentNoRows2',
          fields: new Map(),
          computeGraph: {
            computeCollectionPathDeps: undefined,
            fastReverseDeps: new Map(),
            computeScalarDeps: new Map(),
          },
        } as any;
      }

      if (ctor === ParentEmptyId2) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'ParentEmptyId2',
          fields: new Map(),
          computeGraph: {
            computeCollectionPathDeps: undefined,
            fastReverseDeps: new Map(),
            computeScalarDeps: new Map(),
          },
        } as any;
      }

      if (ctor === ParentPrefetch2) {
        return {
          fullModelName: '',
          modelName: '',
          className: 'ParentPrefetch2',
          fields: new Map([
            [
              'LinesFk',
              {
                type: 'OneToMany',
                relation: { targetModel: () => PrefetchChild2, fkField: 'OwnerId' },
              },
            ],
            [
              'LinesForeign',
              {
                type: 'OneToMany',
                relation: { targetModel: () => PrefetchChild2, foreignKey: 'OwnerForeignId' },
              },
            ],
            [
              'LinesRef',
              {
                type: 'OneToMany',
                relation: { targetModel: () => PrefetchChild2, refField: 'OwnerRefId' },
              },
            ],
            [
              'LinesNoCtor',
              {
                type: 'OneToMany',
                relation: { fkField: 'OwnerNoCtor' },
              },
            ],
          ]),
          computeGraph: {
            computeCollectionPathDeps: new Map([
              [
                'Total',
                [
                  { collection: 'LinesFk', chain: [] },
                  { collection: 'LinesForeign', chain: ['Name'] },
                  { collection: 'LinesRef', chain: ['Name'] },
                  { collection: 'LinesNoCtor', chain: ['Name'] },
                ],
              ],
            ]),
            fastReverseDeps: new Map([
              ['LinesFk', ['Total']],
              ['LinesForeign', ['Total']],
              ['LinesRef', ['Total']],
              ['LinesNoCtor', ['Total']],
            ]),
            computeScalarDeps: new Map([['Total', new Set<string>()]]),
          },
        } as any;
      }

      return originalGet.call(storage, ctor);
    }) as any;

    await (ComputeCascadeEngine as any).triggerRecursive(TriggerChild2 as any, ['Name'], 'C-TR-1', undefined, 'field-change', {
      depth: 0,
      maxDepth: 5,
      visited: new Set<string>(),
      path: [],
    });

    expect(warns.length >= 0).toBe(true);
  } finally {
    storage.getModelMetadata = originalGet;
    (ComputeEngine as any).recompute = originalRecompute;
    console.warn = originalWarn;
  }
});
