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
    store.lang = 'en_US';
    store.terminologyHashByLang = { en_US: 'old' };
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
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
    store.lang = 'en_US';
    store.terminologyHashByLang = { en_US: 'same' };
    store.lastTerminologyLoad = null;
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
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
    store.lang = 'en_US';
    globalThis.fetch = (async () => {
      throw new Error('offline');
    }) as typeof fetch;
    expect(await store.revalidateTerminologyIfStale()).toBeNull();
  });

  test('reloadTerminology: discards stale completions after a newer request', async () => {
    const store = useI18nStore();
    store.lang = 'en_US';
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
        bcp47: 'en-US',
        hash: 'fast',
        unchanged: false,
        messages: { web: { scope: { A: 'a' } } },
      });
    }) as typeof fetch;

    const first = store.reloadTerminology('en_US');
    const second = await store.reloadTerminology('en_US');
    expect(second?.hash).toBe('fast');
    resolveSlow(
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'slow',
        unchanged: false,
        messages: { web: { scope: { B: 'b' } } },
      }),
    );
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('fast');
  });

  test('reloadTerminology: returns null for empty lang', async () => {
    const store = useI18nStore();
    expect(await store.reloadTerminology('')).toBeNull();
    expect(await store.reloadTerminology(undefined)).toBeNull();
  });

  test('reloadTerminology: gateway errors return gatewayError load', async () => {
    const store = useI18nStore();
    store.lang = 'en_US';
    store.terminologyHashByLang = { en_US: 'cached' };
    globalThis.fetch = (async () => {
      throw new Error('offline');
    }) as typeof fetch;
    const load = await store.reloadTerminology('en_US');
    expect(load?.gatewayError).toBe(true);
    expect(load?.hash).toBe('cached');
    expect(store.lastTerminologyLoad?.gatewayError).toBe(true);
  });

  test('reloadTerminology: stale failure after newer request stays quiet', async () => {
    const store = useI18nStore();
    store.lang = 'en_US';
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
        bcp47: 'en-US',
        hash: 'ok',
        unchanged: false,
        messages: { web: { scope: { A: 'a' } } },
      });
    }) as typeof fetch;

    const first = store.reloadTerminology('en_US');
    await store.reloadTerminology('en_US');
    rejectSlow(new Error('offline-stale'));
    expect(await first).toBeNull();
    expect(store.terminologyHashByLang.en_US).toBe('ok');
  });

  test('revalidateTerminologyIfStale: falls back to activeLanguage when lang empty', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = null;
    store.terminologyHashByLang = { en_US: 'old' };
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
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
    store.lang = 'en_US';
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
        bcp47: 'en-US',
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
        bcp47: 'en-US',
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
    store.lang = 'en_US';
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
        bcp47: 'en-US',
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

  test('getPersonNameFormat: returns overlay for POSIX code or current active language', () => {
    const store = useI18nStore();
    store.lang = 'zh_CN';
    store.activeLanguages = [{ Code: 'zh_CN' }, { Code: 'en_US' }];
    store.formatByCode = {
      zh_CN: { PersonNameFormat: '{LastName}{FirstName}' },
      en_US: { PersonNameFormat: '{FirstName} {LastName}' },
    } as any;
    expect(store.getPersonNameFormat('zh_CN')).toBe('{LastName}{FirstName}');
    expect(store.getPersonNameFormat('en_US')).toBe('{FirstName} {LastName}');
    expect(store.getPersonNameFormat()).toBe('{LastName}{FirstName}');
    expect(store.getPersonNameFormat('missing')).toBeUndefined();
  });

  test('setLang: no-ops when already on lang and initialized', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }];
    let calls = 0;
    globalThis.fetch = (async () => {
      calls += 1;
      return jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'skip',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    expect(await store.setLang('en_US')).toBe(true);
    expect(calls).toBe(0);
  });

  test('loadActiveLanguages: applies GetActiveLanguages formats and UI keys', async () => {
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
    await store.loadActiveLanguages();
    expect(store.activeLanguages.map((r) => r.Code)).toEqual(['zh_CN', 'en_US']);
    expect(store.formatByCode.zh_CN?.PersonNameFormat).toBe('{LastName}{FirstName}');
    expect(store.formatByCode.en_US?.PersonNameFormat).toBe('{FirstName} {LastName}');
  });

  test('setLang: pending switch is not no-opped by a same-as-current request', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
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
        bcp47: 'en-US',
        hash: 'en-fast',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setLang('zh_CN');
    await firstFetchGate;
    const second = await store.setLang('en_US');
    expect(second).toBe(true);
    resolveSlow(
      jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh-slow',
        unchanged: true,
        messages: null,
      }),
    );
    expect(await first).toBe(false);
    expect(store.lang).toBe('en_US');
  });

  test('setLang: discards stale gateway failure after a newer request', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
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
        bcp47: 'en-US',
        hash: 'en-ok',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setLang('zh_CN');
    await firstFetchGate;
    await store.setLang('en_US');
    rejectSlow(new Error('gateway-stale'));
    expect(await first).toBe(false);
    expect(store.lang).toBe('en_US');
  });

  test('setLang: discards stale success after yield before lang apply', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
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
          bcp47: 'zh-CN',
          hash: 'zh',
          unchanged: true,
          messages: null,
        });
      }
      return jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'en',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setLang('zh_CN');
    await languagesEnteredGate;
    const second = store.setLang('en_US');
    releaseLanguages();
    expect(await first).toBe(false);
    expect(await second).toBe(true);
    expect(store.lang).toBe('en_US');
  });

  test('revalidateTerminologyIfStale: applies changed catalog without remount hook', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'remount',
        unchanged: false,
        messages: { web: { scope: { Hello: 'Hi' } } },
      })) as typeof fetch;
    const load = await store.revalidateTerminologyIfStale();
    expect(load?.hash).toBe('remount');
    expect(store.lastTerminologyLoad?.hash).toBe('remount');
  });

  test('loadActiveLanguages: preserves rows on failure when already populated', async () => {
    const store = useI18nStore();
    store.activeLanguages = [{ Code: 'zh_CN', Name: 'Chinese' }];
    store.formatByCode = { zh_CN: { PersonNameFormat: '{LastName}' } };
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            throw new Error('offline');
          },
        }) as any,
    );
    const ok = await store.loadActiveLanguages();
    expect(ok).toBe(false);
    expect(store.activeLanguages.map(r => r.Code)).toEqual(['zh_CN']);
    expect(store.formatByCode.zh_CN?.PersonNameFormat).toBe('{LastName}');
  });

  test('setLang: returns false when code stays outside active set after reload', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.activeLanguages = [{ Code: 'en_US' }];
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [{ Code: 'en_US' }],
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'de_DE',
        bcp47: 'de-DE',
        hash: 'de',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    expect(await store.setLang('de_DE')).toBe(false);
    expect(store.lang).not.toBe('de_DE');
  });

  test('setLang: supersede during loadActiveLanguages before terminology fetch', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }];
    let releaseLanguages!: () => void;
    const languagesGate = new Promise<void>(resolve => {
      releaseLanguages = resolve;
    });
    let languageCalls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            languageCalls += 1;
            if (languageCalls === 1) {
              await languagesGate;
            }
            return [
              { Code: 'en_US' },
              { Code: 'zh_CN' },
            ];
          },
        }) as any,
    );
    globalThis.fetch = (async (_input, init) => {
      const url = String(init?.body || '');
      void url;
      return jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const first = store.setLang('zh_CN');
    const second = store.setLang('en_US');
    releaseLanguages();
    expect(await first).toBe(false);
    expect(await second).toBe(true);
    expect(store.lang).toBe('en_US');
  });

  test('initialize: falls back when setLang rejects and assign allowed lang', async () => {
    const store = useI18nStore();
    store.lang = 'zh_CN';
    store.activeLanguages = [];
    let languageCalls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            languageCalls += 1;
            if (languageCalls === 1) {
              throw new Error('offline');
            }
            return [{ Code: 'en_US' }];
          },
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh',
        unchanged: true,
        messages: null,
      })) as typeof fetch;

    await store.initialize();
    expect(store.isInitialized).toBe(true);
    expect(store.lang).toBe('en_US');
    expect(languageCalls).toBeGreaterThanOrEqual(2);
  });

  test('initialize: keeps persisted lang when active load fails', async () => {
    const store = useI18nStore();
    store.lang = 'zh_CN';
    store.activeLanguages = [];
    let loadCalls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            loadCalls += 1;
            throw new Error('offline');
          },
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    await store.initialize();
    expect(loadCalls).toBeGreaterThanOrEqual(1);
    expect(store.isInitialized).toBe(true);
    expect(store.activeLanguages.some(r => r.Code === 'zh_CN')).toBe(true);
    expect(store.lang).toBe('zh_CN');
  });
});
