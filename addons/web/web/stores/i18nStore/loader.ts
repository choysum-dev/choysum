// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { SUPPORTED_LOCALES } from './locales';
import { SupportedLocale } from './types';

/**
 * Dynamically import the Element Plus locale bundle.
 */
export async function loadElementLocale(locale: string): Promise<any> {
  try {
    const localeConfig = SUPPORTED_LOCALES[locale as SupportedLocale];

    if (localeConfig?.importElement) {
      // Use the import hook defined by the locale config.
      return (await localeConfig.importElement()).default;
    }

    // Fall back to English.
    return (await SUPPORTED_LOCALES.en.importElement!()).default;
  } catch (e) {
    console.warn(`Failed to load Element Plus locale for ${locale}, falling back to English`);
    return (await import('element-plus/es/locale/lang/en')).default;
  }
}

/**
 * Dynamically import the DayJS locale bundle.
 */
export async function loadDayjsLocale(locale: string): Promise<void> {
  try {
    const localeConfig = SUPPORTED_LOCALES[locale as SupportedLocale];

    if (localeConfig?.importDayjs) {
      await localeConfig.importDayjs();
    }
  } catch (e) {
    console.warn(`Failed to load dayjs locale for ${locale}`);
  }
}

/**
 * Load Vue I18n messages for the requested locale.
 */
export async function loadVueI18nMessages(locale: string): Promise<any> {
  try {
    const localeConfig = SUPPORTED_LOCALES[locale as SupportedLocale];

    if (localeConfig?.importVueI18n) {
      return (await localeConfig.importVueI18n()).default;
    }

    // Fall back to English.
    return (await SUPPORTED_LOCALES.en.importVueI18n!()).default;
  } catch (e) {
    console.warn(`Failed to load Vue I18n messages for ${locale}, falling back to English`);
    return (await import('../../i18n/source')).default;
  }
}
