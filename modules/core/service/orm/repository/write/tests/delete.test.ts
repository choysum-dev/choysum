// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryDeletePostWrite,
  executeRepositoryDelete,
  executeRepositoryDeleteRuntime,
  executeRepositoryHardDelete,
  prepareRepositoryDeleteQuery,
  prepareRepositorySoftDeleteWrite,
} from '../delete';
import { MetadataStorage } from '../../../metadata/storage';
import { withUser } from '../../../../runtime/context';

function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => Promise<T> | T): Promise<T> | T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  const restore = () => {
    storage.getModelMetadata = original;
  };

  try {
    const result = fn();
    if (result && typeof (result as Promise<T>).then === 'function') {
      return (result as Promise<T>).finally(restore);
    }
    restore();
    return result;
  } catch (error) {
    restore();
    throw error;
  }
}

test('repository delete soft-delete pre-write builds local soft-delete query from ids', async () => {
  const calls: Array<Record<string, any>> = [];

  const query = await prepareRepositorySoftDeleteWrite(
    {
      meta: { fields: new Map() } as any,
      db: {
        updateTable(table: string) {
          calls.push({ method: 'updateTable', table });
          return {
            set(input: Record<string, unknown>) {
              calls.push({ method: 'set', keys: Object.keys(input).sort() });
              return {
                where(callback: ({ eb }: any) => unknown) {
                  calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
                  return { kind: 'soft-delete-query' };
                },
              };
            },
          };
        },
      },
      table: 'demo_table',
      softField: 'DeletedAt',
      createRepository(meta) {
        calls.push({ method: 'createRepository', meta });
        return {
          softDeleteEnabled: () => true,
          delete: async () => [],
          hardDelete: async () => [],
          count: async () => 0,
          withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
          update: async () => [],
        } as any;
      },
      applySoftLayer(condition) {
        calls.push({ method: 'softLayer', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
    },
    ['row_1']
  );

  expect(query).toEqual({ kind: 'soft-delete-query' });
  expect(calls).toEqual([
    { method: 'updateTable', table: 'demo_table' },
    { method: 'set', keys: ['DeletedAt', 'UpdatedAt'] },
    { method: 'softLayer', condition: ['Id', 'in', ['row_1']] },
    { method: 'isEmpty', condition: ['Id', 'in', ['row_1']] },
    { method: 'convert', eb: 'EB', condition: ['Id', 'in', ['row_1']], selfTable: 'demo_table' },
    { method: 'where', result: { eb: 'EB', condition: ['Id', 'in', ['row_1']], selfTable: 'demo_table' } },
  ]);
});

test('repository delete soft-delete pre-write stamps DeletedUid/UpdatedUid when actor present', async () => {
  const calls: Array<Record<string, any>> = [];

  await withUser('U-DEL', async () => {
    await prepareRepositorySoftDeleteWrite(
      {
        meta: { fields: new Map() } as any,
        db: {
          updateTable() {
            return {
              set(input: Record<string, unknown>) {
                calls.push({ method: 'set', keys: Object.keys(input).sort(), input });
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'demo_table',
        softField: 'DeletedAt',
        createRepository() {
          return {
            softDeleteEnabled: () => true,
            delete: async () => [],
            hardDelete: async () => [],
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return true;
        },
        convertCondition(eb, condition, selfTable) {
          return { eb, condition, selfTable };
        },
      },
      ['row_1']
    );
  });

  expect(calls[0].keys).toEqual(['DeletedAt', 'DeletedUid', 'UpdatedAt', 'UpdatedUid']);
  expect(calls[0].input.DeletedUid).toBe('U-DEL');
  expect(calls[0].input.UpdatedUid).toBe('U-DEL');
});

test('repository delete query prepare builds conditioned delete query', async () => {
  const calls: Array<Record<string, any>> = [];

  const query = await prepareRepositoryDeleteQuery(
    {
      db: {
        deleteFrom(table: string) {
          calls.push({ method: 'deleteFrom', table });
          return {
            where(callback: ({ eb }: any) => unknown) {
              calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
              return { kind: 'delete-query' };
            },
          };
        },
      },
      table: 'demo_table',
      async applyRecordRuleToCondition(condition, op) {
        calls.push({ method: 'applyRecordRule', condition, op });
        return condition;
      },
      applyDefaultLayers(condition) {
        calls.push({ method: 'defaultLayers', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
    },
    ['Id', '=', '1'] as any
  );

  expect(query).toEqual({ kind: 'delete-query' });
  expect(calls).toEqual([
    { method: 'deleteFrom', table: 'demo_table' },
    { method: 'applyRecordRule', condition: ['Id', '=', '1'], op: 'delete' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
    { method: 'where', result: { eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' } },
  ]);
});

test('repository delete runtime executes query and returns rows', async () => {
  const calls: Array<Record<string, any>> = [];
  const rows = await executeRepositoryDeleteRuntime(
    {
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ numDeletedRows: 1 }] as any;
      },
      wrapSqlWriteError(error, mode) {
        calls.push({ method: 'wrap', error, mode });
        throw error;
      },
    },
    { kind: 'delete-query' }
  );

  expect(rows).toEqual([{ numDeletedRows: 1 }]);
  expect(calls).toEqual([{ method: 'execute', query: { kind: 'delete-query' } }]);
});

test('repository delete runtime delegates wrapped execution errors to sql write wrapper', async () => {
  const sqlError = new Error('boom');
  const wrappedError = new Error('wrapped');
  const calls: Array<Record<string, any>> = [];
  let actualError: unknown;

  try {
    await executeRepositoryDeleteRuntime(
      {
        async execute(query) {
          calls.push({ method: 'execute', query });
          throw sqlError;
        },
        wrapSqlWriteError(error, mode) {
          calls.push({ method: 'wrap', error, mode });
          throw wrappedError;
        },
      },
      { kind: 'delete-query' },
      'update'
    );
  } catch (error) {
    actualError = error;
  }

  expect(actualError).toBe(wrappedError);
  expect(calls).toEqual([
    { method: 'execute', query: { kind: 'delete-query' } },
    { method: 'wrap', error: sqlError, mode: 'update' },
  ]);
});

test('repository delete post-write invalidates cache when rows are returned', () => {
  const calls: Array<Record<string, any>> = [];
  const rows = applyRepositoryDeletePostWrite(
    {
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
    },
    [{ numDeletedRows: 1 }] as any
  );

  expect(rows).toEqual([{ numDeletedRows: 1 }]);
  expect(calls).toEqual([{ method: 'invalidate' }]);
});

test('repository delete post-write skips cache invalidation when no rows are returned', () => {
  const calls: Array<Record<string, any>> = [];
  const rows = applyRepositoryDeletePostWrite(
    {
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
    },
    [] as any
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository delete soft-delete pre-write skips where when layered condition is empty', async () => {
  const calls: Array<Record<string, any>> = [];

  const query = await prepareRepositorySoftDeleteWrite(
    {
      meta: { fields: new Map() } as any,
      db: {
        updateTable(table: string) {
          calls.push({ method: 'updateTable', table });
          return {
            set(input: Record<string, unknown>) {
              calls.push({ method: 'set', keys: Object.keys(input).sort() });
              return {
                where() {
                  calls.push({ method: 'where' });
                  return { kind: 'should-not-happen' };
                },
                kind: 'soft-delete-query-no-where',
              };
            },
          };
        },
      },
      table: 'demo_table',
      softField: 'DeletedAt',
      createRepository() {
        throw new Error('not used');
      },
      applySoftLayer() {
        return [] as any;
      },
      isEmptyCondition(condition) {
        return Array.isArray(condition) && condition.length === 0;
      },
      convertCondition() {
        throw new Error('convertCondition should not be called');
      },
    },
    ['row_1']
  );

  expect((query as any).kind).toBe('soft-delete-query-no-where');
  expect(calls.some(call => call.method === 'where')).toBe(false);
});

test('repository delete soft-delete pre-write blocks RESTRICT one2many cascade with referencing children', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          relation: {
            onDelete: 'RESTRICT',
          },
        },
      ],
    ]),
  } as any;

  const message = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    try {
      await prepareRepositorySoftDeleteWrite(
        {
          meta: parentMeta,
          db: {
            updateTable() {
              throw new Error('update should not run when restricted');
            },
          },
          table: 'parent_table',
          softField: 'DeletedAt',
          createRepository(meta) {
            expect(meta).toBe(childMeta);
            return {
              softDeleteEnabled: () => true,
              delete: async () => [],
              hardDelete: async () => [],
              count: async () => 2,
              withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
              update: async () => [],
            } as any;
          },
          applySoftLayer(condition) {
            return condition;
          },
          isEmptyCondition() {
            return false;
          },
          convertCondition() {
            return { kind: 'unused' };
          },
        },
        ['parent_1']
      );
      return '';
    } catch (error) {
      return String((error as Error)?.message || error);
    }
  });

  expect(message).toBe('Delete restricted by ChildModel: 2 referencing record(s)');
});

test('repository delete runtime with wrap mode returns rows when execution succeeds', async () => {
  const calls: Array<Record<string, any>> = [];
  const rows = await executeRepositoryDeleteRuntime(
    {
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ numDeletedRows: 3 }] as any;
      },
      wrapSqlWriteError(error, mode) {
        calls.push({ method: 'wrap', error, mode });
        throw error;
      },
    },
    { kind: 'delete-query' },
    'update'
  );

  expect(rows).toEqual([{ numDeletedRows: 3 }]);
  expect(calls).toEqual([{ method: 'execute', query: { kind: 'delete-query' } }]);
});

test('repository delete runtime without wrap mode returns empty array when execute resolves undefined', async () => {
  const rows = await executeRepositoryDeleteRuntime(
    {
      async execute() {
        return undefined as any;
      },
      wrapSqlWriteError(error) {
        throw error;
      },
    },
    { kind: 'delete-query' }
  );

  expect(rows).toEqual([]);
});

test('repository delete runtime with wrap mode returns empty array when execute resolves undefined', async () => {
  const rows = await executeRepositoryDeleteRuntime(
    {
      async execute() {
        return undefined as any;
      },
      wrapSqlWriteError(error) {
        throw error;
      },
    },
    { kind: 'delete-query' },
    'update'
  );

  expect(rows).toEqual([]);
});

test('repository delete post-write returns empty array when rows is undefined', () => {
  const calls: Array<string> = [];
  const rows = applyRepositoryDeletePostWrite(
    {
      invalidateCache() {
        calls.push('invalidate');
      },
    },
    undefined as any
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository hard delete short-circuits when resolved target ids are empty', async () => {
  const calls: Array<string> = [];

  const rows = await executeRepositoryHardDelete(
    {
      meta: { companyField: undefined } as any,
      locateIdsForCondition: async () => [],
      assertCompanyWriteAccessForCondition: async () => {
        throw new Error('should not run company guard');
      },
      assertRecordRuleAllTargetsAllowed: async () => {
        calls.push('record-rule');
      },
      db: {
        deleteFrom() {
          calls.push('deleteFrom');
          return {};
        },
      },
      table: 'demo_table',
      applyRecordRuleToCondition: async (c: any) => c,
      applyDefaultLayers: (c: any) => c,
      isEmptyCondition: () => false,
      convertCondition: () => ({ kind: 'cond' }),
      softField: 'DeletedAt',
      softDeleteEnabled: () => false,
      applySoftLayer: (c: any) => c,
      execute: async () => {
        calls.push('execute');
        return [] as any;
      },
      invalidateCache: () => {
        calls.push('invalidate');
      },
      wrapSqlWriteError: () => {
        throw new Error('should not wrap');
      },
      createRepository: () => {
        throw new Error('should not create child repo');
      },
    } as any,
    ['Id', '=', 'missing'] as any
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository delete soft-delete pre-write falls back to NO ACTION when child relation policy metadata is missing', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          relation: {},
        },
      ],
    ]),
  } as any;

  const query = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    return await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository() {
          return {
            softDeleteEnabled: () => true,
            delete: async () => [],
            hardDelete: async () => [],
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(query).toEqual({ kind: 'soft-delete-query' });
});

test('repository delete soft-delete pre-write falls back when inverse field metadata is missing', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map(),
  } as any;

  const query = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    return await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository() {
          return {
            softDeleteEnabled: () => true,
            delete: async () => [],
            hardDelete: async () => [],
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(query).toEqual({ kind: 'soft-delete-query' });
});

test('repository delete soft-delete pre-write blocks SET NULL when inverse field is NOT NULL', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          column: { notNull: true },
          relation: {
            onDelete: 'SET NULL',
          },
        },
      ],
    ]),
  } as any;

  const message = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    try {
      await prepareRepositorySoftDeleteWrite(
        {
          meta: parentMeta,
          db: {
            updateTable() {
              throw new Error('should not execute parent update');
            },
          },
          table: 'parent_table',
          softField: 'DeletedAt',
          createRepository() {
            return {
              softDeleteEnabled: () => true,
              delete: async () => [],
              hardDelete: async () => [],
              count: async () => 0,
              withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
              update: async () => [],
            } as any;
          },
          applySoftLayer(condition) {
            return condition;
          },
          isEmptyCondition() {
            return false;
          },
          convertCondition() {
            return { kind: 'unused' };
          },
        },
        ['parent_1']
      );
      return '';
    } catch (error) {
      return String((error as Error)?.message || error);
    }
  });

  expect(message).toBe('SET NULL blocked: ChildModel.ParentId is NOT NULL');
});

test('repository delete soft-delete pre-write cascades many2many join via hardDelete when join repo has no soft delete', async () => {
  class ParentModel {}
  class JoinModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Tags',
        {
          type: 'ManyToMany',
          relation: {
            joinModel: () => JoinModel,
            joinField: 'OwnerId',
          },
        },
      ],
    ]),
  } as any;

  const calls: Array<Record<string, any>> = [];
  await withFakeMetadata(new Map([[JoinModel, { modelName: 'JoinModel', fields: new Map() } as any]]), async () => {
    await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository(meta) {
          calls.push({ method: 'createRepository', modelName: meta.modelName });
          return {
            softDeleteEnabled: () => false,
            delete: async (condition: any) => {
              calls.push({ method: 'delete', condition });
              return [];
            },
            hardDelete: async (condition: any) => {
              calls.push({ method: 'hardDelete', condition });
              return [];
            },
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(calls).toEqual([
    { method: 'createRepository', modelName: 'JoinModel' },
    { method: 'hardDelete', condition: ['OwnerId', 'in', ['parent_1']] },
  ]);
});

test('repository delete runtime without wrap mode rethrows raw execute error', async () => {
  const raw = new Error('raw-delete-failure');
  let actual: unknown;

  try {
    await executeRepositoryDeleteRuntime(
      {
        async execute() {
          throw raw;
        },
        wrapSqlWriteError(error) {
          throw error;
        },
      },
      { kind: 'delete-query' }
    );
  } catch (error) {
    actual = error;
  }

  expect(actual).toBe(raw);
});

test('repository delete soft-delete pre-write cascades one2many via delete when child repo enables soft delete', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          relation: {
            onDelete: 'CASCADE',
          },
        },
      ],
    ]),
  } as any;

  const calls: Array<Record<string, any>> = [];
  await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository(meta) {
          calls.push({ method: 'createRepository', modelName: meta.modelName });
          return {
            softDeleteEnabled: () => true,
            delete: async (condition: any) => {
              calls.push({ method: 'delete', condition });
              return [];
            },
            hardDelete: async (condition: any) => {
              calls.push({ method: 'hardDelete', condition });
              return [];
            },
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(calls).toEqual([
    { method: 'createRepository', modelName: 'ChildModel' },
    { method: 'delete', condition: ['ParentId', 'in', ['parent_1']] },
  ]);
});

test('repository delete soft-delete pre-write allows NO ACTION when no referencing child exists', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          relation: {
            onDelete: 'NO ACTION',
          },
        },
      ],
    ]),
  } as any;

  const query = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    return await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository() {
          return {
            softDeleteEnabled: () => true,
            delete: async () => [],
            hardDelete: async () => [],
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(query).toEqual({ kind: 'soft-delete-query' });
});

test('repository delete soft-delete pre-write cascades many2many join via delete when join repo has soft delete', async () => {
  class ParentModel {}
  class JoinModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Tags',
        {
          type: 'ManyToMany',
          relation: {
            joinModel: () => JoinModel,
            joinField: 'OwnerId',
          },
        },
      ],
    ]),
  } as any;

  const calls: Array<Record<string, any>> = [];
  await withFakeMetadata(new Map([[JoinModel, { modelName: 'JoinModel', fields: new Map() } as any]]), async () => {
    await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository(meta) {
          calls.push({ method: 'createRepository', modelName: meta.modelName });
          return {
            softDeleteEnabled: () => true,
            delete: async (condition: any) => {
              calls.push({ method: 'delete', condition });
              return [];
            },
            hardDelete: async (condition: any) => {
              calls.push({ method: 'hardDelete', condition });
              return [];
            },
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(calls).toEqual([
    { method: 'createRepository', modelName: 'JoinModel' },
    { method: 'delete', condition: ['OwnerId', 'in', ['parent_1']] },
  ]);
});

test('repository delete soft-delete pre-write cascades one2many via hardDelete when child repo disables soft delete', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          relation: {
            onDelete: 'CASCADE',
          },
        },
      ],
    ]),
  } as any;

  const calls: Array<Record<string, any>> = [];
  await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository(meta) {
          calls.push({ method: 'createRepository', modelName: meta.modelName });
          return {
            softDeleteEnabled: () => false,
            delete: async (condition: any) => {
              calls.push({ method: 'delete', condition });
              return [];
            },
            hardDelete: async (condition: any) => {
              calls.push({ method: 'hardDelete', condition });
              return [];
            },
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async () => [],
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(calls).toEqual([
    { method: 'createRepository', modelName: 'ChildModel' },
    { method: 'hardDelete', condition: ['ParentId', 'in', ['parent_1']] },
  ]);
});

test('repository delete soft-delete pre-write allows SET NULL when inverse field is nullable', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = {
    modelName: 'ParentModel',
    fields: new Map([
      [
        'Children',
        {
          type: 'OneToMany',
          relation: {
            targetModel: () => ChildModel,
            inverseField: 'ParentId',
          },
        },
      ],
    ]),
  } as any;

  const childMeta = {
    modelName: 'ChildModel',
    fields: new Map([
      [
        'ParentId',
        {
          column: { notNull: false },
          relation: {
            onDelete: 'SET NULL',
          },
        },
      ],
    ]),
  } as any;

  const calls: Array<Record<string, any>> = [];
  const query = await withFakeMetadata(new Map([[ChildModel, childMeta]]), async () => {
    return await prepareRepositorySoftDeleteWrite(
      {
        meta: parentMeta,
        db: {
          updateTable() {
            return {
              set() {
                return {
                  where() {
                    return { kind: 'soft-delete-query' };
                  },
                };
              },
            };
          },
        },
        table: 'parent_table',
        softField: 'DeletedAt',
        createRepository(meta) {
          calls.push({ method: 'createRepository', modelName: meta.modelName });
          return {
            softDeleteEnabled: () => true,
            delete: async () => [],
            hardDelete: async () => [],
            count: async () => 0,
            withFieldRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
            update: async (patch: any, condition: any) => {
              calls.push({ method: 'update', condition, patch });
              return [];
            },
          } as any;
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition(_eb: any, condition: any) {
          return condition;
        },
      },
      ['parent_1']
    );
  });

  expect(query).toEqual({ kind: 'soft-delete-query' });
  expect(calls).toEqual([
    { method: 'createRepository', modelName: 'ChildModel' },
    { method: 'update', condition: ['ParentId', 'in', ['parent_1']], patch: { ParentId: null } },
  ]);
});
