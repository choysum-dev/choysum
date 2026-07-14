// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../orm/model/model';
import { Compute, Field } from '../../orm/decorator';
import { REL_ALIAS_PREFIX } from '../../orm/relation/relation_alias';
import { ModelProxyFactory } from './proxy';
import { MODEL_SYMBOLS } from './symbols';
import { Dep } from './dep';

class ProxyHydratedUser extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

class ProxyHydratedPost extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => ProxyHydratedUser } })
  Owner?: ProxyHydratedUser;

  @Field({ type: 'OneToMany', relation: { targetModel: () => ProxyHydratedTag, inverseField: 'PostId' } })
  Tags?: ProxyHydratedTag[];
}

class ProxyHydratedTag extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => ProxyHydratedPost } })
  PostId?: ProxyHydratedPost;

  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

class ProxyEdgeModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 64 })
  ComputedName?: string;

  @Compute<ProxyEdgeModel>('ComputedName', { deps: ['Name' as any] })
  computeComputedName() {
    return String(this.Name || '').toUpperCase();
  }

  @Field({ type: 'OneToMany', relation: { targetModel: () => ProxyHydratedTag, inverseField: 'PostId' } })
  Tags?: ProxyHydratedTag[];
}

class ProxyBrokenComputedModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 64 })
  BrokenComputed?: string;

  @Compute<ProxyBrokenComputedModel>('BrokenComputed', { deps: ['Name' as any] })
  computeBroken() {
    throw new Error('compute exploded');
  }
}

class ProxyDecimalModel extends BaseModel {
  @Field({ type: 'decimal', precision: 16, scale: 2 })
  Amount?: any;
}

class ProxyNoTargetRelationModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => undefined as any } as any })
  Owner?: any;

  @Field({ type: 'OneToMany', relation: { targetModel: () => undefined as any, inverseField: 'ParentId' } as any })
  Children?: any[];
}

class ProxyManyToManyModel extends BaseModel {
  @Field({ type: 'ManyToMany', relation: { targetModel: () => ProxyHydratedTag } as any })
  Tags?: ProxyHydratedTag[];
}

function createProxyModel<T extends BaseModel>(ModelCtor: { new (...args: any[]): T } & typeof BaseModel, entity: Record<string, any>): T {
  const factoryToken = (ModelCtor as any).FACTORY_TOKEN;
  const instance = new ModelCtor(factoryToken, entity, undefined as any);
  return new ModelProxyFactory(instance, entity as any, undefined).create();
}

function createProxyWithFactory<T extends BaseModel>(ModelCtor: { new (...args: any[]): T } & typeof BaseModel, entity: Record<string, any>) {
  const factoryToken = (ModelCtor as any).FACTORY_TOKEN;
  const instance = new ModelCtor(factoryToken, entity, undefined as any);
  const factory = new ModelProxyFactory(instance, entity as any, undefined);
  return { proxy: factory.create(), factory };
}

test('proxy preloaded many2one hydration does not call public Model.Hydrate', () => {
  const originalHydrate = ProxyHydratedUser.Hydrate;
  let hydrateCalls = 0;

  try {
    ProxyHydratedUser.Hydrate = ((_entity: Record<string, any>) => {
      hydrateCalls += 1;
      throw new Error('public hydrate should not be called');
    }) as any;

    const entity = { Id: 'POST-1' } as Record<string, any>;
    const post = createProxyModel(ProxyHydratedPost, entity);
    entity[`${REL_ALIAS_PREFIX}Owner`] = { Id: 'USER-1', Name: 'Alice' };

    const owner = post.Owner;

    expect(hydrateCalls).toBe(0);
    expect(owner instanceof ProxyHydratedUser).toBe(true);
    expect(owner?.Id).toBe('USER-1');
    expect(owner?.Name).toBe('Alice');
  } finally {
    ProxyHydratedUser.Hydrate = originalHydrate;
  }
});

test('proxy preloaded to-many hydration does not call public Model.Hydrate', () => {
  const originalHydrate = ProxyHydratedTag.Hydrate;
  let hydrateCalls = 0;

  try {
    ProxyHydratedTag.Hydrate = ((_entity: Record<string, any>) => {
      hydrateCalls += 1;
      throw new Error('public hydrate should not be called');
    }) as any;

    const entity = { Id: 'POST-2' } as Record<string, any>;
    const post = createProxyModel(ProxyHydratedPost, entity);
    entity[`${REL_ALIAS_PREFIX}Tags`] = [
      { Id: 'TAG-1', Name: 'alpha' },
      { Id: 'TAG-2', Name: 'beta' },
    ];

    const tags = post.Tags || [];

    expect(hydrateCalls).toBe(0);
    expect(tags.length).toBe(2);
    expect(tags[0] instanceof ProxyHydratedTag).toBe(true);
    expect(tags[0]?.Id).toBe('TAG-1');
    expect(tags[1]?.Name).toBe('beta');
  } finally {
    ProxyHydratedTag.Hydrate = originalHydrate;
  }
});

test('proxy throws when accessing unloaded relation without preload or explicit value', () => {
  const edge = createProxyModel(ProxyHydratedPost, { Id: 'POST-ERR-1' });

  let thrown: unknown;
  try {
    void edge.Owner;
  } catch (error) {
    thrown = error;
  }

  expect(String((thrown as any)?.message || '').includes('Accessing unloaded relation "Owner"')).toBe(true);
});

test('proxy forbids writing computed field directly', () => {
  const edge = createProxyModel(ProxyEdgeModel, { Id: 'EDGE-1', Name: 'alpha', ComputedName: 'ALPHA' });

  let thrown: unknown;
  try {
    (edge as any).ComputedName = 'MANUAL';
  } catch (error) {
    thrown = error;
  }

  expect(String((thrown as any)?.message || '').includes('Cannot set computed property "ComputedName"')).toBe(true);
});

test('proxy tracks relation array mutations and supports reset via symbols', () => {
  const entity = { Id: 'POST-TRACK-1' } as Record<string, any>;
  entity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-1', Name: 'a' }];
  const edge = createProxyModel(ProxyHydratedPost, entity as any);

  const tags = (edge as any).Tags as any[];
  tags.push({ Id: 'TAG-2', Name: 'b' });
  tags.splice(0, 1);

  const changes = (edge as any)[MODEL_SYMBOLS.collectRelationChanges]() as Record<string, any[]>;
  expect(Array.isArray(changes.Tags)).toBe(true);
  expect(changes.Tags.length).toBe(2);
  expect(changes.Tags[0]?.method).toBe('push');
  expect(changes.Tags[1]?.method).toBe('splice');
  expect(Array.isArray(changes.Tags[1]?.snapshot)).toBe(true);

  (edge as any)[MODEL_SYMBOLS.resetRelationChanges]();
  const afterReset = (edge as any)[MODEL_SYMBOLS.collectRelationChanges]() as Record<string, any[]>;
  expect(afterReset).toEqual({});
});

test('proxy allows accessing manually assigned relation values without preload aliases', () => {
  const post = createProxyModel(ProxyHydratedPost, { Id: 'POST-MANUAL-1' });
  const owner = createProxyModel(ProxyHydratedUser, { Id: 'USER-M1', Name: 'Manual User' });

  (post as any).Owner = owner;
  const readOwner = (post as any).Owner;

  expect(readOwner).toBe(owner);
  expect(readOwner?.Name).toBe('Manual User');
});

test('proxy returns undefined for unknown symbol getters via reflect fallback', () => {
  const post = createProxyModel(ProxyHydratedPost, { Id: 'POST-SYM-1' });
  const unknown = Symbol('unknown-handler');

  expect((post as any)[unknown]).toBe(undefined);
});

test('proxy resolves relation using alias candidates when canonical alias is absent', () => {
  const entity = { Id: 'POST-ALIAS-1' } as Record<string, any>;
  const post = createProxyModel(ProxyHydratedPost, entity);
  entity[`${REL_ALIAS_PREFIX}owner`] = { Id: 'USER-ALIAS-1', Name: 'Alias User' };

  const owner = post.Owner;
  expect(owner instanceof ProxyHydratedUser).toBe(true);
  expect(owner?.Id).toBe('USER-ALIAS-1');
});

test('proxy preloaded to-many returns empty array when payload is not an array', () => {
  const entity = { Id: 'POST-EMPTY-LIST-1' } as Record<string, any>;
  const post = createProxyModel(ProxyHydratedPost, entity);
  entity[`${REL_ALIAS_PREFIX}Tags`] = { not: 'array' };

  const tags = post.Tags || [];
  expect(Array.isArray(tags)).toBe(true);
  expect(tags.length).toBe(0);
});

test('proxy relation array index assignment is tracked as set operation', () => {
  const entity = { Id: 'POST-SET-1' } as Record<string, any>;
  entity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-SET-1', Name: 'before' }];
  const post = createProxyModel(ProxyHydratedPost, entity as any);

  const tags = (post as any).Tags as any[];
  tags[0] = { Id: 'TAG-SET-2', Name: 'after' };

  const changes = (post as any)[MODEL_SYMBOLS.collectRelationChanges]() as Record<string, any[]>;
  expect(Array.isArray(changes.Tags)).toBe(true);
  expect(changes.Tags.some(item => item?.method === 'set')).toBe(true);
});

test('proxy wraps computed evaluation failures with field context', () => {
  const broken = createProxyModel(ProxyBrokenComputedModel, { Id: 'BROKEN-1', Name: 'x' });

  let thrown: unknown;
  try {
    void (broken as any).BrokenComputed;
  } catch (error) {
    thrown = error;
  }

  const message = String((thrown as any)?.message || '');
  expect(message.includes('Failed to compute property BrokenComputed')).toBe(true);
  expect(message.includes('compute exploded')).toBe(true);
});

test('proxy preloaded many2one null returns null and is cached', () => {
  const entity = { Id: 'POST-NULL-REL-1' } as Record<string, any>;
  const post = createProxyModel(ProxyHydratedPost, entity);
  entity[`${REL_ALIAS_PREFIX}Owner`] = null;

  const first = (post as any).Owner;
  const second = (post as any).Owner;

  expect(first).toBe(null);
  expect(second).toBe(null);
});

test('proxy treats manually assigned null/array relations as already loaded values', () => {
  const post = createProxyModel(ProxyHydratedPost, { Id: 'POST-LOADED-1' });

  (post as any).Owner = null;
  (post as any).Tags = [];

  expect((post as any).Owner).toBe(null);
  expect((post as any).Tags).toEqual([]);
});

test('proxy wraps cached relation array into tracked relation proxy on access', () => {
  const { proxy, factory } = createProxyWithFactory(ProxyHydratedPost, { Id: 'POST-CACHE-ARR-1' });
  (factory as any).relationCache.set('Tags', [{ Id: 'TAG-CACHED-1', Name: 'cached' }]);

  const tags = (proxy as any).Tags as any[];
  expect(Array.isArray(tags)).toBe(true);
  expect((tags as any).__isRelationProxy).toBe(true);

  tags.push({ Id: 'TAG-CACHED-2', Name: 'cached-2' });
  const changes = (proxy as any)[MODEL_SYMBOLS.collectRelationChanges]() as Record<string, any[]>;
  expect(changes.Tags.some(op => op?.method === 'push')).toBe(true);
});

test('proxy returns stable bound handlers for known symbol methods', () => {
  const post = createProxyModel(ProxyHydratedPost, { Id: 'POST-SYM-CACHE-1' });

  const first = (post as any)[MODEL_SYMBOLS.getChangedFields];
  const second = (post as any)[MODEL_SYMBOLS.getChangedFields];

  expect(typeof first).toBe('function');
  expect(first).toBe(second);
});

test('proxy cleanup clears tracked scalar and relation changes', () => {
  const entity = { Id: 'POST-CLEAN-1' } as Record<string, any>;
  entity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-CLEAN-1', Name: 'before' }];
  const { proxy, factory } = createProxyWithFactory(ProxyHydratedPost, entity);

  (proxy as any).Id = 'POST-CLEAN-2';
  const tags = (proxy as any).Tags as any[];
  tags.push({ Id: 'TAG-CLEAN-2', Name: 'after' });

  expect((proxy as any)[MODEL_SYMBOLS.hasChanged]('Id')).toBe(true);
  expect(((proxy as any)[MODEL_SYMBOLS.collectRelationChanges]() as Record<string, any[]>).Tags?.length).toBe(1);

  factory.cleanup();

  expect((proxy as any)[MODEL_SYMBOLS.getChangedFields]()).toEqual([]);
  expect((proxy as any)[MODEL_SYMBOLS.collectRelationChanges]()).toEqual({});
});

test('proxy decimal field read is lazy-normalized and write is quantized by metadata', () => {
  const model = createProxyModel(ProxyDecimalModel, { Id: 'DEC-1', Amount: '1.236' });

  const readValue = (model as any).Amount;
  expect(typeof readValue?.toString).toBe('function');
  expect(readValue.toString()).toBe('1.24');
  expect((model as any)[MODEL_SYMBOLS.getChangedFields]()).toEqual([]);

  (model as any).Amount = '2.345';
  const updated = (model as any).Amount;
  expect(updated.toString()).toBe('2.35');
  expect((model as any)[MODEL_SYMBOLS.hasChanged]('Amount')).toBe(true);
});

test('proxy ensureDep creates dep once and reuses it for the same key', () => {
  const { factory } = createProxyWithFactory(ProxyEdgeModel, { Id: 'DEP-1', Name: 'n1' });

  const depA = (factory as any).ensureDep('Name');
  const depB = (factory as any).ensureDep('Name');

  expect(depA).toBe(depB);
  expect((factory as any).deps.get('Name')).toBe(depA);
});

test('proxy symbol getOriginalValue returns previous scalar value after change tracking', () => {
  const model = createProxyModel(ProxyEdgeModel, { Id: 'ORIG-1', Name: 'before' });

  (model as any).Name = 'after';

  expect((model as any)[MODEL_SYMBOLS.hasChanged]('Name')).toBe(true);
  expect((model as any)[MODEL_SYMBOLS.getOriginalValue]('Name')).toBe('before');
});

test('proxy relation hydration branches handle missing target ctor and many2many payload', () => {
  const noTargetEntity = { Id: 'NT-1' } as Record<string, any>;
  noTargetEntity[`${REL_ALIAS_PREFIX}Owner`] = { Id: 'RAW-OWNER-1' };
  noTargetEntity[`${REL_ALIAS_PREFIX}Children`] = [{ Id: 'RAW-CHILD-1' }];

  const noTarget = createProxyModel(ProxyNoTargetRelationModel as any, noTargetEntity);
  expect((noTarget as any).Owner).toEqual({ Id: 'RAW-OWNER-1' });
  expect((noTarget as any).Children).toEqual([{ Id: 'RAW-CHILD-1' }]);

  const manyEntity = { Id: 'MM-1' } as Record<string, any>;
  manyEntity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-MM-1', Name: 'mm' }];
  const many = createProxyModel(ProxyManyToManyModel as any, manyEntity);
  const tags = (many as any).Tags as any[];
  expect(Array.isArray(tags)).toBe(true);
  expect(tags[0] instanceof ProxyHydratedTag).toBe(true);
  expect(tags[0]?.Id).toBe('TAG-MM-1');
});

test('proxy relation array internal marker set path is accepted', () => {
  const entity = { Id: 'MARK-1' } as Record<string, any>;
  entity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-MARK-1', Name: 'one' }];
  const model = createProxyModel(ProxyHydratedPost, entity as any);

  const tags = (model as any).Tags as any[];
  (tags as any).__isRelationProxy = true;
  expect((tags as any).__isRelationProxy).toBe(true);
});

test('proxy decimal read handles null and non-normalizable objects, and non-decimal set path is tracked', () => {
  const decimalModel = createProxyModel(ProxyDecimalModel, { Id: 'DEC-EDGE-1', Amount: null });
  expect((decimalModel as any).Amount).toBe(null);

  (decimalModel as any).Amount = { raw: 'not-a-decimal' };
  expect((decimalModel as any).Amount).toEqual({ raw: 'not-a-decimal' });

  const edge = createProxyModel(ProxyEdgeModel, { Id: 'EDGE-SET-1', Name: 'a' });
  (edge as any).Name = 'b';
  expect((edge as any)[MODEL_SYMBOLS.hasChanged]('Name')).toBe(true);
});

test('proxy private computed and relation helper branches return undefined for non-computed or non-relation fields', () => {
  const { proxy, factory } = createProxyWithFactory(ProxyEdgeModel, { Id: 'PRV-1', Name: 'x' });

  const nonComputed = (factory as any).handleComputed(proxy, 'Name');
  expect(nonComputed).toBe(undefined);

  (factory as any).summary.computedKeys.add('Ghost');
  const missingExpr = (factory as any).handleComputed(proxy, 'Ghost');
  expect(missingExpr).toBe(undefined);

  const nonRelation = (factory as any).handleRelation(proxy, 'Name');
  expect(nonRelation).toBe(undefined);
});

test('proxy private fallback branches: proxyRef undefined, field selection object, cleared originals and missing hasChanged', () => {
  const factoryToken = (ProxyEdgeModel as any).FACTORY_TOKEN;
  const instance = new ProxyEdgeModel(factoryToken, { Id: 'PRV-2', Name: 'x' } as any, undefined as any);
  const factory = new ModelProxyFactory(instance as any, { Id: 'PRV-2', Name: 'x' } as any, [{ Owner: ['Id'] } as any] as any);

  // Cover the target branch of self = this.proxyRef ?? target.
  (factory as any).summary.computedKeys.add('ComputedName');
  const beforeCreate = (factory as any).handleComputed(instance, 'ComputedName');
  expect(beforeCreate).toBe('X');

  // Cover the object branch in getFieldFields.
  const nested = (factory as any).getFieldFields('Owner');
  expect(nested).toEqual(['Id']);

  // Cover the branch where originalValues is missing in trackValueChange.
  (factory as any).cleanup();
  (factory as any).trackValueChange(instance, 'Name', 'after');

  // Cover the false fallback in hasChanged.
  expect((factory as any).hasChanged(instance, 'Name')).toBe(false);
});

test('proxy computed branch falls through when summary key has no compute expr', () => {
  const { proxy, factory } = createProxyWithFactory(ProxyEdgeModel, { Id: 'PRV-3', Name: 'x' });
  (factory as any).summary.computedKeys.add('Ghost');

  expect((proxy as any).Ghost).toBeUndefined();
});

test('proxy getter collects dependency when Dep.target is active', () => {
  const { proxy, factory } = createProxyWithFactory(ProxyEdgeModel, { Id: 'DEP-GET-1', Name: 'dep' });
  const originalTarget = Dep.target;

  try {
    Dep.target = { update: () => {} } as any;
    expect((proxy as any).Name).toBe('dep');
    expect((factory as any).deps.has('Name')).toBe(true);
  } finally {
    Dep.target = originalTarget;
  }
});

test('proxy to-many preloaded list falls back to raw item when hydration returns undefined', () => {
  const entity = { Id: 'POST-RAW-FALLBACK-1' } as Record<string, any>;
  entity[`${REL_ALIAS_PREFIX}Tags`] = [{ Id: 'TAG-RAW-1', Name: 'raw' }];
  const post = createProxyModel(ProxyHydratedPost, entity as any);

  const originalCreate = ModelProxyFactory.prototype.create;
  try {
    ModelProxyFactory.prototype.create = (() => undefined) as any;

    const tags = (post as any).Tags as any[];
    expect(tags).toEqual([{ Id: 'TAG-RAW-1', Name: 'raw' }]);
  } finally {
    ModelProxyFactory.prototype.create = originalCreate;
  }
});

test('proxy computed getter returns stored value when instance is pristine', () => {
  const model = createProxyModel(ProxyEdgeModel, {
    Id: 'CMP-STORED-1',
    Name: 'alpha',
    ComputedName: 'CACHED',
  });

  expect((model as any).ComputedName).toBe('CACHED');
});
