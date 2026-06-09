// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage, getEffectiveConstraints } from './storage';

class StorageMergeModel extends BaseModel {}
class StorageParentModel extends BaseModel {}
class StorageChildModel extends StorageParentModel {}
class StorageTreeModel extends BaseModel {}
class StorageFreshModel extends BaseModel {}

function resetModelMetadata(ctor: any) {
  const storage = MetadataStorage.instance as any;
  if (storage.models?.delete) {
    storage.models.delete(ctor);
  }
}

test('metadata storage builder merge handles handlers, map and undefined values', () => {
  const storage = MetadataStorage.instance as any;

  resetModelMetadata(StorageMergeModel as any);

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      application: 'core',
      services: new Map([['S1', { name: 'S1' }]]),
      constraintHandlers: [
        {
          method: 'ruleOne',
          fields: [' Name ', null as any],
          preview: true,
          priority: 7,
        },
      ],
      onchangeHandlers: [
        {
          method: 'onAmount',
          triggers: ['Amount', 'Amount'],
          reads: ['Currency', 'Currency'],
          priority: 5,
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      application: undefined,
      services: new Map([['S2', { name: 'S2' }]]),
      constraintHandlers: [
        {
          method: 'ruleOne',
          fields: ['Code', 'Code'],
        },
        {
          method: 'ruleTwo',
          fields: ['Value'],
          alwaysOnCreate: true,
        },
      ],
      onchangeHandlers: [
        {
          method: 'onAmount',
          triggers: ['Tax', 'Tax'],
        },
        {
          method: 'onName',
          triggers: ['Name'],
          reads: ['DisplayName'],
        },
      ],
    } as any
  );

  const meta = storage.getModelMetadata(StorageMergeModel as any) as any;

  expect(meta.application).toBe('core');
  expect(Array.from(meta.services.keys()).sort()).toEqual(['S1', 'S2']);

  const c1 = (meta.constraintHandlers || []).find((x: any) => x.method === 'ruleOne');
  const c2 = (meta.constraintHandlers || []).find((x: any) => x.method === 'ruleTwo');
  expect(c1?.fields).toEqual(['Code']);
  expect(c1?.preview).toBe(true);
  expect(c1?.priority).toBe(7);
  expect(c2?.alwaysOnCreate).toBe(true);
  expect(c2?.priority).toBe(100);

  const oc1 = (meta.onchangeHandlers || []).find((x: any) => x.method === 'onAmount');
  const oc2 = (meta.onchangeHandlers || []).find((x: any) => x.method === 'onName');
  expect(oc1?.triggers).toEqual(['Tax']);
  expect(oc1?.reads).toEqual(['Currency']);
  expect(oc1?.priority).toBe(5);
  expect(oc2?.triggers).toEqual(['Name']);
  expect(oc2?.reads).toEqual(['DisplayName']);
});

test('metadata storage cache is reused and cleared for target and subclass static metadata', () => {
  const storage = MetadataStorage.instance as any;

  resetModelMetadata(StorageParentModel as any);
  resetModelMetadata(StorageChildModel as any);

  storage.setModelMetadata(
    StorageParentModel as any,
    {
      fields: new Map([['Name', { type: 'varchar', column: {} }]]),
    } as any
  );
  storage.setModelMetadata(
    StorageChildModel as any,
    {
      fields: new Map([['Code', { type: 'varchar', column: {} }]]),
    } as any
  );

  const first = storage.getModelMetadata(StorageChildModel as any);
  const second = storage.getModelMetadata(StorageChildModel as any);
  expect(first).toBe(second);

  (StorageParentModel as any).metadata = { stale: true };
  (StorageChildModel as any).metadata = { stale: true };

  storage.setModelMetadata(
    StorageParentModel as any,
    {
      fields: new Map([['Amount', { type: 'int', column: {} }]]),
    } as any
  );

  const third = storage.getModelMetadata(StorageChildModel as any);
  expect(third).not.toBe(first);
  expect((StorageParentModel as any).metadata).toBe(undefined);
  expect((StorageChildModel as any).metadata).toBe(undefined);
  const mergedKeys = Array.from(third.fields.keys());
  expect(mergedKeys.includes('Amount')).toBe(true);
  expect(mergedKeys.includes('Code')).toBe(true);
  expect(mergedKeys.includes('Name')).toBe(true);
});

test('metadata storage injects ParentPath compute and checks cycle guard', () => {
  const storage = MetadataStorage.instance as any;

  resetModelMetadata(StorageTreeModel as any);

  storage.setModelMetadata(
    StorageTreeModel as any,
    {
      parentField: 'ParentId',
      fields: new Map([['ParentId', { type: 'ManyToOne', relation: { targetModel: () => StorageTreeModel } }]]),
    } as any
  );

  const meta = storage.getModelMetadata(StorageTreeModel as any);
  const parentPath = meta.fields.get('ParentPath') as any;

  expect(Boolean(parentPath)).toBe(true);
  expect(parentPath?.column?.compute?.deps).toEqual(['Id', 'ParentId', 'ParentId.ParentPath']);

  const expr = parentPath?.column?.compute?.expr;
  expect(expr({ Id: 'n1', ParentId: { ParentPath: 'root/' } })).toBe('root/n1/');
  expect(expr({ Id: 'n2', ParentId: null })).toBe('n2/');
  expect(expr({ ParentId: { ParentPath: 'root/' } })).toBe('');

  expect(() => expr({ Id: 'n3', ParentId: { ParentPath: 'root/n3/' } })).toThrow('Cycle detected');
});

test('getEffectiveConstraints fallback path works when instance method is unavailable', () => {
  const storage = MetadataStorage.instance as any;
  const original = storage.getEffectiveConstraints;

  resetModelMetadata(StorageParentModel as any);
  resetModelMetadata(StorageChildModel as any);

  storage.setModelMetadata(
    StorageParentModel as any,
    {
      fullModelName: 'core.parent',
      constraintHandlers: [
        {
          method: 'shared',
          fields: ['Name', 'Name', ''],
          priority: 40,
        },
        {
          method: 'parentOnly',
          fields: ['Code'],
          priority: 10,
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageChildModel as any,
    {
      modelName: 'core.child',
      constraintHandlers: [
        {
          method: 'shared',
          fields: ['ChildName'],
          priority: 5,
        },
        {
          method: 'childOnly',
          fields: ['Value'],
          preview: true,
        },
      ],
    } as any
  );

  try {
    storage.getEffectiveConstraints = undefined;

    const out = getEffectiveConstraints(StorageChildModel as any);
    expect(out.map((x: any) => x.method)).toEqual(['shared', 'parentOnly', 'childOnly']);

    const shared = out.find((x: any) => x.method === 'shared');
    const childOnly = out.find((x: any) => x.method === 'childOnly');
    const parentOnly = out.find((x: any) => x.method === 'parentOnly');

    expect(shared?.fields).toEqual(['ChildName']);
    expect(shared?.source).toBe('core.child');
    expect(childOnly?.priority).toBe(100);
    expect(childOnly?.preview).toBe(true);
    expect(parentOnly?.source).toBe('core.parent');
  } finally {
    storage.getEffectiveConstraints = original;
  }
});

test('metadata storage initializes default tableName and clearCache invalidates merged cache', () => {
  const storage = MetadataStorage.instance as any;

  resetModelMetadata(StorageFreshModel as any);

  storage.setModelMetadata(
    StorageFreshModel as any,
    {
      fields: new Map([['Name', { type: 'varchar', column: {} }]]),
    } as any
  );

  const first = storage.getModelMetadata(StorageFreshModel as any);
  expect(typeof first.tableName).toBe('function');
  expect(first.tableName()).toBe('');

  storage.clearCache();

  const second = storage.getModelMetadata(StorageFreshModel as any);
  expect(second).not.toBe(first);
  expect(second.tableName()).toBe('');
});

test('metadata storage builder merge normalizes malformed handlers and keeps undefined scalars untouched', () => {
  const storage = MetadataStorage.instance as any;
  resetModelMetadata(StorageMergeModel as any);

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      application: 'core',
      services: new Map([['A', { enabled: true }]]),
      constraintHandlers: [
        {
          method: 'ruleX',
          fields: ['Name'],
          preview: true,
          alwaysOnCreate: true,
          priority: 9,
          isStatic: true,
        },
      ],
      onchangeHandlers: [
        {
          method: 'onX',
          triggers: ['Qty'],
          reads: ['Amount'],
          priority: 8,
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      application: undefined,
      constraintHandlers: [
        {
          method: 'ruleX',
          // Non-array input should hit the empty-array branch in ensureStringArray.
          fields: 'Name' as any,
        },
        null,
      ],
      onchangeHandlers: [
        {
          method: 'onX',
          triggers: 'Name' as any,
          // Non-array input should fall back to existing reads.
          reads: 'ReadName' as any,
        },
        undefined,
      ],
    } as any
  );

  const meta = storage.getModelMetadata(StorageMergeModel as any) as any;
  expect(meta.application).toBe('core');
  expect(Array.from(meta.services.keys())).toEqual(['A']);

  const c = (meta.constraintHandlers || []).find((x: any) => x.method === 'ruleX');
  expect(c?.fields).toEqual([]);
  expect(c?.preview).toBe(true);
  expect(c?.priority).toBe(9);
  expect(c?.alwaysOnCreate).toBe(true);
  expect(c?.isStatic).toBe(true);

  const oc = (meta.onchangeHandlers || []).find((x: any) => x.method === 'onX');
  expect(oc?.triggers).toEqual(['Qty']);
  expect(oc?.reads).toEqual(['Amount']);
  expect(oc?.priority).toBe(8);
});

test('metadata storage getEffectiveConstraints resolves source fallback to class name and unknown', () => {
  const storage = MetadataStorage.instance as any;

  class StorageAnonymousLikeModel extends BaseModel {}
  class StorageUnknownSourceModel extends BaseModel {}

  resetModelMetadata(StorageAnonymousLikeModel as any);
  resetModelMetadata(StorageUnknownSourceModel as any);

  storage.setModelMetadata(
    StorageAnonymousLikeModel as any,
    {
      fullModelName: '',
      modelName: '',
      name: '',
      constraintHandlers: [
        {
          method: 'classNameSource',
          fields: [' Name ', '', null as any],
          priority: undefined,
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageUnknownSourceModel as any,
    {
      fullModelName: '',
      modelName: '',
      name: '',
      // Simulate an extreme runtime where the constructor name is empty.
      constraintHandlers: [
        {
          method: 'unknownSource',
          fields: 'bad-fields' as any,
        },
      ],
    } as any
  );
  Object.defineProperty(StorageUnknownSourceModel, 'name', {
    value: '',
    configurable: true,
  });

  const classOut = storage.getEffectiveConstraints(StorageAnonymousLikeModel as any);
  const unknownOut = storage.getEffectiveConstraints(StorageUnknownSourceModel as any);

  expect(classOut[0]?.source).toBe('StorageAnonymousLikeModel');
  expect(classOut[0]?.priority).toBe(100);
  expect(classOut[0]?.fields).toEqual(['Name']);

  expect(unknownOut[0]?.source).toBe('unknown');
  expect(unknownOut[0]?.fields).toEqual([]);
});

test('metadata storage builder merge keeps existing handler attributes when incoming payload is sparse', () => {
  const storage = MetadataStorage.instance as any;
  resetModelMetadata(StorageMergeModel as any);

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      constraintHandlers: [
        {
          method: 'ruleKeep',
          fields: ['Name'],
          preview: true,
          alwaysOnCreate: true,
          priority: 9,
          isStatic: true,
        },
      ],
      onchangeHandlers: [
        {
          method: 'onKeep',
          triggers: ['Qty'],
          reads: ['Amount'],
          priority: 8,
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageMergeModel as any,
    {
      constraintHandlers: [
        {
          method: 'ruleKeep',
          fields: undefined,
          preview: undefined,
          alwaysOnCreate: undefined,
          priority: undefined,
          isStatic: undefined,
        },
        {
          method: '',
          fields: ['Ignore'],
        } as any,
        undefined as any,
      ],
      onchangeHandlers: [
        {
          method: 'onKeep',
          triggers: undefined,
          reads: undefined,
          priority: undefined,
        },
        {
          method: '',
          triggers: ['Ignore'],
        } as any,
        null as any,
      ],
    } as any
  );

  const meta = storage.getModelMetadata(StorageMergeModel as any) as any;
  const c = (meta.constraintHandlers || []).find((x: any) => x.method === 'ruleKeep');
  const o = (meta.onchangeHandlers || []).find((x: any) => x.method === 'onKeep');

  expect(c).toEqual({
    method: 'ruleKeep',
    fields: ['Name'],
    preview: true,
    alwaysOnCreate: true,
    priority: 9,
    isStatic: true,
  });
  expect(o).toEqual({
    method: 'onKeep',
    triggers: ['Qty'],
    reads: ['Amount'],
    priority: 8,
  });
});

test('metadata storage effective constraints source fallback prefers metadata name then constructor name', () => {
  const storage = MetadataStorage.instance as any;

  class StorageSourceFromMetadataName extends BaseModel {}
  class StorageSourceFromCtorName extends BaseModel {}

  resetModelMetadata(StorageSourceFromMetadataName as any);
  resetModelMetadata(StorageSourceFromCtorName as any);

  storage.setModelMetadata(
    StorageSourceFromMetadataName as any,
    {
      fullModelName: '',
      modelName: '',
      name: 'metadata.name.source',
      constraintHandlers: [
        {
          method: 'byMetaName',
          fields: ['  A  '],
        },
      ],
    } as any
  );

  storage.setModelMetadata(
    StorageSourceFromCtorName as any,
    {
      fullModelName: '',
      modelName: '',
      name: '',
      constraintHandlers: [
        {
          method: 'byCtorName',
          fields: ['  B  '],
        },
      ],
    } as any
  );

  const byMeta = storage.getEffectiveConstraints(StorageSourceFromMetadataName as any);
  const byCtor = storage.getEffectiveConstraints(StorageSourceFromCtorName as any);

  expect(byMeta[0]?.source).toBe('metadata.name.source');
  expect(byMeta[0]?.fields).toEqual(['A']);
  expect(byCtor[0]?.source).toBe('StorageSourceFromCtorName');
  expect(byCtor[0]?.fields).toEqual(['B']);
});
