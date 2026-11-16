// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineStore, type PiniaPluginContext } from 'pinia';
import { ref, computed, shallowRef } from 'vue';
import { isClient } from '@vueuse/core';
import { DEFAULT_ACTIVE_LANGS } from './activeLangs';
import { EN_US_FORMAT_DEFAULTS, type ActiveLanguageRow, type DateTimeFormatType } from './types';
import {
  detectBestLang,
  updateDocumentDirection,
  formatDateTime,
  formatNumber,
  formatCurrency,
  getDateTimeFormats,
  getNumberFormats,
} from './utils';
import { bcp47FromCode } from './bcp47';
import { fetchWebTranslations, type TerminologyLoadResult } from './terminologyLoader';
import { revalidateTerminologyCatalog } from './terminologyRevalidate';
import { shouldNoopSetLang } from './setLangGuard';
import { createModelStore } from '@/web/web/stores/registry';
import {
  collectActiveLanguages,
  resolveFormatConfig,
  type LanguageFormatOverlay,
} from './languageFormat';

type CreateModelStore = typeof createModelStore;
let createModelStoreImpl: CreateModelStore = createModelStore;

/**
 * Overrides createModelStore used by i18n RPC helpers (FE unit tests).
 * Pass null to restore the default registry lookup.
 */
export function setI18nCreateStoreForTests(fn: CreateModelStore | null): void {
  createModelStoreImpl = fn ?? createModelStore;
}

// Re-export types for external consumers.
export * from './types';
export { bcp47FromCode } from './bcp47';
export { DEFAULT_ACTIVE_LANGS } from './activeLangs';
export { fetchWebTranslations } from './terminologyLoader';
export { shouldNoopSetLang } from './setLangGuard';
export { detectBestLang, matchBestLang } from './utils';

export { componentHintFromScope } from './componentHint';
export { afterLangChange } from './langRemount';
export {
  resolveFormatConfig,
  formatNumberFromConfig,
  formatFixedDecimalString,
  formatCurrencyFromConfig,
  parseGrouping,
  applyGrouping,
  collectActiveLanguages,
  overlayFromActiveLanguageRow,
  type LanguageFormatOverlay,
} from './languageFormat';

export type { TerminologyLoadResult } from './terminologyLoader';
export type { ActiveLanguageRow };

/**
 * I18n state store — identity is POSIX Language.Code.
 */
export const useI18nStore = defineStore(
  'choysum.web.i18n',
  () => {
    const lang = ref<string | null>(null);

    const isInitialized = ref(false);
    const isLoading = ref(false);

    const terminologyHashByLang = ref<Record<string, string>>({});
    const lastTerminologyLoad = shallowRef<TerminologyLoadResult | null>(null);

    let terminologyCatalogGeneration = 0;
    let setLangGeneration = 0;
    let pendingLang: string | null = null;

    const activeLanguages = ref<ActiveLanguageRow[]>([]);
    const formatByCode = ref<Record<string, LanguageFormatOverlay>>({});

    const activeCodes = computed(() =>
      activeLanguages.value.map(r => r.Code).filter(Boolean)
    );

    const activeLanguage = computed(() => {
      const code =
        lang.value && activeCodes.value.includes(lang.value)
          ? lang.value
          : detectBestLang(activeCodes.value.length ? activeCodes.value : [...DEFAULT_ACTIVE_LANGS]);

      const row = activeLanguages.value.find(r => r.Code === code) || null;
      const overlay = formatByCode.value[code] || null;
      const resolved = resolveFormatConfig(overlay);
      const defaults = EN_US_FORMAT_DEFAULTS;

      return {
        code,
        name: row?.Name || overlay?.Name || code,
        textDirection: (row?.Direction || overlay?.Direction || 'ltr') as 'ltr' | 'rtl',
        bcp47: bcp47FromCode(code),
        numberFormat: resolved.numberFormat,
        dateTimeFormat: resolved.dateTimeFormat,
        currencyFormat: {
          ...defaults.currencyFormat,
          ...(resolved.currencyCodeOverride ? { code: resolved.currencyCodeOverride } : {}),
          ...(resolved.currencySymbolPosition ? { position: resolved.currencySymbolPosition } : {}),
          ...(resolved.currencySymbolSpacing != null ? { spacing: resolved.currencySymbolSpacing } : {}),
        },
      };
    });

    const bcp47Locale = computed(() => activeLanguage.value.bcp47);

    /**
     * Set the application language (POSIX Language.Code + Gateway terminology).
     * Same lang after initialize is a no-op.
     */
    async function setLang(code: string) {
      const next = String(code || '').trim();
      if (!next) {
        console.warn(`Language ${next} is not in the active set`);
        return false;
      }

      const inActive = activeCodes.value.includes(next);
      if (
        inActive &&
        shouldNoopSetLang({
          lang: next,
          currentLang: lang.value,
          isInitialized: isInitialized.value,
        }) &&
        pendingLang == null
      ) {
        return true;
      }

      let generation: number;

      if (!inActive) {
        generation = ++setLangGeneration;
        pendingLang = next;
        isLoading.value = true;
        await loadActiveLanguages();
        if (generation !== setLangGeneration) {
          return false;
        }
        const allowed = activeCodes.value.length ? activeCodes.value : [...DEFAULT_ACTIVE_LANGS];
        if (!allowed.includes(next)) {
          console.warn(`Language ${next} is not in the active set`);
          pendingLang = null;
          isLoading.value = false;
          return false;
        }
      } else {
        generation = ++setLangGeneration;
        pendingLang = next;
        isLoading.value = true;
      }

      try {
        const prevHash = terminologyHashByLang.value[next] || '';
        try {
          const res = await fetchWebTranslations(next, prevHash || undefined);
          if (generation !== setLangGeneration) {
            return false;
          }
          if (res.hash) {
            terminologyHashByLang.value = {
              ...terminologyHashByLang.value,
              [next]: res.hash,
            };
          }
          lastTerminologyLoad.value = {
            lang: res.lang,
            bcp47: res.bcp47 || bcp47FromCode(next),
            hash: res.hash,
            unchanged: res.unchanged,
            messages: res.unchanged ? null : res.messages,
          };
        } catch (error) {
          if (generation !== setLangGeneration) {
            return false;
          }
          console.warn('Failed to load terminology from Gateway; falling back to msgid', error);
          lastTerminologyLoad.value = {
            lang: next,
            bcp47: bcp47FromCode(next),
            hash: prevHash,
            unchanged: false,
            messages: null,
            gatewayError: true,
          };
        }

        if (isClient) {
          try {
            await loadActiveLanguages();
          } catch {
            // Best-effort; EN_US defaults remain.
          }
        }

        await Promise.resolve();
        if (generation !== setLangGeneration) {
          return false;
        }

        if (activeCodes.value.length && !activeCodes.value.includes(next)) {
          console.warn(`Language ${next} is not in the active set`);
          return false;
        }

        lang.value = next;

        if (isClient) {
          updateDocumentDirection(activeLanguage.value.textDirection);
        }

        return true;
      } catch (error) {
        console.error('Failed to set language:', error);
        return false;
      } finally {
        if (generation === setLangGeneration) {
          pendingLang = null;
          isLoading.value = false;
        }
      }
    }

    function wrappedFormatDateTime(
      date: Date | string | number,
      options?: {
        type?: DateTimeFormatType;
        format?: string;
        isLong?: boolean;
        timeZone?: string;
      }
    ): string {
      return formatDateTime(date, activeLanguage.value.dateTimeFormat, {
        ...options,
        bcp47: activeLanguage.value.bcp47,
      });
    }

    function wrappedFormatNumber(value: number, options?: { digits?: number }) {
      return formatNumber(
        value,
        activeLanguage.value.bcp47,
        activeLanguage.value.numberFormat,
        options
      );
    }

    function wrappedFormatCurrency(value: number, currencyCode?: string) {
      const loc = activeLanguage.value;
      return formatCurrency(
        value,
        loc.bcp47,
        {
          ...loc.currencyFormat,
          thousandsSeparator: loc.numberFormat?.thousandsSeparator,
          decimalSeparator: loc.numberFormat?.decimalSeparator,
          grouping: loc.numberFormat?.grouping,
        },
        currencyCode
      );
    }

    function wrappedGetDateTimeFormats() {
      return getDateTimeFormats(activeLanguage.value.code, activeLanguage.value.dateTimeFormat);
    }

    function wrappedGetNumberFormats() {
      return getNumberFormats(
        activeLanguage.value.code,
        activeLanguage.value.numberFormat,
        activeLanguage.value.currencyFormat
      );
    }

    async function reloadTerminology(targetLang?: string): Promise<TerminologyLoadResult | null> {
      const next = String(targetLang || lang.value || '').trim();
      if (!next) {
        return null;
      }
      const generation = ++terminologyCatalogGeneration;
      try {
        const res = await fetchWebTranslations(next, undefined);
        if (generation !== terminologyCatalogGeneration) {
          return null;
        }
        if (res.hash) {
          terminologyHashByLang.value = {
            ...terminologyHashByLang.value,
            [next]: res.hash,
          };
        }
        const load: TerminologyLoadResult = {
          lang: res.lang,
          bcp47: res.bcp47 || bcp47FromCode(next),
          hash: res.hash,
          unchanged: false,
          messages: res.messages ?? {},
        };
        lastTerminologyLoad.value = load;
        return load;
      } catch (error) {
        console.warn('Failed to reload terminology from Gateway', error);
        if (generation !== terminologyCatalogGeneration) {
          return null;
        }
        const load: TerminologyLoadResult = {
          lang: next,
          bcp47: bcp47FromCode(next),
          hash: terminologyHashByLang.value[next] || '',
          unchanged: false,
          messages: null,
          gatewayError: true,
        };
        lastTerminologyLoad.value = load;
        return load;
      }
    }

    async function revalidateTerminologyIfStale(): Promise<TerminologyLoadResult | null> {
      if (!isInitialized.value) {
        return null;
      }
      const next =
        lang.value && activeCodes.value.includes(lang.value)
          ? lang.value
          : activeLanguage.value.code;
      const prevHash = terminologyHashByLang.value[next] || '';
      const generation = ++terminologyCatalogGeneration;
      try {
        const load = await revalidateTerminologyCatalog({
          lang: next,
          prevHash,
        });
        if (generation !== terminologyCatalogGeneration) {
          return null;
        }
        if (load.hash) {
          terminologyHashByLang.value = {
            ...terminologyHashByLang.value,
            [next]: load.hash,
          };
        }
        if (!load.unchanged) {
          lastTerminologyLoad.value = load;
        }
        return load;
      } catch (error) {
        if (generation !== terminologyCatalogGeneration) {
          return null;
        }
        console.warn('Failed to revalidate terminology from Gateway', error);
        return null;
      }
    }

    /**
     * Load active languages from base.Language/GetActiveLanguages (gRPC-Web).
     */
    async function loadActiveLanguages(): Promise<boolean> {
      try {
        const languageStore = createModelStoreImpl('base.Language');
        const rows = await (languageStore as any).GetActiveLanguages();
        const { rows: nextRows, byCode } = collectActiveLanguages(rows);
        formatByCode.value = byCode;
        activeLanguages.value =
          nextRows.length > 0
            ? nextRows
            : DEFAULT_ACTIVE_LANGS.map(code => ({ Code: code }));
        return true;
      } catch (error) {
        console.warn('Failed to load active languages; using defaults', error);
        if (activeLanguages.value.length === 0) {
          formatByCode.value = {};
          activeLanguages.value = DEFAULT_ACTIVE_LANGS.map(code => ({ Code: code }));
        }
        return false;
      }
    }

    async function initialize() {
      if (isInitialized.value) {
        return;
      }

      const loaded = isClient ? await loadActiveLanguages() : true;

      const allowed = activeCodes.value.length ? activeCodes.value : [...DEFAULT_ACTIVE_LANGS];
      let langToSet = lang.value;

      if (!langToSet || !allowed.includes(langToSet)) {
        if (!langToSet) {
          langToSet = detectBestLang(allowed);
        } else if (loaded) {
          langToSet = detectBestLang(allowed);
        } else if (!activeLanguages.value.some(r => r.Code === langToSet)) {
          activeLanguages.value = [...activeLanguages.value, { Code: langToSet }];
        }
      }

      let ok = await setLang(langToSet);
      if (!ok) {
        const allowedNow = activeCodes.value.length ? activeCodes.value : [...DEFAULT_ACTIVE_LANGS];
        ok = await setLang(detectBestLang(allowedNow));
      }
      if (!ok) {
        const allowedNow = activeCodes.value.length ? activeCodes.value : [...DEFAULT_ACTIVE_LANGS];
        lang.value = allowedNow.includes('en_US') ? 'en_US' : allowedNow[0] || 'en_US';
      }

      if (isClient) {
        updateDocumentDirection(activeLanguage.value.textDirection);
      }

      isInitialized.value = true;
    }

    return {
      lang,
      activeLanguage,
      bcp47Locale,
      isInitialized,
      isLoading,
      terminologyHashByLang,
      lastTerminologyLoad,
      activeLanguages,
      formatByCode,
      initialize,
      setLang,
      loadActiveLanguages,
      reloadTerminology,
      revalidateTerminologyIfStale,
      formatDateTime: wrappedFormatDateTime,
      formatNumber: wrappedFormatNumber,
      formatCurrency: wrappedFormatCurrency,
      getDateTimeFormats: wrappedGetDateTimeFormats,
      getNumberFormats: wrappedGetNumberFormats,
      getFirstDayOfWeek: () => activeLanguage.value.dateTimeFormat?.firstDayOfWeek ?? 1,
      getPersonNameFormat: (posixCode?: string) => {
        const code = String(posixCode || '').trim() || activeLanguage.value.code;
        const raw = formatByCode.value[code]?.PersonNameFormat;
        return raw != null ? String(raw) : undefined;
      },
    };
  },
  {
    persist: {
      key: 'choysum.web.i18n',
      storage: localStorage,
      pick: ['lang'],
      afterHydrate: (ctx: PiniaPluginContext) => {
        if (ctx.store.$id === 'choysum.web.i18n') {
          const i18nStore = ctx.store;
          void i18nStore.initialize();
        }
      },
    },
  }
);
