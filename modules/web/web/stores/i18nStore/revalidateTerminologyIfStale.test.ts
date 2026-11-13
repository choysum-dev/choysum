// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { useI18nStore } from './index';

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
