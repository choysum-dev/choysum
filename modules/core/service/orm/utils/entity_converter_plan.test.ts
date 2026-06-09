// @ts-nocheck
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from 'decimal.js';
import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { EntityConverter } from './converter';

@Model('test.EntityConverterPlanOwner')
class EntityConverterPlanOwner extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.EntityConverterPlanLine')
class EntityConverterPlanLine extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.EntityConverterPlanModel')
class EntityConverterPlanModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'datetime', column: {} })
  CreatedAt?: Date;

  @Field({ type: 'decimal', column: { precision: 16, scale: 2 } })
  Amount?: any;

  @Field({ type: 'bigint', column: {} })
  CountBig?: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => EntityConverterPlanOwner }, column: { notNull: false } })
  OwnerId?: EntityConverterPlanOwner;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => EntityConverterPlanLine, inverseField: 'ParentId' as any },
  })
  Lines?: EntityConverterPlanLine[];
}

function createModelInstance<T extends BaseModel>(ModelCtor: { new (...args: any[]): T } & typeof BaseModel, entity: Record<string, any>): T {
  const factoryToken = (ModelCtor as any).FACTORY_TOKEN;
  return new ModelCtor(factoryToken, entity, undefined as any);
}

test('EntityConverter entityToPlainObject applies scalar conversions for default public fields', () => {
  const row = {
    Id: 'row_1',
    Name: 'alpha',
    CreatedAt: new Date('2024-02-01T10:00:00.000Z'),
    Amount: new Decimal('12.34'),
    CountBig: 9n,
    internal: 'hidden',
  } as any;

  const out = EntityConverter.entityToPlainObject(EntityConverterPlanModel as any, row);

  expect(out.Name).toBe('alpha');
  expect(out.CreatedAt).toBe('2024-02-01T10:00:00.000Z');
  expect(out.Amount).toEqual({ $bigdecimal: '12.34' });
  expect(out.CountBig).toEqual({ $bigint: '9' });
  expect((out as any).internal).toBeUndefined();
  expect((out as any).OwnerId).toBeUndefined();
});

test('EntityConverter many2one fallback returns {Id} when relation payload is absent or null', () => {
  const rowNoRel = {
    Id: 'row_2',
    OwnerId: 'owner_1',
  } as any;
  const outNoRel = EntityConverter.entityToPlainObject(EntityConverterPlanModel as any, rowNoRel, ['Id', 'OwnerId'] as any);
  expect((outNoRel as any).OwnerId).toEqual({ Id: 'owner_1' });

  const rowNullRel = {
    Id: 'row_3',
    OwnerId: 'owner_2',
    $rel$OwnerId: null,
  } as any;
  const outNullRel = EntityConverter.entityToPlainObject(EntityConverterPlanModel as any, rowNullRel, ['Id', 'OwnerId'] as any);
  expect((outNullRel as any).OwnerId).toEqual({ Id: 'owner_2' });
});

test('EntityConverter to-many normalization handles null payload and primitive items', () => {
  const rowNull = {
    Id: 'row_4',
    Lines: null,
  } as any;
  const outNull = EntityConverter.entityToPlainObject(EntityConverterPlanModel as any, rowNull, ['Id', 'Lines'] as any);
  expect((outNull as any).Lines).toEqual([]);

  const rowMixed = {
    Id: 'row_5',
    Lines: JSON.stringify(['x', { Id: 'line_1', Name: 'line-1' }]),
  } as any;
  const outMixed = EntityConverter.entityToPlainObject(EntityConverterPlanModel as any, rowMixed, ['Id', 'Lines'] as any);
  expect((outMixed as any).Lines).toEqual(['x', { Id: 'line_1', Name: 'line-1' }]);
});

test('EntityConverter modelToPlainObject respects nested field selection and ignores private keys', () => {
  const child = createModelInstance(EntityConverterPlanLine as any, {
    Id: 'line_2',
    Name: 'child-visible',
  });

  const parent = createModelInstance(EntityConverterPlanModel as any, {
    Id: 'row_6',
    Name: 'parent-visible',
    Lines: [child],
    OwnerId: createModelInstance(EntityConverterPlanOwner as any, { Id: 'owner_3', Name: 'owner-visible' }),
  });

  (parent as any).hiddenLowercase = 'secret';

  const out = EntityConverter.modelToPlainObject(parent as any, ['Name', { Lines: ['Name'] }, { OwnerId: ['Name'] }, 'hiddenLowercase'] as any);
  expect(out).toEqual({
    Name: 'parent-visible',
    Lines: [{ Name: 'child-visible' }],
    OwnerId: { Name: 'owner-visible' },
  });
});
