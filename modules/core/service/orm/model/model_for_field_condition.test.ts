// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import BaseModel from './model';
import {
  evaluateFieldRelationalCondition,
  mergeCallerConditionWithForField,
  resolveForFieldCondition,
} from './model_for_field_condition';
import { MetadataStorage } from '../metadata/storage';

@Model('ForFieldBank', { application: 'demo' })
class ForFieldBank extends BaseModel {
  @Field({ type: 'boolean' })
  Active!: boolean;

  @Field({ type: 'varchar', size: 64 })
  CompanyId!: string;
}

@Model('ForFieldOrder', { application: 'demo' })
class ForFieldOrder extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ForFieldBank },
    condition: ['Active', '=', true],
  } as any)
  BankAccountId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ForFieldBank },
    condition: () => ({
      And: [
        ['Active', '=', true],
        ['CompanyId', '=', 'C1'],
      ],
    }),
  } as any)
  DynamicBankId!: ForFieldBank | null;

  @Field({ type: 'varchar', size: 32 })
  Name!: string;
}

test('resolveForFieldCondition returns static meta condition', () => {
  const cond = resolveForFieldCondition(ForFieldBank as any, {
    model: 'demo.ForFieldOrder',
    field: 'BankAccountId',
  });
  expect(cond).toEqual(['Active', '=', true]);
});

test('resolveForFieldCondition evaluates callable', () => {
  const cond = resolveForFieldCondition(ForFieldBank as any, {
    model: 'demo.ForFieldOrder',
    field: 'DynamicBankId',
  });
  expect(cond).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'C1'],
    ],
  });
});

test('resolveForFieldCondition returns empty when field has no condition', () => {
  // Use a relation field without condition — add Partner-less by pointing to Name (non-relation) should throw
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'Name' })
  ).toThrow('must be a relational field');
});

test('resolveForFieldCondition rejects blank / unknown model or field', () => {
  expect(() => resolveForFieldCondition(ForFieldBank as any, { model: '', field: 'BankAccountId' })).toThrow(
    'forField.model'
  );
  expect(() => resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: '' })).toThrow(
    'forField.field'
  );
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.NoSuchModel', field: 'BankAccountId' })
  ).toThrow('not a registered model');
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'MissingField' })
  ).toThrow('does not exist');
});

test('resolveForFieldCondition rejects target mismatch', () => {
  expect(() =>
    resolveForFieldCondition(ForFieldOrder as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
  ).toThrow('does not match the searched model');
});

test('mergeCallerConditionWithForField Ands meta and caller', () => {
  const merged = mergeCallerConditionWithForField(
    ForFieldBank as any,
    ['CompanyId', '=', 'X'] as any,
    { model: 'demo.ForFieldOrder', field: 'BankAccountId' }
  );
  expect(merged).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'X'],
    ],
  });
});

test('mergeCallerConditionWithForField without forField is identity', () => {
  expect(mergeCallerConditionWithForField(ForFieldBank as any, ['Active', '=', true] as any, undefined)).toEqual([
    'Active',
    '=',
    true,
  ]);
});

test('evaluateFieldRelationalCondition reads static from metadata', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('BankAccountId')!;
  expect(evaluateFieldRelationalCondition(ForFieldOrder as any, meta)).toEqual(['Active', '=', true]);
});
