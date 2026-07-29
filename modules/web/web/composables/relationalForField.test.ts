// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { buildRelationalForField } from './relationalForField';

describe('buildRelationalForField', () => {
  it('builds forField from fullModelName + top-level prop', () => {
    expect(buildRelationalForField({ fullModelName: 'sale.SaleOrder', modelName: 'SaleOrder' }, 'BankAccountId')).toEqual({
      forField: { model: 'sale.SaleOrder', field: 'BankAccountId' },
    });
  });

  it('falls back to modelName', () => {
    expect(buildRelationalForField({ modelName: 'SaleOrder' }, 'PartnerId')).toEqual({
      forField: { model: 'SaleOrder', field: 'PartnerId' },
    });
  });

  it('skips nested dotted props', () => {
    expect(buildRelationalForField({ fullModelName: 'sale.SaleOrder' }, 'PartnerId.CountryId')).toEqual({});
  });

  it('returns empty when model or field missing', () => {
    expect(buildRelationalForField({}, 'PartnerId')).toEqual({});
    expect(buildRelationalForField({ modelName: 'SaleOrder' }, '')).toEqual({});
  });
});
