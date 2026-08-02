// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveEffectiveFieldDefaults } from './field_default_resolve';

test('resolveEffectiveFieldDefaults prefers user+company over weaker scopes', () => {
  const out = resolveEffectiveFieldDefaults([
    { Id: 'g', Field: 'Name', UserId: null, CompanyId: null, Value: 'global' },
    { Id: 'u', Field: 'Name', UserId: 'U1', CompanyId: null, Value: 'user' },
    { Id: 'c', Field: 'Name', UserId: null, CompanyId: 'C1', Value: 'company' },
    { Id: 'uc', Field: 'Name', UserId: 'U1', CompanyId: 'C1', Value: 'user-co' },
  ]);
  expect(out.Name).toBe('user-co');
});

test('resolveEffectiveFieldDefaults filters by fieldNames and ties break on smallest Id', () => {
  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(x => String(x)).join(' '));
  };

  try {
    const out = resolveEffectiveFieldDefaults(
      [
        { Id: 'b', Field: 'Name', UserId: null, CompanyId: null, Value: 'B' },
        { Id: 'a', Field: 'Name', UserId: null, CompanyId: null, Value: 'A' },
        { Id: 'x', Field: 'Code', UserId: null, CompanyId: null, Value: 'skip-me' },
      ],
      ['Name']
    );
    expect(out).toEqual({ Name: 'A' });
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_SCOPE_TIE'))).toBe(true);

    // Opposite row order must keep the same winner (smallest Id) and still warn.
    warnings.length = 0;
    const outReversed = resolveEffectiveFieldDefaults(
      [
        { Id: 'a', Field: 'Name', UserId: null, CompanyId: null, Value: 'A' },
        { Id: 'b', Field: 'Name', UserId: null, CompanyId: null, Value: 'B' },
      ],
      ['Name']
    );
    expect(outReversed).toEqual({ Name: 'A' });
    expect(warnings.some(msg => msg.includes('FIELD_DEFAULT_SCOPE_TIE'))).toBe(true);
  } finally {
    console.warn = originalWarn;
  }
});
