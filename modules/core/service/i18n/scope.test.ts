// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test, beforeEach } from 'bun:test';
import { formatScope, resolveI18nScope, withI18nScope, __resetI18nScopeStackForTests } from './scope';

beforeEach(() => {
  __resetI18nScopeStackForTests();
});

describe('formatScope', () => {
  test('path only', () => {
    expect(formatScope('web/pages/Login')).toBe('web/pages/Login');
  });
  test('path@location', () => {
    expect(formatScope('web/pages/Login', 'title')).toBe('web/pages/Login@title');
  });
});

describe('resolveI18nScope', () => {
  test('manual scope wins', () => {
    expect(resolveI18nScope({ scope: 'game.rescue', path: 'a', location: 'b' })).toBe('game.rescue');
  });
  test('withI18nScope stack', () => {
    const got = withI18nScope('game.rescue', () => resolveI18nScope({ path: 'ignored' }));
    expect(got).toBe('game.rescue');
  });
  test('path@location fallback', () => {
    expect(resolveI18nScope({ path: 'web/pages/Login', location: 'title' })).toBe('web/pages/Login@title');
  });
});
