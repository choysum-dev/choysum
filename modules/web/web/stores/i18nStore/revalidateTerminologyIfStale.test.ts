// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import {
  runI18nAfterHydrate,
  runI18nPersistAfterHydrate,
  setI18nCreateStoreForTests,
  useI18nStore,
} from './index';

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

  test('setLang: empty code returns false', async () => {
    const store = useI18nStore();
    expect(await store.setLang('')).toBe(false);
    expect(await store.setLang('   ')).toBe(false);
    expect(store.isLoading).toBe(false);
  });

  test('setLang: unknown code clears loading after reload', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.activeLanguages = [{ Code: 'en_US' }];
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [{ Code: 'en_US' }],
        }) as any,
    );
    const ok = await store.setLang('xx_YY');
    expect(ok).toBe(false);
    expect(store.isLoading).toBe(false);
    expect(store.lang).not.toBe('xx_YY');
  });

  test('loadActiveLanguages: stale generation discards RPC result', async () => {
    const store = useI18nStore();
    store.activeLanguages = [{ Code: 'en_US', Name: 'English' }];
    store.formatByCode = { en_US: { Name: 'English' } };
    let releaseSlow!: () => void;
    const slowGate = new Promise<void>(resolve => {
      releaseSlow = resolve;
    });
    let calls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            calls += 1;
            if (calls === 1) {
              await slowGate;
              return [{ Code: 'zh_CN', Name: 'Chinese' }];
            }
            return [{ Code: 'de_DE', Name: 'German' }];
          },
        }) as any,
    );
    const first = store.loadActiveLanguages();
    const second = store.loadActiveLanguages();
    releaseSlow();
    expect(await first).toBe(false);
    expect(await second).toBe(true);
    expect(store.activeLanguages.map(r => r.Code)).toEqual(['de_DE']);
  });

  test('initialize: persisted lang not in active set falls back after load', async () => {
    const store = useI18nStore();
    store.lang = 'zh_CN';
    store.activeLanguages = [];
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [{ Code: 'en_US' }],
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'en',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    await store.initialize();
    expect(store.lang).toBe('en_US');
    expect(store.isInitialized).toBe(true);
  });

  test('setLang: returns false when code stays outside active set after reload', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.activeLanguages = [{ Code: 'en_US' }];
    store.terminologyHashByLang = {};
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
    expect(store.lastTerminologyLoad).toBeNull();
    expect(store.terminologyHashByLang.de_DE).toBeUndefined();
  });

  test('setLang: restores previous terminology hash when rejected after reload', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.activeLanguages = [{ Code: 'en_US' }];
    store.terminologyHashByLang = { de_DE: 'old-de' };
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
        hash: 'new-de',
        unchanged: false,
        messages: { web: {} },
      })) as typeof fetch;
    expect(await store.setLang('de_DE')).toBe(false);
    expect(store.terminologyHashByLang.de_DE).toBe('old-de');
    expect(store.lastTerminologyLoad).toBeNull();
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
    // Use a code outside DEFAULT_ACTIVE_LANGS so the ensure-active branch runs.
    store.lang = 'de_DE';
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
        lang: 'de_DE',
        bcp47: 'de-DE',
        hash: 'de',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    await store.initialize();
    expect(loadCalls).toBeGreaterThanOrEqual(1);
    expect(store.isInitialized).toBe(true);
    expect(store.activeLanguages.some(r => r.Code === 'de_DE')).toBe(true);
    expect(store.lang).toBe('de_DE');
  });

  test('initialize: empty lang uses detectBestLang', async () => {
    const store = useI18nStore();
    store.lang = '';
    store.activeLanguages = [];
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [{ Code: 'en_US' }, { Code: 'zh_CN' }],
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'en_US',
        bcp47: 'en-US',
        hash: 'en',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    await store.initialize();
    expect(store.isInitialized).toBe(true);
    expect(store.lang).toBeTruthy();
  });

  test('initialize: no-ops when already initialized', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    await store.initialize();
    expect(store.isInitialized).toBe(true);
  });

  test('initialize: force-assigns lang when setLang cannot apply', async () => {
    const store = useI18nStore();
    store.lang = 'fr_FR';
    store.activeLanguages = [];
    let loads = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            loads += 1;
            // Init allows fr_FR; each setLang post-fetch reload removes the selected code.
            if (loads === 1) {
              return [{ Code: 'fr_FR' }];
            }
            if (loads === 2) {
              return [{ Code: 'aa_AA' }];
            }
            return [{ Code: 'bb_BB' }];
          },
        }) as any,
    );
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'x',
        bcp47: 'x',
        hash: 'x',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    await store.initialize();
    expect(store.isInitialized).toBe(true);
    expect(store.lang).toBe('bb_BB');
  });

  test('initialize: finally assigns default when lang stays empty after supersede', async () => {
    const store = useI18nStore();
    store.lang = null as any;
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    let releaseLoad!: () => void;
    const loadGate = new Promise<void>(resolve => {
      releaseLoad = resolve;
    });
    let releaseFetch!: () => void;
    const fetchGate = new Promise<void>(resolve => {
      releaseFetch = resolve;
    });
    let loads = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            loads += 1;
            if (loads === 1) {
              await loadGate;
            }
            return [
              { Code: 'en_US' },
              { Code: 'zh_CN' },
            ];
          },
        }) as any,
    );
    globalThis.fetch = (async () => {
      await fetchGate;
      return jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh',
        unchanged: true,
        messages: null,
      });
    }) as typeof fetch;

    const init = store.initialize();
    for (let i = 0; i < 10 && loads < 1; i += 1) {
      await Promise.resolve();
    }
    // Bump generation while terminology fetch is still pending so initialize's finally
    // runs with lang still empty.
    store.lang = null as any;
    const switched = store.setLang('zh_CN');
    releaseLoad();
    await init;
    expect(store.isInitialized).toBe(true);
    expect(store.lang).toBe('en_US');
    releaseFetch();
    await switched;
  });

  test('initialize: skips selection when superseded during active-language load', async () => {
    const store = useI18nStore();
    store.lang = 'en_US';
    store.activeLanguages = [];
    let release!: () => void;
    const gate = new Promise<void>(resolve => {
      release = resolve;
    });
    let loads = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            loads += 1;
            if (loads === 1) {
              await gate;
              return [{ Code: 'en_US' }];
            }
            return [
              { Code: 'en_US' },
              { Code: 'zh_CN' },
            ];
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

    const init = store.initialize();
    const switched = store.setLang('zh_CN');
    release();
    await Promise.all([init, switched]);
    expect(store.isInitialized).toBe(true);
    expect(store.lang).toBe('zh_CN');
  });

  test('setLang: empty code returns false', async () => {
    const store = useI18nStore();
    expect(await store.setLang('')).toBe(false);
    expect(await store.setLang('   ')).toBe(false);
  });

  test('setLang: gateway failure falls back to msgid catalog', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    globalThis.fetch = (async () => {
      throw new Error('gateway down');
    }) as typeof fetch;
    expect(await store.setLang('zh_CN')).toBe(true);
    expect(store.lang).toBe('zh_CN');
    expect(store.lastTerminologyLoad?.gatewayError).toBe(true);
  });

  test('setLang: outer catch when document direction update throws', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    globalThis.fetch = (async () =>
      jsonResponse({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
        hash: 'zh',
        unchanged: true,
        messages: null,
      })) as typeof fetch;
    const el = document.documentElement;
    const prev = el.setAttribute.bind(el);
    el.setAttribute = (() => {
      throw new Error('dir boom');
    }) as typeof el.setAttribute;
    try {
      expect(await store.setLang('zh_CN')).toBe(false);
    } finally {
      el.setAttribute = prev;
    }
  });

  test('loadActiveLanguages: stale generation discards catch path', async () => {
    const store = useI18nStore();
    store.activeLanguages = [{ Code: 'en_US' }];
    let releaseSlow!: () => void;
    const slowGate = new Promise<void>(resolve => {
      releaseSlow = resolve;
    });
    let calls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            calls += 1;
            if (calls === 1) {
              await slowGate;
              throw new Error('stale failure');
            }
            return [{ Code: 'de_DE' }];
          },
        }) as any,
    );
    const first = store.loadActiveLanguages();
    const second = store.loadActiveLanguages();
    releaseSlow();
    expect(await first).toBe(false);
    expect(await second).toBe(true);
    expect(store.activeLanguages.map(r => r.Code)).toEqual(['de_DE']);
  });

  test('store format helpers: date/number/currency wrappers', () => {
    const store = useI18nStore();
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US', Name: 'English' }];
    store.formatByCode = {
      en_US: {
        Name: 'English',
        DecimalSeparator: '.',
        ThousandSeparator: ',',
        Grouping: '[3,0]',
        DateFormat: 'YYYY-MM-DD',
        TimeFormat: 'HH:mm',
      },
    };
    expect(String(store.formatDateTime(new Date(2026, 0, 2), { type: 'date' }))).toMatch(/2026/);
    expect(String(store.formatNumber(12.5, { digits: 1 }))).not.toBe('');
    expect(String(store.formatCurrency(12.5, 'USD'))).not.toBe('');
    expect(store.getDateTimeFormats().en_US).toBeTruthy();
    expect(store.getNumberFormats().en_US).toBeTruthy();
  });

  test('runI18nAfterHydrate: invokes initialize for i18n store id', async () => {
    const calls: string[] = [];
    runI18nAfterHydrate({
      $id: 'choysum.web.i18n',
      initialize: () => {
        calls.push('init');
      },
    });
    runI18nAfterHydrate({ $id: 'other' });
    expect(calls).toEqual(['init']);
  });

  test('runI18nPersistAfterHydrate: forwards pinia ctx.store', () => {
    const calls: string[] = [];
    runI18nPersistAfterHydrate({
      store: {
        $id: 'choysum.web.i18n',
        initialize: () => {
          calls.push('init');
        },
      },
    } as any);
    expect(calls).toEqual(['init']);
  });

  test('setLang: late supersede after terminology returns false', async () => {
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    let releaseLoad!: () => void;
    const loadGate = new Promise<void>(resolve => {
      releaseLoad = resolve;
    });
    let loadCalls = 0;
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => {
            loadCalls += 1;
            if (loadCalls === 1) {
              await loadGate;
            }
            return [
              { Code: 'en_US' },
              { Code: 'zh_CN' },
            ];
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

    const first = store.setLang('zh_CN');
    // Let the first call reach the post-fetch loadActiveLanguages wait.
    for (let i = 0; i < 10 && loadCalls < 1; i += 1) {
      await Promise.resolve();
    }
    const second = store.setLang('en_US');
    expect(await second).toBe(true);
    releaseLoad();
    expect(await first).toBe(false);
    expect(store.lang).toBe('en_US');
  });
});
