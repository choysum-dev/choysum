// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { setI18nCreateStoreForTests, useI18nStore } from './index';

function jsonResponse(body: unknown, ok = true): Response {
  return {
    ok,
    status: ok ? 200 : 500,
    json: async () => body,
  } as Response;
}

describe('useI18nStore.revalidateTerminologyIfStale / reloadTerminology', () => {
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    setActivePinia(createPinia());
    originalFetch = globalThis.fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    setI18nCreateStoreForTests(null);
  });

  test('revalidateTerminologyIfStale: returns null when not initialized', async () => {
    const store = useI18nStore();
    store.isInitialized = false;
    expect(await store.revalidateTerminologyIfStale()).toBeNull();
  });

  test('revalidateTerminologyIfStale: merges changed catalog and updates hash', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    store.terminologyHashByLang = { en_US: 'old' };
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'new',
        unchanged: false,
        messages: { web: { scope: { Hello: 'Hi' } } },
      })) as typeof fetch;

    const load = await store.revalidateTerminologyIfStale();
    expect(load?.hash).toBe('new');
    expect(load?.unchanged).toBe(false);
    expect(store.terminologyHashByLang.en_US).toBe('new');
    expect(store.lastTerminologyLoad?.hash).toBe('new');
  });

  test('revalidateTerminologyIfStale: unchanged leaves lastTerminologyLoad alone', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    store.terminologyHashByLang = { en_US: 'same' };
    store.lastTerminologyLoad = null;
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'same',
        unchanged: true,
        messages: null,
      })) as typeof fetch;

    const load = await store.revalidateTerminologyIfStale();
    expect(load?.unchanged).toBe(true);
    expect(store.lastTerminologyLoad).toBeNull();
  });

  test('revalidateTerminologyIfStale: gateway errors return null', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    globalThis.fetch = (async () => {
      throw new Error('offline');
    }) as typeof fetch;
    expect(await store.revalidateTerminologyIfStale()).toBeNull();
  });

  test('reloadTerminology: discards stale completions after a newer request', async () => {
    const store = useI18nStore();
    store.localeCode = 'en';
    let resolveSlow: (v: Response) => void = () => undefined;
    const slow = new Promise<Response>(resolve => {
      resolveSlow = resolve;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) return slow;
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'fast',
        unchanged: false,
        messages: { web: { scope: { A: 'a' } } },
      });
    }) as typeof fetch;

    const first = store.reloadTerminology('en');
    const second = await store.reloadTerminology('en');
    expect(second?.hash).toBe('fast');
    resolveSlow(
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'slow',
        unchanged: false,
        messages: { web: { scope: { B: 'b' } } },
      }),
    );
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('fast');
  });

  test('reloadTerminology: returns null for unsupported locale', async () => {
    const store = useI18nStore();
    expect(await store.reloadTerminology('nope-XX')).toBeNull();
  });

  test('reloadTerminology: gateway errors return gatewayError load', async () => {
    const store = useI18nStore();
    store.localeCode = 'en';
    store.terminologyHashByLang = { en_US: 'cached' };
    globalThis.fetch = (async () => {
      throw new Error('offline');
    }) as typeof fetch;
    const load = await store.reloadTerminology('en');
    expect(load?.gatewayError).toBe(true);
    expect(load?.hash).toBe('cached');
    expect(store.lastTerminologyLoad?.gatewayError).toBe(true);
  });

  test('reloadTerminology: stale failure after newer request stays quiet', async () => {
    const store = useI18nStore();
    store.localeCode = 'en';
    let rejectSlow: (err: Error) => void = () => undefined;
    const slow = new Promise<Response>((_resolve, reject) => {
      rejectSlow = reject;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) return slow;
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'ok',
        unchanged: false,
        messages: { web: { scope: { A: 'a' } } },
      });
    }) as typeof fetch;

    const first = store.reloadTerminology('en');
    await store.reloadTerminology('en');
    rejectSlow(new Error('offline-stale'));
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('ok');
  });

  test('revalidateTerminologyIfStale: falls back to currentLocale when localeCode empty', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = null;
    store.terminologyHashByLang = { en_US: 'old' };
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'from-current',
        unchanged: true,
        messages: null,
      })) as typeof fetch;

    const load = await store.revalidateTerminologyIfStale();
    expect(load?.hash).toBe('from-current');
    expect(store.terminologyHashByLang.en_US).toBe('from-current');
  });

  test('revalidateTerminologyIfStale: discards stale completions after a newer request', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    store.terminologyHashByLang = { en_US: 'old' };
    let resolveSlow: (v: Response) => void = () => undefined;
    const slow = new Promise<Response>(resolve => {
      resolveSlow = resolve;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) return slow;
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'fast',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.revalidateTerminologyIfStale();
    const second = await store.revalidateTerminologyIfStale();
    expect(second?.hash).toBe('fast');
    resolveSlow(
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'slow',
        unchanged: false,
        messages: { web: { scope: { B: 'b' } } },
      }),
    );
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('fast');
  });

  test('revalidateTerminologyIfStale: stale failure after newer request stays quiet', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    let rejectSlow: (err: Error) => void = () => undefined;
    const slow = new Promise<Response>((_resolve, reject) => {
      rejectSlow = reject;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) return slow;
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'ok',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.revalidateTerminologyIfStale();
    await store.revalidateTerminologyIfStale();
    rejectSlow(new Error('offline-stale'));
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('ok');
  });

  test('getPersonNameFormat: returns overlay for POSIX code or current UI language', () => {
    const store = useI18nStore();
    store.localeCode = 'zh-CN';
    store.languageFormatByCode = {
      zh_CN: { PersonNameFormat: '{LastName}{FirstName}' },
      en_US: { PersonNameFormat: '{FirstName} {LastName}' },
    } as any;
    expect(store.getPersonNameFormat('zh_CN')).toBe('{LastName}{FirstName}');
    expect(store.getPersonNameFormat('en_US')).toBe('{FirstName} {LastName}');
    expect(store.getPersonNameFormat()).toBe('{LastName}{FirstName}');
    expect(store.getPersonNameFormat('missing')).toBeUndefined();
  });

  test('setUiKey: no-ops when already on locale and initialized', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'skip',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    expect(await store.setUiKey('en')).toBe(true);
    expect(calls).toBe(0);
  });

  test('loadActiveUiKeysFromServer: applies GetActiveLanguages formats and UI keys', async () => {
    const store = useI18nStore();
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [
            {
              Code: 'zh_CN',
              ThousandSeparator: ',',
              DecimalSeparator: '.',
              PersonNameFormat: '{LastName}{FirstName}',
            },
            {
              Code: 'en_US',
              ThousandSeparator: ',',
              DecimalSeparator: '.',
              PersonNameFormat: '{FirstName} {LastName}',
            },
          ],
        }) as any,
    );
    await store.loadActiveUiKeysFromServer();
    expect(store.activeUiKeys).toEqual(['zh-CN', 'en']);
    expect(store.languageFormatByCode.zh_CN?.PersonNameFormat).toBe('{LastName}{FirstName}');
    expect(store.languageFormatByCode.en_US?.PersonNameFormat).toBe('{FirstName} {LastName}');
  });

  test('setUiKey: pending switch is not no-opped by a same-as-current request', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    let resolveSlow: (v: Response) => void = () => undefined;
    const slow = new Promise<Response>(resolve => {
      resolveSlow = resolve;
    });
    let firstFetchStarted!: () => void;
    const firstFetchGate = new Promise<void>(resolve => {
      firstFetchStarted = resolve;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) {
        firstFetchStarted();
        return slow;
      }
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'en-fast',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setUiKey('zh-CN');
    await firstFetchGate;
    const second = await store.setUiKey('en');
    expect(second).toBe(true);
    resolveSlow(
      jsonResponse({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'zh-slow',
        unchanged: true,
        messages: null,
      }),
    );
    expect(await first).toBe(false);
    expect(store.localeCode).toBe('en');
  });

  test('setUiKey: discards stale gateway failure after a newer request', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    let rejectSlow: (err: Error) => void = () => undefined;
    const slow = new Promise<Response>((_resolve, reject) => {
      rejectSlow = reject;
    });
    let firstFetchStarted!: () => void;
    const firstFetchGate = new Promise<void>(resolve => {
      firstFetchStarted = resolve;
    });
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      if (calls === 1) {
        firstFetchStarted();
        return slow;
      }
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'en-ok',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setUiKey('zh-CN');
    await firstFetchGate;
    await store.setUiKey('en');
    rejectSlow(new Error('gateway-stale'));
    expect(await first).toBe(false);
    expect(store.localeCode).toBe('en');
  });

  test('setUiKey: discards stale success after yield before locale apply', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    let releaseLanguages!: () => void;
    const languagesGate = new Promise<void>(resolve => {
      releaseLanguages = resolve;
    });
    let languagesEntered!: () => void;
    const languagesEnteredGate = new Promise<void>(resolve => {
      languagesEntered = resolve;
    });
    let languageCalls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            languageCalls += 1;
            if (languageCalls === 1) {
              languagesEntered();
              await languagesGate;
            }
            return [
              {
                Code: 'en_US',
                ThousandSeparator: ',',
                DecimalSeparator: '.',
              },
            ];
          },
        }) as any,
    );
    let fetchCalls = 0;
    globalThis.fetch = (async () => {
      fetchCalls += 1;
      if (fetchCalls === 1) {
        return jsonResponse({
          lang: 'zh_CN',
          locale: 'zh-CN',
          hash: 'zh',
          unchanged: true,
          messages: null,
        });
      }
      return jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'en',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setUiKey('zh-CN');
    await languagesEnteredGate;
    const second = store.setUiKey('en');
    releaseLanguages();
    expect(await first).toBe(false);
    expect(await second).toBe(true);
    expect(store.localeCode).toBe('en');
  });

  test('revalidateTerminologyIfStale: soft-remounts when remount mode is enabled', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.localeCode = 'en';
    const g = globalThis as { __CHOYSUM_I18N_REMOUNT_MODE__?: string };
    const prev = g.__CHOYSUM_I18N_REMOUNT_MODE__;
    g.__CHOYSUM_I18N_REMOUNT_MODE__ = 'remount';
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        locale: 'en',
        hash: 'remount',
        unchanged: false,
        messages: { web: { scope: { Hello: 'Hi' } } },
      })) as typeof fetch;
    try {
      const load = await store.revalidateTerminologyIfStale();
      expect(load?.hash).toBe('remount');
      expect(store.lastTerminologyLoad?.hash).toBe('remount');
    } finally {
      if (prev === undefined) delete g.__CHOYSUM_I18N_REMOUNT_MODE__;
      else g.__CHOYSUM_I18N_REMOUNT_MODE__ = prev;
    }
  });
});
