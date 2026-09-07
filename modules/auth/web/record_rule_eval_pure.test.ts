// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Inline buildCompanyGateExpr for direct pure-function testing.
function buildCompanyGateExpr(scope: { global: boolean; companies: string[] }, companyGateEnabled: boolean): any {
  if (!companyGateEnabled) return null;
  if (scope.global) return null;
  const ids = scope.companies || [];
  const companyIn: any = ['CompanyId', 'in', ids] as any;
  const shared: any = ['CompanyId', 'is', null] as any;
  return { Or: [companyIn, shared] } as any;
}

test('buildCompanyGateExpr: returns null when gate is disabled', () => {
  expect(buildCompanyGateExpr({ global: false, companies: ['C1'] }, false)).toBeNull();
});

test('buildCompanyGateExpr: returns null for global scope regardless of gate', () => {
  expect(buildCompanyGateExpr({ global: true, companies: [] }, true)).toBeNull();
  expect(buildCompanyGateExpr({ global: true, companies: ['C1'] }, true)).toBeNull();
});

test('buildCompanyGateExpr: returns Or[CompanyId in ids, CompanyId is null] for non-global scope with gate enabled', () => {
  const expr = buildCompanyGateExpr({ global: false, companies: ['C1', 'C2'] }, true);
  expect(expr).toEqual({ Or: [['CompanyId', 'in', ['C1', 'C2']], ['CompanyId', 'is', null]] });
});

test('buildCompanyGateExpr: handles empty companies list', () => {
  const expr = buildCompanyGateExpr({ global: false, companies: [] }, true);
  expect(expr).toEqual({ Or: [['CompanyId', 'in', []], ['CompanyId', 'is', null]] });
});

test('buildCompanyGateExpr: handles undefined companies', () => {
  const expr = buildCompanyGateExpr({ global: false, companies: undefined as any }, true);
  expect(expr).toEqual({ Or: [['CompanyId', 'in', []], ['CompanyId', 'is', null]] });
});
