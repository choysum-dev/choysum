// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildRelationConditionSource } from './relationalForField';

test('buildRelationConditionSource: builds relationConditionSource from fullModelName + top-level prop', () => {
  expect(buildRelationConditionSource({ fullModelName: 'sale.SaleOrder', modelName: 'SaleOrder' }, 'BankAccountId')).toEqual({
    relationConditionSource: { model: 'sale.SaleOrder', field: 'BankAccountId' },
  });
});

test('buildRelationConditionSource: falls back to modelName', () => {
  expect(buildRelationConditionSource({ modelName: 'SaleOrder' }, 'PartnerId')).toEqual({
    relationConditionSource: { model: 'SaleOrder', field: 'PartnerId' },
  });
});

test('buildRelationConditionSource: skips nested dotted props', () => {
  expect(buildRelationConditionSource({ fullModelName: 'sale.SaleOrder' }, 'PartnerId.CountryId')).toEqual({});
});

test('buildRelationConditionSource: returns empty when model or field missing', () => {
  expect(buildRelationConditionSource({}, 'PartnerId')).toEqual({});
  expect(buildRelationConditionSource({ modelName: 'SaleOrder' }, '')).toEqual({});
  expect(buildRelationConditionSource({ modelName: 'SaleOrder' }, undefined)).toEqual({});
  expect(buildRelationConditionSource({ modelName: 'SaleOrder' }, null)).toEqual({});
});

