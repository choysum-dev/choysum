// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../../../metadata/storage';
import { aliasSelection, buildSelectionTree, getScalarFields } from '..';

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

test('repository selection tree collects only scalar persisted and virtual display fields', () => {
  const meta = {
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['DisplayName', {}],
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => null } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  expect(getScalarFields(meta)).toEqual(['Name', 'DisplayName']);
});

test('repository selection tree treats owner binary/image as attachment-backed virtual fields', () => {
  const meta = {
    application: 'auth',
    modelName: 'User',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Avatar', { type: 'image', column: { name: 'Avatar' } }],
      ['Resume', { type: 'binary', column: { name: 'Resume' } }],
    ]),
  } as any;

  expect(getScalarFields(meta)).toEqual(['Name']);

  const wildcardTree = buildSelectionTree(meta, ['*']);
  expect(Array.from(wildcardTree.columns)).toEqual(['Name']);

  const explicitTree = buildSelectionTree(meta, ['Name', 'Avatar', 'Resume']);
  expect(Array.from(explicitTree.columns)).toEqual(['Name']);
});

test('repository selection tree keeps storage blob carrier binary/image scalar fields', () => {
  const meta = {
    application: 'document',
    modelName: 'AttachmentObject',
    fullModelName: 'document.AttachmentObject',
    fields: new Map([
      ['Backend', { column: { name: 'Backend' } }],
      ['BlobData', { type: 'binary', column: { name: 'BlobData' } }],
      ['Preview', { type: 'image', column: { name: 'Preview' } }],
    ]),
  } as any;

  expect(getScalarFields(meta)).toEqual(['Backend', 'BlobData', 'Preview']);

  const tree = buildSelectionTree(meta, ['*']);
  expect(Array.from(tree.columns)).toEqual(['Backend', 'BlobData', 'Preview']);
});

test('repository selection tree expands shorthand relations to child scalar fields and preserves Id fallback', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map([
      ['Title', { column: { name: 'Title' } }],
      ['DisplayName', {}],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['DisplayName', {}],
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const tree = buildSelectionTree(demoMeta, ['*', 'Owner']);
      expect(Array.from(tree.columns)).toEqual(['Name', 'DisplayName', 'Owner']);
      const owner = tree.relations.get('Owner')!;
      expect(owner.fieldType).toBe('ManyToOne');
      expect(Array.from(owner.node.columns)).toEqual(['Title', 'DisplayName']);
    }
  );
});

test('repository selection tree uses explicit nested fields and adds Id when child selection is empty', () => {
  class DemoModel {}
  class TaskModel {}

  const taskMeta = {
    type: TaskModel,
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['State', { column: { name: 'State' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([['Tasks', { type: 'OneToMany', relation: { targetModel: () => TaskModel, inverseField: 'ParentId' } }]]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [TaskModel, taskMeta],
    ]),
    () => {
      const explicitTree = buildSelectionTree(demoMeta, [{ Tasks: ['State'] }]);
      expect(Array.from(explicitTree.relations.get('Tasks')!.node.columns)).toEqual(['State']);

      const fallbackTree = buildSelectionTree(demoMeta, [{ Tasks: [] }]);
      expect(Array.from(fallbackTree.relations.get('Tasks')!.node.columns)).toEqual(['Name', 'State']);
    }
  );
});

test('repository selection tree prefers SqlCompute over OneToMany relation load', () => {
  class DemoModel {}
  class ChildModel {}

  const childMeta = {
    type: ChildModel,
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      [
        'Childs',
        {
          type: 'OneToMany',
          relation: { targetModel: () => ChildModel, inverseField: 'ParentId' },
        },
      ],
    ]),
    sqlComputeHandlers: new Map([['Childs', { field: 'Childs', method: 'sqlChilds' }]]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [ChildModel, childMeta],
    ]),
    () => {
      const tree = buildSelectionTree(demoMeta, ['Childs']);
      expect(tree.columns.has('Childs')).toBe(true);
      expect(tree.relations.has('Childs')).toBe(false);

      expect(() => buildSelectionTree(demoMeta, [{ Childs: ['Name'] }], { strict: true })).toThrow(
        /SqlCompute field does not support nested relation selection/
      );

      const nestedLoose = buildSelectionTree(demoMeta, [{ Childs: ['Name'] }], { strict: false });
      expect(nestedLoose.relations.has('Childs')).toBe(false);
      expect(nestedLoose.columns.has('Childs')).toBe(false);
    }
  );
});

test('repository selection tree aliasSelection handles function/as/plain branches', () => {
  const fnAliased = aliasSelection(
    () => ({
      as(alias: string) {
        return { alias, tagged: true };
      },
    }),
    'A'
  );
  expect(fnAliased('EB')).toEqual({ alias: 'A', tagged: true });

  const fnPlain = aliasSelection(() => 'raw', 'B');
  expect(fnPlain('EB')).toBe('raw');

  const objAliased = aliasSelection(
    {
      as(alias: string) {
        return { alias, from: 'obj' };
      },
    },
    'C'
  );
  expect(objAliased).toEqual({ alias: 'C', from: 'obj' });

  expect(aliasSelection('plain', 'D')).toBe('plain');
});

test('repository selection tree skips invalid relation metadata and supports many2many Id fallback', () => {
  class DemoModel {}
  class TagModel {}

  const tagMeta = {
    type: TagModel,
    fields: new Map<string, any>(),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['MissingRel', { type: 'ManyToOne' }],
      ['BrokenM2O', { type: 'ManyToOne', relation: {}, column: { name: 'BrokenM2O' } }],
      ['BrokenO2M', { type: 'OneToMany', relation: { targetModel: () => TagModel } }],
      ['BrokenM2M', { type: 'ManyToMany', relation: { targetModel: () => TagModel, joinModel: () => DemoModel, joinField: 'A' } }],
      ['Tags', { type: 'ManyToMany', relation: { targetModel: () => TagModel, joinModel: () => DemoModel, joinField: 'LeftId', inverseJoinField: 'RightId' } }],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [TagModel, tagMeta],
    ]),
    () => {
      const tree = buildSelectionTree(demoMeta, [
        'MissingField',
        { MissingRel: ['Name'] },
        { BrokenM2O: ['Name'] },
        { BrokenO2M: ['Name'] },
        { BrokenM2M: ['Name'] },
        { Tags: [] },
      ]);
      expect(tree.relations.has('MissingRel')).toBe(false);
      expect(tree.relations.has('BrokenM2O')).toBe(false);
      expect(tree.relations.has('BrokenO2M')).toBe(false);
      expect(tree.relations.has('BrokenM2M')).toBe(false);
      expect(Array.from(tree.relations.get('Tags')!.node.columns)).toEqual(['Id']);
    }
  );
});

test('repository selection tree strict mode throws on unknown fields and invalid nested fields', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;

  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
    ]),
  } as any;

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SELECTION_TREE_STRICT: true };

    withFakeMetadata(
      new Map([
        [DemoModel, demoMeta],
        [OwnerModel, ownerMeta],
      ]),
      () => {
        expect(() => buildSelectionTree(demoMeta, ['MissingField'] as any)).toThrow('Selection field does not exist');
        expect(() => buildSelectionTree(demoMeta, [{ Owner: ['MissingNested'] }] as any)).toThrow('Owner.MissingNested');
      }
    );
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository selection tree strict mode auto-enables in development CHOYSUM_ENV', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const meta = {
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_ENV: 'development' };
    expect(() => buildSelectionTree(meta, ['MissingField'] as any)).toThrow('Selection field does not exist');
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository selection tree strict option false overrides strict env and keeps backward-compatible skip behavior', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const meta = {
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SELECTION_TREE_STRICT: true };
    const tree = buildSelectionTree(meta, ['MissingField'] as any, { strict: false });
    expect(Array.from(tree.columns)).toEqual([]);
    expect(tree.relations.size).toBe(0);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository selection tree resolves strict mode when import.meta.env is missing', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const meta = {
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = undefined;
    const tree = buildSelectionTree(meta, ['MissingField'] as any);
    expect(Array.from(tree.columns)).toEqual([]);
    expect(tree.relations.size).toBe(0);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository selection tree strict env parser supports truthy/falsey tokens', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const meta = {
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const strictTokens: any[] = [true, '1', 'yes', 'on', 'TRUE'];
  const nonStrictTokens: any[] = [false, '0', 'no', 'off', 'FALSE', ' ', 'unexpected'];

  try {
    for (const token of strictTokens) {
      (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SELECTION_TREE_STRICT: token };
      expect(() => buildSelectionTree(meta, ['MissingField'] as any)).toThrow('Selection field does not exist');
    }

    for (const token of nonStrictTokens) {
      (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SELECTION_TREE_STRICT: token };
      const tree = buildSelectionTree(meta, ['MissingField'] as any);
      expect(Array.from(tree.columns)).toEqual([]);
      expect(tree.relations.size).toBe(0);
    }
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository selection tree strict mode validates relation-object shape and missing relation fields', () => {
  const meta = {
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => null }, column: { name: 'Owner' } }],
    ]),
  } as any;

  expect(() => buildSelectionTree(meta, [{} as any], { strict: true })).toThrow('Selection relation object cannot be empty');
  expect(() => buildSelectionTree(meta, [{ Owner: 'bad' } as any], { strict: true })).toThrow('Selection relation sub-selection must be an array');
  expect(() => buildSelectionTree(meta, [{ MissingRel: ['Name'] } as any], { strict: true })).toThrow('Selection relation field does not exist');
});

test('repository selection tree relation fallback branches cover many2one/one2many/many2many child defaults and explicit subfields', () => {
  class DemoModel {}
  class OwnerModel {}
  class TaskModel {}
  class TagModel {}

  const ownerMeta = {
    type: OwnerModel,
    fields: new Map<string, any>(),
  } as any;
  const taskMeta = {
    type: TaskModel,
    fields: new Map<string, any>(),
  } as any;
  const tagMeta = {
    type: TagModel,
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;
  const demoMeta = {
    type: DemoModel,
    fields: new Map([
      ['BrokenOwner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel }, column: { name: 'Owner' } }],
      ['Tasks', { type: 'OneToMany', relation: { targetModel: () => TaskModel, inverseField: 'ParentId' } }],
      [
        'Tags',
        {
          type: 'ManyToMany',
          relation: {
            targetModel: () => TagModel,
            joinModel: () => DemoModel,
            joinField: 'LeftId',
            inverseJoinField: 'RightId',
          },
        },
      ],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
      [TaskModel, taskMeta],
      [TagModel, tagMeta],
    ]),
    () => {
      expect(() => buildSelectionTree(demoMeta, [{ BrokenOwner: [] } as any], { strict: true })).toThrow('missing column metadata');

      const ownerFallback = buildSelectionTree(demoMeta, ['Owner']);
      expect(Array.from(ownerFallback.relations.get('Owner')!.node.columns)).toEqual(['Id']);

      const tasksFallback = buildSelectionTree(demoMeta, [{ Tasks: null } as any]);
      expect(Array.from(tasksFallback.relations.get('Tasks')!.node.columns)).toEqual(['Id']);

      const tagsExplicit = buildSelectionTree(demoMeta, [{ Tags: ['Name'] }]);
      expect(Array.from(tagsExplicit.relations.get('Tags')!.node.columns)).toEqual(['Name']);
    }
  );
});
