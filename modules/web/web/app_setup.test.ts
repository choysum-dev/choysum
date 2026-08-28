// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const exposeBrowserI18nOnWindow = vi.hoisted(() => vi.fn());
const registerGlobalDirectives = vi.hoisted(() => vi.fn());
const setGlobalRequestContextProvider = vi.hoisted(() => vi.fn());
const setUserTimeZoneResolver = vi.hoisted(() => vi.fn());
const mergeLocaleMessage = vi.hoisted(() => vi.fn());
const loadVueI18nMessages = vi.hoisted(() => vi.fn());
const createAppRouter = vi.hoisted(() => vi.fn(() => ({ name: 'router' })));
const createAppMenu = vi.hoisted(() => vi.fn(() => ({ name: 'menu' })));
const createTerminologyCatalogMerger = vi.hoisted(() => vi.fn());
const resolveRequestTimezone = vi.hoisted(() => vi.fn());
const detectBrowserTimezone = vi.hoisted(() => vi.fn());

const storeState = vi.hoisted(() => {
  const { reactive, ref } = require('vue') as typeof import('vue');
  return {
    currentLocale: reactive({ code: 'en', elementLocale: { name: 'en' } }),
    lastTerminologyLoad: ref<unknown>(null),
    i18nLocale: ref('en'),
  };
});

let mockAuthStore: Record<string, unknown>;
let requestContextProvider: (() => Record<string, string>) | undefined;
let userTimeZoneResolver: (() => string | null) | undefined;
let terminologyMerger: ((terminology: unknown, locale: string) => void) | undefined;
let catalogMerge: ((locale: string, messages: unknown) => void) | undefined;

vi.mock('./i18n', async importOriginal => {
  const actual = await importOriginal<typeof import('./i18n')>();
  return {
    ...actual,
    exposeBrowserI18nOnWindow,
  };
});
vi.mock('./directives', () => ({ registerGlobalDirectives }));
vi.mock('pinia', () => ({ createPinia: () => ({ use: () => ({}) }) }));
vi.mock('pinia-plugin-persistedstate', () => ({ default: vi.fn() }));
vi.mock('./stores/i18nStore', () => ({
  useI18nStore: () => ({
    get currentLocale() {
      return storeState.currentLocale;
    },
    terminologyLang: 'en_US',
    get lastTerminologyLoad() {
      return storeState.lastTerminologyLoad.value;
    },
    getDateTimeFormats: () => ({ short: {} }),
    getNumberFormats: () => ({ currency: {} }),
    loadVueI18nMessages,
  }),
}));
vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => mockAuthStore,
}));
vi.mock('@/core/rpc/context', () => ({
  setGlobalRequestContextProvider: (provider: () => Record<string, string>) => {
    setGlobalRequestContextProvider(provider);
    requestContextProvider = provider;
  },
}));
vi.mock('./utils/request_timezone', () => ({
  detectBrowserTimezone,
  resolveRequestTimezone,
}));
vi.mock('./utils/datetime', () => ({
  setUserTimeZoneResolver: (resolver: () => string | null) => {
    setUserTimeZoneResolver(resolver);
    userTimeZoneResolver = resolver;
  },
}));
vi.mock('vue-i18n', () => ({
  createI18n: vi.fn(() => ({
    global: {
      locale: storeState.i18nLocale,
      mergeLocaleMessage,
    },
  })),
}));
vi.mock('./i18n/source', () => ({ default: { hello: 'Hello' } }));
vi.mock('./stores/i18nStore/merge', () => ({
  createTerminologyCatalogMerger: (opts: {
    merge: (locale: string, messages: unknown) => void;
    notify: () => void;
  }) => {
    createTerminologyCatalogMerger(opts);
    catalogMerge = opts.merge;
    terminologyMerger = vi.fn((terminology: unknown, locale: string) => {
      opts.merge(locale, terminology);
    });
    return terminologyMerger;
  },
}));
vi.mock('./router', () => ({ createAppRouter }));
vi.mock('./menu', () => ({ createAppMenu }));
vi.mock('element-plus', () => ({ default: { name: 'ElementPlus' } }));

import { setupApp } from './app_setup';
import { notifyComposerMessagesChanged } from './i18n';

function makeApp(elementLocale: Record<string, unknown> = { name: 'en' }) {
  return {
    config: {
      globalProperties: {
        $ELEMENT: { locale: elementLocale },
      },
    },
    usePlugin: vi.fn(),
  };
}

describe('setupApp', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storeState.currentLocale.code = 'en';
    storeState.currentLocale.elementLocale = { name: 'en' };
    storeState.lastTerminologyLoad.value = null;
    storeState.i18nLocale.value = 'en';
    mockAuthStore = {};
    requestContextProvider = undefined;
    userTimeZoneResolver = undefined;
    terminologyMerger = undefined;
    catalogMerge = undefined;
    loadVueI18nMessages.mockResolvedValue(null);
    resolveRequestTimezone.mockImplementation((userTz: string, browserTz: string | null) => userTz || browserTz || '');
    detectBrowserTimezone.mockReturnValue('Asia/Shanghai');
  });

  it('registers plugins and exposes browser i18n globals', () => {
    const app = makeApp();

    setupApp(app as any);

    expect(registerGlobalDirectives).toHaveBeenCalledWith(app);
    expect(exposeBrowserI18nOnWindow).toHaveBeenCalledTimes(1);
    expect(app.usePlugin).toHaveBeenCalledWith('pinia', expect.anything(), {}, false);
    expect(app.usePlugin).toHaveBeenCalledWith('i18n', expect.anything());
    expect(app.usePlugin).toHaveBeenCalledWith('router', expect.anything());
    expect(app.usePlugin).toHaveBeenCalledWith('menu', expect.anything());
    expect(app.usePlugin).toHaveBeenCalledWith('element-plus', expect.anything(), expect.anything());
    expect(createAppRouter).toHaveBeenCalled();
    expect(createAppMenu).toHaveBeenCalled();
    expect(createTerminologyCatalogMerger).toHaveBeenCalledWith({
      merge: expect.any(Function),
      notify: notifyComposerMessagesChanged,
    });
  });

  it('skips browser i18n expose without window', () => {
    const originalWindow = globalThis.window;
    // @ts-expect-error simulate non-browser runtime
    delete globalThis.window;
    setupApp(makeApp() as any);
    expect(exposeBrowserI18nOnWindow).not.toHaveBeenCalled();
    globalThis.window = originalWindow;
  });

  it('resolves user timezone from auth store', () => {
    mockAuthStore = {
      currentUser: { Timezone: 'Europe/Berlin' },
      identity: { metadata: { timezone: 'UTC' } },
    };

    setupApp(makeApp() as any);

    expect(userTimeZoneResolver?.()).toBe('Europe/Berlin');
  });

  it('falls back to identity timezone and swallows auth lookup failures', () => {
    mockAuthStore = {
      get currentUser() {
        throw new Error('auth unavailable');
      },
      identity: { metadata: { timezone: 'America/New_York' } },
    };

    setupApp(makeApp() as any);

    expect(userTimeZoneResolver?.()).toBeNull();
  });

  it('builds request context with terminology lang and resolved tz', () => {
    mockAuthStore = {
      currentUser: { Timezone: 'Europe/Berlin' },
    };
    resolveRequestTimezone.mockReturnValue('Europe/Berlin');
    storeState.currentLocale.code = 'zh-CN';
    storeState.currentLocale.elementLocale = { name: 'zh-CN' };

    setupApp(makeApp() as any);

    expect(requestContextProvider?.()).toEqual({
      locale: 'zh-CN',
      lang: 'en_US',
      tz: 'Europe/Berlin',
    });
  });

  it('omits tz from request context when unresolved', () => {
    resolveRequestTimezone.mockReturnValue('');

    setupApp(makeApp() as any);

    expect(requestContextProvider?.()).toEqual({
      locale: 'en',
      lang: 'en_US',
    });
  });

  it('swallows auth errors while building request context timezone', () => {
    mockAuthStore = {
      get currentUser() {
        throw new Error('auth unavailable');
      },
    };
    resolveRequestTimezone.mockReturnValue('UTC');

    setupApp(makeApp() as any);

    expect(requestContextProvider?.()).toEqual({
      locale: 'en',
      lang: 'en_US',
      tz: 'UTC',
    });
  });

  it('updates Element Plus locale and legacy messages on locale change', async () => {
    const elementLocale = { name: 'zh-CN' };
    const app = makeApp(elementLocale);
    loadVueI18nMessages.mockResolvedValue({ legacy: 'messages' });

    setupApp(app as any);
    storeState.currentLocale.code = 'zh-CN';
    storeState.currentLocale.elementLocale = elementLocale;

    await nextTick();
    await nextTick();

    expect(app.config.globalProperties.$ELEMENT.locale).toStrictEqual(elementLocale);
    expect(loadVueI18nMessages).toHaveBeenCalledWith('zh-CN');
    expect(mergeLocaleMessage).toHaveBeenCalledWith('zh-CN', { legacy: 'messages' });
    expect(storeState.i18nLocale.value).toBe('zh-CN');
  });

  it('warns when legacy locale messages fail to load', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    loadVueI18nMessages.mockRejectedValue(new Error('network'));

    setupApp(makeApp() as any);
    storeState.currentLocale.code = 'zh-CN';
    storeState.currentLocale.elementLocale = { name: 'zh-CN' };

    await nextTick();
    await nextTick();

    expect(warn).toHaveBeenCalledWith(
      'Failed to load legacy locale messages for zh-CN',
      expect.any(Error)
    );
    warn.mockRestore();
  });

  it('merges terminology catalog updates from the i18n store', async () => {
    setupApp(makeApp() as any);

    storeState.lastTerminologyLoad.value = { auth: { menu: { Users: '用户' } } };
    storeState.currentLocale.code = 'zh-CN';
    storeState.currentLocale.elementLocale = { name: 'zh-CN' };

    await nextTick();

    expect(terminologyMerger).toHaveBeenCalledWith(
      { auth: { menu: { Users: '用户' } } },
      'zh-CN'
    );
    expect(mergeLocaleMessage).toHaveBeenCalled();
  });
});
