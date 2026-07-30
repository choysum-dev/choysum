// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../../../runtime/context';
import { MetadataStorage } from '../../../metadata/storage';
import type { SelectionNode } from '../selection_tree';
import {
  applyRepositoryRelationCompanyFilter,
  applyRepositoryRelationSoftDeleteFilter,
  buildRelationJsonSelect,
  buildRepositoryRelationChildSelect,
} from '..';
import { applyRepositoryRelationFieldConditionFilter } from '../relation_projection';

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

test('repository relation projection applies soft-delete filter only when enabled for model', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(left: any, op: any, right: any) {
      calls.push({ left, op, right });
      return this;
    },
  };

  applyRepositoryRelationSoftDeleteFilter(
    {
      softDelete: true,
    } as any,
    'demo_table',
    query as any
  );

  expect(calls).toEqual([{ left: 'demo_table.DeletedAt', op: 'is', right: null }]);
});

test('repository relation projection applies parent field condition onto child subquery', () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    type: ParentModel,
    fields: new Map([
      [
        'LineIds',
        {
          name: 'LineIds',
          type: 'OneToMany',
          condition: ['Active', '=', true],
          conditionKind: 'static',
        },
      ],
      [
        'EmptyLines',
        {
          name: 'EmptyLines',
          type: 'OneToMany',
        },
      ],
    ]),
  } as any;

  const targetMeta = {
    type: ChildModel,
    tableName: () => 'child_table',
    fields: new Map([['Active', { type: 'boolean', column: { name: 'Active' } }]]),
  } as any;

  const db: any = { fn: { any: (v: any) => ({ any: v }) } };
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.and = (parts: any[]) => ({ kind: 'and', parts });
  eb.or = (parts: any[]) => ({ kind: 'or', parts });
  eb.ref = (value: string) => `ref:${value}`;
  eb.fn = (name: string, args: any[]) => ({ fn: name, args });

  const emptyCalls: any[] = [];
  const emptyQuery = {
    where(predicate: any) {
      emptyCalls.push(predicate);
      return this;
    },
  };
  const unchanged = applyRepositoryRelationFieldConditionFilter(
    db,
    () => 'postgres',
    parentMeta,
    'EmptyLines',
    targetMeta,
    'child_table',
    emptyQuery as any
  );
  expect(unchanged).toBe(emptyQuery);
  expect(emptyCalls).toEqual([]);

  const whereResults: any[] = [];
  const query = {
    where(predicate: any) {
      whereResults.push(predicate({ eb }));
      return this;
    },
  };

  applyRepositoryRelationFieldConditionFilter(
    db,
    () => 'postgres',
    parentMeta,
    'LineIds',
    targetMeta,
    'child_table',
    query as any
  );

  expect(whereResults).toEqual([{ lhs: 'Active', op: '=', rhs: true }]);
});

test('repository relation projection applies company filter from runtime context when model is company scoped', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(predicate: any) {
      calls.push({ predicate });
      return this;
    },
  };

  withContext({ enabledCompanyIds: ['company_a', 'company_b'] }, () => {
    applyRepositoryRelationCompanyFilter(
      {
        companyField: 'CompanyId',
        fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
      } as any,
      'child_table',
      query as any
    );
  });

  expect(calls.length).toBe(1);
  const predicate = calls[0].predicate;
  const eb = ((left: any, op: any, right: any) => ({ left, op, right })) as any;
  eb.or = (parts: any[]) => ({ parts });
  expect(predicate(eb)).toEqual({
    parts: [
      { left: 'child_table.CompanyId', op: 'in', right: ['company_a', 'company_b'] },
      { left: 'child_table.CompanyId', op: 'is', right: null },
    ],
  });
});

test('repository relation projection child select adds Id, scalar/select fields, hidden scale alias and nested relation alias', () => {
  class DemoModel {
    sqlDisplayName() {
      return {
        as(alias: string) {
          return { type: 'display', alias };
        },
      };
    }
  }

  const targetMeta = {
    type: DemoModel,
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Name', { column: { name: 'Name' } }],
      ['DisplayName', {}],
      ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
      ['AmountScale', { column: { name: 'AmountScale' } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const node: SelectionNode = {
    columns: new Set(['Name', 'DisplayName', 'Amount']),
    relations: new Map([
      [
        'Owner',
        {
          node: { columns: new Set(['Name']), relations: new Map() },
          fieldType: 'ManyToOne',
          relation: {} as any,
        },
      ],
    ]),
  };

  const db = {
    selectFrom() {
      return {};
    },
  };

  const sqb = {
    ref(value: string) {
      return {
        as(alias: string) {
          return { type: 'ref', value, alias };
        },
      };
    },
    fn: {},
  };

  const selections = buildRepositoryRelationChildSelect(db as any, () => 'postgres', targetMeta, 'demo_table', node, {
    buildRelationJsonSelect() {
      return {
        as(alias: string) {
          return { type: 'relation', alias };
        },
      };
    },
  })(sqb as any);

  expect(selections).toEqual([
    { type: 'ref', value: 'demo_table.Id', alias: 'Id' },
    { type: 'ref', value: 'demo_table.Name', alias: 'Name' },
    { type: 'display', alias: 'DisplayName' },
    { type: 'ref', value: 'demo_table.Amount', alias: 'Amount' },
    { type: 'ref', value: 'demo_table.AmountScale', alias: '$dec$Amount__scale' },
    { type: 'relation', alias: '$rel$Owner' },
  ]);
});

test('repository relation projection skips company filter for non-scoped/empty ids and keeps soft-delete disabled model unchanged', () => {
  const whereCalls: any[] = [];
  const query = {
    where(predicate: any) {
      whereCalls.push(predicate);
      return this;
    },
  };

  const noScope = applyRepositoryRelationCompanyFilter(
    {
      companyField: undefined,
      fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
    } as any,
    'child_table',
    query as any
  );
  expect(noScope).toBe(query);

  withContext({ enabledCompanyIds: [] }, () => {
    expect(() =>
      applyRepositoryRelationCompanyFilter(
        {
          companyField: 'CompanyId',
          fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
        } as any,
        'child_table',
        query as any
      )
    ).toThrow(/missing ctx\.enabledCompanyIds\/activeCompanyId/);
  });

  const soft = applyRepositoryRelationSoftDeleteFilter(
    {
      softDelete: false,
    } as any,
    'demo_table',
    query as any
  );
  expect(soft).toBe(query);
  expect(whereCalls.length).toBe(0);
});

test('repository relation projection supports scalar ActiveCompanyId fallback and unsupported relation type returns null', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(predicate: any) {
      calls.push({ predicate });
      return this;
    },
  };

  withContext({ ActiveCompanyId: 'company_one' }, () => {
    applyRepositoryRelationCompanyFilter(
      {
        companyField: 'CompanyId',
        fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
      } as any,
      'child_table',
      query as any
    );
  });

  expect(calls.length).toBe(1);
  const predicate = calls[0].predicate;
  const eb = ((left: any, op: any, right: any) => ({ left, op, right })) as any;
  eb.or = (parts: any[]) => ({ parts });
  expect(predicate(eb)).toEqual({
    parts: [
      { left: 'child_table.CompanyId', op: 'in', right: ['company_one'] },
      { left: 'child_table.CompanyId', op: 'is', right: null },
    ],
  });

  const unsupported = buildRelationJsonSelect(
    {
      selectFrom() {
        return {};
      },
    } as any,
    () => 'postgres',
    {
      tableName: () => 'demo_table',
    } as any,
    'UnknownRel',
    {
      fieldType: 'UnknownType' as any,
      relation: {} as any,
      node: { columns: new Set(), relations: new Map() } as any,
    } as any
  );
  expect(unsupported).toBeNull();
});

test('repository relation projection respects global filter switches', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
    ...(originalEnv || {}),
    CHOYSUM_SOFT_DELETE_ENABLED: 'false',
    CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: false,
  };

  const whereCalls: any[] = [];
  const query = {
    where(...args: any[]) {
      whereCalls.push(args);
      return this;
    },
  };

  try {
    const soft = applyRepositoryRelationSoftDeleteFilter({ softDelete: true } as any, 'demo_table', query as any);
    const company = applyRepositoryRelationCompanyFilter(
      {
        companyField: 'CompanyId',
        fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
      } as any,
      'demo_table',
      query as any
    );
    expect(soft).toBe(query);
    expect(company).toBe(query);
    expect(whereCalls.length).toBe(0);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository relation projection child select skips missing decimal-scale metadata and null child relation expr', () => {
  class DemoModel {}

  const targetMeta = {
    type: DemoModel,
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
    ]),
  } as any;

  const node: SelectionNode = {
    columns: new Set(['Amount']),
    relations: new Map([
      [
        'Owner',
        {
          node: { columns: new Set(['Id']), relations: new Map() },
          fieldType: 'ManyToOne',
          relation: {} as any,
        },
      ],
    ]),
  };

  const sqb = {
    ref(value: string) {
      return {
        as(alias: string) {
          return { type: 'ref', value, alias };
        },
      };
    },
    fn: {},
  };

  const selections = buildRepositoryRelationChildSelect({} as any, () => 'postgres', targetMeta, 'demo_table', node, {
    buildRelationJsonSelect() {
      return null;
    },
  })(sqb as any);

  expect(selections).toEqual([
    { type: 'ref', value: 'demo_table.Id', alias: 'Id' },
    { type: 'ref', value: 'demo_table.Amount', alias: 'Amount' },
  ]);
});

test('repository relation projection builds m2o/o2m/m2m subqueries with alias, join and orderBy chain', () => {
  class ParentModel {}
  class ChildModel {}
  class JoinModel {}

  const parentMeta = {
    type: ParentModel,
    tableName: () => 'parent_table',
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => ParentModel } }],
    ]),
  } as any;

  const childMeta = {
    type: ChildModel,
    tableName: () => 'child_table',
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Name', { column: { name: 'Name' } }],
      ['ParentId', { column: { name: 'ParentId' } }],
    ]),
  } as any;

  const joinMeta = {
    type: JoinModel,
    tableName: () => 'join_table',
    fields: new Map<string, any>([
      ['LeftId', { column: { name: 'LeftId' } }],
      ['RightId', { column: { name: 'RightId' } }],
    ]),
  } as any;

  const selects: string[] = [];
  const joins: string[] = [];
  const wheres: string[] = [];
  const orderBys: string[] = [];

  function makeBuilder(from: string) {
    selects.push(`from:${from}`);
    return {
      from,
      select(arg: any) {
        if (typeof arg === 'function') {
          const sqb = {
            ref(value: string) {
              return {
                as(alias: string) {
                  return { value, alias };
                },
              };
            },
            fn: {},
          } as any;
          arg(sqb);
        }
        return this;
      },
      whereRef(left: string, op: string, right: string) {
        wheres.push(`${left}${op}${right}`);
        return this;
      },
      where(left: any, op: any, right: any) {
        wheres.push(`${String(left)}${String(op)}${String(right)}`);
        return this;
      },
      innerJoin(table: string, left: string, right: string) {
        joins.push(`${table}:${left}=${right}`);
        return this;
      },
      orderBy(field: any, order: any) {
        orderBys.push(`${String(field)} ${String(order)}`);
        return this;
      },
    } as any;
  }

  const db = {
    selectFrom(table: string) {
      return makeBuilder(table);
    },
  };

  const node: SelectionNode = {
    columns: new Set(['Id']),
    relations: new Map(),
  };

  withFakeMetadata(
    new Map([
      [ParentModel, parentMeta],
      [ChildModel, childMeta],
      [JoinModel, joinMeta],
    ]),
    () => {
      const m2o = buildRelationJsonSelect(db as any, () => 'postgres', parentMeta, 'Owner', {
        fieldType: 'ManyToOne',
        relation: { targetModel: () => ParentModel },
        node,
      } as any);

      const o2m = buildRelationJsonSelect(db as any, () => 'postgres', parentMeta, 'Children', {
        fieldType: 'OneToMany',
        relation: { targetModel: () => ChildModel, inverseField: 'ParentId', orderBy: [{ field: 'Name', order: 'asc' }] },
        node,
      } as any);

      const m2m = buildRelationJsonSelect(db as any, () => 'postgres', parentMeta, 'Tags', {
        fieldType: 'ManyToMany',
        relation: { targetModel: () => ChildModel, joinModel: () => JoinModel, joinField: 'LeftId', inverseJoinField: 'RightId' },
        node,
      } as any);

      expect(m2o).toBeTruthy();
      expect(o2m).toBeTruthy();
      expect(m2m).toBeTruthy();
    }
  );

  expect(selects.some(s => s.includes('from:parent_table as parent_table__rel_Owner'))).toBe(true);
  expect(joins.some(j => j.includes('child_table.Id=join_table.RightId'))).toBe(true);
  expect(wheres.some(w => w.includes('parent_table__rel_Owner.Id=parent_table.Owner'))).toBe(true);
  expect(wheres.some(w => w.includes('child_table.ParentId=parent_table.Id'))).toBe(true);
  expect(wheres.some(w => w.includes('join_table.LeftId=parent_table.Id'))).toBe(true);
  expect(orderBys.some(v => v.includes('child_table.Name asc'))).toBe(true);
});

test('repository relation projection company filter requires CompanyId field and normalizes enabled ids', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(predicate: any) {
      calls.push({ predicate });
      return this;
    },
  };

  withContext({ EnabledCompanyIds: ['  company_a  ', 'company_a', null, '', 'company_b'] }, () => {
    expect(() =>
      applyRepositoryRelationCompanyFilter(
        {
          companyField: 'CompanyId',
          fields: new Map([['Name', { column: { name: 'Name' } }]]),
        } as any,
        'child_table',
        query as any
      )
    ).toThrow(/missing ownership field/);

    applyRepositoryRelationCompanyFilter(
      {
        companyField: 'CompanyId',
        fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
      } as any,
      'child_table',
      query as any
    );
  });

  expect(calls.length).toBe(1);
  const predicate = calls[0].predicate;
  const eb = ((left: any, op: any, right: any) => ({ left, op, right })) as any;
  eb.or = (parts: any[]) => ({ parts });
  expect(predicate(eb)).toEqual({
    parts: [
      { left: 'child_table.CompanyId', op: 'in', right: ['company_a', 'company_b'] },
      { left: 'child_table.CompanyId', op: 'is', right: null },
    ],
  });
});

test('repository relation projection child select uses decimal scale select expression alias', () => {
  class DemoModel {
    sqlAmountScale() {
      return {
        as(alias: string) {
          return { type: 'scale-select', alias };
        },
      };
    }
  }

  const targetMeta = {
    type: DemoModel,
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
      ['AmountScale', {}],
    ]),
    sqlComputeHandlers: new Map([['AmountScale', { field: 'AmountScale', method: 'sqlAmountScale' }]]),
  } as any;

  const node: SelectionNode = {
    columns: new Set(['Amount']),
    relations: new Map(),
  };

  const sqb = {
    ref(value: string) {
      return {
        as(alias: string) {
          return { type: 'ref', value, alias };
        },
      };
    },
    fn: {},
  };

  const selections = buildRepositoryRelationChildSelect({} as any, () => 'postgres', targetMeta, 'demo_table', node, {
    buildRelationJsonSelect() {
      return null;
    },
  })(sqb as any);

  expect(selections).toEqual([
    { type: 'ref', value: 'demo_table.Id', alias: 'Id' },
    { type: 'ref', value: 'demo_table.Amount', alias: 'Amount' },
    { type: 'scale-select', alias: '$dec$Amount__scale' },
  ]);
});

test('repository relation projection uses lowercase activeCompanyId and defaults soft-delete model flag to enabled', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(arg1: any, arg2?: any, arg3?: any) {
      calls.push({ arg1, arg2, arg3 });
      return this;
    },
  };

  withContext({ activeCompanyId: 'company_lowercase' }, () => {
    applyRepositoryRelationCompanyFilter(
      {
        companyField: 'CompanyId',
        fields: new Map([['CompanyId', { column: { name: 'CompanyId' } }]]),
      } as any,
      'child_table',
      query as any
    );
  });

  applyRepositoryRelationSoftDeleteFilter(
    {
      softDelete: undefined,
    } as any,
    'demo_table',
    query as any
  );

  expect(calls.length).toBe(2);
  const predicate = calls[0].arg1;
  const eb = ((left: any, op: any, right: any) => ({ left, op, right })) as any;
  eb.or = (parts: any[]) => ({ parts });
  expect(predicate(eb)).toEqual({
    parts: [
      { left: 'child_table.CompanyId', op: 'in', right: ['company_lowercase'] },
      { left: 'child_table.CompanyId', op: 'is', right: null },
    ],
  });
  expect(calls[1]).toEqual({ arg1: 'demo_table.DeletedAt', arg2: 'is', arg3: null });
});

test('repository relation projection executes path/select order resolvers for one2many and many2many', () => {
  class ParentModel {}
  class ChildModel {
    sqlDisplayName() {
      return { kind: 'display-expr' };
    }
  }
  class OwnerModel {}
  class JoinModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['Name', { column: { name: 'Name' } }],
    ]),
  } as any;

  const childMeta = {
    type: ChildModel,
    tableName: () => 'child_table',
    fields: new Map<string, any>([
      ['Id', { column: { name: 'Id' } }],
      ['ParentId', { column: { name: 'ParentId' } }],
      ['OwnerId', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel }, column: { name: 'OwnerId' } }],
      ['DisplayName', {}],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const parentMeta = {
    type: ParentModel,
    tableName: () => 'parent_table',
    fields: new Map<string, any>([['Id', { column: { name: 'Id' } }]]),
  } as any;

  const joinMeta = {
    type: JoinModel,
    tableName: () => 'join_table',
    fields: new Map<string, any>([
      ['LeftId', { column: { name: 'LeftId' } }],
      ['RightId', { column: { name: 'RightId' } }],
    ]),
  } as any;

  const callbackOrderHits: Array<'path' | 'select'> = [];

  function createSubQuery() {
    const createInnerBuilder = () => ({
      ref(value: string) {
        return {
          value,
          as(alias: string) {
            return { value, alias };
          },
        };
      },
      fn: {},
    });

    return {
      innerJoin() {
        return this;
      },
      select(arg: any) {
        if (typeof arg === 'function') arg(createInnerBuilder());
        return this;
      },
      whereRef() {
        return this;
      },
      where() {
        return this;
      },
      orderBy(field: any) {
        if (typeof field === 'function') {
          const resolved = field(createInnerBuilder());
          if (resolved?.kind === 'display-expr') callbackOrderHits.push('select');
          else callbackOrderHits.push('path');
        }
        return this;
      },
    } as any;
  }

  const db = {
    selectFrom() {
      return createSubQuery();
    },
  } as any;

  const node: SelectionNode = {
    columns: new Set(['Id']),
    relations: new Map(),
  };

  withFakeMetadata(
    new Map([
      [ParentModel, parentMeta],
      [ChildModel, childMeta],
      [OwnerModel, ownerMeta],
      [JoinModel, joinMeta],
    ]),
    () => {
      buildRelationJsonSelect(db, () => 'postgres', parentMeta, 'Children', {
        fieldType: 'OneToMany',
        relation: {
          targetModel: () => ChildModel,
          inverseField: 'ParentId',
          orderBy: [
            { field: 'OwnerId.Name', order: 'asc' },
            { field: 'DisplayName', order: 'desc' },
          ],
        },
        node,
      } as any);

      buildRelationJsonSelect(db, () => 'postgres', parentMeta, 'Tags', {
        fieldType: 'ManyToMany',
        relation: {
          targetModel: () => ChildModel,
          joinModel: () => JoinModel,
          joinField: 'LeftId',
          inverseJoinField: 'RightId',
          orderBy: [
            { field: 'OwnerId.Name', order: 'asc' },
            { field: 'DisplayName', order: 'desc' },
          ],
        },
        node,
      } as any);
    }
  );

  expect(callbackOrderHits).toEqual(['path', 'select', 'path', 'select']);
});
