// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildPartnerPageInitialValues } from './partner_page_initial_values';

test('buildPartnerPageInitialValues: prefers active company when it is enabled', () => {
  const values = buildPartnerPageInitialValues({
    activeCompanyId: 'cmp-2',
    enabledCompanyIds: ['cmp-1', 'cmp-2'],
  });
  expect(values.CompanyId).toBe('cmp-2');
  expect(values.IsActive).toBe(true);
  expect(values.IsCompany).toBe(true);
  expect(values.CustomerRank).toBe(0);
  expect(values.SupplierRank).toBe(0);
  expect(values.Contacts).toEqual([]);
  expect(values.Sequence).toBe(10);
});

test('buildPartnerPageInitialValues: falls back to first enabled company', () => {
  const values = buildPartnerPageInitialValues({
    activeCompanyId: 'missing',
    enabledCompanyIds: ['cmp-1', 'cmp-2'],
  });
  expect(values.CompanyId).toBe('cmp-1');
});

test('buildPartnerPageInitialValues: leaves CompanyId undefined without enabled companies', () => {
  const values = buildPartnerPageInitialValues({ activeCompanyId: 'cmp-1' });
  expect(values.CompanyId).toBeUndefined();
});
