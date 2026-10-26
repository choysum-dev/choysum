// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { __attachMonetaryCurrencyCompanionsForTest } from './formController';

test('attachMonetaryCurrencyCompanions restores CurrencyId cleared by display-only draft', () => {
  const payload: Record<string, unknown> = {
    Amount: '12.3456',
    CurrencyId: null,
  };
  __attachMonetaryCurrencyCompanionsForTest(
    payload,
    { CurrencyId: { DisplayName: '人民币' } },
    name => (name === 'Amount' ? { type: 'monetary', currencyField: 'CurrencyId' } : undefined),
    { CurrencyId: 'cur-cny-1' }
  );
  expect(payload.CurrencyId).toBe('cur-cny-1');
});

test('attachMonetaryCurrencyCompanions prefers draft id over original', () => {
  const payload: Record<string, unknown> = { Amount: '1' };
  __attachMonetaryCurrencyCompanionsForTest(
    payload,
    { CurrencyId: { Id: 'from-draft' } },
    name => (name === 'Amount' ? { type: 'monetary', currencyField: 'CurrencyId' } : undefined),
    { CurrencyId: 'from-original' }
  );
  expect(payload.CurrencyId).toBe('from-draft');
});

test('attachMonetaryCurrencyCompanions no-ops without currencyField meta', () => {
  const payload: Record<string, unknown> = { Amount: '1' };
  __attachMonetaryCurrencyCompanionsForTest(
    payload,
    { CurrencyId: 'cur-1' },
    name => (name === 'Amount' ? { type: 'monetary' } : undefined),
    { CurrencyId: 'cur-1' }
  );
  expect(payload.CurrencyId).toBeUndefined();
});
