// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { runPreferencesSubmit } from './preferences_submit';

function deps(overrides?: Partial<Parameters<typeof runPreferencesSubmit>[0]>) {
  const calls: string[] = [];
  return {
    calls,
    opts: {
      userId: 'user-1',
      hasCurrentUserId: true,
      missingUserMessage: 'missing id',
      defaultSubmit: async () => {
        calls.push('defaultSubmit');
        return { LanguageId: 'lang-1', Timezone: 'Asia/Shanghai' };
      },
      formData: { LanguageId: 'lang-draft', Timezone: 'UTC' } as Record<string, unknown>,
      patchCurrentUser: (languageId: unknown, timezone: unknown) => {
        calls.push(`patch:${String(languageId)}:${String(timezone)}`);
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

test('runPreferencesSubmit: missing user id errors without Write', async () => {
  const { calls, opts } = deps({ userId: '' });
  const ok = await runPreferencesSubmit(opts);
  expect(ok).toBe(false);
  expect(calls).toEqual(['error:missing id']);
});

test('runPreferencesSubmit: defaultSubmit then language/token side effects', async () => {
  const { calls, opts } = deps();
  const ok = await runPreferencesSubmit(opts);
  expect(ok).toBe(true);
  expect(calls).toEqual([
    'defaultSubmit',
    'patch:lang-1:Asia/Shanghai',
    'apply:lang-1',
    'refreshToken',
    'afterLocaleChange',
    'success',
  ]);
});

test('runPreferencesSubmit: loadUser when current user id is empty', async () => {
  const { calls, opts } = deps({
    hasCurrentUserId: false,
    loadUser: async () => {
      calls.push('loadUser');
    },
  });
  const ok = await runPreferencesSubmit(opts);
  expect(ok).toBe(true);
  expect(calls[0]).toBe('loadUser');
  expect(calls).toContain('defaultSubmit');
});

test('runPreferencesSubmit: defaultSubmit failure surfaces the error', async () => {
  const { calls, opts } = deps({
    defaultSubmit: async () => {
      calls.push('defaultSubmit');
      throw new Error('write failed');
    },
  });
  const ok = await runPreferencesSubmit(opts);
  expect(ok).toBe(false);
  expect(calls).toEqual(['defaultSubmit', 'error:write failed']);
});
