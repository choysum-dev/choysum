// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { EntityConverter } from './converter';

@Model('test.EntityConverterTransportParent')
class EntityConverterTransportParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => EntityConverterTransportChild, inverseField: 'ParentId' },
  })
  Childs?: EntityConverterTransportChild[];
}

@Model('test.EntityConverterTransportChild')
class EntityConverterTransportChild extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => EntityConverterTransportParent },
    column: { notNull: false },
  })
  ParentId?: EntityConverterTransportParent;
}

@Model('test.EntityConverterTransportNoTarget')
class EntityConverterTransportNoTarget extends BaseModel {
  @Field({ type: 'ManyToOne', relation: {} as any, column: { notNull: false } })
  OwnerId?: any;
}

@Model('test.EntityConverterTransportNoTargetMany')
class EntityConverterTransportNoTargetMany extends BaseModel {
  @Field({ type: 'ManyToMany', relation: {} as any })
  Tags?: any[];
}

@Model('test.EntityConverterTransportBrokenOne')
class EntityConverterTransportBrokenOne extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => undefined as any }, column: { notNull: false } })
  OwnerId?: any;
}

@Model('test.EntityConverterTransportBrokenMany')
class EntityConverterTransportBrokenMany extends BaseModel {
  @Field({ type: 'OneToMany', relation: { targetModel: () => undefined as any, inverseField: 'ParentId' } as any })
  Childs?: any[];
}

@Model('test.EntityConverterTransportTag')
class EntityConverterTransportTag extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.EntityConverterTransportGroup')
class EntityConverterTransportGroup extends BaseModel {
  @Field({ type: 'ManyToMany', relation: { targetModel: () => EntityConverterTransportTag } as any })
  Tags?: EntityConverterTransportTag[];
}

@Model('test.EntityConverterTransportPrivate')
class EntityConverterTransportPrivate extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  privateField?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => EntityConverterTransportParent }, column: {} })
  OwnerId?: EntityConverterTransportParent;
}

function createModelInstance<T extends BaseModel>(ModelCtor: { new (...args: any[]): T } & typeof BaseModel, entity: Record<string, any>): T {
  const factoryToken = (ModelCtor as any).FACTORY_TOKEN;
  return new ModelCtor(factoryToken, entity as any, undefined as any);
}

test('EntityConverter transport fallback uses direct to-many field when no $rel$ alias exists (json string)', () => {
  const row = {
    Id: 'parent_json',
    Childs: JSON.stringify([{ Id: 'child_json_1', Name: 'child-from-json' }]),
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportParent as any, row as any, ['Id', 'Childs'] as any);
  const childs = (out as any).Childs;

  expect(Array.isArray(childs)).toBe(true);
  expect((childs || []).length).toBe(1);
  expect(childs[0]?.Id).toBe('child_json_1');
  expect(childs[0]?.Name).toBe('child-from-json');
});

test('EntityConverter transport fallback uses direct to-many field object wrapper (values)', () => {
  const row = {
    Id: 'parent_values',
    Childs: {
      values: [{ Id: 'child_values_1', Name: 'child-from-values' }],
    },
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportParent as any, row as any, ['Id', 'Childs'] as any);
  const childs = (out as any).Childs;

  expect(Array.isArray(childs)).toBe(true);
  expect((childs || []).length).toBe(1);
  expect(childs[0]?.Id).toBe('child_values_1');
  expect(childs[0]?.Name).toBe('child-from-values');
});

test('EntityConverter transport fallback uses direct to-many field numeric-key object', () => {
  const row = {
    Id: 'parent_numeric',
    Childs: {
      0: { Id: 'child_numeric_1', Name: 'child-from-numeric-1' },
      1: { Id: 'child_numeric_2', Name: 'child-from-numeric-2' },
    },
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportParent as any, row as any, ['Id', 'Childs'] as any);
  const childs = (out as any).Childs;

  expect(Array.isArray(childs)).toBe(true);
  expect((childs || []).length).toBe(2);
  expect(childs[0]?.Id).toBe('child_numeric_1');
  expect(childs[1]?.Id).toBe('child_numeric_2');
});

test('EntityConverter transport fallback uses direct to-many field object wrapper (items)', () => {
  const row = {
    Id: 'parent_items',
    Childs: {
      items: [{ Id: 'child_items_1', Name: 'child-from-items' }],
    },
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportParent as any, row as any, ['Id', 'Childs'] as any);
  const childs = (out as any).Childs;

  expect(Array.isArray(childs)).toBe(true);
  expect((childs || []).length).toBe(1);
  expect(childs[0]?.Id).toBe('child_items_1');
  expect(childs[0]?.Name).toBe('child-from-items');
});

test('EntityConverter transport fallback normalizes unsupported to-many object payload to empty array', () => {
  const row = {
    Id: 'parent_invalid_obj',
    Childs: {
      foo: 'bar',
    },
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportParent as any, row as any, ['Id', 'Childs'] as any);
  const childs = (out as any).Childs;

  expect(Array.isArray(childs)).toBe(true);
  expect((childs || []).length).toBe(0);
});

test('EntityConverter transport many2one keeps invalid JSON alias payload as raw string', () => {
  const row = {
    Id: 'child_alias_invalid',
    ParentId: 'parent_scalar',
    $rel$_parent_id: '{invalid-json',
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportChild as any, row as any, ['Id', 'ParentId'] as any);

  expect((out as any).ParentId).toBe('{invalid-json');
});

test('EntityConverter transport entityArrayToPlainObject applies the same compiled plan for each row', () => {
  const rows = [
    {
      Id: 'parent_batch_1',
      Childs: JSON.stringify([{ Id: 'child_batch_1', Name: 'first' }]),
    },
    {
      Id: 'parent_batch_2',
      Childs: {
        items: [{ Id: 'child_batch_2', Name: 'second' }],
      },
    },
  ];

  const out = EntityConverter.entityArrayToPlainObject(EntityConverterTransportParent as any, rows as any, ['Id', 'Childs'] as any);

  expect(out.length).toBe(2);
  expect((out[0] as any).Childs?.[0]?.Id).toBe('child_batch_1');
  expect((out[1] as any).Childs?.[0]?.Name).toBe('second');
});

test('EntityConverter transport keeps scalar many2one value when relation target model is missing', () => {
  const row = {
    Id: 'notarget_1',
    OwnerId: 'owner_scalar_1',
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportNoTarget as any, row as any, ['Id', 'OwnerId'] as any);

  expect((out as any).OwnerId).toBe('owner_scalar_1');
});

test('EntityConverter transport normalizes alias payload when relation target model is missing', () => {
  const row = {
    Id: 'notarget_2',
    $rel$_owner_id: new Date('2024-03-02T08:09:10.000Z'),
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportNoTarget as any, row as any, ['Id', 'OwnerId'] as any);

  expect((out as any).OwnerId).toBe('2024-03-02T08:09:10.000Z');
});

test('EntityConverter transport normalizes to-many relation payload when target model is missing', () => {
  const row = {
    Id: 'notarget_many_1',
    Tags: [new Date('2024-03-03T01:02:03.000Z'), BigInt('9'), { nested: { a: 1 } }],
  };

  const out = EntityConverter.entityToPlainObject(EntityConverterTransportNoTargetMany as any, row as any, ['Id', 'Tags'] as any);

  expect(Array.isArray((out as any).Tags)).toBe(true);
  expect((out as any).Tags?.[0]).toBe('2024-03-03T01:02:03.000Z');
  expect((out as any).Tags?.[1]).toEqual({ $bigint: '9' });
  expect((out as any).Tags?.[2]?.nested?.a).toBe(1);
});

test('EntityConverter private scalar serialize conversion normalizes primitives and rich values', () => {
  const applyScalarConv = (EntityConverter as any).applyScalarConv as (value: any, conv: string) => any;

  expect(applyScalarConv('plain', 'serialize')).toBe('plain');
  expect(applyScalarConv(true, 'serialize')).toBe(true);
  expect(applyScalarConv(new Date('2024-03-04T00:00:00.000Z'), 'serialize')).toBe('2024-03-04T00:00:00.000Z');
  expect(applyScalarConv(BigInt('123'), 'serialize')).toEqual({ $bigint: '123' });
  expect(applyScalarConv({ v: 1 }, 'serialize')).toEqual({ v: 1 });
});

test('EntityConverter private parse/canonicalize/infer helpers cover fallback branches', () => {
  const parseJsonRelationValue = (EntityConverter as any).parseJsonRelationValue as (value: any) => any;
  const canonicalizeFields = (EntityConverter as any).canonicalizeFields as (sel?: any[]) => string;
  const inferScalarConvByType = (EntityConverter as any).inferScalarConvByType as (type?: string) => string;

  expect(parseJsonRelationValue('')).toBe('');
  expect(parseJsonRelationValue('plain')).toBe('plain');
  expect(parseJsonRelationValue('{invalid')).toBe('{invalid');
  expect(parseJsonRelationValue('{"a":1}')).toEqual({ a: 1 });

  expect(canonicalizeFields(undefined)).toBe('__ALL_PUBLIC__');
  expect(canonicalizeFields([{ OwnerId: ['Name'] }, 'Id', { OwnerId: ['DisplayName'] }])).toBe('["Id",{"OwnerId":["Name"]},{"OwnerId":["DisplayName"]}]');

  expect(inferScalarConvByType('datetime')).toBe('date');
  expect(inferScalarConvByType('decimal')).toBe('decimal');
  expect(inferScalarConvByType('bigint')).toBe('bigint');
  expect(inferScalarConvByType('varchar')).toBe('none');
});

test('EntityConverter private relation helpers cover alias lookup and to-many normalization branches', () => {
  const getPreloadedRelationValue = (EntityConverter as any).getPreloadedRelationValue as (row: Record<string, any>, fieldName: string) => any;
  const normalizeToManyRelationValue = (EntityConverter as any).normalizeToManyRelationValue as (raw: any) => any[] | undefined;

  expect(getPreloadedRelationValue({ $rel$_owner_id: { Id: 'U1' } }, 'OwnerId')).toEqual({ Id: 'U1' });
  expect(getPreloadedRelationValue({}, 'OwnerId')).toBe(undefined);

  const callNormalize = (raw: any) => normalizeToManyRelationValue.call(EntityConverter, raw);

  expect(callNormalize(undefined)).toBe(undefined);
  expect(callNormalize(null)).toEqual([]);
  expect(callNormalize('[{"Id":"A"}]')).toEqual([{ Id: 'A' }]);
  expect(callNormalize('{"value":[{"Id":"B"}]}')).toEqual([{ Id: 'B' }]);
  expect(callNormalize('{"0":{"Id":"C"},"2":{"Id":"D"}}')).toEqual([{ Id: 'C' }, { Id: 'D' }]);
  expect(callNormalize('raw-text')).toEqual([]);
});

test('EntityConverter private transport helper branches handle null/date/bigint and undefined relation target', () => {
  const applyScalarConv = (EntityConverter as any).applyScalarConv as (value: any, conv: string) => any;
  const hydrateRelationTarget = (EntityConverter as any).hydrateRelationTarget as (targetCtor: any, value: any) => any;

  expect(applyScalarConv(null, 'serialize')).toBe(null);
  expect(applyScalarConv(new Date('2024-03-05T00:00:00.000Z'), 'serialize')).toBe('2024-03-05T00:00:00.000Z');
  expect(applyScalarConv(BigInt('7'), 'serialize')).toEqual({ $bigint: '7' });
  expect(applyScalarConv({ nested: { ok: true } }, 'serialize')).toEqual({ nested: { ok: true } });

  expect(hydrateRelationTarget(undefined, { Id: 'X' })).toBe(undefined);
});

test('EntityConverter private non-rel public field cache branch reuses cached field list', () => {
  const getNonRelPublicFields = (EntityConverter as any).getNonRelPublicFields as (ctor: any) => string[];

  const first = getNonRelPublicFields.call(EntityConverter, EntityConverterTransportParent as any);
  const second = getNonRelPublicFields.call(EntityConverter, EntityConverterTransportParent as any);

  expect(second).toBe(first);
  expect(first.includes('Name')).toBe(true);
  expect(first.includes('Childs')).toBe(false);
});

test('EntityConverter entityToModel handles relation null/undefined/non-array and missing target branches', () => {
  const brokenOne = createModelInstance(EntityConverterTransportBrokenOne as any, {
    Id: 'BROKEN-ONE',
    OwnerId: { Id: 'U-1', Name: 'ignored' },
  });
  expect((brokenOne as any).OwnerId).toBe(undefined);

  const brokenMany = createModelInstance(EntityConverterTransportBrokenMany as any, {
    Id: 'BROKEN-MANY',
    Childs: 'not-array',
  });
  expect((brokenMany as any).Childs).toBe(undefined);

  const childNull = createModelInstance(EntityConverterTransportChild as any, {
    Id: 'C-NULL',
    ParentId: null,
  });
  expect((childNull as any).ParentId).toBe(null);

  const childUndefined = createModelInstance(EntityConverterTransportChild as any, {
    Id: 'C-UNDEF',
    ParentId: undefined,
  });
  expect((childUndefined as any).ParentId).toBe(undefined);

  const parentPreloadedNull = createModelInstance(
    EntityConverterTransportParent as any,
    {
      Id: 'P-PRELOAD-NULL',
      $rel$Childs: null,
    } as any
  );
  expect((parentPreloadedNull as any).Childs).toBe(null);
});

test('EntityConverter executePlan jsonSafe relation branches cover one/many fallback paths', () => {
  const executePlan = (EntityConverter as any).executePlan as (plan: any, row: Record<string, any>) => Record<string, any>;

  const plan: any = {
    jsonSafe: true,
    ops: [
      { kind: 'scalar', key: 'Id', conv: 'none' },
      { kind: 'relation', key: 'OwnerId', cardinality: 'one' },
      { kind: 'relation', key: 'Tags', cardinality: 'many' },
      {
        kind: 'relation',
        key: 'Childs',
        cardinality: 'many',
        childPlan: { jsonSafe: true, ops: [{ kind: 'scalar', key: 'Id', conv: 'none' }] },
      },
    ],
  };

  const row = {
    Id: 'R1',
    OwnerId: 'U1',
    Tags: undefined,
    Childs: [{ Id: 'C1', Name: 'ignore' }, 'raw'],
  } as any;

  const out = executePlan.call(EntityConverter, plan, row);

  expect(out.Id).toBe('R1');
  expect(out.OwnerId).toBe('U1');
  expect(out.Tags).toBe(undefined);
  expect(out.Childs).toEqual([{ Id: 'C1' }, 'raw']);
});

test('EntityConverter applyScalarConv covers undefined and non-matching date/bigint branches', () => {
  const applyScalarConv = (EntityConverter as any).applyScalarConv as (value: any, conv: string) => any;

  expect(applyScalarConv(undefined, 'none')).toBe(undefined);
  expect(applyScalarConv(BigInt('5'), 'none')).toEqual({ $bigint: '5' });
  expect(applyScalarConv('2024-03-01', 'date')).toBe('2024-03-01');
  expect(applyScalarConv(9, 'bigint')).toBe(9);
});

test('EntityConverter modelToPlainObject without fields keeps public data and skips private/functions', () => {
  const rec = createModelInstance(EntityConverterTransportParent as any, {
    Id: 'P-MODEL-1',
    Name: 'Parent',
  });

  (rec as any).Name = 'Parent';
  (rec as any).Childs = [createModelInstance(EntityConverterTransportChild as any, { Id: 'C-MODEL-1', Name: 'Child' })];
  (rec as any).privateName = 'hidden';
  (rec as any).Fn = () => 'ignore';

  const plain = EntityConverter.modelToPlainObject(rec as any);
  expect((plain as any).Name).toBe('Parent');
  expect(Array.isArray((plain as any).Childs)).toBe(true);
  expect((plain as any).Childs?.[0]?.Id).toBe('C-MODEL-1');
  expect((plain as any).privateName).toBe(undefined);
  expect((plain as any).Fn).toBe(undefined);
});

test('EntityConverter entityToModel covers direct one-to-many and many-to-many mapping fallbacks', () => {
  const originalHydrateRelationTarget = (EntityConverter as any).hydrateRelationTarget;

  try {
    (EntityConverter as any).hydrateRelationTarget = (_targetCtor: any, value: any) => {
      if (value && typeof value === 'object' && 'Id' in value) {
        return value;
      }
      return undefined;
    };

    const group = createModelInstance(EntityConverterTransportGroup as any, {
      Id: 'G1',
      Tags: [{ Id: 'T1', Name: 'tag-1' }, 'raw-tag'],
    });
    expect((group as any).Tags).toEqual([{ Id: 'T1', Name: 'tag-1' }, 'raw-tag']);

    const parent = createModelInstance(
      EntityConverterTransportParent as any,
      {
        Id: 'P1',
        Childs: [{ Id: 'C1', Name: 'child-1' }, 'raw-child'],
      } as any
    );
    expect((parent as any).Childs).toEqual([{ Id: 'C1', Name: 'child-1' }, 'raw-child']);
  } finally {
    (EntityConverter as any).hydrateRelationTarget = originalHydrateRelationTarget;
  }
});

test('EntityConverter relation alias and JSON parse fallbacks cover null JSON and sparse field signatures', () => {
  const normalizeToManyRelationValue = (EntityConverter as any).normalizeToManyRelationValue as (raw: any) => any[] | undefined;
  const canonicalizeFields = (EntityConverter as any).canonicalizeFields as (sel?: any[]) => string;

  expect(normalizeToManyRelationValue.call(EntityConverter, 'null')).toEqual([]);
  expect(normalizeToManyRelationValue.call(EntityConverter, '{"items":null}')).toEqual([]);

  // When a relation object's sub-selection is not an array, canonicalization falls back to an empty array.
  expect(canonicalizeFields([{ OwnerId: 'bad-subsel' } as any])).toBe('[{"OwnerId":[]}]');
});

test('EntityConverter compile and execute plan guard branches keep public data and fallback raw relation values', () => {
  const plain = EntityConverter.entityToPlainObject(
    EntityConverterTransportPrivate as any,
    {
      Id: 'PR-1',
      Name: 'public',
      privateField: 'hidden',
      OwnerId: 'U-RAW',
      $rel$_owner_id: undefined,
    } as any,
    undefined as any
  );

  // Without explicit fields, only uppercase-leading non-relation fields are emitted.
  expect((plain as any).Name).toBe('public');
  expect((plain as any).privateField).toBe(undefined);

  const loweredRelation = EntityConverter.entityToPlainObject(
    EntityConverterTransportPrivate as any,
    {
      Name: 'keep',
      ownerId: { Id: 'U-1' },
    } as any,
    [{ ownerId: ['Id'] } as any]
  );
  expect(loweredRelation).toEqual({});

  const group = createModelInstance(EntityConverterTransportGroup as any, {
    Id: 'G-RAW',
    Tags: [{ Id: 'T1', Name: 'tag-1' }, 'raw-tag'],
  });
  expect((group as any).Tags?.[0] instanceof EntityConverterTransportTag).toBe(true);
  const secondTag = (group as any).Tags?.[1];
  expect(secondTag == null).toBe(false);
  if (typeof secondTag === 'string') {
    expect(secondTag).toBe('raw-tag');
  } else {
    expect((secondTag as any).entity ?? (secondTag as any).Id).toBe('raw-tag');
  }
});
