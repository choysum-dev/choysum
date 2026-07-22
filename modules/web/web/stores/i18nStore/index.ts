// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore, type PiniaPluginContext } from 'pinia';
import { ref, computed, shallowRef } from 'vue';
import { isClient } from '@vueuse/core';
import dayjs from 'dayjs';
import { SUPPORTED_LOCALES } from './locales';
import { DEFAULT_ACTIVE_UI_KEYS } from './active_ui_keys';
import { SupportedLocale, DateTimeFormatType } from './types';
import { detectBestUiKey, updateDocumentDirection, formatDateTime, formatNumber, formatCurrency, getDateTimeFormats, getNumberFormats } from './utils';
import { loadElementLocale, loadDayjsLocale, loadVueI18nMessages } from './loader';
import { uiKeyToLang, langToUiKey } from './lang';
import { fetchWebTranslations, type TerminologyLoadResult } from './terminology_loader';
import { afterLocaleChange } from './locale_remount';
import { createStoreByModel } from '@/web/web/stores/registry';

// Re-export types for external consumers.
export * from './types';
export { SUPPORTED_LOCALES } from './locales';
export { uiKeyToLang, langToUiKey } from './lang';
export { fetchWebTranslations } from './terminology_loader';
export { fetchTerms, patchTerms, downloadTerminologyPo } from './terms_api';

export { componentHintFromScope } from './component_hint';
export { afterLocaleChange, resolveLocaleRemountMode, softLocaleRemount } from './locale_remount';

export type { TerminologyLoadResult } from './terminology_loader';
export type { TermItem, TermsListResponse } from './terms_api';
export type { SupportedLocale };

/**
 * I18n state store.
 */
export const useI18nStore = defineStore(
  'choysum.web.i18n',
  () => {
    const localeCode = ref<SupportedLocale | null>(null);

    // Initialization state.
    const isInitialized = ref(false);

    // Cache of loaded locale resources.
    const loadedLocales = shallowRef<Record<string, any>>({});

    // Current loading state.
    const isLoading = ref(false);

    // Terminology catalogHash by lang (for Gateway hash negotiation).
    const terminologyHashByLang = ref<Record<string, string>>({});

    // Last Gateway load result (consumed by app.ts for mergeLocaleMessage).
    const lastTerminologyLoad = shallowRef<TerminologyLoadResult | null>(null);

    // Language switcher codes (Language.IsActive ∩ format catalog; fallback DEFAULT_ACTIVE_UI_KEYS).
    const activeUiKeys = ref<string[]>([...DEFAULT_ACTIVE_UI_KEYS]);

    // Current locale config, combining the code with its metadata.
    const currentLocale = computed(() => {
      if (!localeCode.value || !(localeCode.value in SUPPORTED_LOCALES)) {
        localeCode.value = detectBestUiKey();
      }

      const code = localeCode.value;
      const config = SUPPORTED_LOCALES[code];

      return {
        code,
        ...config,
        elementLocale: loadedLocales.value[code] || null,
      };
    });

    const terminologyLang = computed(() => uiKeyToLang(currentLocale.value.code));

    /**
     * Set the application locale (format UI + Gateway terminology).
     */
    async function setUiKey(locale: string) {
      // Check whether the locale is supported for format metadata.
      if (!(locale in SUPPORTED_LOCALES)) {
        console.warn(`Locale ${locale} is not supported`);
        return false;
      }

      isLoading.value = true;

      try {
        // Load resources on demand when they are not cached yet.
        if (!loadedLocales.value[locale]) {
          // Load the Element Plus locale bundle.
          const elementLocaleData = await loadElementLocale(locale);

          // Load the DayJS locale bundle.
          await loadDayjsLocale(locale);

          // Cache loaded resources.
          loadedLocales.value = {
            ...loadedLocales.value,
            [locale]: elementLocaleData,
          };
        }

        // Load terminology from host Gateway (S4-1); failures fall back to msgid.
        const lang = uiKeyToLang(locale);
        const prevHash = terminologyHashByLang.value[lang] || '';
        try {
          const res = await fetchWebTranslations(lang, prevHash || undefined);
          if (res.hash) {
            terminologyHashByLang.value = {
              ...terminologyHashByLang.value,
              [lang]: res.hash,
            };
          }
          lastTerminologyLoad.value = {
            lang: res.lang,
            locale: res.locale || locale,
            hash: res.hash,
            unchanged: res.unchanged,
            messages: res.unchanged ? null : res.messages,
          };
        } catch (error) {
          console.warn('Failed to load terminology from Gateway; falling back to msgid', error);
          lastTerminologyLoad.value = {
            lang,
            locale,
            hash: prevHash,
            unchanged: false,
            messages: null,
            gatewayError: true,
          };
        }

        // Update the active locale.
        localeCode.value = locale as SupportedLocale;

        // Configure the dayjs locale.
        const dayjsLocaleCode = SUPPORTED_LOCALES[locale as SupportedLocale].dayjsLocaleCode;
        dayjs.locale(dayjsLocaleCode);

        // Update the document direction on the client only.
        if (isClient) {
          updateDocumentDirection(currentLocale.value.textDirection);
        }

        return true;
      } catch (error) {
        console.error('Failed to set locale:', error);
        return false;
      } finally {
        isLoading.value = false;
      }
    }

    /**
     * Restrict the language switcher to active terminology locales.
     */
    function setActiveUiKeys(codes: string[]) {
      const next = codes
        .map(c => String(c || '').trim())
        .filter(c => c && c in SUPPORTED_LOCALES);
      activeUiKeys.value = next.length > 0 ? next : [...DEFAULT_ACTIVE_UI_KEYS];
    }

    /**
     * Wrapped date and time formatter.
     */
    function wrappedFormatDateTime(
      date: Date | string | number,
      options?: {
        type?: DateTimeFormatType;
        format?: string;
        isLong?: boolean;
      }
    ): string {
      return formatDateTime(date, currentLocale.value.dateTimeFormat, options);
    }

    /**
     * Wrapped number formatter.
     */
    function wrappedFormatNumber(value: number, options?: { digits?: number }) {
      return formatNumber(value, currentLocale.value.code, currentLocale.value.numberFormat, options);
    }

    /**
     * Wrapped currency formatter.
     */
    function wrappedFormatCurrency(value: number, currencyCode?: string) {
      return formatCurrency(value, currentLocale.value.code, currentLocale.value.currencyFormat, currencyCode);
    }

    /**
     * Wrapped accessor for date and time format config.
     */
    function wrappedGetDateTimeFormats() {
      return getDateTimeFormats(currentLocale.value.code, currentLocale.value.dateTimeFormat);
    }

    /**
     * Wrapped accessor for number format config.
     */
    function wrappedGetNumberFormats() {
      return getNumberFormats(currentLocale.value.code, currentLocale.value.numberFormat, currentLocale.value.currencyFormat);
    }

    /**
     * Reload terminology for the current (or given) locale from Gateway and
     * refresh lastTerminologyLoad for vue-i18n merge (post Terminology Editor save).
     */
    async function reloadTerminology(locale?: string): Promise<TerminologyLoadResult | null> {
      const targetLocale = String(locale || localeCode.value || '').trim();
      if (!targetLocale || !(targetLocale in SUPPORTED_LOCALES)) {
        return null;
      }
      const lang = uiKeyToLang(targetLocale);
      // Force catalog refresh: omit client hash so Gateway always returns messages.
      try {
        const res = await fetchWebTranslations(lang, undefined);
        if (res.hash) {
          terminologyHashByLang.value = {
            ...terminologyHashByLang.value,
            [lang]: res.hash,
          };
        }
        const load: TerminologyLoadResult = {
          lang: res.lang,
          locale: res.locale || targetLocale,
          hash: res.hash,
          unchanged: false,
          messages: res.messages ?? {},
        };
        lastTerminologyLoad.value = load;
        return load;
      } catch (error) {
        console.warn('Failed to reload terminology from Gateway', error);
        const load: TerminologyLoadResult = {
          lang,
          locale: targetLocale,
          hash: terminologyHashByLang.value[lang] || '',
          unchanged: false,
          messages: null,
          gatewayError: true,
        };
        lastTerminologyLoad.value = load;
        return load;
      }
    }

    /**
     * Load active languages from base.Language/GetActiveLanguages (gRPC-Web).
     * Failures keep DEFAULT_ACTIVE_UI_KEYS.
     */
    async function loadActiveUiKeysFromServer() {
      try {
        const languageStore = createStoreByModel('base.Language');
        const rows = await (languageStore as any).GetActiveLanguages();
        const keys = (rows || [])
          .map((row: any) => langToUiKey(String(row?.Code || '')))
          .filter((code: string) => !!code && code in SUPPORTED_LOCALES);
        setActiveUiKeys(keys);
      } catch (error) {
        console.warn('Failed to load active languages; using defaults', error);
        setActiveUiKeys([...DEFAULT_ACTIVE_UI_KEYS]);
      }
    }

    /**
     * Initialize locale resources.
     */
    async function initialize() {
      if (isInitialized.value) {
        return;
      }

      if (isClient) {
        await loadActiveUiKeysFromServer();
      }

      // Detect only when localeCode is missing or invalid for the active set.
      if (!localeCode.value || !(localeCode.value in SUPPORTED_LOCALES) || !activeUiKeys.value.includes(localeCode.value)) {
        // Honor manual localStorage session: only auto-detect when unset/invalid.
        if (!localeCode.value || !(localeCode.value in SUPPORTED_LOCALES)) {
          localeCode.value = detectBestUiKey(activeUiKeys.value);
        } else if (!activeUiKeys.value.includes(localeCode.value)) {
          localeCode.value = detectBestUiKey(activeUiKeys.value);
        }
      }

      // During SSR, preload only the default locale.
      if (!isClient && localeCode.value === 'en') {
        const enLocale = await loadElementLocale('en');
        loadedLocales.value = { en: enLocale };
        return;
      }

      // Load the current locale resources on the client.
      await setUiKey(localeCode.value);

      // Sync the document direction during client initialization.
      if (isClient) {
        updateDocumentDirection(currentLocale.value.textDirection);
      }

      // Ensure post-async state changes are captured so Pinia tracking reacts correctly.
      isInitialized.value = true;
    }

    // Store state and public methods.
    return {
      // Core state.
      currentLocale,
      isInitialized,
      localeCode,
      initialize,
      terminologyLang,
      terminologyHashByLang,
      lastTerminologyLoad,
      activeUiKeys,
      // Locale support information.
      supportedLocales: Object.keys(SUPPORTED_LOCALES) as SupportedLocale[],

      // Locale management methods.
      setUiKey,
      setActiveUiKeys,
      loadActiveUiKeysFromServer,
      reloadTerminology,

      // Formatting helpers.
      formatDateTime: wrappedFormatDateTime,
      formatNumber: wrappedFormatNumber,
      formatCurrency: wrappedFormatCurrency,

      // Vue I18n integration methods.
      loadVueI18nMessages,
      getDateTimeFormats: wrappedGetDateTimeFormats,
      getNumberFormats: wrappedGetNumberFormats,

      // Utility methods.
      getFirstDayOfWeek: () => currentLocale.value.dateTimeFormat?.firstDayOfWeek ?? 1,
    };
  },
  {
    // Persistence config.
    persist: {
      key: 'choysum.web.i18n',
      storage: localStorage,
      pick: ['localeCode'], // Persist only the locale code.
      afterHydrate: (ctx: PiniaPluginContext) => {
        if (ctx.store.$id === 'choysum.web.i18n') {
          const i18nStore = ctx.store;
          // Set critical attributes such as direction immediately to avoid flicker.
          const code = i18nStore.localeCode;
          if (code && isClient && SUPPORTED_LOCALES[code]) {
            updateDocumentDirection(SUPPORTED_LOCALES[code].textDirection);
          }

          // Start the full initialization flow without blocking.
          i18nStore.initialize();
        }
      },
    },
  }
);
