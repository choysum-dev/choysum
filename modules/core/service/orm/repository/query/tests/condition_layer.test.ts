// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  andRepositoryConditions,
  applyRepositoryDefaultLayers,
  applyRepositorySoftDeleteLayer,
  isEmptyRepositoryCondition,
  repositorySoftDeleteEnabled,
} from '..';

test('repository condition layer empty-condition helpers cover empty object and nested Not/Or forms', () => {
  expect(isEmptyRepositoryCondition(undefined as any)).toBe(true);
  expect(isEmptyRepositoryCondition({} as any)).toBe(true);
  expect(isEmptyRepositoryCondition({ Not: {} } as any)).toBe(true);
  expect(isEmptyRepositoryCondition({ Or: [[], { And: [[]] }] } as any)).toBe(true);
  expect(isEmptyRepositoryCondition({ Not: ['Id', '=', '1'] } as any)).toBe(false);

  expect(
    andRepositoryConditions([] as any, { And: [['Name', '=', 'demo'] as any, [] as any] } as any, { And: [['Status', '=', 'draft'] as any] } as any)
  ).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['Status', '=', 'draft'],
    ],
  });
});

test('repository condition layer applies default layers and respects soft-delete global disable', () => {
  const meta = {
    softDelete: true,
  } as any;
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SOFT_DELETE_ENABLED: 'false' };
    expect(repositorySoftDeleteEnabled(meta)).toBe(false);
    expect(applyRepositorySoftDeleteLayer({ meta, softField: 'DeletedAt', includeDeleted: false, onlyDeletedMode: false }, ['Id', '=', '1'] as any)).toEqual([
      'Id',
      '=',
      '1',
    ]);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SOFT_DELETE_ENABLED: true };
    expect(
      applyRepositoryDefaultLayers(
        {
          meta,
          softField: 'DeletedAt',
          includeDeleted: false,
          onlyDeletedMode: true,
          applyCompanyLayer: condition => ({ And: [condition, ['CompanyId', '=', 'company_a']] }),
        },
        ['Id', '=', '1'] as any
      )
    ).toEqual({
      And: [
        ['Id', '=', '1'],
        ['CompanyId', '=', 'company_a'],
        ['DeletedAt', 'is not', null],
      ],
    });
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository condition layer covers global env string-true/model default and and-composer tail branches', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SOFT_DELETE_ENABLED: 'TRUE' };
    expect(repositorySoftDeleteEnabled({} as any)).toBe(true);

    const makeFlakyAnd = () => {
      let reads = 0;
      return {
        get And() {
          reads += 1;
          return reads === 1 ? [['Id', '=', '1']] : [];
        },
      } as any;
    };

    expect(andRepositoryConditions(makeFlakyAnd(), makeFlakyAnd())).toEqual([]);
    expect(andRepositoryConditions(['Name', '=', 'demo'] as any, makeFlakyAnd())).toEqual(['Name', '=', 'demo']);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('repository condition layer falls back when env is non-boolean/non-string and supports fully empty and-composition', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_SOFT_DELETE_ENABLED: 0 as any };
    expect(repositorySoftDeleteEnabled({ softDelete: true } as any)).toBe(true);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }

  expect(andRepositoryConditions(undefined as any, [] as any, {} as any)).toEqual([]);
});
