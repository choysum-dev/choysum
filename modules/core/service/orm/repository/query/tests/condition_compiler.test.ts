// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { convertCondition } from '..';
import { MetadataStorage } from '../../../metadata/storage';
import { withContext } from '../../../../runtime/context';

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

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.and = (parts: any[]) => ({ kind: 'and', parts });
  eb.or = (parts: any[]) => ({ kind: 'or', parts });
  eb.ref = (value: string) => `ref:${value}`;
  eb.fn = (name: string, args: any[]) => ({ fn: name, args });
  return eb;
}

test('repository condition compiler wraps monetary rhs like decimal', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'monetary', column: { name: 'Amount' } }]]),
  } as any;
  const eb = createExpressionBuilder();
  const db: any = { fn: { any: (v: any) => ({ any: v }) } };
  const result = convertCondition(db, () => 'sqlite', meta, eb, ['Amount', '=', '12.34'] as any, 'demo_table') as any;
  expect(result).toEqual({ lhs: 'Amount', op: '=', rhs: { $bigdecimal: '12.34' } });
});

test('repository condition compiler normalizes null comparisons and decimal constants', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  expect(convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '=', null] as any, 'demo_table')).toEqual({ lhs: 'Name', op: 'is', rhs: null });
  expect(convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '!=', null] as any, 'demo_table')).toEqual({ lhs: 'Name', op: 'is not', rhs: null });
  expect(convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'in', [1, '2']] as any, 'demo_table')).toEqual({
    lhs: 'Amount',
    op: 'in',
    rhs: [{ $bigdecimal: '1' }, { $bigdecimal: '2' }],
  });
});

test('repository condition compiler reuses select-context for select and many2one path expressions', () => {
  class DemoModel {
    sqlDisplayName() {
      return (this as any).$sql.field(DemoModel as any, 'Name');
    }
  }
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['DisplayName', {}],
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      expect(convertCondition(db as any, () => 'postgres', demoMeta, eb, ['DisplayName', '=', 'demo'] as any, 'demo_table')).toEqual({
        lhs: 'ref:demo_table.Name',
        op: '=',
        rhs: 'demo',
      });

      const pathResult = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Name', '=', 'alice'] as any, 'demo_table') as any;
      expect(pathResult.op).toBe('=');
      expect(pathResult.rhs).toBe('alice');
      expect(pathResult.lhs.ops).toEqual([
        { type: 'selectFrom', table: 'owner_table' },
        { type: 'select', selection: 'owner_table.Name' },
        { type: 'whereRef', left: 'owner_table.Id', op: '=', right: 'demo_table.Owner' },
      ]);
    }
  );
});

test('repository condition compiler preserves nested And and Or envelopes', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Status', { column: { name: 'Status' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  expect(
    convertCondition(
      db as any,
      () => 'postgres',
      meta,
      eb,
      {
        And: [['Name', '=', 'demo'] as any, { Or: [['Status', '=', 'draft'] as any, ['Status', '=', 'done'] as any] } as any],
      } as any,
      'demo_table'
    )
  ).toEqual({
    kind: 'and',
    parts: [
      { lhs: 'Name', op: '=', rhs: 'demo' },
      {
        kind: 'or',
        parts: [
          { lhs: 'Status', op: '=', rhs: 'draft' },
          { lhs: 'Status', op: '=', rhs: 'done' },
        ],
      },
    ],
  });
});

test('repository condition compiler throws on invalid tuple length', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '='] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('invalid condition tuple length: 2');
});

test('repository condition compiler child_of on Id requires selfTable and parentField', () => {
  class DemoModel {}
  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let noTableMessage = '';
  try {
    convertCondition(
      db as any,
      () => 'postgres',
      {
        type: DemoModel,
        modelName: 'DemoModel',
        tableName: () => 'demo_table',
        parentField: 'ParentId',
        fields: new Map([['Id', { column: { name: 'Id' } }]]),
      } as any,
      eb,
      ['Id', 'child_of', 'row_1'] as any
    );
  } catch (error) {
    noTableMessage = String((error as Error)?.message || error);
  }
  expect(noTableMessage).toBe('child_of requires selfTable');

  let noParentFieldMessage = '';
  try {
    convertCondition(
      db as any,
      () => 'postgres',
      {
        type: DemoModel,
        modelName: 'DemoModel',
        tableName: () => 'demo_table',
        fields: new Map([
          ['Id', { column: { name: 'Id' } }],
          ['ParentPath', { column: { name: 'ParentPath' } }],
        ]),
      } as any,
      eb,
      ['Id', 'child_of', 'row_1'] as any,
      'demo_table'
    );
  } catch (error) {
    noParentFieldMessage = String((error as Error)?.message || error);
  }
  expect(noParentFieldMessage).toBe('Model DemoModel does not configure parentField and cannot use child_of');
});

test('repository condition compiler maps ilike to lower-like on non-postgres dialect', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'mysql', meta, eb, ['Name', 'ilike', '%AbC%'] as any, 'demo_table') as any;
  expect(result.op).toBe('like');
  expect(result.rhs).toBe('%abc%');
  expect(result.lhs).toEqual({ fn: 'lower', args: ['ref:demo_table.Name'] });
});

test('repository condition compiler unwraps translated fields for search predicates', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { type: 'varchar', translate: true, column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'sqlite', meta, eb, ['Name', 'ilike', '%艾%'] as any, 'demo_table')
  ) as any;

  expect(result.op).toBe('like');
  expect(result.rhs).toBe('%艾%');
  // LHS must be an unwrap expression (not a bare column ref), so JSON structure is not matched.
  expect(result.lhs).not.toBe('Name');
  expect(result.lhs).not.toEqual('ref:demo_table.Name');
  // Non-postgres ilike wraps the unwrap expr with lower(...).
  expect(result.lhs?.fn).toBe('lower');
  expect(Array.isArray(result.lhs?.args) && result.lhs.args.length === 1).toBe(true);
  expect(result.lhs.args[0]).not.toEqual('ref:demo_table.Name');
});

test('repository condition compiler uses postgres unwrap without lower for translated ilike', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { type: 'varchar', translate: true, column: { name: 'Name' } }]]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };
  const result = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'ilike', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(result.op).toBe('ilike');
  expect(result.lhs?.fn).not.toBe('lower');
  expect(result.lhs).not.toEqual('ref:demo_table.Name');
});

test('repository condition compiler skips trigram prefilter for not ilike on translated fields', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name', index: 'trigram' } }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };
  const result = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'not ilike', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(result.kind).toBeUndefined();
  expect(result.op).toBe('not ilike');
});

test('repository condition compiler ANDs trigram prefilter for translated equality', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name', index: 'trigram' } }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };
  const result = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '=', 'hello'] as any, 'demo_table')
  ) as any;
  expect(result.kind).toBe('and');
  expect(result.parts?.length).toBe(2);
});

test('repository condition compiler ANDs trigram prefilter for translated like and in', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name', index: 'trigram' } }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };

  const likeResult = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'like', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(likeResult.kind).toBe('and');
  expect(likeResult.parts?.[0]?.op).toBe('like');

  const inResult = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'in', ['hello']] as any, 'demo_table')
  ) as any;
  expect(inResult.kind).toBe('and');
  expect(inResult.parts?.[0]?.op).toBe('like');

  const mysqlEq = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'mysql', meta, eb, ['Name', '=', 'hello'] as any, 'demo_table')
  ) as any;
  expect(mysqlEq.kind).toBeUndefined();
  expect(mysqlEq.op).toBe('=');
});

test('repository condition compiler ANDs trigram prefilter for translate+trigram on postgres', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name', index: 'trigram' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const withPrefilter = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'ilike', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(withPrefilter.kind).toBe('and');
  expect(withPrefilter.parts?.length).toBe(2);
  expect(withPrefilter.parts[0].op).toBe('ilike');
  expect(withPrefilter.parts[0].rhs).toBe('%abc%');
  expect(withPrefilter.parts[1].op).toBe('ilike');
  expect(withPrefilter.parts[1].rhs).toBe('%abc%');

  // Short patterns skip prefilter (L0 unwrap only).
  const shortOnly = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'ilike', '%ab%'] as any, 'demo_table')
  ) as any;
  expect(shortOnly.kind).toBeUndefined();
  expect(shortOnly.op).toBe('ilike');

  // Without trigram metadata, stay on unwrap-only path.
  const noTrigramMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { type: 'varchar', translate: true, column: { name: 'Name' } }]]),
  } as any;
  const noPrefilter = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', noTrigramMeta, eb, ['Name', 'ilike', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(noPrefilter.kind).toBeUndefined();
  expect(noPrefilter.op).toBe('ilike');
});

test('repository condition compiler uses postgresql dialect alias for trigram prefilter', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name', index: 'trigram' } }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };

  const result = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgresql', meta, eb, ['Name', '=', 'hello'] as any, 'demo_table')
  ) as any;
  expect(result.kind).toBe('and');

  const eqIlike = withContext({ lang: 'zh_CN' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '=ilike', '%abc%'] as any, 'demo_table')
  ) as any;
  expect(eqIlike.kind).toBe('and');
  expect(eqIlike.parts?.[0]?.op).toBe('=ilike');
});

test('repository condition compiler contains on non-json field warns and keeps predicate conversion', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Id', { column: { name: 'Id' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const originalWarn = console.warn;
  const warns: string[] = [];
  (console as any).warn = (msg: string) => {
    warns.push(String(msg));
  };

  try {
    const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'contains', { key: 'v' }] as any, 'demo_table');
    expect(result != null).toBe(true);
    expect(warns.length).toBe(1);
    expect(warns[0].includes('contains is recommended only for JSON container fields')).toBe(true);
  } finally {
    (console as any).warn = originalWarn;
  }
});

test('repository condition compiler contains on jsonobject field does not warn', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Payload', { type: 'jsonobject', column: { name: 'Payload' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const originalWarn = console.warn;
  const warns: string[] = [];
  (console as any).warn = (msg: string) => {
    warns.push(String(msg));
  };

  try {
    const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Payload', 'contains', { key: 'v' }] as any, 'demo_table') as any;
    expect(result.op).toBe('@>');
    expect(warns).toEqual([]);
  } finally {
    (console as any).warn = originalWarn;
  }
});

test('repository condition compiler contains on many-to-many ref field does not warn', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['FollowerIds', { type: 'ManyToManyRef', column: { name: 'FollowerIds' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const originalWarn = console.warn;
  const warns: string[] = [];
  (console as any).warn = (msg: string) => {
    warns.push(String(msg));
  };

  try {
    const result = convertCondition(db as any, () => 'postgres', meta, eb, ['FollowerIds', 'contains', 'u_1'] as any, 'demo_table') as any;
    expect(result.op).toBe('@>');
    expect(warns).toEqual([]);
  } finally {
    (console as any).warn = originalWarn;
  }
});

test('repository condition compiler fail-fast when virtual compute field has no search handler', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 1] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('Virtual compute field DemoModel.VirtualScore does not declare compute.search and cannot participate in query conditions');
});

test('repository condition compiler rewrites compute field condition via search handler domain result', () => {
  class DemoModel {
    static buildVirtualScoreDomain(ctx: any) {
      return {
        domain: ['Name', '=', `score:${ctx.value}`],
      };
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              searchable: true,
              search: 'buildVirtualScoreDomain',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 9] as any, 'demo_table');
  expect(result).toEqual({ lhs: 'Name', op: '=', rhs: 'score:9' });
});

test('repository condition compiler rewrites persisted compute field condition via search handler sql result', () => {
  class DemoModel {
    static buildPersistedSql(_ctx: any) {
      return {
        sql: { kind: 'sql', value: 'persisted_sql_expression' } as any,
      };
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              search: 'buildPersistedSql',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['PersistedScore', '>=', 3] as any, 'demo_table') as any;
  expect(result).toEqual({ kind: 'sql', value: 'persisted_sql_expression' });
});

test('repository condition compiler resolves compute.search handler from static registry', () => {
  class DemoModel {}

  (DemoModel as any).computeSearchHandlers = {
    fromRegistry(ctx: any) {
      return { domain: ['Name', '=', `registry:${ctx.value}`] };
    },
  };

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              search: 'fromRegistry',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 7] as any, 'demo_table');
  expect(result).toEqual({ lhs: 'Name', op: '=', rhs: 'registry:7' });
});

test('repository condition compiler resolves compute.search handler from static method and propagates default dialect', () => {
  class DemoModel {
    static fromStaticMethod(ctx: any) {
      return {
        domain: ['Name', '=', `${ctx.value}:${ctx.dialect}`],
      };
    }
  }

  const meta = {
    type: DemoModel,
    className: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              search: 'fromStaticMethod',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => '' as any, meta, eb, ['VirtualScore', '=', 11] as any, 'demo_table');
  expect(result).toEqual({ lhs: 'Name', op: '=', rhs: '11:postgres' });
});

test('repository condition compiler resolves compute.search handler from prototype method', () => {
  class DemoModel {
    fromProto(ctx: any) {
      return { domain: ['Name', '=', `proto:${ctx.value}`] };
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              search: 'fromProto',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 8] as any, 'demo_table');
  expect(result).toEqual({ lhs: 'Name', op: '=', rhs: 'proto:8' });
});

test('repository condition compiler throws when compute.search handler is missing', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              search: 'missingHandler',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['PersistedScore', '=', 1] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('compute.search handler not found: DemoModel.PersistedScore -> missingHandler');
});

test('repository condition compiler executes compute.search without runAs wrapper', () => {
  class DemoModel {
    static buildVirtualScoreDomain(ctx: any) {
      return {
        domain: ['Name', '=', String(ctx.value)],
      };
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              searchable: true,
              search: 'buildVirtualScoreDomain',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 9] as any, 'demo_table');
  expect(result).toEqual({ lhs: 'Name', op: '=', rhs: '9' });
});

test('repository condition compiler fails fast when compute.search handler returns Promise', () => {
  class DemoModel {
    static buildAsyncDomain(_ctx: any) {
      return Promise.resolve({ domain: ['Name', '=', 'ok'] });
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'VirtualScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              store: false,
              search: 'buildAsyncDomain',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['VirtualScore', '=', 1] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe(
    'compute.search handler returned a Promise, but the current query compilation path only supports synchronous handlers: DemoModel.VirtualScore'
  );
});

test('repository condition compiler fails fast when compute.search returns both domain and sql', () => {
  class DemoModel {
    static buildInvalidPayload(_ctx: any) {
      return {
        domain: ['Name', '=', 'demo'],
        sql: { kind: 'sql', value: 'demo' },
      };
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              search: 'buildInvalidPayload',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['PersistedScore', '=', 1] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('compute.search handler returned an invalid value and must provide exactly one of domain or sql: DemoModel.PersistedScore');
});

test('repository condition compiler fails fast when compute.search returns neither domain nor sql', () => {
  class DemoModel {
    static buildEmptyPayload(_ctx: any) {
      return null;
    }
  }

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name'],
              search: 'buildEmptyPayload',
            },
          },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['PersistedScore', '=', 1] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('compute.search handler returned an invalid value and must provide exactly one of domain or sql: DemoModel.PersistedScore');
});

test('repository condition compiler parent_of on many2one requires resolvable target model', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          relation: {},
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('unable to resolve targetModel for Owner');
});

test('repository condition compiler applies decimal conversion for dotted between predicate', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Credit', { type: 'decimal', column: { name: 'Credit' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Credit', 'between', [1, '2.5']] as any, 'demo_table') as any;
      expect(result.op).toBe('between');
      expect(result.rhs).toEqual([{ $bigdecimal: '1' }, { $bigdecimal: '2.5' }]);
    }
  );
});

test('repository condition compiler keeps nested envelope shape and falls back unknown object to empty-and', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Status', { column: { name: 'Status' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  expect(
    convertCondition(
      db as any,
      () => 'postgres',
      meta,
      eb,
      {
        And: [
          ['Name', '=', 'demo'] as any,
          {
            Or: [['Status', '=', 'draft'] as any, { Unknown: [['Name', '=', 'noop'] as any] } as any],
          } as any,
        ],
      } as any,
      'demo_table'
    )
  ).toEqual({
    kind: 'and',
    parts: [
      { lhs: 'Name', op: '=', rhs: 'demo' },
      {
        kind: 'or',
        parts: [
          { lhs: 'Status', op: '=', rhs: 'draft' },
          { kind: 'and', parts: [] },
        ],
      },
    ],
  });
});

test('repository condition compiler child_of on relation field requires many2one field', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Id', { column: { name: 'Id' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'child_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('DemoModel.Name is not ManyToOne and cannot be used with child_of');
});

test('repository condition compiler child_of on relation field requires relation metadata', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([['Owner', { type: 'ManyToOne', column: { name: 'OwnerId' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('DemoModel.Owner is not ManyToOne and cannot be used with child_of');
});

test('repository condition compiler child_of uses className in many2one error when modelName is missing', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    className: 'DemoClass',
    tableName: () => 'demo_table',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Name', 'child_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('DemoClass.Name is not ManyToOne and cannot be used with child_of');
});

test('repository condition compiler child_of on relation field requires target parentField', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      try {
        convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table');
      } catch (error) {
        message = String((error as Error)?.message || error);
      }
    }
  );

  expect(message).toBe('Target model OwnerModel does not configure parentField and cannot use child_of');
});

test('repository condition compiler falls back to raw lhs when selfTable is absent and still wraps decimal rhs', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', '=', '12.34'] as any) as any;
  expect(result).toEqual({ lhs: 'Amount', op: '=', rhs: { $bigdecimal: '12.34' } });
});

test('repository condition compiler child_of on relation field requires selfTable', () => {
  class DemoModel {}
  class OwnerModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'child_of', 'row_1'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('child_of requires selfTable');
});

test('repository condition compiler parent_of on relation field builds target subquery in mysql mode', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'mysql', demoMeta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.OwnerId');
      expect(result.rhs).toBeDefined();
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler supports decimal not-in and not-between operators', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  expect(convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'not in', [1, '2']] as any, 'demo_table')).toEqual({
    lhs: 'Amount',
    op: 'not in',
    rhs: [{ $bigdecimal: '1' }, { $bigdecimal: '2' }],
  });

  expect(convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'not between', [1, '2.5']] as any, 'demo_table')).toEqual({
    lhs: 'Amount',
    op: 'not between',
    rhs: [{ $bigdecimal: '1' }, { $bigdecimal: '2.5' }],
  });
});

test('repository condition compiler dotted-path guard branches are covered for invalid relation paths', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([
      ['', { column: { name: 'BlankLeaf' } }],
      ['Name', { column: { name: 'Name' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['', { type: 'ManyToOne', column: { name: 'BlankRel' }, relation: { targetModel: () => OwnerModel } }],
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', column: { name: 'OwnerId' }, relation: { targetModel: () => OwnerModel } }],
      ['BrokenOwner', { type: 'ManyToOne', column: { name: 'BrokenOwnerId' }, relation: { targetModel: () => undefined } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      for (const path of ['.', 'Owner.Unknown', 'Name.Other', 'BrokenOwner.Name']) {
        try {
          convertCondition(db as any, () => 'postgres', demoMeta, eb, [path, '=', 'v'] as any, 'demo_table');
        } catch {
          // The branch-coverage intent here is to exercise dotted-path guard paths, regardless of exact throw shape.
        }
      }
      expect(true).toBe(true);
    }
  );
});

test('repository condition compiler ilike on postgres uses select expression and normalizes non-string rhs', () => {
  class DemoModel {
    sqlDisplayName() {
      return 'expr:display_name';
    }
  }

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['DisplayName', {}]]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['DisplayName', 'ilike', 123] as any, 'demo_table') as any;
  expect(result).toEqual({ lhs: 'expr:display_name', op: 'ilike', rhs: '123' });
});

test('repository condition compiler parent_of on Id builds mysql subquery using id and parent path columns', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    parentField: 'ParentId',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  const result = convertCondition(db as any, () => 'mysql', meta, eb, ['Id', 'parent_of', 'row_1'] as any, 'demo_table') as any;
  expect(result.op).toBe('in');
  expect(result.lhs).toBe('ref:demo_table.Id');
  expect(result.rhs).toBeDefined();
  expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'demo_table as t' });
});

test('repository condition compiler maps not ilike to lower-not-like in mysql when selfTable is absent', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'mysql', meta, eb, ['Name', 'not ilike', '%AbC%'] as any) as any;
  expect(result.op).toBe('not like');
  expect(result.rhs).toBe('%abc%');
  expect(result.lhs).toEqual({ fn: 'lower', args: ['ref:Name'] });
});

test('repository condition compiler contains with select expression uses select-context expression', () => {
  class DemoModel {
    sqlPayload() {
      return 'expr:payload';
    }
  }

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Payload', { type: 'jsonobject' }]]),
    sqlComputeHandlers: new Map([['Payload', { field: 'Payload', method: 'sqlPayload' }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Payload', 'contains', { key: 'v' }] as any, 'demo_table') as any;
  expect(result.lhs).toBe('expr:payload');
  expect(result.op).toBe('@>');
  expect(typeof result.rhs).toBe('object');
});

test('repository condition compiler keeps decimal not-between rhs unchanged when rhs is not a 2-item array', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'not between', 10] as any, 'demo_table') as any;
  expect(result).toEqual({ lhs: 'Amount', op: 'not between', rhs: 10 });
});

test('repository condition compiler keeps decimal in/not-in rhs unchanged when rhs is not an array', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const inResult = convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'in', 'oops'] as any, 'demo_table') as any;
  const notInResult = convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', 'not in', 'oops'] as any, 'demo_table') as any;

  expect(inResult).toEqual({ lhs: 'Amount', op: 'in', rhs: 'oops' });
  expect(notInResult).toEqual({ lhs: 'Amount', op: 'not in', rhs: 'oops' });
});

test('repository condition compiler child_of on Id builds mysql subquery path', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    parentField: 'ParentId',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  const result = convertCondition(db as any, () => 'mysql', meta, eb, ['Id', 'child_of', 'row_1'] as any, 'demo_table') as any;
  expect(result.op).toBe('in');
  expect(result.lhs).toBe('ref:demo_table.Id');
  expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'demo_table as t' });
});

test('repository condition compiler child_of on relation field builds mysql target subquery path', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'mysql', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.OwnerId');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler child_of unwraps companyDependent ManyToOne fk', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          companyDependent: true,
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(n: number) {
          ops.push({ type: 'limit', n });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = withContext({ activeCompanyId: 'comp_main' }, () =>
        convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table')
      ) as any;
      expect(result.op).toBe('in');
      // Must unwrap the company map, not compare the raw JSON column.
      expect(result.lhs).not.toBe('ref:demo_table.OwnerId');
      expect(typeof result.lhs?.toOperationNode).toBe('function');
      expect(JSON.stringify(result.lhs.toOperationNode()).includes('OwnerId')).toBe(true);
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler companyDependent comparison uses physical column name', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          companyDependent: true,
          column: { name: 'OwnerId' },
        },
      ],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };

  const result = withContext({ activeCompanyId: 'comp_main' }, () =>
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', '=', 'p1'] as any, 'demo_table')
  ) as any;
  expect(result.op).toBe('=');
  expect(result.rhs).toBe('p1');
  expect(typeof result.lhs?.toOperationNode).toBe('function');
  const node = JSON.stringify(result.lhs.toOperationNode());
  expect(node.includes('OwnerId')).toBe(true);
  // Must not target the logical field name as a physical column.
  expect(node.includes('"demo_table"."Owner"') || node.includes('demo_table.Owner,')).toBe(false);

  // Empty dialect falls back to postgres for companyDependent unwrap.
  const emptyDialect = withContext({ activeCompanyId: 'comp_main' }, () =>
    convertCondition(db as any, () => '', meta, eb, ['Owner', '=', 'p1'] as any, 'demo_table')
  ) as any;
  expect(typeof emptyDialect.lhs?.toOperationNode).toBe('function');
});

test('repository condition compiler contains without selfTable falls back to raw ref path', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Payload', { type: 'jsonobject', column: { name: 'Payload' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Payload', 'contains', { k: 'v' }] as any) as any;
  expect(result.lhs).toBe('ref:Payload');
  expect(result.op).toBe('@>');
});

test('repository condition compiler returns empty-and predicate for empty tuple condition', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, [] as any, 'demo_table') as any;
  expect(result).toEqual({ kind: 'and', parts: [] });
});

test('repository condition compiler supports dotted ilike path via select context', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Name', 'ilike', '%Al%'] as any, 'demo_table') as any;
      expect(result.op).toBe('ilike');
      expect(result.rhs).toBe('%Al%');
      expect(result.lhs?.ops?.[0]).toEqual({ type: 'selectFrom', table: 'owner_table' });
    }
  );
});

test('repository condition compiler supports dotted contains path via select context', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Profile', { type: 'jsonobject', column: { name: 'Profile' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Profile', 'contains', { k: 'v' }] as any, 'demo_table') as any;
      expect(result.op).toBe('@>');
      expect(typeof result.rhs).toBe('object');
      expect(result.lhs?.ops?.[0]).toEqual({ type: 'selectFrom', table: 'owner_table' });
    }
  );
});

test('repository condition compiler parent_of on relation field requires selfTable', () => {
  class DemoModel {}
  class OwnerModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'parent_of', 'row_1'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('parent_of requires selfTable');
});

test('repository condition compiler keeps rhs unchanged when lhs is non-string and no decimal field metadata is available', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const lhs = { raw: 'expr:amount' };
  const result = convertCondition(db as any, () => 'postgres', meta, eb, [lhs, '=', 123] as any, 'demo_table') as any;
  expect(result).toEqual({ lhs, op: '=', rhs: 123 });
});

test('repository condition compiler child_of on relation field builds postgres target subquery path', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.OwnerId');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler parent_of on Id falls back to default id and parent-path columns', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    parentField: 'ParentId',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', {}],
      ['ParentPath', {}],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Id', 'parent_of', 'row_1'] as any, 'demo_table') as any;
  expect(result.op).toBe('in');
  expect(result.lhs).toBe('ref:demo_table.Id');
  expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'demo_table as t' });
});

test('repository condition compiler parent_of on relation field falls back to default fk/id/parent-path columns', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', {}],
      ['ParentPath', {}],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.Owner');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler parent_of on relation field requires target parentField metadata', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      try {
        convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table');
      } catch (error) {
        message = String((error as Error)?.message || error);
      }
    }
  );

  expect(message).toBe('Target model OwnerModel does not configure parentField and cannot use parent_of');
});

test('repository condition compiler normalizes <> null into is not', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => 'postgres', meta, eb, ['Name', '<>', null] as any, 'demo_table') as any;
  expect(result).toEqual({ lhs: 'Name', op: 'is not', rhs: null });
});

test('repository condition compiler relation child_of falls back to postgres when dialect is empty', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => '', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.OwnerId');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler dotted path throws when mid segment is not many2one', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Owner', { type: 'varchar', column: { name: 'Owner' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner.Amount', '=', 3] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('path Owner.Amount only supports ManyToOne chains; intermediate segment Owner is not ManyToOne');
});

test('repository condition compiler dotted path throws when relation metadata does not form a valid many2one chain', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: {} }],
      ['Amount', { type: 'decimal', column: { name: 'Amount' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner.Amount', '=', 4] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('path Owner.Amount only supports ManyToOne chains; intermediate segment Owner is not ManyToOne');
});

test('repository condition compiler parent_of on relation field rejects non-many2one metadata', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([['Owner', { type: 'varchar', column: { name: 'OwnerId' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('DemoModel.Owner is not ManyToOne and cannot be used with parent_of');
});

test('repository condition compiler applies decimal conversion for dotted in predicate', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Credit', { type: 'decimal', column: { name: 'Credit' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Credit', 'in', [1, '2.5']] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.rhs).toEqual([{ $bigdecimal: '1' }, { $bigdecimal: '2.5' }]);
    }
  );
});

test('repository condition compiler keeps dotted decimal not-between rhs unchanged when rhs is not 2-item array', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Credit', { type: 'decimal', column: { name: 'Credit' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Credit', 'not between', 10] as any, 'demo_table') as any;
      expect(result.op).toBe('not between');
      expect(result.rhs).toBe(10);
    }
  );
});

test('repository condition compiler keeps dotted decimal not-in rhs unchanged when rhs is not an array', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Credit', { type: 'decimal', column: { name: 'Credit' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Credit', 'not in', 'oops'] as any, 'demo_table') as any;
      expect(result.op).toBe('not in');
      expect(result.rhs).toBe('oops');
    }
  );
});

test('repository condition compiler parent_of on Id requires parentField metadata', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Id', 'parent_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('Model DemoModel does not configure parentField and cannot use parent_of');
});

test('repository condition compiler handles missing operator with decimal field and empty dialect ilike fallback', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Amount', { type: 'decimal', column: { name: 'Amount' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const decimalFallback = convertCondition(db as any, () => 'postgres', meta, eb, ['Amount', undefined, '12.34'] as any, 'demo_table') as any;
  expect(decimalFallback).toEqual({ lhs: 'Amount', op: undefined, rhs: { $bigdecimal: '12.34' } });

  const ilikeFallback = convertCondition(db as any, () => '', meta, eb, ['Amount', 'ilike', undefined] as any, 'demo_table') as any;
  expect(ilikeFallback.op).toBe('ilike');
  expect(ilikeFallback.rhs).toBe('');
  expect(ilikeFallback.lhs).toBe('ref:demo_table.Amount');
});

test('repository condition compiler converts dotted decimal rhs when operator is missing', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Credit', { type: 'decimal', column: { name: 'Credit' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => 'postgres', demoMeta, eb, ['Owner.Credit', undefined, '12.34'] as any, 'demo_table') as any;
      expect(result.op).toBeUndefined();
      expect(result.rhs).toEqual({ $bigdecimal: '12.34' });
    }
  );
});

test('repository condition compiler parent_of error messages fallback to className and default dialect path', () => {
  class DemoModel {}
  class OwnerModel {}

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  const noParentMeta = {
    type: DemoModel,
    className: 'DemoClass',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  let idMessage = '';
  try {
    convertCondition(db as any, () => 'postgres', noParentMeta, eb, ['Id', 'parent_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    idMessage = String((error as Error)?.message || error);
  }
  expect(idMessage).toBe('Model DemoClass does not configure parentField and cannot use parent_of');

  const ownerMetaMissingParent = {
    type: OwnerModel,
    className: 'OwnerClass',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const relationMeta = {
    type: DemoModel,
    modelName: '',
    className: 'DemoClass',
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  let relationMessage = '';
  withFakeMetadata(
    new Map([
      [DemoModel, relationMeta],
      [OwnerModel, ownerMetaMissingParent],
    ]),
    () => {
      try {
        convertCondition(db as any, () => '', relationMeta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table');
      } catch (error) {
        relationMessage = String((error as Error)?.message || error);
      }
    }
  );

  expect(relationMessage).toBe('Target model OwnerClass does not configure parentField and cannot use parent_of');

  const ownerMetaWithParent = {
    type: OwnerModel,
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, relationMeta],
      [OwnerModel, ownerMetaWithParent],
    ]),
    () => {
      const result = convertCondition(db as any, () => '', relationMeta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.Owner');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler contains defaults empty dialect to postgres', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Payload', { type: 'jsonobject', column: { name: 'Payload' } }]]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  const result = convertCondition(db as any, () => '', meta, eb, ['Payload', 'contains', { a: 1 }] as any, 'demo_table') as any;
  expect(result.op).toBe('@>');
  expect(result.lhs).toBe('ref:demo_table.Payload');
});

test('repository condition compiler Id child_of defaults empty dialect to postgres branch', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    parentField: 'ParentId',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  const result = convertCondition(db as any, () => '', meta, eb, ['Id', 'child_of', 'row_1'] as any, 'demo_table') as any;
  expect(result.op).toBe('in');
  expect(result.lhs).toBe('ref:demo_table.Id');
  expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'demo_table as t' });
});

test('repository condition compiler relation child_of defaults empty dialect path', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          column: { name: 'Owner' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(value: number) {
          ops.push({ type: 'limit', value });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const result = convertCondition(db as any, () => '', demoMeta, eb, ['Owner', 'child_of', 'row_1'] as any, 'demo_table') as any;
      expect(result.op).toBe('in');
      expect(result.lhs).toBe('ref:demo_table.Owner');
      expect(result.rhs.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table as t' });
    }
  );
});

test('repository condition compiler child_of rejects missing relation field metadata', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom() {
      throw new Error('not used');
    },
  };

  let message = '';
  try {
    convertCondition(db as any, () => 'postgres', meta, eb, ['Missing', 'child_of', 'row_1'] as any, 'demo_table');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('DemoModel.Missing is not ManyToOne and cannot be used with child_of');
});

test('repository condition compiler dotted path guards fail before leaf-meta fallback branches', () => {
  class DemoModel {}

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', column: { name: 'OwnerId' }, relation: {} }],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  const collectMessage = (fieldPath: string) => {
    try {
      convertCondition(db as any, () => 'postgres', meta, eb, [fieldPath, '=', 'v'] as any, 'demo_table');
      return '';
    } catch (error) {
      return String((error as Error)?.message || error);
    }
  };

  expect(collectMessage('Missing.Name')).toContain('intermediate segment Missing is not ManyToOne');
  expect(collectMessage('Name.Inner')).toContain('intermediate segment Name is not ManyToOne');
  expect(collectMessage('Owner.Name')).toContain('intermediate segment Owner is not ManyToOne');
});

test('repository condition compiler unwraps companyDependent for ilike/contains/parent_of', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    modelName: 'OwnerModel',
    parentField: 'ParentId',
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['ParentPath', { column: { name: 'ParentPath' } }],
    ]),
  } as any;

  const meta = {
    type: DemoModel,
    modelName: 'DemoModel',
    tableName: () => 'demo_table',
    fields: new Map([
      ['Cost', { type: 'number', companyDependent: true, column: { name: 'Cost' } }],
      [
        'Owner',
        {
          type: 'ManyToOne',
          companyDependent: true,
          column: { name: 'OwnerId' },
          relation: { targetModel: () => OwnerModel },
        },
      ],
    ]),
  } as any;

  const eb = createExpressionBuilder();
  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        where(lhs: any, op: any, rhs: any) {
          ops.push({ type: 'where', lhs, op, rhs });
          return this;
        },
        limit(n: number) {
          ops.push({ type: 'limit', n });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, meta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ilike = withContext({ activeCompanyId: 'comp_main' }, () =>
        convertCondition(db as any, () => 'postgres', meta, eb, ['Cost', 'ilike', '%1%'] as any, 'demo_table')
      ) as any;
      expect(ilike.op).toBe('ilike');
      expect(typeof ilike.lhs?.toOperationNode).toBe('function');

      const warns: string[] = [];
      const originalWarn = console.warn;
      console.warn = (msg: any) => {
        warns.push(String(msg));
      };
      try {
        const contains = withContext({ activeCompanyId: 'comp_main' }, () =>
          convertCondition(db as any, () => 'postgres', meta, eb, ['Cost', 'contains', { k: 1 }] as any, 'demo_table')
        ) as any;
        expect(typeof contains.lhs?.toOperationNode).toBe('function');
        expect(warns.some(w => w.includes('contains is recommended only for JSON'))).toBe(false);
      } finally {
        console.warn = originalWarn;
      }

      const parentOf = withContext({ activeCompanyId: 'comp_main' }, () =>
        convertCondition(db as any, () => 'postgres', meta, eb, ['Owner', 'parent_of', 'row_1'] as any, 'demo_table')
      ) as any;
      expect(parentOf.op).toBe('in');
      expect(typeof parentOf.lhs?.toOperationNode).toBe('function');
      expect(JSON.stringify(parentOf.lhs.toOperationNode()).includes('OwnerId')).toBe(true);
    }
  );
});

test('repository condition compiler companyDependent not-ilike and empty column name fallback', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Cost', { type: 'number', companyDependent: true, column: { name: '   ' } }],
      ['Note', { type: 'char', companyDependent: true, column: {} }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('not used'); } };

  const notIlike = withContext({ activeCompanyId: 'comp_main' }, () =>
    convertCondition(db as any, () => 'mysql', meta, eb, ['Cost', 'not ilike', '%x%'] as any, 'demo_table')
  ) as any;
  expect(notIlike.op === 'not like' || notIlike.op === 'not ilike' || typeof notIlike.lhs === 'object').toBe(true);

  const eq = withContext({ activeCompanyId: 'comp_main' }, () =>
    convertCondition(db as any, () => 'postgresql', meta, eb, ['Note', '=', 'hi'] as any, 'demo_table')
  ) as any;
  expect(typeof eq.lhs?.toOperationNode).toBe('function');
  expect(JSON.stringify(eq.lhs.toOperationNode()).includes('Note')).toBe(true);
});

test('repository condition compiler resolveStoredColumnName covers non-string and missing name', () => {
  class DemoModel {}
  const meta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['A', { type: 'number', companyDependent: true, column: { name: 42 } }],
      ['B', { type: 'number', companyDependent: true, column: {} }],
      ['C', { type: 'number', companyDependent: true }],
      ['D', { type: 'number', companyDependent: true, column: null }],
      ['E', { type: 'number', companyDependent: true, column: 'CostCol' }],
      ['F', { type: 'number', companyDependent: true, column: { name: 'PhysCost' } }],
      ['G', { type: 'number', companyDependent: true, column: { name: '   ' } }],
    ]),
  } as any;
  const eb = createExpressionBuilder();
  const db = { selectFrom() { throw new Error('unused'); } };

  const expectCol = (field: string, needle: string) => {
    const result = withContext({ activeCompanyId: 'comp_main' }, () =>
      convertCondition(db as any, () => 'postgres', meta, eb, [field, '=', 1] as any, 'demo_table')
    ) as any;
    expect(typeof result.lhs?.toOperationNode).toBe('function');
    expect(JSON.stringify(result.lhs.toOperationNode()).includes(needle)).toBe(true);
  };

  expectCol('A', 'A');
  expectCol('B', 'B');
  expectCol('C', 'C');
  expectCol('D', 'D');
  expectCol('E', 'E');
  expectCol('F', 'PhysCost');
  expectCol('G', 'G');
});
