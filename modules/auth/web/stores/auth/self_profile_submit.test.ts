// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { runSelfProfileSubmit } from './self_profile_submit';

function deps(overrides?: Partial<Parameters<typeof runSelfProfileSubmit>[0]>) {
  const calls: string[] = [];
  return {
    calls,
    opts: {
      userId: 'user-1',
      hasCurrentUserId: true,
      missingUserMessage: 'missing id',
      defaultSubmit: async () => {
        calls.push('defaultSubmit');
        return {
          LanguageId: 'lang-1',
          Timezone: 'Asia/Shanghai',
          Theme: 'dark',
          Density: 'compact',
        };
      },
      formData: {
        LanguageId: 'lang-draft',
        Timezone: 'UTC',
        Theme: 'light',
        Density: 'standard',
      } as Record<string, unknown>,
      patchCurrentUser: (languageId: unknown, timezone: unknown, theme: unknown, density: unknown) => {
        calls.push(`patch:${String(languageId)}:${String(timezone)}:${String(theme)}:${String(density)}`);
      },
      applyAppearance: (theme: unknown, density: unknown) => {
        calls.push(`appearance:${String(theme)}:${String(density)}`);
      },
      applyLanguage: async (languageId: unknown) => {
        calls.push(`apply:${String(languageId)}`);
      },
      refreshToken: async () => {
        calls.push('refreshToken');
      },
      afterLocaleChange: async () => {
        calls.push('afterLocaleChange');
      },
      onSuccess: () => {
        calls.push('success');
      },
      onError: (message: string) => {
        calls.push(`error:${message}`);
      },
      failedMessage: 'failed',
      ...overrides,
    },
  };
}

test('runSelfProfileSubmit: missing user id errors without Write', async () => {
  const { calls, opts } = deps({ userId: '' });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(false);
  expect(calls).toEqual(['error:missing id']);
});

test('runSelfProfileSubmit: defaultSubmit then language/token side effects', async () => {
  const { calls, opts } = deps();
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(true);
  expect(calls).toEqual([
    'defaultSubmit',
    'patch:lang-1:Asia/Shanghai:dark:compact',
    'appearance:dark:compact',
    'apply:lang-1',
    'refreshToken',
    'afterLocaleChange',
    'success',
  ]);
});

test('runSelfProfileSubmit: normalizes ManyToOne LanguageId before side effects', async () => {
  const { calls, opts } = deps({
    defaultSubmit: async () => {
      calls.push('defaultSubmit');
      return { LanguageId: { Id: 'lang-obj' }, Timezone: 'UTC' };
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(true);
  expect(calls).toEqual([
    'defaultSubmit',
    'patch:lang-obj:UTC:light:standard',
    'appearance:light:standard',
    'apply:lang-obj',
    'refreshToken',
    'afterLocaleChange',
    'success',
  ]);
});

test('runSelfProfileSubmit: loadUser when current user id is empty', async () => {
  const { calls, opts } = deps({
    hasCurrentUserId: false,
    loadUser: async () => {
      calls.push('loadUser');
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(true);
  expect(calls[0]).toBe('loadUser');
  expect(calls).toContain('defaultSubmit');
});

test('runSelfProfileSubmit: defaultSubmit failure surfaces the error', async () => {
  const { calls, opts } = deps({
    defaultSubmit: async () => {
      calls.push('defaultSubmit');
      throw new Error('write failed');
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(false);
  expect(calls).toEqual(['defaultSubmit', 'error:write failed']);
});

test('runSelfProfileSubmit: applyLanguage failure still refreshes and succeeds', async () => {
  const { calls, opts } = deps({
    applyLanguage: async () => {
      calls.push('apply:lang-1');
      throw new Error('locale failed');
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(true);
  expect(calls).toEqual([
    'defaultSubmit',
    'patch:lang-1:Asia/Shanghai:dark:compact',
    'appearance:dark:compact',
    'apply:lang-1',
    'refreshToken',
    'afterLocaleChange',
    'success',
  ]);
});

test('runSelfProfileSubmit: refreshToken failure after Write still succeeds', async () => {
  const { calls, opts } = deps({
    refreshToken: async () => {
      calls.push('refreshToken');
      throw new Error('token failed');
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(true);
  expect(calls).toEqual([
    'defaultSubmit',
    'patch:lang-1:Asia/Shanghai:dark:compact',
    'appearance:dark:compact',
    'apply:lang-1',
    'refreshToken',
    'success',
  ]);
});

test('runSelfProfileSubmit: non-Error Write throws keep their string message', async () => {
  const { calls, opts } = deps({
    defaultSubmit: async () => {
      throw 'plain failure';
    },
  });
  const ok = await runSelfProfileSubmit(opts);
  expect(ok).toBe(false);
  expect(calls).toEqual(['error:plain failure']);
});
