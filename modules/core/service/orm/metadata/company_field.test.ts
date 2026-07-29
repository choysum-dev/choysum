// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { validateModelCompanyField } from './company_field';

test('validateModelCompanyField returns early when companyField is empty', () => {
  expect(() => validateModelCompanyField({ fields: new Map() } as any)).not.toThrow();
  expect(() => validateModelCompanyField({ companyField: '  ', fields: new Map() } as any)).not.toThrow();
});

test('validateModelCompanyField rejects missing field and non-Map fields', () => {
  expect(() =>
    validateModelCompanyField({
      fullModelName: 'demo.A',
      companyField: 'CompanyId',
      fields: new Map([['Name', {}]]),
    } as any)
  ).toThrow(/companyField "CompanyId" does not exist/);

  expect(() =>
    validateModelCompanyField({
      modelName: 'B',
      companyField: 'CompanyId',
      fields: { CompanyId: {} } as any,
    } as any)
  ).toThrow(/@Model\(B\) companyField "CompanyId"/);

  expect(() =>
    validateModelCompanyField({
      name: '',
      companyField: 'OwningCompanyId',
      fields: undefined as any,
    } as any)
  ).toThrow(/@Model\(model\) companyField "OwningCompanyId"/);
});

test('validateModelCompanyField accepts existing ownership field', () => {
  expect(() =>
    validateModelCompanyField({
      fullModelName: 'demo.Ok',
      companyField: 'CompanyId',
      fields: new Map([['CompanyId', { type: 'char' }]]),
    } as any)
  ).not.toThrow();
});
