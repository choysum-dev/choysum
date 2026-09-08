// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import type { ChoysumWebApp } from '@/core/web/application';
import { createAppMenu } from './menu';
import { createI18n } from 'vue-i18n';
import { useI18nStore } from './stores/i18nStore';
import { watch } from 'vue';
import sourceMessages from './i18n/source';
import { createAppRouter } from './router';
import { registerGlobalDirectives } from './directives';
import { setGlobalRequestContextProvider } from '@/core/rpc/context';
import { createTerminologyCatalogMerger } from './stores/i18nStore/merge';
import { projectTerminologyMessages } from './i18n/terminology';
import {
  exposeBrowserI18nOnWindow,
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
} from './i18n';
import { detectBrowserTimezone, resolveRequestTimezone } from './utils/request_timezone';
import { setUserTimeZoneResolver } from './utils/datetime';
import { useAuthStore } from '@/auth/web/stores/auth';
import ElementPlus from 'element-plus';

/** Optional overrides for unit tests; production callers omit this. */
export type SetupAppDeps = {
  registerGlobalDirectives?: typeof registerGlobalDirectives;
  createPinia?: typeof createPinia;
  piniaPluginPersistedstate?: typeof piniaPluginPersistedstate;
  useI18nStore?: typeof useI18nStore;
  setUserTimeZoneResolver?: typeof setUserTimeZoneResolver;
  setGlobalRequestContextProvider?: typeof setGlobalRequestContextProvider;
  resolveRequestTimezone?: typeof resolveRequestTimezone;
  detectBrowserTimezone?: typeof detectBrowserTimezone;
  useAuthStore?: typeof useAuthStore;
  createI18n?: typeof createI18n;
  sourceMessages?: typeof sourceMessages;
  createTerminologyCatalogMerger?: typeof createTerminologyCatalogMerger;
  projectTerminologyMessages?: typeof projectTerminologyMessages;
  exposeBrowserI18nOnWindow?: typeof exposeBrowserI18nOnWindow;
  notifyComposerMessagesChanged?: typeof notifyComposerMessagesChanged;
  trackComposerMessageRevision?: typeof trackComposerMessageRevision;
  createAppRouter?: typeof createAppRouter;
  createAppMenu?: typeof createAppMenu;
  ElementPlus?: typeof ElementPlus;
  baseUrl?: string;
  hasWindow?: () => boolean;
};

function pickDep<T>(override: T | undefined, fallback: T): T {
  return override !== undefined ? override : fallback;
}

export function setupApp(app: ChoysumWebApp, deps: SetupAppDeps = {}): void {
  const registerDirectives = pickDep(deps.registerGlobalDirectives, registerGlobalDirectives);
  const makePinia = pickDep(deps.createPinia, createPinia);
  const piniaPersist = pickDep(deps.piniaPluginPersistedstate, piniaPluginPersistedstate);
  const resolveI18nStore = pickDep(deps.useI18nStore, useI18nStore);
  const setTzResolver = pickDep(deps.setUserTimeZoneResolver, setUserTimeZoneResolver);
  const setRequestContext = pickDep(deps.setGlobalRequestContextProvider, setGlobalRequestContextProvider);
  const resolveTz = pickDep(deps.resolveRequestTimezone, resolveRequestTimezone);
  const detectBrowserTz = pickDep(deps.detectBrowserTimezone, detectBrowserTimezone);
  const resolveAuthStore = pickDep(deps.useAuthStore, useAuthStore);
  const makeI18n = pickDep(deps.createI18n, createI18n);
  const messages = pickDep(deps.sourceMessages, sourceMessages);
  const makeTerminologyMerger = pickDep(deps.createTerminologyCatalogMerger, createTerminologyCatalogMerger);
  const projectTerminology = pickDep(deps.projectTerminologyMessages, projectTerminologyMessages);
  const exposeBrowserI18n = pickDep(deps.exposeBrowserI18nOnWindow, exposeBrowserI18nOnWindow);
  const notifyMessagesChanged = pickDep(deps.notifyComposerMessagesChanged, notifyComposerMessagesChanged);
  const trackRevision = pickDep(deps.trackComposerMessageRevision, trackComposerMessageRevision);
  const makeRouter = pickDep(deps.createAppRouter, createAppRouter);
  const makeMenu = pickDep(deps.createAppMenu, createAppMenu);
  const elementPlus = pickDep(deps.ElementPlus, ElementPlus);
  const baseUrl = pickDep(deps.baseUrl, import.meta.env?.BASE_URL ?? '/');
  const hasWindow = pickDep(deps.hasWindow, () => typeof window !== 'undefined');

  registerDirectives(app);

  const pinia = makePinia().use(piniaPersist);
  app.usePlugin('pinia', pinia, {}, false);

  const i18nStore = resolveI18nStore();

  setTzResolver(() => {
    try {
      const authStore = resolveAuthStore();
      return (authStore.currentUser as any)?.Timezone ?? (authStore.identity as any)?.metadata?.timezone;
    } catch {
      return null;
    }
  });

  setRequestContext(() => {
    let userTz = '';
    try {
      const authStore = resolveAuthStore();
      userTz = resolveTz(
        (authStore.currentUser as any)?.Timezone ?? (authStore.identity as any)?.metadata?.timezone,
        null
      );
    } catch {
      userTz = '';
    }
    const tz = resolveTz(userTz, detectBrowserTz());
    return {
      locale: i18nStore.currentLocale.code,
      lang: i18nStore.terminologyLang,
      ...(tz ? { tz } : {}),
    };
  });

  const i18n = makeI18n<false, { [key: string]: any }>({
    legacy: false,
    locale: i18nStore.currentLocale.code,
    fallbackLocale: 'en',
    missingWarn: false,
    fallbackWarn: false,
    messages: {
      en: messages,
    },
    postTranslation: trackRevision,
    datetimeFormats: i18nStore.getDateTimeFormats(),
    numberFormats: i18nStore.getNumberFormats(),
  });
  const mergeTerminologyCatalog = makeTerminologyMerger({
    merge: (locale, messagesToMerge) => {
      i18n.global.mergeLocaleMessage(locale, projectTerminology(messagesToMerge));
    },
    notify: notifyMessagesChanged,
  });

  if (hasWindow()) {
    exposeBrowserI18n(i18n.global);
  }

  watch(
    () => i18nStore.currentLocale.code,
    async newLocale => {
      if (app.config.globalProperties.$ELEMENT) {
        app.config.globalProperties.$ELEMENT.locale = i18nStore.currentLocale.elementLocale;
      }

      if (newLocale !== 'en') {
        try {
          const legacy = await i18nStore.loadVueI18nMessages(newLocale);
          if (legacy) {
            i18n.global.mergeLocaleMessage(newLocale, legacy);
          }
        } catch (error) {
          console.warn(`Failed to load legacy locale messages for ${newLocale}`, error);
        }
      }

      i18n.global.locale.value = newLocale;
    },
    { immediate: true }
  );

  watch(
    () => i18nStore.lastTerminologyLoad,
    terminology => {
      mergeTerminologyCatalog(terminology, i18nStore.currentLocale.code);
    }
  );

  app.usePlugin('i18n', i18n);

  const router = makeRouter(baseUrl, i18n.global);
  app.usePlugin('router', router);

  const menuPlugin = makeMenu();
  app.usePlugin('menu', menuPlugin);

  app.usePlugin('element-plus', elementPlus, {
    locale: i18nStore.currentLocale.elementLocale,
  });
}
