// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore, type PiniaPluginContext } from 'pinia';
import { ref, computed, shallowRef } from 'vue';
import { isClient } from '@vueuse/core';
import dayjs from 'dayjs';
import { SUPPORTED_LOCALES } from './locales';
import { SupportedLocale, DateTimeFormatType } from './types';
import { detectBestLocale, updateDocumentDirection, formatDateTime, formatNumber, formatCurrency, getDateTimeFormats, getNumberFormats } from './utils';
import { loadElementLocale, loadDayjsLocale, loadVueI18nMessages } from './loader';

// Re-export types for external consumers.
export * from './types';
export { SUPPORTED_LOCALES } from './locales';
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

    /**
     * Set the application locale.
     */
    async function setLocale(locale: string) {
      // Check whether the locale is supported.
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
      // Locale support information.
      supportedLocales: Object.keys(SUPPORTED_LOCALES) as SupportedLocale[],

      // Locale management methods.
      setLocale,

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
