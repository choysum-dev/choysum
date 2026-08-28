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

export function setupApp(app: ChoysumWebApp): void {
  registerGlobalDirectives(app);

  const pinia = createPinia().use(piniaPluginPersistedstate);
  app.usePlugin('pinia', pinia, {}, false);

  const i18nStore = useI18nStore();

  setUserTimeZoneResolver(() => {
    try {
      const authStore = useAuthStore();
      return (authStore.currentUser as any)?.Timezone ?? (authStore.identity as any)?.metadata?.timezone;
    } catch {
      return null;
    }
  });

  setGlobalRequestContextProvider(() => {
    let userTz = '';
    try {
      const authStore = useAuthStore();
      userTz = resolveRequestTimezone(
        (authStore.currentUser as any)?.Timezone ?? (authStore.identity as any)?.metadata?.timezone,
        null
      );
    } catch {
      userTz = '';
    }
    const tz = resolveRequestTimezone(userTz, detectBrowserTimezone());
    return {
      locale: i18nStore.currentLocale.code,
      lang: i18nStore.terminologyLang,
      ...(tz ? { tz } : {}),
    };
  });

  const i18n = createI18n<false, { [key: string]: any }>({
    legacy: false,
    locale: i18nStore.currentLocale.code,
    fallbackLocale: 'en',
    missingWarn: false,
    fallbackWarn: false,
    messages: {
      en: sourceMessages,
    },
    postTranslation: trackComposerMessageRevision,
    datetimeFormats: i18nStore.getDateTimeFormats(),
    numberFormats: i18nStore.getNumberFormats(),
  });
  const mergeTerminologyCatalog = createTerminologyCatalogMerger({
    merge: (locale, messages) => {
      i18n.global.mergeLocaleMessage(locale, projectTerminologyMessages(messages));
    },
    notify: notifyComposerMessagesChanged,
  });

  if (typeof window !== 'undefined') {
    exposeBrowserI18nOnWindow(i18n.global);
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

  const router = createAppRouter(import.meta.env.BASE_URL, i18n.global);
  app.usePlugin('router', router);

  const menuPlugin = createAppMenu();
  app.usePlugin('menu', menuPlugin);

  app.usePlugin('element-plus', ElementPlus, {
    locale: i18nStore.currentLocale.elementLocale,
  });
}
