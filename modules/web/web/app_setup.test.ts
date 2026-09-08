// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, reactive, ref } from 'vue';
import { setupApp, type SetupAppDeps } from './app_setup';
import { notifyComposerMessagesChanged } from './i18n';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function makeApp(elementLocale: Record<string, unknown> = { name: 'en' }) {
  return {
    config: {
      globalProperties: {
        $ELEMENT: { locale: elementLocale },
      },
    },
    usePlugin: fnRecorder(),
  };
}

function pluginNames(app: ReturnType<typeof makeApp>): string[] {
  return app.usePlugin.calls.map(call => String(call[0]));
}

test('setupApp > registers plugins and exposes browser i18n globals', () => {
  const registerGlobalDirectives = fnRecorder();
  const exposeBrowserI18nOnWindow = fnRecorder();
  const createAppRouter = fnRecorder(() => ({ name: 'router' }));
  const createAppMenu = fnRecorder(() => ({ name: 'menu' }));
  const createTerminologyCatalogMerger = fnRecorder(() => fnRecorder());
  const pinia = { name: 'pinia' };
  const createPinia = () => ({ use: () => pinia }) as any;
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const lastTerminologyLoad = ref<unknown>(null);
  const i18nLocale = ref('en');
  const mergeLocaleMessage = fnRecorder();
  const loadVueI18nMessages = fnRecorder(async () => null);
  const createI18n = fnRecorder(() => ({
    global: {
      locale: i18nLocale,
      mergeLocaleMessage,
    },
  })) as any;

  const app = makeApp();
  setupApp(app as any, {
    registerGlobalDirectives: registerGlobalDirectives as any,
    createPinia,
    piniaPluginPersistedstate: (() => {}) as any,
    useI18nStore: (() => ({
      currentLocale,
      terminologyLang: 'en_US',
      lastTerminologyLoad: lastTerminologyLoad.value,
      getDateTimeFormats: () => ({ short: {} }),
      getNumberFormats: () => ({ currency: {} }),
      loadVueI18nMessages,
    })) as any,
    setUserTimeZoneResolver: fnRecorder() as any,
    setGlobalRequestContextProvider: fnRecorder() as any,
    resolveRequestTimezone: ((userTz: string, browserTz: string | null) => userTz || browserTz || '') as any,
    detectBrowserTimezone: (() => 'Asia/Shanghai') as any,
    useAuthStore: (() => ({})) as any,
    createI18n,
    sourceMessages: { hello: 'Hello' } as any,
    createTerminologyCatalogMerger: createTerminologyCatalogMerger as any,
    projectTerminologyMessages: (m: unknown) => m as any,
    exposeBrowserI18nOnWindow: exposeBrowserI18nOnWindow as any,
    notifyComposerMessagesChanged,
    trackComposerMessageRevision: ((v: unknown) => v) as any,
    createAppRouter: createAppRouter as any,
    createAppMenu: createAppMenu as any,
    ElementPlus: { name: 'ElementPlus' } as any,
    baseUrl: '/',
    hasWindow: () => true,
  });

  expect(registerGlobalDirectives.calls.length).toBe(1);
  expect(registerGlobalDirectives.calls[0][0]).toBe(app);
  expect(exposeBrowserI18nOnWindow.calls.length).toBe(1);
  expect(pluginNames(app)).toEqual(['pinia', 'i18n', 'router', 'menu', 'element-plus']);
  expect(createAppRouter.calls.length).toBe(1);
  expect(createAppMenu.calls.length).toBe(1);
  expect(createTerminologyCatalogMerger.calls.length).toBe(1);
  const mergerOpts = createTerminologyCatalogMerger.calls[0][0] as {
    merge: unknown;
    notify: unknown;
  };
  expect(typeof mergerOpts.merge).toBe('function');
  expect(mergerOpts.notify).toBe(notifyComposerMessagesChanged);
});

test('setupApp > skips browser i18n expose without window', () => {
  const exposeBrowserI18nOnWindow = fnRecorder();
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const i18nLocale = ref('en');

  setupApp(makeApp() as any, {
    registerGlobalDirectives: fnRecorder() as any,
    createPinia: (() => ({ use: () => ({}) })) as any,
    piniaPluginPersistedstate: (() => {}) as any,
    useI18nStore: (() => ({
      currentLocale,
      terminologyLang: 'en_US',
      lastTerminologyLoad: null,
      getDateTimeFormats: () => ({}),
      getNumberFormats: () => ({}),
      loadVueI18nMessages: async () => null,
    })) as any,
    setUserTimeZoneResolver: fnRecorder() as any,
    setGlobalRequestContextProvider: fnRecorder() as any,
    resolveRequestTimezone: ((a: string, b: string | null) => a || b || '') as any,
    detectBrowserTimezone: (() => '') as any,
    useAuthStore: (() => ({})) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage: fnRecorder() },
    })) as any,
    sourceMessages: {} as any,
    createTerminologyCatalogMerger: (() => fnRecorder()) as any,
    projectTerminologyMessages: (m: unknown) => m as any,
    exposeBrowserI18nOnWindow: exposeBrowserI18nOnWindow as any,
    notifyComposerMessagesChanged,
    trackComposerMessageRevision: ((v: unknown) => v) as any,
    createAppRouter: (() => ({})) as any,
    createAppMenu: (() => ({})) as any,
    ElementPlus: {} as any,
    baseUrl: '/',
    hasWindow: () => false,
  });

  expect(exposeBrowserI18nOnWindow.calls.length).toBe(0);
});

test('setupApp > resolves user timezone from auth store', () => {
  let userTimeZoneResolver: (() => string | null) | undefined;
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const i18nLocale = ref('en');

  setupApp(makeApp() as any, {
    registerGlobalDirectives: fnRecorder() as any,
    createPinia: (() => ({ use: () => ({}) })) as any,
    piniaPluginPersistedstate: (() => {}) as any,
    useI18nStore: (() => ({
      currentLocale,
      terminologyLang: 'en_US',
      lastTerminologyLoad: null,
      getDateTimeFormats: () => ({}),
      getNumberFormats: () => ({}),
      loadVueI18nMessages: async () => null,
    })) as any,
    setUserTimeZoneResolver: (resolver: () => string | null) => {
      userTimeZoneResolver = resolver;
    },
    setGlobalRequestContextProvider: fnRecorder() as any,
    resolveRequestTimezone: ((a: string, b: string | null) => a || b || '') as any,
    detectBrowserTimezone: (() => '') as any,
    useAuthStore: (() => ({
      currentUser: { Timezone: 'Europe/Berlin' },
      identity: { metadata: { timezone: 'UTC' } },
    })) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage: fnRecorder() },
    })) as any,
    sourceMessages: {} as any,
    createTerminologyCatalogMerger: (() => fnRecorder()) as any,
    projectTerminologyMessages: (m: unknown) => m as any,
    exposeBrowserI18nOnWindow: fnRecorder() as any,
    notifyComposerMessagesChanged,
    trackComposerMessageRevision: ((v: unknown) => v) as any,
    createAppRouter: (() => ({})) as any,
    createAppMenu: (() => ({})) as any,
    ElementPlus: {} as any,
    baseUrl: '/',
    hasWindow: () => false,
  });

  expect(userTimeZoneResolver?.()).toBe('Europe/Berlin');
});

test('setupApp > falls back to identity timezone and swallows auth lookup failures', () => {
  let userTimeZoneResolver: (() => string | null) | undefined;
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const i18nLocale = ref('en');

  setupApp(makeApp() as any, {
    registerGlobalDirectives: fnRecorder() as any,
    createPinia: (() => ({ use: () => ({}) })) as any,
    piniaPluginPersistedstate: (() => {}) as any,
    useI18nStore: (() => ({
      currentLocale,
      terminologyLang: 'en_US',
      lastTerminologyLoad: null,
      getDateTimeFormats: () => ({}),
      getNumberFormats: () => ({}),
      loadVueI18nMessages: async () => null,
    })) as any,
    setUserTimeZoneResolver: (resolver: () => string | null) => {
      userTimeZoneResolver = resolver;
    },
    setGlobalRequestContextProvider: fnRecorder() as any,
    resolveRequestTimezone: ((a: string, b: string | null) => a || b || '') as any,
    detectBrowserTimezone: (() => '') as any,
    useAuthStore: (() => ({
      get currentUser() {
        throw new Error('auth unavailable');
      },
      identity: { metadata: { timezone: 'America/New_York' } },
    })) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage: fnRecorder() },
    })) as any,
    sourceMessages: {} as any,
    createTerminologyCatalogMerger: (() => fnRecorder()) as any,
    projectTerminologyMessages: (m: unknown) => m as any,
    exposeBrowserI18nOnWindow: fnRecorder() as any,
    notifyComposerMessagesChanged,
    trackComposerMessageRevision: ((v: unknown) => v) as any,
    createAppRouter: (() => ({})) as any,
    createAppMenu: (() => ({})) as any,
    ElementPlus: {} as any,
    baseUrl: '/',
    hasWindow: () => false,
  });

  expect(userTimeZoneResolver?.()).toBeNull();
});

function baseDeps(overrides: Partial<SetupAppDeps> & Record<string, unknown> = {}): SetupAppDeps {
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const lastTerminologyLoad = ref<unknown>(null);
  const i18nLocale = ref('en');
  const store = {
    currentLocale,
    terminologyLang: 'en_US',
    get lastTerminologyLoad() {
      return lastTerminologyLoad.value;
    },
    getDateTimeFormats: () => ({}),
    getNumberFormats: () => ({}),
    loadVueI18nMessages: async () => null as unknown,
  };
  const deps: SetupAppDeps & { _store: typeof store; _lastTerminologyLoad: typeof lastTerminologyLoad; _i18nLocale: typeof i18nLocale } = {
    registerGlobalDirectives: fnRecorder() as any,
    createPinia: (() => ({ use: () => ({}) })) as any,
    piniaPluginPersistedstate: (() => {}) as any,
    useI18nStore: (() => store) as any,
    setUserTimeZoneResolver: fnRecorder() as any,
    setGlobalRequestContextProvider: fnRecorder() as any,
    resolveRequestTimezone: ((userTz: string, browserTz: string | null) => userTz || browserTz || '') as any,
    detectBrowserTimezone: (() => 'Asia/Shanghai') as any,
    useAuthStore: (() => ({})) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage: fnRecorder() },
    })) as any,
    sourceMessages: {} as any,
    createTerminologyCatalogMerger: (() => fnRecorder()) as any,
    projectTerminologyMessages: (m: unknown) => m as any,
    exposeBrowserI18nOnWindow: fnRecorder() as any,
    notifyComposerMessagesChanged,
    trackComposerMessageRevision: ((v: unknown) => v) as any,
    createAppRouter: (() => ({})) as any,
    createAppMenu: (() => ({})) as any,
    ElementPlus: {} as any,
    baseUrl: '/',
    hasWindow: () => false,
    _store: store,
    _lastTerminologyLoad: lastTerminologyLoad,
    _i18nLocale: i18nLocale,
    ...overrides,
  };
  return deps;
}

test('setupApp > builds request context with terminology lang and resolved tz', () => {
  let requestContextProvider: (() => Record<string, string>) | undefined;
  const deps = baseDeps({
    useAuthStore: (() => ({ currentUser: { Timezone: 'Europe/Berlin' } })) as any,
    resolveRequestTimezone: (() => 'Europe/Berlin') as any,
    setGlobalRequestContextProvider: (provider: () => Record<string, string>) => {
      requestContextProvider = provider;
    },
  });
  (deps as any)._store.currentLocale.code = 'zh-CN';
  (deps as any)._store.currentLocale.elementLocale = { name: 'zh-CN' };

  setupApp(makeApp() as any, deps);

  expect(requestContextProvider?.()).toEqual({
    locale: 'zh-CN',
    lang: 'en_US',
    tz: 'Europe/Berlin',
  });
});

test('setupApp > omits tz from request context when unresolved', () => {
  let requestContextProvider: (() => Record<string, string>) | undefined;
  setupApp(
    makeApp() as any,
    baseDeps({
      resolveRequestTimezone: (() => '') as any,
      setGlobalRequestContextProvider: (provider: () => Record<string, string>) => {
        requestContextProvider = provider;
      },
    })
  );

  expect(requestContextProvider?.()).toEqual({
    locale: 'en',
    lang: 'en_US',
  });
});

test('setupApp > swallows auth errors while building request context timezone', () => {
  let requestContextProvider: (() => Record<string, string>) | undefined;
  setupApp(
    makeApp() as any,
    baseDeps({
      useAuthStore: (() => ({
        get currentUser() {
          throw new Error('auth unavailable');
        },
      })) as any,
      resolveRequestTimezone: (() => 'UTC') as any,
      setGlobalRequestContextProvider: (provider: () => Record<string, string>) => {
        requestContextProvider = provider;
      },
    })
  );

  expect(requestContextProvider?.()).toEqual({
    locale: 'en',
    lang: 'en_US',
    tz: 'UTC',
  });
});

test('setupApp > updates Element Plus locale and legacy messages on locale change', async () => {
  const elementLocale = { name: 'zh-CN' };
  const app = makeApp(elementLocale);
  const mergeLocaleMessage = fnRecorder();
  const loadVueI18nMessages = fnRecorder(async () => ({ legacy: 'messages' }));
  const i18nLocale = ref('en');
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const store = {
    currentLocale,
    terminologyLang: 'en_US',
    lastTerminologyLoad: null as unknown,
    getDateTimeFormats: () => ({}),
    getNumberFormats: () => ({}),
    loadVueI18nMessages,
  };

  setupApp(app as any, {
    ...baseDeps(),
    useI18nStore: (() => store) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage },
    })) as any,
  });

  currentLocale.code = 'zh-CN';
  currentLocale.elementLocale = elementLocale;
  await nextTick();
  await nextTick();

  expect(app.config.globalProperties.$ELEMENT.locale).toEqual(elementLocale);
  expect(loadVueI18nMessages.calls.map(c => c[0])).toEqual(['zh-CN']);
  expect(mergeLocaleMessage.calls).toEqual([['zh-CN', { legacy: 'messages' }]]);
  expect(i18nLocale.value).toBe('zh-CN');
});

test('setupApp > warns when legacy locale messages fail to load', async () => {
  const warns: unknown[][] = [];
  const originalWarn = console.warn;
  console.warn = (...args: unknown[]) => {
    warns.push(args);
  };
  try {
    const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
    const i18nLocale = ref('en');
    const loadVueI18nMessages = fnRecorder(async () => {
      throw new Error('network');
    });
    setupApp(makeApp() as any, {
      ...baseDeps(),
      useI18nStore: (() => ({
        currentLocale,
        terminologyLang: 'en_US',
        lastTerminologyLoad: null,
        getDateTimeFormats: () => ({}),
        getNumberFormats: () => ({}),
        loadVueI18nMessages,
      })) as any,
      createI18n: (() => ({
        global: { locale: i18nLocale, mergeLocaleMessage: fnRecorder() },
      })) as any,
    });

    currentLocale.code = 'zh-CN';
    currentLocale.elementLocale = { name: 'zh-CN' };
    await nextTick();
    await nextTick();

    expect(warns.length).toBe(1);
    expect(String(warns[0][0])).toContain('Failed to load legacy locale messages for zh-CN');
    expect(warns[0][1] instanceof Error).toBe(true);
  } finally {
    console.warn = originalWarn;
  }
});

test('setupApp > merges terminology catalog updates from the i18n store', async () => {
  const mergeLocaleMessage = fnRecorder();
  const terminologyMerger = fnRecorder();
  const lastTerminologyLoad = ref<unknown>(null);
  const currentLocale = reactive({ code: 'en', elementLocale: { name: 'en' } });
  const i18nLocale = ref('en');
  const store = {
    currentLocale,
    terminologyLang: 'en_US',
    get lastTerminologyLoad() {
      return lastTerminologyLoad.value;
    },
    getDateTimeFormats: () => ({}),
    getNumberFormats: () => ({}),
    loadVueI18nMessages: async () => null,
  };

  setupApp(makeApp() as any, {
    ...baseDeps(),
    useI18nStore: (() => store) as any,
    createI18n: (() => ({
      global: { locale: i18nLocale, mergeLocaleMessage },
    })) as any,
    createTerminologyCatalogMerger: ((opts: { merge: (locale: string, messages: unknown) => void }) => {
      return (terminology: unknown, locale: string) => {
        terminologyMerger(terminology, locale);
        opts.merge(locale, terminology);
      };
    }) as any,
    projectTerminologyMessages: (m: unknown) => m as any,
  });

  lastTerminologyLoad.value = { auth: { menu: { Users: '用户' } } };
  currentLocale.code = 'zh-CN';
  currentLocale.elementLocale = { name: 'zh-CN' };
  await nextTick();

  expect(terminologyMerger.calls).toEqual([[{ auth: { menu: { Users: '用户' } } }, 'zh-CN']]);
  expect(mergeLocaleMessage.calls.length).toBeGreaterThan(0);
});
