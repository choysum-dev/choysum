// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { LocaleConfig } from './types';

/**
 * Supported UI-key catalog: Element / dayjs package names, direction, and soft format fallbacks.
 * Runtime number/date authority is Language (+ optional Preferences.display); catalog is not SSOT.
 * Terminology comes from Gateway + modules/<module>/i18n/*.po (S4).
 */
export const SUPPORTED_LOCALES: Record<string, LocaleConfig> = {
  // East Asian locales.
  'zh-CN': {
    name: '简体中文',
    textDirection: 'ltr',
    dayjsLocaleCode: 'zh-cn',
    importDayjs: () => import('dayjs/locale/zh-cn'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '¥',
      position: 'before',
      code: 'CNY',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'YYYY-MM-DD',
      longDate: 'YYYY年MM月DD日',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1, // Monday.
    },
  },
  'zh-TW': {
    name: '繁體中文',
    textDirection: 'ltr',
    dayjsLocaleCode: 'zh-tw',
    importDayjs: () => import('dayjs/locale/zh-tw'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: 'NT$',
      position: 'before',
      code: 'TWD',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'YYYY/MM/DD',
      longDate: 'YYYY年MM月DD日',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1,
    },
  },
  ja: {
    name: '日本語',
    textDirection: 'ltr',
    dayjsLocaleCode: 'ja',
    importDayjs: () => import('dayjs/locale/ja'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 0,
    },
    currencyFormat: {
      symbol: '¥',
      position: 'before',
      code: 'JPY',
      decimalDigits: 0,
    },
    dateTimeFormat: {
      shortDate: 'YYYY/MM/DD',
      longDate: 'YYYY年MM月DD日',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 0, // Sunday.
    },
  },
  ko: {
    name: '한국어',
    textDirection: 'ltr',
    dayjsLocaleCode: 'ko',
    importDayjs: () => import('dayjs/locale/ko'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 0,
    },
    currencyFormat: {
      symbol: '₩',
      position: 'before',
      code: 'KRW',
      decimalDigits: 0,
    },
    dateTimeFormat: {
      shortDate: 'YYYY. MM. DD',
      longDate: 'YYYY년 MM월 DD일',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 0,
    },
  },

  // European locales.
  en: {
    name: 'English',
    textDirection: 'ltr',
    dayjsLocaleCode: 'en',
    importDayjs: () => import('dayjs/locale/en'),
    importVueI18n: () => import('../../i18n/source'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '$',
      position: 'before',
      code: 'USD',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'MM/DD/YYYY',
      longDate: 'MMMM D, YYYY',
      shortTime: 'h:mm A',
      longTime: 'h:mm:ss A',
      firstDayOfWeek: 0, // Sunday.
    },
  },
  'en-GB': {
    name: 'British English',
    textDirection: 'ltr',
    dayjsLocaleCode: 'en-gb',
    importDayjs: () => import('dayjs/locale/en-gb'),
    numberFormat: {
      thousandsSeparator: ',',
      decimalSeparator: '.',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '£',
      position: 'before',
      code: 'GBP',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD/MM/YYYY',
      longDate: 'D MMMM YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1, // Monday.
    },
  },
  de: {
    name: 'Deutsch',
    textDirection: 'ltr',
    dayjsLocaleCode: 'de',
    importDayjs: () => import('dayjs/locale/de'),
    numberFormat: {
      thousandsSeparator: '.',
      decimalSeparator: ',',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '€',
      position: 'after',
      code: 'EUR',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD.MM.YYYY',
      longDate: 'D. MMMM YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1,
    },
  },
  fr: {
    name: 'Français',
    textDirection: 'ltr',
    dayjsLocaleCode: 'fr',
    importDayjs: () => import('dayjs/locale/fr'),
    numberFormat: {
      thousandsSeparator: ' ',
      decimalSeparator: ',',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '€',
      position: 'after',
      code: 'EUR',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD/MM/YYYY',
      longDate: 'D MMMM YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1,
    },
  },
  es: {
    name: 'Español',
    textDirection: 'ltr',
    dayjsLocaleCode: 'es',
    importDayjs: () => import('dayjs/locale/es'),
    numberFormat: {
      thousandsSeparator: '.',
      decimalSeparator: ',',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '€',
      position: 'after',
      code: 'EUR',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD/MM/YYYY',
      longDate: 'D [de] MMMM [de] YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1,
    },
  },
  it: {
    name: 'Italiano',
    textDirection: 'ltr',
    dayjsLocaleCode: 'it',
    importDayjs: () => import('dayjs/locale/it'),
    numberFormat: {
      thousandsSeparator: '.',
      decimalSeparator: ',',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: '€',
      position: 'after',
      code: 'EUR',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD/MM/YYYY',
      longDate: 'D MMMM YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 1,
    },
  },

  // Other European locales.
  pt: {
    name: 'Português',
    textDirection: 'ltr',
    dayjsLocaleCode: 'pt',
    importDayjs: () => import('dayjs/locale/pt'),
  },
  'pt-BR': {
    name: 'Português (Brasil)',
    textDirection: 'ltr',
    dayjsLocaleCode: 'pt-br',
    importDayjs: () => import('dayjs/locale/pt-br'),
  },
  ru: {
    name: 'Русский',
    textDirection: 'ltr',
    dayjsLocaleCode: 'ru',
    importDayjs: () => import('dayjs/locale/ru'),
  },
  uk: {
    name: 'Українська',
    textDirection: 'ltr',
    dayjsLocaleCode: 'uk',
    importDayjs: () => import('dayjs/locale/uk'),
  },
  pl: {
    name: 'Polski',
    textDirection: 'ltr',
    dayjsLocaleCode: 'pl',
    importDayjs: () => import('dayjs/locale/pl'),
  },
  nl: {
    name: 'Nederlands',
    textDirection: 'ltr',
    dayjsLocaleCode: 'nl',
    importDayjs: () => import('dayjs/locale/nl'),
  },
  sv: {
    name: 'Svenska',
    textDirection: 'ltr',
    dayjsLocaleCode: 'sv',
    importDayjs: () => import('dayjs/locale/sv'),
  },
  da: {
    name: 'Dansk',
    textDirection: 'ltr',
    dayjsLocaleCode: 'da',
    importDayjs: () => import('dayjs/locale/da'),
  },
  no: {
    name: 'Norsk',
    textDirection: 'ltr',
    dayjsLocaleCode: 'nb',
    importDayjs: () => import('dayjs/locale/nb'),
  },
  fi: {
    name: 'Suomi',
    textDirection: 'ltr',
    dayjsLocaleCode: 'fi',
    importDayjs: () => import('dayjs/locale/fi'),
  },
  cs: {
    name: 'Čeština',
    textDirection: 'ltr',
    dayjsLocaleCode: 'cs',
    importDayjs: () => import('dayjs/locale/cs'),
  },
  hu: {
    name: 'Magyar',
    textDirection: 'ltr',
    dayjsLocaleCode: 'hu',
    importDayjs: () => import('dayjs/locale/hu'),
  },
  ro: {
    name: 'Română',
    textDirection: 'ltr',
    dayjsLocaleCode: 'ro',
    importDayjs: () => import('dayjs/locale/ro'),
  },
  sk: {
    name: 'Slovenčina',
    textDirection: 'ltr',
    dayjsLocaleCode: 'sk',
    importDayjs: () => import('dayjs/locale/sk'),
  },
  el: {
    name: 'Ελληνικά',
    textDirection: 'ltr',
    dayjsLocaleCode: 'el',
    importDayjs: () => import('dayjs/locale/el'),
  },
  tr: {
    name: 'Türkçe',
    textDirection: 'ltr',
    dayjsLocaleCode: 'tr',
    importDayjs: () => import('dayjs/locale/tr'),
  },

  // RTL locales.
  ar: {
    name: 'العربية',
    textDirection: 'rtl',
    dayjsLocaleCode: 'ar',
    importDayjs: () => import('dayjs/locale/ar'),
    numberFormat: {
      thousandsSeparator: '٬',
      decimalSeparator: '٫',
      grouping: [3, 0],
      decimalDigits: 2,
    },
    currencyFormat: {
      symbol: 'ر.س.‏',
      position: 'after',
      code: 'SAR',
      decimalDigits: 2,
    },
    dateTimeFormat: {
      shortDate: 'DD/MM/YYYY',
      longDate: 'D MMMM YYYY',
      shortTime: 'HH:mm',
      longTime: 'HH:mm:ss',
      firstDayOfWeek: 6, // Saturday.
    },
  },
  fa: {
    name: 'فارسی',
    textDirection: 'rtl',
    dayjsLocaleCode: 'fa',
    importDayjs: () => import('dayjs/locale/fa'),
  },
  he: {
    name: 'עברית',
    textDirection: 'rtl',
    dayjsLocaleCode: 'he',
    importDayjs: () => import('dayjs/locale/he'),
  },
  ur: {
    name: 'اردو',
    textDirection: 'rtl',
    dayjsLocaleCode: 'ur',
    importDayjs: () => import('dayjs/locale/ur'),
  },

  // Southeast and South Asian locales.
  th: {
    name: 'ไทย',
    textDirection: 'ltr',
    dayjsLocaleCode: 'th',
    importDayjs: () => import('dayjs/locale/th'),
  },
  vi: {
    name: 'Tiếng Việt',
    textDirection: 'ltr',
    dayjsLocaleCode: 'vi',
    importDayjs: () => import('dayjs/locale/vi'),
  },
  id: {
    name: 'Bahasa Indonesia',
    textDirection: 'ltr',
    dayjsLocaleCode: 'id',
    importDayjs: () => import('dayjs/locale/id'),
  },
  ms: {
    name: 'Bahasa Melayu',
    textDirection: 'ltr',
    dayjsLocaleCode: 'ms',
    importDayjs: () => import('dayjs/locale/ms'),
  },
  hi: {
    name: 'हिन्दी',
    textDirection: 'ltr',
    dayjsLocaleCode: 'hi',
    importDayjs: () => import('dayjs/locale/hi'),
  },
  bn: {
    name: 'বাংলা',
    textDirection: 'ltr',
    dayjsLocaleCode: 'bn',
    importDayjs: () => import('dayjs/locale/bn'),
  },
};
