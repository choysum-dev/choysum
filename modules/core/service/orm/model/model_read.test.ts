// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import {
  __aliasCandidatesForTest,
  __buildGroupConditionForTest,
  __fillTemporalGapsForLevelForTest,
  __formatGroupDisplayForTest,
  __pickAliasedValueForTest,
  __rangeFromGroupedValueForTest,
  __toArrayForTest,
  __toTreeResultForTest,
  ReadOperations,
} from './model_read';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { GrpcCode, ChoysumError } from '@/core/service/error';

@Model('ModelReadUser', { application: 'test' })
class ModelReadUser extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  DisplayName!: string;
}

@Model('ModelReadOrder', { application: 'test' })
class ModelReadOrder extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => ModelReadUser }, column: {} })
  OwnerId?: ModelReadUser;

  @Field({ type: 'datetime', column: {} })
  OrderedAt?: Date;

  @Field({ type: 'decimal', column: { precision: 16, scale: 2 } })
  Amount?: string;

  @Field({
    type: 'varchar',
    column: {
      compute: {
        expr: (self: any) => `${String(self.Name || '').trim()}-v`,
        deps: ['Name'],
        store: false,
      },
    },
  } as any)
  NameVirtual?: string;

  @Field({
    type: 'varchar',
    column: {
      compute: {
        expr: (self: any) => `${String(self.NameVirtual || '')}-d`,
        deps: ['NameVirtual'],
        store: false,
      },
    },
  } as any)
  NameVirtualDerived?: string;
}

@Model('ModelReadAttachmentOwner', { application: 'test' })
class ModelReadAttachmentOwner extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'image', column: { index: true } })
  Avatar?: string;

  @Field({ type: 'binary' })
  IdentityDoc?: string;
}

@Model('ModelReadBrokenOwner', { application: 'test' })
class ModelReadBrokenOwner extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => undefined as any }, column: {} })
  OwnerId?: any;
}

test('model read search hydrates binary/image fields from document.AttachmentBinding active bindings', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBindingResolver = (ReadOperations as any).resolveAttachmentBindingService;
  const bindingSearchCalls: any[] = [];

  try {
    const repo = {
      async search() {
        return [
          { Id: 'OWNER-1', Name: 'alice' },
          { Id: 'OWNER-2', Name: 'bob' },
        ];
      },
    };
    RepositoryFactory.getRepository = (() => repo) as any;

    (ReadOperations as any).resolveAttachmentBindingService = () => ({
      Search: async (condition: any, options: any) => {
        bindingSearchCalls.push({ condition, options });
        return [
          { Id: 'bind-avatar-1', OwnerRecordId: 'OWNER-1', FieldName: 'Avatar' },
          { Id: 'bind-doc-2', OwnerRecordId: 'OWNER-2', FieldName: 'IdentityDoc' },
        ];
      },
    });

    const rows = await ReadOperations.Search(ModelReadAttachmentOwner as any, [] as any);

    expect(bindingSearchCalls.length).toBe(1);
    const andClauses = bindingSearchCalls[0]?.condition?.And ?? [];
    const hasOwnerModelClause = andClauses.some((entry: any) => JSON.stringify(entry) === JSON.stringify(['OwnerModel', '=', 'test.ModelReadAttachmentOwner']));
    const hasStatusClause = andClauses.some((entry: any) => JSON.stringify(entry) === JSON.stringify(['Status', '=', 'active']));
    const hasFieldNameClause = andClauses.some((entry: any) => JSON.stringify(entry) === JSON.stringify(['FieldName', 'in', ['Avatar', 'IdentityDoc']]));
    expect(hasOwnerModelClause).toBe(true);
    expect(hasStatusClause).toBe(true);
    expect(hasFieldNameClause).toBe(true);

    expect(rows[0]).toMatchObject({
      Id: 'OWNER-1',
      Name: 'alice',
      Avatar: 'bind-avatar-1',
      IdentityDoc: null,
    });
    expect(rows[1]).toMatchObject({
      Id: 'OWNER-2',
      Name: 'bob',
      Avatar: null,
      IdentityDoc: 'bind-doc-2',
    });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    (ReadOperations as any).resolveAttachmentBindingService = originalBindingResolver;
  }
});

test('model read browse hydrates only explicitly requested attachment fields', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBindingResolver = (ReadOperations as any).resolveAttachmentBindingService;
  const bindingSearchCalls: any[] = [];

  try {
    const repo = {
      async search() {
        return [{ Id: 'OWNER-3', Name: 'charlie' }];
      },
    };
    RepositoryFactory.getRepository = (() => repo) as any;

    (ReadOperations as any).resolveAttachmentBindingService = () => ({
      Search: async (condition: any) => {
        bindingSearchCalls.push(condition);
        return [{ Id: 'bind-avatar-3', OwnerRecordId: 'OWNER-3', FieldName: 'Avatar' }];
      },
    });

    const row = await ReadOperations.Browse(ModelReadAttachmentOwner as any, 'OWNER-3', ['Avatar'] as any);

    expect(bindingSearchCalls.length).toBe(1);
    const andClauses = bindingSearchCalls[0]?.And ?? [];
    const hasFieldNameClause = andClauses.some((entry: any) => JSON.stringify(entry) === JSON.stringify(['FieldName', 'in', ['Avatar']]));
    expect(hasFieldNameClause).toBe(true);
    expect(row).toMatchObject({
      Id: 'OWNER-3',
      Avatar: 'bind-avatar-3',
    });
    expect((row as any).IdentityDoc).toBeUndefined();
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    (ReadOperations as any).resolveAttachmentBindingService = originalBindingResolver;
  }
});

test('model read browse uses soft-delete repository selection and raises NotFound with model domain', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];
  let withDeletedCalls = 0;

  try {
    const withDeletedRepo = {
      async search(condition: any, options?: any) {
        searchCalls.push({ condition, options });
        return [];
      },
    };

    const repo = {
      withDeleted() {
        withDeletedCalls++;
        return withDeletedRepo;
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    let thrown: any;
    try {
      await ReadOperations.Browse(ModelReadOrder as any, 'MISS-1', ['Name'] as any, { withDeleted: true } as any);
    } catch (error) {
      thrown = error;
    }

    expect(withDeletedCalls).toBe(1);
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.condition).toEqual(['Id', '=', 'MISS-1']);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Name'] });
    expect(thrown instanceof ChoysumError).toBe(true);
    expect(thrown?.domain).toBe('test');
    expect(thrown?.code).toBe('NotFound');
    expect(thrown?.grpcCode).toBe(GrpcCode.NotFound);
    expect(thrown?.message).toBe('test.ModelReadOrder not found');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read search and count normalize empty conditions and honor onlyDeleted or withDeleted repository scopes', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];
  const countCalls: any[] = [];
  let onlyDeletedCalls = 0;
  let withDeletedCalls = 0;

  try {
    const onlyDeletedRepo = {
      async search(condition: any, options?: any) {
        searchCalls.push({ condition, options });
        return [{ Id: 'OD-1', Name: 'archived' }];
      },
    };
    const withDeletedRepo = {
      async count(condition: any) {
        countCalls.push({ condition });
        return 9;
      },
    };
    const repo = {
      onlyDeleted() {
        onlyDeletedCalls++;
        return onlyDeletedRepo;
      },
      withDeleted() {
        withDeletedCalls++;
        return withDeletedRepo;
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const rows = await ReadOperations.Search(ModelReadOrder as any, undefined as any, { onlyDeleted: true, fields: ['Name'] as any } as any);
    const total = await ReadOperations.Count(ModelReadOrder as any, null as any, { withDeleted: true } as any);

    expect(onlyDeletedCalls).toBe(1);
    expect(withDeletedCalls).toBe(1);
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.condition).toEqual([]);
    expect(searchCalls[0]?.options).toEqual({ onlyDeleted: true, fields: ['Name'] });
    expect(countCalls.length).toBe(1);
    expect(countCalls[0]?.condition).toEqual(null);
    expect(rows).toEqual([{ Id: 'OD-1', Name: 'archived' }]);
    expect(total).toBe(9);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read search injects virtual compute fields before returning plain rows', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async search() {
        return [{ Id: 'S-1', Name: 'demo' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;
    const rows = await ReadOperations.Search(ModelReadOrder as any, ['Id', '=', 'S-1'] as any);
    expect(rows[0]?.NameVirtual).toBe('demo-v');
    expect(rows[0]?.NameVirtualDerived).toBe('demo-v-d');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read browse/search applies fields pruning for virtual compute injection', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async search() {
        return [{ Id: 'B-1', Name: 'demo' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const row1 = await ReadOperations.Browse(ModelReadOrder as any, 'B-1', ['Name'] as any);
    expect(row1.NameVirtual).toBeUndefined();
    expect(row1.NameVirtualDerived).toBeUndefined();

    const row2 = await ReadOperations.Browse(ModelReadOrder as any, 'B-1', ['Name', 'NameVirtualDerived'] as any);
    expect(row2.NameVirtual).toBe('demo-v');
    expect(row2.NameVirtualDerived).toBe('demo-v-d');

    const rows = await ReadOperations.Search(
      ModelReadOrder as any,
      ['Id', '=', 'B-1'] as any,
      {
        fields: ['Name', 'NameVirtual'] as any,
      } as any
    );
    expect(rows[0]?.NameVirtual).toBe('demo-v');
    expect(rows[0]?.NameVirtualDerived).toBeUndefined();
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read search skips virtual injection when requested fields is an empty array', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async search() {
        return [{ Id: 'E-1', Name: 'demo' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const rows = await ReadOperations.Search(
      ModelReadOrder as any,
      ['Id', '=', 'E-1'] as any,
      {
        fields: [] as any,
      } as any
    );

    expect(rows[0]?.NameVirtual).toBeUndefined();
    expect(rows[0]?.NameVirtualDerived).toBeUndefined();
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read search recomputes virtual fields and overwrites stale row values', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async search() {
        return [{ Id: 'C-1', Name: 'demo', NameVirtual: 'stale', NameVirtualDerived: 'stale-d' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;
    const rows = await ReadOperations.Search(ModelReadOrder as any, ['Id', '=', 'C-1'] as any);

    expect(rows[0]?.NameVirtual).toBe('demo-v');
    expect(rows[0]?.NameVirtualDerived).toBe('demo-v-d');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup returns totals row when groupby is empty', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const totalCalls: any[] = [];

  try {
    const repo = {
      async readTotals(options: any) {
        totalCalls.push(options);
        return {
          Amount__sum: '10.50',
          __count: '3',
        };
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [] as any,
      ['Active', '=', true] as any,
      {
        fields: ['Amount:sum'] as any,
        timezone: 'UTC',
      } as any
    );

    expect(totalCalls.length).toBe(1);
    expect(totalCalls[0]?.condition).toEqual(['Active', '=', true]);
    expect(totalCalls[0]?.timezone).toBe('UTC');
    expect(Array.isArray(totalCalls[0]?.fields)).toBe(true);
    expect(totalCalls[0]?.fields[0]?.field).toBe('Amount');
    expect(totalCalls[0]?.fields[0]?.agg).toBe('sum');
    expect(totalCalls[0]?.fields[0]?.alias).toBe('Amount__sum');

    expect(result.length).toBe(1);
    expect(result[0]?.depth).toBe(0);
    expect(result[0]?.keys).toEqual({});
    expect(result[0]?.labels).toEqual({});
    expect(result[0]?.metrics).toEqual({ Amount__sum: '10.50' });
    expect(result[0]?.count).toBe(3);
    expect(result[0]?.remainingGroupby).toEqual([]);
    expect(result[0]?.children).toEqual([]);
    expect(result[0]?.condition).toEqual(['Active', '=', true]);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup builds nested rows, fills temporal gaps, and resolves ManyToOne labels', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];
  const labelCalls: any[] = [];

  try {
    const orderRepo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        if (readGroupCalls.length === 1) {
          return [
            {
              OwnerId: 'USER-1',
              Amount__sum: '5.00',
              __count: '1',
            },
          ];
        }
        return [
          {
            OrderedAt__day: '2026-04-10T00:00:00Z',
            Amount__sum: '5.00',
            __count: '1',
          },
        ];
      },
    };

    const userRepo = {
      async search(condition: any, options?: any) {
        labelCalls.push({ condition, options });
        return [{ Id: 'USER-1', DisplayName: 'Alice' }];
      },
    };

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ModelReadUser) return userRepo;
      return orderRepo;
    }) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        'OwnerId',
        {
          field: 'OrderedAt',
          granularity: 'day',
          range: {
            start: '2026-04-10T00:00:00Z',
            end: '2026-04-11T23:59:59Z',
          },
        },
      ] as any,
      ['Active', '=', true] as any,
      {
        fields: ['Amount:sum'] as any,
        fillTemporalGaps: true,
        timezone: 'UTC',
      } as any
    );

    expect(readGroupCalls.length).toBe(2);
    expect(readGroupCalls[0]?.groupby).toBe('OwnerId');
    expect(readGroupCalls[0]?.condition).toEqual(['Active', '=', true]);
    expect(readGroupCalls[1]?.groupby?.field).toBe('OrderedAt');
    expect(readGroupCalls[1]?.groupby?.granularity).toBe('day');
    expect(labelCalls.length).toBe(1);
    expect(labelCalls[0]?.condition).toEqual(['Id', 'in', ['USER-1']]);
    expect(labelCalls[0]?.options).toEqual({ fields: ['Id', 'DisplayName'] });

    expect(result.length).toBe(1);
    expect(result[0]?.keys).toEqual({ OwnerId: 'USER-1' });
    expect(result[0]?.labels).toEqual({ OwnerId: 'Alice' });
    expect(result[0]?.metrics).toEqual({ Amount__sum: '5.00' });
    expect(result[0]?.count).toBe(1);
    expect(Array.isArray(result[0]?.children)).toBe(true);
    expect(result[0]?.children?.length).toBe(2);
    expect(result[0]?.children?.[0]?.labels).toEqual({ OrderedAt__day: '2026-04-10' });
    expect(result[0]?.children?.[0]?.count).toBe(1);
    expect(result[0]?.children?.[0]?.metrics).toEqual({ Amount__sum: '5.00' });
    expect(result[0]?.children?.[1]?.labels).toEqual({ OrderedAt__day: '2026-04-11' });
    expect(result[0]?.children?.[1]?.count).toBe(0);
    expect(result[0]?.children?.[1]?.metrics).toEqual({ Amount__sum: 0 });
    expect(result[0]?.children?.[1]?.remainingGroupby).toEqual([]);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroupCount returns one for empty groupby and delegates only the first level otherwise', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const countCalls: any[] = [];

  try {
    const repo = {
      async readGroupCount(options: any) {
        countCalls.push(options);
        return '7';
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const emptyCount = await ReadOperations.ReadGroupCount(ModelReadOrder as any, [] as any, ['Active', '=', true] as any, {} as any);
    const total = await ReadOperations.ReadGroupCount(
      ModelReadOrder as any,
      [
        [
          'OwnerId',
          {
            field: 'OrderedAt',
            granularity: 'month',
          },
        ],
        'Name',
      ] as any,
      ['Active', '=', true] as any,
      {
        fields: ['Amount:sum'] as any,
        having: [[['Amount__sum', '>', 1] as any], ['Name', '=', 'ignored'] as any] as any,
        timezone: 'UTC',
      } as any
    );

    expect(emptyCount).toBe(1);
    expect(total).toBe(7);
    expect(countCalls.length).toBe(1);
    expect(Array.isArray(countCalls[0]?.groupby)).toBe(true);
    expect(countCalls[0]?.groupby?.length).toBe(2);
    expect(countCalls[0]?.groupby?.[0]).toBe('OwnerId');
    expect(countCalls[0]?.groupby?.[1]?.field).toBe('OrderedAt');
    expect(countCalls[0]?.groupby?.[1]?.alias).toBe('OrderedAt__month');
    expect(countCalls[0]?.condition).toEqual(['Active', '=', true]);
    expect(Array.isArray(countCalls[0]?.fields)).toBe(true);
    expect(countCalls[0]?.fields?.[0]?.alias).toBe('Amount__sum');
    expect(countCalls[0]?.having).toEqual([['Amount__sum', '>', 1]]);
    expect(countCalls[0]?.timezone).toBe('UTC');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read browse falls back to core domain and Record label when metadata is empty', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;

  try {
    RepositoryFactory.getRepository = (() => ({ search: async () => [] })) as any;
    storage.getModelMetadata = () => ({
      application: '   ',
      fullModelName: '',
      modelName: '',
      name: '',
    });

    let thrown: any;
    try {
      await ReadOperations.Browse(ModelReadOrder as any, 'MISS-CORE');
    } catch (error) {
      thrown = error;
    }

    expect(thrown instanceof ChoysumError).toBe(true);
    expect(thrown?.domain).toBe('core');
    expect(thrown?.grpcCode).toBe(GrpcCode.NotFound);
    expect(thrown?.message).toBe('Record not found');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    storage.getModelMetadata = originalGetModelMetadata;
  }
});

test('model read readGroup rejects empty composite group level', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({ readGroup: async () => [] })) as any;

    let message = '';
    try {
      await ReadOperations.ReadGroup(ModelReadOrder as any, [[]] as any, [] as any, {} as any);
    } catch (error) {
      message = String((error as Error)?.message || error);
    }

    expect(message).toBe('Composite groupby level cannot be empty');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup routes per-level options and keeps single-bucket temporal level stable', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];
  const labelCalls: any[] = [];

  try {
    const orderRepo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        if (readGroupCalls.length === 1) {
          return [
            {
              OwnerId: 'USER-MISS',
              Amount__sum: '2.00',
              __count: '1',
            },
          ];
        }

        return [
          {
            OrderedAt__day: '2026-04-12T00:00:00Z',
            Amount__sum: '2.00',
            __count: '1',
          },
        ];
      },
    };

    const userRepo = {
      async search(condition: any, options?: any) {
        labelCalls.push({ condition, options });
        return [];
      },
    };

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ModelReadUser) return userRepo;
      return orderRepo;
    }) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        'OwnerId',
        {
          field: 'OrderedAt',
          granularity: 'day',
        },
      ] as any,
      ['Active', '=', true] as any,
      {
        fields: ['Amount:sum'] as any,
        having: [['Amount__sum', '>', 0] as any, ['Amount__sum', '>', 0] as any] as any,
        orderBy: [['Amount__sum', 'desc'] as any, ['OrderedAt__day', 'asc'] as any] as any,
        limit: { perLevel: [1, 2] } as any,
        offset: 5,
        fillTemporalGaps: true,
        timezone: 'UTC',
      } as any
    );

    expect(readGroupCalls.length).toBe(2);
    expect(readGroupCalls[0]?.limit).toBe(1);
    expect(readGroupCalls[0]?.offset).toBe(5);
    expect(readGroupCalls[0]?.having).toEqual(['Amount__sum', '>', 0]);
    expect(readGroupCalls[0]?.orderBy).toEqual(['Amount__sum', 'desc']);

    expect(readGroupCalls[1]?.limit).toBe(2);
    expect(readGroupCalls[1]?.offset).toBe(undefined);
    expect(readGroupCalls[1]?.having).toEqual(['Amount__sum', '>', 0]);
    expect(readGroupCalls[1]?.orderBy).toEqual(['OrderedAt__day', 'asc']);

    expect(labelCalls.length).toBe(1);
    expect(labelCalls[0]?.condition).toEqual(['Id', 'in', ['USER-MISS']]);
    expect(result[0]?.labels).toEqual({ OwnerId: 'USER-MISS' });
    expect(result[0]?.children?.length).toBe(1);
    expect(result[0]?.children?.[0]?.labels).toEqual({ OrderedAt__day: '2026-04-12' });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup skips many2one label query for non-string keys and readGroupCount normalizes NaN to zero', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];
  const countCalls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        return [
          {
            OwnerId: 123,
            Amount__sum: '1.00',
            __count: '1',
          },
        ];
      },
      async readGroupCount(options: any) {
        countCalls.push(options);
        return 'not-a-number';
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const grouped = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      ['OwnerId'] as any,
      [] as any,
      {
        fields: ['Amount:sum'] as any,
      } as any
    );

    const counted = await ReadOperations.ReadGroupCount(
      ModelReadOrder as any,
      ['OwnerId'] as any,
      [] as any,
      {
        having: {
          And: [['Amount__sum', '>', 1] as any],
        } as any,
      } as any
    );

    expect(readGroupCalls.length).toBe(1);
    expect(grouped[0]?.labels).toEqual({ OwnerId: '123' });
    expect(counted).toBe(0);
    expect(countCalls.length).toBe(1);
    expect(countCalls[0]?.having).toEqual({
      And: [['Amount__sum', '>', 1]],
    });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup empty groupby accepts pascalized aggregate/count aliases', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async readTotals() {
        return {
          AmountSum: '12.00',
          Count: '6',
        };
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [] as any,
      [] as any,
      {
        fields: ['Amount:sum'] as any,
      } as any
    );

    expect(result.length).toBe(1);
    expect(result[0]?.metrics).toEqual({ Amount__sum: '12.00' });
    expect(result[0]?.count).toBe(6);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup resolves collapsed aliases for grouped key/metric/count', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        return [
          {
            orderedAtDay: '2026-04-13T00:00:00Z',
            amountSum: '3.00',
            count: '2',
          },
        ];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        {
          field: 'OrderedAt',
          granularity: 'day',
        },
      ] as any,
      [] as any,
      {
        fields: ['Amount:sum'] as any,
        timezone: 'UTC',
      } as any
    );

    expect(readGroupCalls.length).toBe(1);
    expect(result.length).toBe(1);
    expect(result[0]?.keys).toEqual({ OrderedAt__day: '2026-04-13T00:00:00Z' });
    expect(result[0]?.labels).toEqual({ OrderedAt__day: '2026-04-13' });
    expect(result[0]?.metrics).toEqual({ Amount__sum: '3.00' });
    expect(result[0]?.count).toBe(2);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup formats temporal labels for year and week granularities', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        if (readGroupCalls.length === 1) {
          return [{ OrderedAt__year: '2026-01-01T00:00:00Z', __count: '1' }];
        }
        return [{ OrderedAt__week: '2026-01-05T00:00:00Z', __count: '1' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        { field: 'OrderedAt', granularity: 'year' },
        { field: 'OrderedAt', granularity: 'week' },
      ] as any,
      [] as any,
      {} as any
    );

    expect(readGroupCalls.length).toBe(2);
    expect(result[0]?.labels).toEqual({ OrderedAt__year: '2026' });
    expect(result[0]?.children?.[0]?.labels).toEqual({ OrderedAt__week: '2026-W02' });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup supports composite first-level groupby payload', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];
  const labelCalls: any[] = [];

  try {
    const orderRepo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        return [
          {
            OwnerId: 'USER-COMPOSITE',
            OrderedAt__day: '2026-04-14T00:00:00Z',
            __count: '1',
          },
        ];
      },
    };

    const userRepo = {
      async search(condition: any, options?: any) {
        labelCalls.push({ condition, options });
        return [{ Id: 'USER-COMPOSITE', DisplayName: 'Composite User' }];
      },
    };

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ModelReadUser) return userRepo;
      return orderRepo;
    }) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        [
          'OwnerId',
          {
            field: 'OrderedAt',
            granularity: 'day',
          },
        ],
      ] as any,
      ['Active', '=', true] as any,
      {
        timezone: 'UTC',
      } as any
    );

    expect(readGroupCalls.length).toBe(1);
    expect(Array.isArray(readGroupCalls[0]?.groupby)).toBe(true);
    expect(readGroupCalls[0]?.groupby?.length).toBe(2);
    expect(readGroupCalls[0]?.groupby?.[0]).toBe('OwnerId');
    expect(readGroupCalls[0]?.groupby?.[1]?.field).toBe('OrderedAt');
    expect(readGroupCalls[0]?.groupby?.[1]?.alias).toBe('OrderedAt__day');

    expect(labelCalls.length).toBe(1);
    expect(labelCalls[0]?.condition).toEqual(['Id', 'in', ['USER-COMPOSITE']]);

    expect(result.length).toBe(1);
    expect(result[0]?.keys).toEqual({ OwnerId: 'USER-COMPOSITE', OrderedAt__day: '2026-04-14T00:00:00Z' });
    expect(result[0]?.labels).toEqual({ OwnerId: 'Composite User', OrderedAt__day: '2026-04-14' });
    expect(result[0]?.count).toBe(1);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup fillTemporalGaps infers and sorts range from unsorted nodes', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async readGroup() {
        return [
          { OrderedAt__day: '2026-04-12T00:00:00Z', __count: '1' },
          { OrderedAt__day: '2026-04-10T00:00:00Z', __count: '1' },
        ];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        {
          field: 'OrderedAt',
          granularity: 'day',
        },
      ] as any,
      [] as any,
      {
        fillTemporalGaps: true,
        timezone: 'UTC',
      } as any
    );

    expect(result.map(row => row.labels?.OrderedAt__day)).toEqual(['2026-04-10', '2026-04-11', '2026-04-12']);
    expect(result.map(row => row.count)).toEqual([1, 0, 1]);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup formats temporal labels for quarter and month granularities', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const readGroupCalls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        readGroupCalls.push(options);
        if (readGroupCalls.length === 1) {
          return [{ OrderedAt__quarter: '2026-04-01T00:00:00Z', __count: '1' }];
        }
        return [{ OrderedAt__month: '2026-04-01T00:00:00Z', __count: '1' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(
      ModelReadOrder as any,
      [
        { field: 'OrderedAt', granularity: 'quarter' },
        { field: 'OrderedAt', granularity: 'month' },
      ] as any,
      [] as any,
      {} as any
    );

    expect(readGroupCalls.length).toBe(2);
    expect(result[0]?.labels).toEqual({ OrderedAt__quarter: '2026-Q2' });
    expect(result[0]?.children?.[0]?.labels).toEqual({ OrderedAt__month: '2026-04' });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroup day granularity accepts timezone-offset grouped values', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    const repo = {
      async readGroup() {
        return [{ OrderedAt__day: '2026-04-15T00:00:00+08:00', __count: '1' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(ModelReadOrder as any, [{ field: 'OrderedAt', granularity: 'day' }] as any, [] as any, {} as any);

    expect(result[0]?.labels).toEqual({ OrderedAt__day: '2026-04-15' });
    expect(result[0]?.count).toBe(1);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read readGroupCount throws for malformed non-empty groupby first level', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RepositoryFactory.getRepository = (() => ({ readGroupCount: async () => 1 })) as any;

    let message = '';
    try {
      await ReadOperations.ReadGroupCount(ModelReadOrder as any, [undefined as any] as any, [] as any, {} as any);
    } catch (error) {
      message = String((error as Error)?.message || error);
    }

    expect(message).toBe('ReadGroupCount requires non-empty groupby');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read helper alias/pick-value branches handle empty alias and private-count forms', () => {
  expect(__aliasCandidatesForTest('')).toEqual([]);

  const privateCountCandidates = __aliasCandidatesForTest('__count');
  expect(privateCountCandidates.includes('__count')).toBe(true);
  expect(privateCountCandidates.includes('count')).toBe(true);
  expect(privateCountCandidates.includes('Count')).toBe(true);

  const row = {
    orderedAtDay: '2026-05-01T00:00:00Z',
    Amount__sum: '7.00',
  } as any;
  expect(__pickAliasedValueForTest(row, 'OrderedAt__day')).toBe('2026-05-01T00:00:00Z');
  expect(__pickAliasedValueForTest(row, 'Amount__sum')).toBe('7.00');
  expect(__pickAliasedValueForTest({} as any, 'MissingAlias')).toBe(undefined);
});

test('model read helper temporal condition branches support non-time fallback and week range parsing', () => {
  const nonTime = __buildGroupConditionForTest({ field: 'Name', isTime: false } as any, 'A', 'UTC');
  expect(nonTime).toEqual(['Name', '=', 'A']);

  const weekRange = __rangeFromGroupedValueForTest('2026-05-14T00:00:00+08:00', 'week');
  expect(Boolean(weekRange)).toBe(true);
  expect(weekRange?.start instanceof Date).toBe(true);
  expect(weekRange?.end instanceof Date).toBe(true);

  const invalidRange = __rangeFromGroupedValueForTest(123 as any, 'day');
  expect(invalidRange).toBe(undefined);
});

test('model read helper temporal condition falls back to bucket coercion for Date input', () => {
  const condition = __buildGroupConditionForTest(
    {
      field: 'OrderedAt',
      alias: 'OrderedAt__day',
      isTime: true,
      granularity: 'day',
    } as any,
    new Date('2026-05-15T12:00:00Z'),
    'UTC'
  ) as any;

  expect(Array.isArray(condition?.And)).toBe(true);
  expect(condition.And[0][0]).toBe('OrderedAt');
  expect(condition.And[0][1]).toBe('>=');
  expect(String(condition.And[0][2])).toBe('2026-05-15T00:00:00.000Z');
  expect(condition.And[1][1]).toBe('<');
});

test('model read helper fill-gaps/tree-array-format branches cover non-time passthrough and null metrics', () => {
  const passthrough = __fillTemporalGapsForLevelForTest(
    [{ keyAliases: ['Name'], keyValues: ['A'], key: { Name: 'A' }, metrics: {}, count: 1, condition: [], children: [] }] as any,
    { field: 'Name', alias: 'Name', isTime: false } as any,
    [] as any,
    'UTC',
    [] as any
  );
  expect(passthrough.length).toBe(1);

  const gaps = __fillTemporalGapsForLevelForTest(
    [
      {
        keyAliases: ['OrderedAt__day'],
        keyValues: ['2026-05-01T00:00:00Z'],
        key: { OrderedAt__day: '2026-05-01T00:00:00Z' },
        metrics: { Name__avg: 9 },
        count: 1,
        condition: [],
        children: [],
      },
    ] as any,
    {
      field: 'OrderedAt',
      alias: 'OrderedAt__day',
      isTime: true,
      granularity: 'day',
      range: { start: '2026-05-01T00:00:00Z', end: '2026-05-02T23:59:59Z' },
    } as any,
    [{ agg: 'avg', alias: 'Name__avg' }] as any,
    'UTC',
    [] as any
  );
  expect(gaps.length).toBe(2);
  expect(gaps[1]?.metrics?.Name__avg).toBe(null);

  const arr = __toArrayForTest('x');
  expect(arr).toEqual(['x']);

  const tree = __toTreeResultForTest([{ keyAliases: [], key: {}, metrics: {}, count: 0, condition: [], children: [] }] as any);
  expect(tree[0]?.total).toBe(true);

  expect(__formatGroupDisplayForTest('Field', 'raw')).toBe('raw');
  expect(__formatGroupDisplayForTest('OrderedAt__day', 'not-a-date')).toBe('not-a-date');
});

test('model read helper formatGroupDisplay covers year quarter month week day and fallback', () => {
  expect(__formatGroupDisplayForTest('OrderedAt__year', '2026-05-14T00:00:00+08:00')).toBe('2026');
  expect(__formatGroupDisplayForTest('OrderedAt__quarter', '2026-05-14T00:00:00+08:00')).toBe('2026-Q2');
  expect(__formatGroupDisplayForTest('OrderedAt__month', '2026-05-14T00:00:00+08:00')).toBe('2026-05');
  expect(__formatGroupDisplayForTest('OrderedAt__week', '2026-05-14T00:00:00+08:00')).toBe('2026-W20');
  expect(__formatGroupDisplayForTest('OrderedAt__day', '2026-05-14T00:00:00+08:00')).toBe('2026-05-14');

  // When raw is a Date, exercise the moment(raw) branch.
  expect(__formatGroupDisplayForTest('OrderedAt__day', new Date('2026-05-14T00:00:00Z'))).toBe('2026-05-14');

  // Fallback returns the raw string value.
  expect(__formatGroupDisplayForTest('OrderedAt__week', '' as any)).toBe('');

  // Cover the nullish alias/raw branches.
  expect(__formatGroupDisplayForTest(undefined as any, undefined as any)).toBe('');
  expect(__formatGroupDisplayForTest('OrderedAt__day', null as any)).toBe('');
});

test('model read helper normalizeRequestedReadFields handles non-array and falsy entries', () => {
  const normalize = (ReadOperations as any).normalizeRequestedReadFields.bind(ReadOperations as any);

  expect(normalize(undefined)).toBeUndefined();
  const set = normalize([null, ' Name ', '', 0, 'Code'] as any) as Set<string>;
  expect(Array.from(set)).toEqual(['Name', 'Code']);
});

test('model read helper temporal condition fallback covers non-Date value conversion branch', () => {
  const condition = __buildGroupConditionForTest(
    {
      field: 'OrderedAt',
      alias: 'OrderedAt__day',
      isTime: true,
      granularity: 'day',
    } as any,
    1715644800000 as any,
    'UTC'
  ) as any;

  expect(Array.isArray(condition?.And)).toBe(true);
  expect(condition.And[0][0]).toBe('OrderedAt');
  expect(condition.And[0][1]).toBe('>=');
  expect(String(condition.And[0][2])).toBe('2024-05-14T00:00:00.000Z');
});

test('model read helper toTreeResult marks non-total when keyAliases empty but key is non-empty', () => {
  const tree = __toTreeResultForTest([
    {
      keyAliases: [],
      keyValues: ['A'],
      key: { Name: 'A' },
      metrics: {},
      count: 1,
      condition: [],
      children: [],
    } as any,
  ]);

  expect(tree[0]?.total).toBe(undefined);
});

test('model read default argument branches: count/search/group with omitted options', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const countCalls: any[] = [];
  const searchCalls: any[] = [];
  const groupCalls: any[] = [];

  try {
    const repo = {
      async count(condition: any) {
        countCalls.push(condition);
        return 2;
      },
      async search(condition: any) {
        searchCalls.push(condition);
        return [{ Id: 'S-1', Name: 'row' }];
      },
      async readGroup(options: any) {
        groupCalls.push(options);
        return [{ Name: null, __count: undefined }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const count = await ReadOperations.Count(ModelReadOrder as any);
    const rows = await ReadOperations.Search(ModelReadOrder as any, ['Id', '=', 'S-1'] as any);
    const grouped = await ReadOperations.ReadGroup(ModelReadOrder as any, ['Name'] as any);

    expect(count).toBe(2);
    expect(countCalls).toEqual([[]]);
    expect(searchCalls).toEqual([['Id', '=', 'S-1']]);
    expect(rows).toEqual([{ Id: 'S-1', Name: 'row', NameVirtual: 'row-v', NameVirtualDerived: 'row-v-d' }]);
    expect(groupCalls.length).toBe(1);
    expect(groupCalls[0]?.condition).toEqual([]);
    expect(grouped[0]?.count).toBe(0);
    expect(grouped[0]?.labels).toEqual({ Name: '' });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read many2one label path skips lookup when relation target model is missing', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const groupCalls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        groupCalls.push(options);
        return [{ OwnerId: 'BROKEN-U1', __count: '1' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    const result = await ReadOperations.ReadGroup(ModelReadBrokenOwner as any, ['OwnerId'] as any, [] as any);

    expect(groupCalls.length).toBe(1);
    expect(result[0]?.labels).toEqual({ OwnerId: 'BROKEN-U1' });
    expect(result[0]?.count).toBe(1);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read helper toTreeResult and toArray cover non-total and array input branches', () => {
  expect(__toArrayForTest(['x'])).toEqual(['x']);

  const tree = __toTreeResultForTest([
    {
      keyAliases: ['Name'],
      keyValues: ['A'],
      key: { Name: 'A' },
      metrics: {},
      count: 1,
      condition: [],
      children: undefined,
    } as any,
  ]);

  expect(tree[0]?.total).toBe(undefined);
  expect(Array.isArray(tree[0]?.children)).toBe(true);
  expect(tree[0]?.children?.length).toBe(0);
});

test('model read per-level limit branch handles numeric limit and non-numeric perLevel entries', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const calls: any[] = [];

  try {
    const repo = {
      async readGroup(options: any) {
        calls.push(options);
        if (calls.length === 1) {
          return [{ Name: 'A', __count: '1' }];
        }
        return [{ Name: 'A2', __count: '1' }];
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    await ReadOperations.ReadGroup(ModelReadOrder as any, ['Name', 'Name'] as any, [] as any, { limit: 2 } as any);
    await ReadOperations.ReadGroup(ModelReadOrder as any, ['Name', 'Name'] as any, [] as any, { limit: { perLevel: ['x', 3] } as any } as any);

    expect(calls[0]?.limit).toBe(2);
    expect(calls[1]?.limit).toBe(undefined);
    expect(calls[2]?.limit).toBe(undefined);
    expect(calls[3]?.limit).toBe(3);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read search normalizes nullish conditions and readGroupCount default options branch', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];
  const countCalls: any[] = [];

  try {
    const repo = {
      async search(condition: any) {
        searchCalls.push(condition);
        return [];
      },
      async readGroupCount(options: any) {
        countCalls.push(options);
        return 0;
      },
    };

    RepositoryFactory.getRepository = (() => repo) as any;

    await ReadOperations.Search(ModelReadOrder as any, undefined as any);
    await ReadOperations.Search(ModelReadOrder as any, null as any);
    await ReadOperations.ReadGroupCount(ModelReadOrder as any, ['Name'] as any);

    expect(searchCalls).toEqual([[], []]);
    expect(countCalls.length).toBe(1);
    expect(countCalls[0]?.condition).toEqual([]);
    expect(countCalls[0]?.timezone).toBe(undefined);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model read helper alias and temporal display fallback handles nullish and invalid string branches', () => {
  const weird = __aliasCandidatesForTest('__');
  expect(Array.isArray(weird)).toBe(true);
  expect(weird.includes('__')).toBe(true);

  expect(__pickAliasedValueForTest({ Count: 7 } as any, '__count')).toBe(7);
  expect(__rangeFromGroupedValueForTest('invalid-date', 'day')).toBe(undefined);
  expect(/^\d{4}-\d{2}-\d{2}$/.test(__formatGroupDisplayForTest('OrderedAt__day', undefined))).toBe(true);
});
