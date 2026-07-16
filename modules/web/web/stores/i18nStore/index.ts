// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore, type PiniaPluginContext } from 'pinia';
import { ref, computed, shallowRef } from 'vue';
import { isClient } from '@vueuse/core';
import dayjs from 'dayjs';
import { SUPPORTED_LOCALES } from './locales';
import { DEFAULT_ACTIVE_LOCALES } from './active_locales';
import { SupportedLocale, DateTimeFormatType } from './types';
import { detectBestLocale, updateDocumentDirection, formatDateTime, formatNumber, formatCurrency, getDateTimeFormats, getNumberFormats } from './utils';
import { loadElementLocale, loadDayjsLocale, loadVueI18nMessages } from './loader';
import { localeToLang } from './lang';
import { fetchWebTranslations, type TerminologyLoadResult } from './terminology_loader';

// Re-export types for external consumers.
export * from './types';
export { SUPPORTED_LOCALES } from './locales';
export { localeToLang, langToLocale } from './lang';
export { fetchWebTranslations } from './terminology_loader';
export type { TerminologyLoadResult } from './terminology_loader';
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

    // Language switcher codes (Language.IsActive ∩ format catalog; fallback DEFAULT_ACTIVE_LOCALES).
    const activeLocaleCodes = ref<string[]>([...DEFAULT_ACTIVE_LOCALES]);

    // Current locale config, combining the code with its metadata.
    const currentLocale = computed(() => {
      if (!localeCode.value || !(localeCode.value in SUPPORTED_LOCALES)) {
        localeCode.value = detectBestLocale();
      }

      const code = localeCode.value;
      const config = SUPPORTED_LOCALES[code];

      return {
        code,
        ...config,
        elementLocale: loadedLocales.value[code] || null,
      };
    });

    const terminologyLang = computed(() => localeToLang(currentLocale.value.code));

    /**
     * Set the application locale (format UI + Gateway terminology).
     */
    async function setLocale(locale: string) {
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
        const lang = localeToLang(locale);
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
    function setActiveLocales(codes: string[]) {
      const next = codes
        .map(c => String(c || '').trim())
        .filter(c => c && c in SUPPORTED_LOCALES);
      activeLocaleCodes.value = next.length > 0 ? next : [...DEFAULT_ACTIVE_LOCALES];
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
     * Initialize locale resources.
     */
    async function initialize() {
      if (isInitialized.value) {
        return;
      }
      // Detect only when localeCode is missing or invalid.
      if (!localeCode.value || !(localeCode.value in SUPPORTED_LOCALES)) {
        localeCode.value = detectBestLocale();
      }

      // During SSR, preload only the default locale.
      if (!isClient && localeCode.value === 'en') {
        const enLocale = await loadElementLocale('en');
        loadedLocales.value = { en: enLocale };
        return;
      }

      // Load the current locale resources on the client.
      await setLocale(localeCode.value);

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
      activeLocaleCodes,
      // Locale support information.
      supportedLocales: Object.keys(SUPPORTED_LOCALES) as SupportedLocale[],

      // Locale management methods.
      setLocale,
      setActiveLocales,

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
