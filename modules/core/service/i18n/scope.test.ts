// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatScope, resolveI18nScope, withI18nScope, __resetI18nScopeStackForTests } from './scope';

function withResetScope(run: () => void): void {
  __resetI18nScopeStackForTests();
  try {
    run();
  } finally {
    __resetI18nScopeStackForTests();
  }
}

test('i18n formatScope: path only', () => {
  withResetScope(() => {
    expect(formatScope('web/pages/Login')).toBe('web/pages/Login');
  });
});

test('i18n formatScope: path@location', () => {
  withResetScope(() => {
    expect(formatScope('web/pages/Login', 'title')).toBe('web/pages/Login@title');
  });
});

test('i18n resolveI18nScope: manual scope wins', () => {
  withResetScope(() => {
    expect(resolveI18nScope({ scope: 'game.rescue', path: 'a', location: 'b' })).toBe('game.rescue');
  });
});

test('i18n resolveI18nScope: withI18nScope stack', () => {
  withResetScope(() => {
    const got = withI18nScope('game.rescue', () => resolveI18nScope({ path: 'ignored' }));
    expect(got).toBe('game.rescue');
  });
});

test('i18n resolveI18nScope: path@location fallback', () => {
  withResetScope(() => {
    expect(resolveI18nScope({ path: 'web/pages/Login', location: 'title' })).toBe('web/pages/Login@title');
  });
});
