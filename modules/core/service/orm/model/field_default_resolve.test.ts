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

test('resolveEffectiveFieldDefaults covers empty rows, blank fields, and undefined values', () => {
  expect(resolveEffectiveFieldDefaults(null as any)).toEqual({});
  expect(resolveEffectiveFieldDefaults([{ Id: '1', Field: '  ', Value: 'x' }])).toEqual({});
  expect(resolveEffectiveFieldDefaults([{ Id: '1', Field: 'Name', UserId: '  ', CompanyId: '  ', Value: undefined }])).toEqual(
    {}
  );
  // company-only rank
  expect(
    resolveEffectiveFieldDefaults([{ Id: '1', Field: 'Name', UserId: null, CompanyId: 'C1', Value: 'co' }]).Name
  ).toBe('co');
  // nullish row / missing Field/Id optional chains
  expect(
    resolveEffectiveFieldDefaults([null as any, { Field: 'Name', Value: 'ok' } as any] as any).Name
  ).toBe('ok');
  // empty-id winner then same-rank replacement with a real Id (covers !prev.id take path)
  expect(
    resolveEffectiveFieldDefaults([
      { Field: 'Code', UserId: null, CompanyId: null, Value: 'first' } as any,
      { Id: 'z', Field: 'Code', UserId: null, CompanyId: null, Value: 'second' },
    ]).Code
  ).toBe('second');
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
