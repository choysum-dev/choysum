// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RepositoryFactory } from '../../orm/repository/repository_factory';
import { MetadataStorage } from '../../orm/metadata/storage';
import BaseModel from '../../orm/model/model';
import { ComputeEngine } from './engine';
import Decimal from '@/core/utils/decimal';

async function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => Promise<T> | T): Promise<T> {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  try {
    return await fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

test('compute engine preview normalizes decimal fields and computes decimal output', async () => {
  class RootModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map([['Rate', { type: 'decimal', column: { scale: 1 } }]]),
  } as any;

  const rootMeta = {
    type: RootModel,
    fullModelName: 'test.RootModel',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Qty', { type: 'int', column: { name: 'Qty' } }],
      ['Price', { type: 'decimal', column: { scale: 2 } }],
      [
        'Total',
        {
          type: 'decimal',
          column: {
            scale: 1,
            compute: {
              expr: (self: any) => Number(self.Qty || 0) * 1.111,
              deps: ['Qty'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Total']),
      fastReverseDeps: new Map([['Qty', ['Total']]]),
      orderIndex: new Map([['Total', 0]]),
      computePathDeps: new Map(),
    },
  } as any;

  const entity: any = {
    Qty: 2,
    Price: '1.239',
    Owner: { Id: 'o1', Rate: '2.24' },
    Total: '0',
  };
  const changed = new Set<string>(['Qty']);

  await withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
    ]),
    async () => {
      await ComputeEngine.recompute(rootMeta, entity, changed, 'preview');
    }
  );

  expect(String(entity.Price)).toBe('1.24');
  expect(String(entity.Owner.Rate)).toBe('2.2');
  expect(String(entity.Total)).toBe('2.2');
  expect(changed.has('Total')).toBe(false);
});

test('compute engine persist prefetches m2o path and marks changed compute fields', async () => {
  class RootModel {}
  class OwnerModel {}

  const rootMeta = {
    type: RootModel,
    modelName: 'RootModel',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      [
        'Result',
        {
          type: 'decimal',
          column: {
            scale: 2,
            compute: {
              expr: (self: any) => self.Owner?.DiscountRate || 0,
              deps: ['Owner.DiscountRate'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Result']),
      fastReverseDeps: new Map([['Owner', ['Result']]]),
      orderIndex: new Map([['Result', 0]]),
      computePathDeps: new Map([
        [
          'Result',
          [
            {
              root: 'Owner',
              chain: ['DiscountRate'],
            },
          ],
        ],
      ]),
    },
  } as any;

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    OwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'o1', DiscountRate: '1.75' }];
      },
    } as any
  );

  const entity: any = {
    Owner: 'o1',
    Result: '0',
  };
  const changed = new Set<string>(['Owner']);

  await ComputeEngine.recompute(rootMeta, entity, changed, 'persist');

  expect(ownerSearchCalls).toBe(1);
  expect(entity.Owner).toEqual({ Id: 'o1', DiscountRate: '1.75' });
  expect(String(entity.Result)).toBe('1.75');
  expect(changed.has('Result')).toBe(true);
});

test('compute engine wraps compute errors with model and field name', async () => {
  const meta = {
    fullModelName: 'app.Demo',
    fields: new Map([
      ['Trigger', { type: 'int', column: { name: 'Trigger' } }],
      [
        'Broken',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => {
                throw new Error('boom-compute');
              },
              deps: ['Trigger'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Broken']),
      fastReverseDeps: new Map([['Trigger', ['Broken']]]),
      orderIndex: new Map([['Broken', 0]]),
      computePathDeps: new Map(),
    },
  } as any;

  let err = '';
  try {
    await ComputeEngine.recompute(meta, { Trigger: 1 }, new Set(['Trigger']), 'persist');
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('Compute execution failed: app.Demo.Broken')).toBe(true);
  expect(err.includes('boom-compute')).toBe(true);
});

test('compute engine persist keeps running when m2o prefetch fails', async () => {
  class RootModel {}
  class OwnerModel {}

  const meta = {
    type: RootModel,
    className: 'RootModel',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      [
        'Value',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Owner.Id'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Value']),
      fastReverseDeps: new Map([['Owner', ['Value']]]),
      orderIndex: new Map([['Value', 0]]),
      computePathDeps: new Map([
        [
          'Value',
          [
            {
              root: 'Owner',
              chain: ['Id'],
            },
          ],
        ],
      ]),
    },
  } as any;

  RepositoryFactory.setRepository(
    OwnerModel as any,
    {
      async search() {
        throw new Error('prefetch-failed');
      },
    } as any
  );

  const warns: string[] = [];
  const originalWarn = console.warn;
  console.warn = (...args: any[]) => {
    warns.push(args.map(x => String(x)).join(' '));
  };

  const entity: any = { Owner: 'o1', Value: 0 };
  try {
    await ComputeEngine.recompute(meta, entity, new Set(['Owner']), 'persist');
  } finally {
    console.warn = originalWarn;
  }

  expect(entity.Value).toBe(1);
  expect(warns.some(msg => msg.includes('persist multi-hop M2O prefetch failed'))).toBe(true);
});

test('compute engine persist only recomputes persisted compute fields', async () => {
  const entity: any = { Trigger: 1, PersistedValue: 0, VirtualValue: 0 };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        [
          'PersistedValue',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 11,
                deps: ['Trigger'],
              },
            },
          },
        ],
        [
          'VirtualValue',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 22,
                deps: ['Trigger'],
                store: false,
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['PersistedValue', 'VirtualValue']),
        persistedComputeFields: new Set(['PersistedValue']),
        virtualComputeFields: new Set(['VirtualValue']),
        fastReverseDeps: new Map([['Trigger', ['PersistedValue', 'VirtualValue']]]),
        orderIndex: new Map([
          ['PersistedValue', 0],
          ['VirtualValue', 1],
        ]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.PersistedValue).toBe(11);
  expect(entity.VirtualValue).toBe(0);
  expect(changed.has('PersistedValue')).toBe(true);
  expect(changed.has('VirtualValue')).toBe(false);
});

test('compute engine executes compute expression without runAs wrapper', async () => {
  const entity: any = { Trigger: 1, SecureValue: '' };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      modelName: 'DemoModel',
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        [
          'SecureValue',
          {
            type: 'varchar',
            column: {
              compute: {
                expr: () => 'ok',
                deps: ['Trigger'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['SecureValue']),
        persistedComputeFields: new Set(['SecureValue']),
        fastReverseDeps: new Map([['Trigger', ['SecureValue']]]),
        orderIndex: new Map([['SecureValue', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.SecureValue).toBe('ok');
});

test('compute engine returns early when graph or changed set is empty', async () => {
  const entity: any = { Value: 0 };

  await ComputeEngine.recompute(
    {
      fields: new Map(),
      computeGraph: undefined,
    } as any,
    entity,
    new Set(['Value']),
    'persist'
  );

  await ComputeEngine.recompute(
    {
      fields: new Map(),
      computeGraph: {
        computeFields: new Set(['Value']),
        fastReverseDeps: new Map([['Trigger', ['Value']]]),
        orderIndex: new Map([['Value', 0]]),
      },
    } as any,
    entity,
    new Set(),
    'persist'
  );

  expect(entity.Value).toBe(0);
});

test('compute engine skips recompute when no triggers are collected', async () => {
  const entity: any = { Trigger: 1, Result: 0 };

  await ComputeEngine.recompute(
    {
      fields: new Map([['Result', { column: { compute: { expr: () => 9 } } }]]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map(),
        orderIndex: new Map([['Result', 0]]),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'persist'
  );

  expect(entity.Result).toBe(0);
});

test('compute engine does not append unchanged computed field into persist changed set', async () => {
  const entity: any = { Trigger: 1, SameValue: 5 };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([['SameValue', { type: 'int', column: { compute: { expr: () => 5 } } }]]),
      computeGraph: {
        computeFields: new Set(['SameValue']),
        fastReverseDeps: new Map([['Trigger', ['SameValue']]]),
        orderIndex: new Map([['SameValue', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.SameValue).toBe(5);
  expect(changed.has('SameValue')).toBe(false);
});

test('compute engine uses decimal value equality to avoid false positive persist changes', async () => {
  const oldAmount = new Decimal('1.00');
  const entity: any = { Trigger: 1, Amount: oldAmount };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        [
          'Amount',
          {
            type: 'decimal',
            column: {
              scale: 2,
              compute: {
                expr: () => new Decimal('1'),
                deps: ['Trigger'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Amount']),
        fastReverseDeps: new Map([['Trigger', ['Amount']]]),
        orderIndex: new Map([['Amount', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.Amount).toBe(oldAmount);
  expect(changed.has('Amount')).toBe(false);
});

test('compute engine persist handles empty path segment chain and numeric relation id', async () => {
  class EmptySegRootModel {}
  class EmptySegOwnerModel {}

  const rootMeta = {
    type: EmptySegRootModel,
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => EmptySegOwnerModel } }],
      [
        'Echo',
        {
          type: 'varchar',
          column: {
            compute: {
              expr: (self: any) => String(self.Owner?.Id || ''),
              deps: ['Owner.'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Echo']),
      fastReverseDeps: new Map([['Owner', ['Echo']]]),
      orderIndex: new Map([['Echo', 0]]),
      computePathDeps: new Map([
        [
          'Echo',
          [
            {
              root: 'Owner',
              chain: [''],
            },
          ],
        ],
      ]),
    },
  } as any;

  const entity: any = { Owner: 123, Echo: '' };
  const changed = new Set<string>(['Owner']);

  await ComputeEngine.recompute(rootMeta, entity, changed, 'persist');

  expect(entity.Owner).toEqual({ Id: '123' });
  expect(entity.Echo).toBe('123');
  expect(changed.has('Echo')).toBe(true);
});

test('compute engine persist ignores invalid compute paths before prefetch', async () => {
  const entity: any = { Trigger: 1, Owner: 'o1', Result: 0 };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Owner', { type: 'varchar', column: {} }],
        [
          'Result',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 7,
                deps: ['Owner.Name'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map([['Trigger', ['Result']]]),
        orderIndex: new Map([['Result', 0]]),
        computePathDeps: new Map([
          [
            'Result',
            [
              { root: '', chain: ['Name'] },
              { root: 'Owner', chain: ['Name'] },
              { root: 'Owner', chain: [] },
            ],
          ],
        ]),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.Result).toBe(7);
  expect(changed.has('Result')).toBe(true);
});

test('compute engine preview handles primitive entity when trigger collection is empty', async () => {
  await ComputeEngine.recompute(
    {
      fields: new Map([['Amount', { type: 'decimal', column: { scaleField: 'Scale' } }]]),
      computeGraph: {
        computeFields: new Set(['Computed']),
        fastReverseDeps: new Map(),
        orderIndex: new Map([['Computed', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    1 as any,
    new Set(['Trigger']),
    'preview'
  );

  expect(true).toBe(true);
});

test('compute engine preview skips fields without compute spec after trigger collection', async () => {
  const entity: any = {
    Trigger: 1,
    AmountA: '1.23',
    ScaleA: 'x',
    AmountB: '2.34',
  };

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        ['AmountA', { type: 'decimal', column: { scaleField: 'ScaleA' } }],
        ['AmountB', { type: 'decimal', column: { scaleField: 'ScaleB' } }],
        ['NoCompute', { type: 'int', column: {} }],
      ]),
      computeGraph: {
        computeFields: new Set(['NoCompute']),
        fastReverseDeps: new Map([['Trigger', ['NoCompute']]]),
        orderIndex: new Map([['NoCompute', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'preview'
  );

  expect(entity.NoCompute).toBe(undefined);
});

test('compute engine walkAndLoad private helper handles unknown segment and missing next object rows', async () => {
  class WalkRoot {}
  class WalkNext {}

  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;

  RepositoryFactory.setRepository(
    WalkNext as any,
    {
      async search() {
        return [];
      },
    } as any
  );

  try {
    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === WalkRoot) {
        return {
          fields: new Map([['Link', { type: 'ManyToOne', relation: { targetModel: () => WalkNext } }]]),
        };
      }
      return { fields: new Map() };
    }) as any;

    const rootNode: any = { Link: 'NX-1' };
    await (ComputeEngine as any).walkAndLoad(WalkRoot as any, rootNode, ['Unknown'], new Map<string, any>());
    await (ComputeEngine as any).walkAndLoad(WalkRoot as any, rootNode, ['Link', 'Name'], new Map<string, any>());

    expect(rootNode.Link).toBe('NX-1');
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
  }
});

test('compute engine persist skips roots without target model or relation id', async () => {
  class MissingTargetRootModel {}
  class OwnerModel {}

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    OwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'o1', Name: 'owner-1' }];
      },
    } as any
  );

  const entity: any = {
    Trigger: 1,
    OwnerWithNoTarget: 'x1',
    OwnerWithoutId: {},
    Result: 0,
  };

  await ComputeEngine.recompute(
    {
      type: MissingTargetRootModel,
      fields: new Map([
        ['OwnerWithNoTarget', { type: 'ManyToOne', relation: {} }],
        ['OwnerWithoutId', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
        [
          'Result',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 3,
                deps: ['OwnerWithNoTarget.Name', 'OwnerWithoutId.Name'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map([['Trigger', ['Result']]]),
        orderIndex: new Map([['Result', 0]]),
        computePathDeps: new Map([
          [
            'Result',
            [
              { root: 'OwnerWithNoTarget', chain: ['Name'] },
              { root: 'OwnerWithoutId', chain: ['Name'] },
            ],
          ],
        ]),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'persist'
  );

  expect(ownerSearchCalls).toBe(0);
  expect(entity.Result).toBe(3);
});

test('compute engine persist continues when root prefetch row is missing', async () => {
  class MissingRootModel {}
  class MissingOwnerModel {}

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    MissingOwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [];
      },
    } as any
  );

  const entity: any = {
    Trigger: 1,
    Owner: 'missing-owner',
    Result: 0,
  };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      type: MissingRootModel,
      fields: new Map([
        ['Owner', { type: 'ManyToOne', relation: { targetModel: () => MissingOwnerModel } }],
        [
          'Result',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 11,
                deps: ['Owner.Name'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map([['Trigger', ['Result']]]),
        orderIndex: new Map([['Result', 0]]),
        computePathDeps: new Map([
          [
            'Result',
            [
              {
                root: 'Owner',
                chain: ['Name'],
              },
            ],
          ],
        ]),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(ownerSearchCalls).toBe(1);
  expect(entity.Owner).toBe('missing-owner');
  expect(entity.Result).toBe(11);
  expect(changed.has('Result')).toBe(true);
});

test('compute engine persist walkAndLoad stops when next relation target model is missing', async () => {
  class BrokenWalkRootModel {}
  class BrokenWalkOwnerModel {}

  const ownerMeta = {
    type: BrokenWalkOwnerModel,
    fields: new Map([
      ['Child', { type: 'ManyToOne', relation: {} }],
      ['Name', { type: 'varchar', column: {} }],
    ]),
  } as any;

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    BrokenWalkOwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'ow1', Child: 'cw1', Name: 'owner' }];
      },
    } as any
  );

  const rootMeta = {
    type: BrokenWalkRootModel,
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => BrokenWalkOwnerModel } }],
      [
        'Result',
        {
          type: 'varchar',
          column: {
            compute: {
              expr: (self: any) => String(self.Owner?.Name || ''),
              deps: ['Owner.Child.Name'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Result']),
      fastReverseDeps: new Map([['Owner', ['Result']]]),
      orderIndex: new Map([['Result', 0]]),
      computePathDeps: new Map([
        [
          'Result',
          [
            {
              root: 'Owner',
              chain: ['Child', 'Name'],
            },
          ],
        ],
      ]),
    },
  } as any;

  const entity: any = { Owner: 'ow1', Result: '' };
  await withFakeMetadata(new Map([[BrokenWalkOwnerModel, ownerMeta]]), async () => {
    await ComputeEngine.recompute(rootMeta, entity, new Set(['Owner']), 'persist');
  });

  expect(ownerSearchCalls).toBe(1);
  expect(entity.Owner).toEqual({ Id: 'ow1', Child: 'cw1', Name: 'owner' });
  expect(entity.Result).toBe('owner');
});

test('compute engine persist walkAndLoad reuses cache for repeated m2o hop and stops on scalar segment', async () => {
  class MultiHopRootModel {}
  class MultiHopOwnerModel {}
  class MultiHopChildModel {}

  const ownerMeta = {
    type: MultiHopOwnerModel,
    fields: new Map([
      ['Child', { type: 'ManyToOne', relation: { targetModel: () => MultiHopChildModel } }],
      ['Name', { type: 'varchar', column: {} }],
    ]),
  } as any;

  const childMeta = {
    type: MultiHopChildModel,
    fields: new Map([
      ['Code', { type: 'varchar', column: {} }],
      ['Label', { type: 'varchar', column: {} }],
    ]),
  } as any;

  let ownerSearchCalls = 0;
  let childSearchCalls = 0;

  RepositoryFactory.setRepository(
    MultiHopOwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'ow2', Child: 'cw2', Name: 'owner-2' }];
      },
    } as any
  );

  RepositoryFactory.setRepository(
    MultiHopChildModel as any,
    {
      async search() {
        childSearchCalls += 1;
        return [{ Id: 'cw2', Code: 'code-2', Label: 'label-2' }];
      },
    } as any
  );

  const rootMeta = {
    type: MultiHopRootModel,
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => MultiHopOwnerModel } }],
      [
        'Result',
        {
          type: 'varchar',
          column: {
            compute: {
              expr: (self: any) => `${String(self.Owner?.Child?.Code || '')}|${String(self.Owner?.Name || '')}`,
              deps: ['Owner.Child.Code', 'Owner.Child.Label', 'Owner.Name.Suffix'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Result']),
      fastReverseDeps: new Map([['Owner', ['Result']]]),
      orderIndex: new Map([['Result', 0]]),
      computePathDeps: new Map([
        [
          'Result',
          [
            { root: 'Owner', chain: ['Child', 'Code'] },
            { root: 'Owner', chain: ['Child', 'Label'] },
            { root: 'Owner', chain: ['Name', 'Suffix'] },
          ],
        ],
      ]),
    },
  } as any;

  const entity: any = { Owner: 'ow2', Result: '' };
  const changed = new Set<string>(['Owner']);

  await withFakeMetadata(
    new Map([
      [MultiHopOwnerModel, ownerMeta],
      [MultiHopChildModel, childMeta],
    ]),
    async () => {
      await ComputeEngine.recompute(rootMeta, entity, changed, 'persist');
    }
  );

  expect(ownerSearchCalls).toBe(1);
  expect(childSearchCalls).toBe(1);
  expect(entity.Owner).toEqual({
    Id: 'ow2',
    Child: { Id: 'cw2', Code: 'code-2', Label: 'label-2' },
    Name: 'owner-2',
  });
  expect(entity.Result).toBe('code-2|owner-2');
  expect(changed.has('Result')).toBe(true);
});

test('compute engine invokes targetModel resolver for path prefetch roots and keeps trigger traversal stable', async () => {
  class RootModel {}
  class OwnerModel {}

  let targetModelCalls = 0;
  let ownerSearchCalls = 0;

  const meta = {
    type: RootModel,
    fullModelName: 'test.RootModel',
    fields: new Map([
      [
        'Owner',
        {
          type: 'ManyToOne',
          relation: {
            targetModel: () => {
              targetModelCalls += 1;
              return OwnerModel;
            },
          },
        },
      ],
      [
        'Value',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 7,
              deps: ['Owner.Name'],
            },
          },
        },
      ],
      [
        'Mirror',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 8,
              deps: ['Value'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Value', 'Mirror']),
      fastReverseDeps: new Map([
        ['Owner', ['Value']],
        ['Value', ['Mirror']],
      ]),
      orderIndex: new Map([
        ['Value', 0],
        ['Mirror', 1],
      ]),
      computePathDeps: new Map([
        [
          'Value',
          [
            {
              root: 'Owner',
              chain: ['Name'],
            },
          ],
        ],
      ]),
    },
  } as any;

  RepositoryFactory.setRepository(
    OwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'o1', Name: 'Alice' }];
      },
    } as any
  );

  const entity: any = { Owner: 'o1', Value: 0, Mirror: 0 };
  const changed = new Set<string>(['Owner']);

  await ComputeEngine.recompute(meta, entity, changed, 'persist');

  expect(targetModelCalls).toBeGreaterThan(0);
  expect(ownerSearchCalls).toBe(1);
  expect(entity.Owner).toEqual({ Id: 'o1', Name: 'Alice' });
  expect(entity.Value).toBe(7);
  expect(entity.Mirror).toBe(8);
});

test('compute engine preview keeps Decimal values, supports valid scaleField and normalizes $bigdecimal envelope', async () => {
  class PreviewScaleModel {}

  const entity: any = {
    Scale: 3,
    AmountWithScale: '1.2349',
    // The isDecimal branch should keep the original value.
    AmountDecimalAlready: new Decimal('2.5'),
    // Exercise the left branch of new Decimal((val as any)?.$bigdecimal ?? val).
    AmountEnvelope: { $bigdecimal: '3.456' },
  };

  await ComputeEngine.recompute(
    {
      type: PreviewScaleModel,
      fields: new Map([
        ['Scale', { type: 'int', column: {} }],
        ['AmountWithScale', { type: 'decimal', column: { scaleField: 'Scale' } }],
        // Omit column to exercise (metaField?.column) ?? {}.
        ['AmountDecimalAlready', { type: 'decimal' }],
        ['AmountEnvelope', { type: 'decimal', column: { scale: 2 } }],
      ]),
      computeGraph: {
        computeFields: new Set(['Noop']),
        fastReverseDeps: new Map(),
        orderIndex: new Map([['Noop', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'preview'
  );

  expect(String(entity.AmountWithScale)).toBe('1.235');
  expect(String(entity.AmountDecimalAlready)).toBe('2.5');
  expect(String(entity.AmountEnvelope)).toBe('3.46');
});

test('compute engine error wrapping uses className fallback and string error branch', async () => {
  const meta = {
    className: 'OnlyClassNameModel',
    fields: new Map([
      ['Trigger', { type: 'int', column: {} }],
      [
        'Broken',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => {
                throw 'plain-string-error';
              },
              deps: ['Trigger'],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      computeFields: new Set(['Broken']),
      fastReverseDeps: new Map([['Trigger', ['Broken']]]),
      orderIndex: new Map([['Broken', 0]]),
      computePathDeps: new Map([
        [
          'Broken',
          [
            {
              root: 'Owner',
              // Empty chain to exercise if (!p.chain || !p.chain.length) continue.
            },
          ],
        ],
      ]),
    },
  } as any;

  let err = '';
  try {
    await ComputeEngine.recompute(meta, { Trigger: 1, Owner: { Id: 'o1' } }, new Set(['Trigger']), 'persist');
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('Compute execution failed: OnlyClassNameModel.Broken')).toBe(true);
  expect(err.includes('plain-string-error')).toBe(true);
});

test('compute engine walkAndLoad stops when many2one next id is missing and handles isLast branch', async () => {
  class WalkEdgeRoot {}
  class WalkEdgeChild {}

  const storage = MetadataStorage.instance as any;
  const originalGet = storage.getModelMetadata;

  try {
    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === WalkEdgeRoot) {
        return {
          fields: new Map([
            ['Link', { type: 'ManyToOne', relation: { targetModel: () => WalkEdgeChild } }],
            ['Name', { type: 'varchar', column: {} }],
          ]),
        };
      }
      if (ctor === WalkEdgeChild) {
        return {
          fields: new Map([
            ['Id', { type: 'varchar', column: {} }],
            ['Code', { type: 'varchar', column: {} }],
          ]),
        };
      }
      return originalGet.call(storage, ctor);
    }) as any;

    const nodeMissingId: any = { Link: {} };
    await (ComputeEngine as any).walkAndLoad(WalkEdgeRoot as any, nodeMissingId, ['Link', 'Code'], new Map<string, any>());
    expect(nodeMissingId.Link).toEqual({});

    const cached = new Map<string, any>([['WalkEdgeChild#C-1', { Id: 'C-1' }]]);
    const nodeLast: any = { Link: { Id: 'C-1' } };
    await (ComputeEngine as any).walkAndLoad(WalkEdgeRoot as any, nodeLast, ['Link'], cached);
    expect(nodeLast.Link).toEqual({ Id: 'C-1' });
  } finally {
    storage.getModelMetadata = originalGet;
  }
});

test('compute engine preview decimal fallback handles quantize failures and keeps raw fallback values', async () => {
  const throwingMeta: any = { type: 'decimal' };
  Object.defineProperty(throwingMeta, 'column', {
    get() {
      throw new Error('meta-column-getter-failed');
    },
  });

  const badDecimalValue: any = {
    toString() {
      throw new Error('bad-toString');
    },
  };

  const entity: any = {
    AmountFromEnvelope: { $bigdecimal: '1.23' },
    AmountFromPlain: '2.5',
    AmountFromBad: badDecimalValue,
    AmountInvalidFormat: 'not-a-decimal',
    AmountInvalidRound: '7.891',
  };

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['AmountFromEnvelope', throwingMeta],
        ['AmountFromPlain', throwingMeta],
        ['AmountFromBad', throwingMeta],
        ['AmountInvalidFormat', { type: 'decimal', column: { scale: 2 } }],
        ['AmountInvalidRound', { type: 'decimal', column: { scale: 2, round: 999 } }],
      ]),
      computeGraph: {
        computeFields: new Set(['Noop']),
        fastReverseDeps: new Map([['Trigger', ['Noop']]]),
        orderIndex: new Map([['Noop', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'preview'
  );

  expect(String(entity.AmountFromEnvelope)).toBe('1.23');
  expect(String(entity.AmountFromPlain)).toBe('2.5');
  expect(entity.AmountFromBad).toBe(badDecimalValue);
  expect(entity.AmountInvalidFormat).toBe('not-a-decimal');
  expect(entity.AmountInvalidRound).toBe('7.891');
});

test('compute engine persist resolves object root ids and ignores empty-root path deps', async () => {
  class RootObjectIdModel {}
  class OwnerObjectIdModel {}

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    OwnerObjectIdModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'owner-obj-1', Name: 'Owner Obj' }];
      },
    } as any
  );

  const entity: any = {
    Owner: { Id: 'owner-obj-1' },
    Result: '',
  };

  await ComputeEngine.recompute(
    {
      type: RootObjectIdModel,
      fields: new Map([
        ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerObjectIdModel } }],
        [
          'Result',
          {
            type: 'varchar',
            column: {
              compute: {
                expr: (self: any) => String(self.Owner?.Name || ''),
                deps: ['Owner.Name'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map([['Trigger', ['Result']]]),
        orderIndex: new Map([['Result', 0]]),
        computePathDeps: new Map([
          [
            'Result',
            [
              { root: '', chain: ['Name'] },
              { root: 'Owner', chain: ['Name'] },
            ],
          ],
        ]),
      },
    } as any,
    entity,
    new Set(['Trigger']),
    'persist'
  );

  expect(ownerSearchCalls).toBe(1);
  expect(entity.Owner).toEqual({ Id: 'owner-obj-1', Name: 'Owner Obj' });
  expect(entity.Result).toBe('Owner Obj');
});

test('compute engine error wrapping falls back to Unknown model label when metadata names are absent', async () => {
  let message = '';
  try {
    await ComputeEngine.recompute(
      {
        fields: new Map([
          ['Trigger', { type: 'int', column: {} }],
          [
            'Broken',
            {
              type: 'int',
              column: {
                compute: {
                  expr: () => {
                    throw new Error('boom-unknown-model');
                  },
                  deps: ['Trigger'],
                },
              },
            },
          ],
        ]),
        computeGraph: {
          computeFields: new Set(['Broken']),
          fastReverseDeps: new Map([['Trigger', ['Broken']]]),
          orderIndex: new Map([['Broken', 0]]),
          computePathDeps: new Map(),
        },
      } as any,
      { Trigger: 1, Broken: 0 },
      new Set(['Trigger']),
      'persist'
    );
  } catch (e) {
    message = String((e as Error).message || e);
  }

  expect(message.includes('Compute execution failed: Unknown.Broken')).toBe(true);
  expect(message.includes('boom-unknown-model')).toBe(true);
});

test('compute engine persist does not mark decimal changed when invalid payload string stays identical', async () => {
  const entity: any = { Trigger: 1, Amount: 'bad-decimal' };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        [
          'Amount',
          {
            type: 'decimal',
            column: {
              compute: {
                expr: () => 'bad-decimal',
                deps: ['Trigger'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Amount']),
        fastReverseDeps: new Map([['Trigger', ['Amount']]]),
        orderIndex: new Map([['Amount', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.Amount).toBe('bad-decimal');
  expect(changed.has('Amount')).toBe(false);
});

test('compute engine persist skips path deps when chain is undefined even if root relation exists', async () => {
  class UndefinedChainRootModel {}
  class UndefinedChainOwnerModel {}

  let ownerSearchCalls = 0;
  RepositoryFactory.setRepository(
    UndefinedChainOwnerModel as any,
    {
      async search() {
        ownerSearchCalls += 1;
        return [{ Id: 'owner_1', Name: 'Owner 1' }];
      },
    } as any
  );

  const entity: any = { Trigger: 1, Owner: 'owner_1', Result: 0 };
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      type: UndefinedChainRootModel,
      fields: new Map([
        ['Owner', { type: 'ManyToOne', relation: { targetModel: () => UndefinedChainOwnerModel } }],
        [
          'Result',
          {
            type: 'int',
            column: {
              compute: {
                expr: () => 77,
                deps: ['Owner.Name'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Result']),
        fastReverseDeps: new Map([['Trigger', ['Result']]]),
        orderIndex: new Map([['Result', 0]]),
        computePathDeps: new Map([
          [
            'Result',
            [
              {
                root: 'Owner',
              } as any,
            ],
          ],
        ]),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(ownerSearchCalls).toBe(0);
  expect(entity.Result).toBe(77);
  expect(changed.has('Result')).toBe(true);
});

test('compute engine handles falsy primitive entity with decimal scaleField compute safely', async () => {
  const changed = new Set<string>(['Trigger']);

  await ComputeEngine.recompute(
    {
      fields: new Map([
        ['Trigger', { type: 'int', column: {} }],
        [
          'Amount',
          {
            type: 'decimal',
            column: {
              scaleField: 'Scale',
              compute: {
                expr: () => undefined,
                deps: ['Trigger'],
              },
            },
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Amount']),
        fastReverseDeps: new Map([['Trigger', ['Amount']]]),
        orderIndex: new Map([['Amount', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    0 as any,
    changed,
    'persist'
  );

  expect(changed.has('Amount')).toBe(false);
});

test('compute engine executes @Compute handler method and writes back through this', async () => {
  class HandlerComputeModel extends BaseModel {
    Qty?: number;
    Total?: number;

    computeTotal() {
      this.Total = Number((this as any).Qty || 0) + 5;
    }
  }

  const entity: any = { Qty: 3, Total: 0 };
  const changed = new Set<string>(['Qty']);

  await ComputeEngine.recompute(
    {
      type: HandlerComputeModel,
      fields: new Map([
        ['Qty', { type: 'int', column: {} }],
        ['Total', { type: 'int', column: {} }],
      ]),
      computeHandlers: new Map([
        [
          'Total',
          {
            field: 'Total',
            method: 'computeTotal',
            deps: ['Qty'],
            store: true,
          },
        ],
      ]),
      computeGraph: {
        computeFields: new Set(['Total']),
        persistedComputeFields: new Set(['Total']),
        fastReverseDeps: new Map([['Qty', ['Total']]]),
        orderIndex: new Map([['Total', 0]]),
        computePathDeps: new Map(),
      },
    } as any,
    entity,
    changed,
    'persist'
  );

  expect(entity.Total).toBe(8);
  expect(changed.has('Total')).toBe(true);
});

test('compute engine injectVirtualForRead does not execute @SqlCompute handler in runtime read stage', () => {
  class SqlComputeReadModel extends BaseModel {
    Name?: string;
    override DisplayName!: string;

    sqlDisplayName() {
      throw new Error('sql compute should not run in runtime read stage');
    }
  }

  const meta = {
    type: SqlComputeReadModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: { size: 64 } }],
      ['DisplayName', { type: 'varchar', column: { size: 64 } }],
    ]),
    sqlComputeHandlers: new Map([
      [
        'DisplayName',
        {
          field: 'DisplayName',
          method: 'sqlDisplayName',
          deps: ['Name'],
        },
      ],
    ]),
    computeGraph: {
      order: ['DisplayName'],
      virtualComputeFields: new Set(['DisplayName']),
      parsedDeps: new Map([['DisplayName', [{ kind: 'scalar', field: 'Name' }]]]),
    },
  } as any;

  const entity: any = { Name: 'Alice' };
  ComputeEngine.injectVirtualForRead(meta, entity);

  expect(entity.DisplayName).toBeUndefined();
});

test('compute engine injectVirtualForRead keeps prefilled @SqlCompute value and skips runtime sql bridge execution', () => {
  class SqlComputePrefilledModel extends BaseModel {
    override DisplayName!: string;

    sqlDisplayName() {
      const sql = this.$sql as any;
      return sql.col('demo_table', 'DisplayName');
    }
  }

  const meta = {
    type: SqlComputePrefilledModel,
    fields: new Map([['DisplayName', { type: 'varchar', column: { size: 64 } }]]),
    sqlComputeHandlers: new Map([
      [
        'DisplayName',
        {
          field: 'DisplayName',
          method: 'sqlDisplayName',
          deps: ['Id'],
        },
      ],
    ]),
    computeGraph: {
      order: ['DisplayName'],
      virtualComputeFields: new Set(['DisplayName']),
      parsedDeps: new Map([['DisplayName', []]]),
    },
  } as any;

  const entity: any = { DisplayName: 'from-db' };
  ComputeEngine.injectVirtualForRead(meta, entity);

  expect(entity.DisplayName).toBe('from-db');
});
