// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend terminology `_t` and reactive `_tr` bound to vue-i18n (§7.2).
 * Looks up module/scope/msgid in the active locale catalog.
 */

import { computed, type ComputedRef } from 'vue';

import { resolveI18nScope, type ResolveI18nScopeOptions } from '../../../core/service/i18n/scope';

export type TranslateOptions = ResolveI18nScopeOptions;

type VueI18nLike = {
  locale?: { value?: string } | string;
  getLocaleMessage?: (locale: string) => unknown;
};

function getI18n(): VueI18nLike | undefined {
  const g = globalThis as { window?: { $i18n?: VueI18nLike }; $i18n?: VueI18nLike };
  if (g.window?.$i18n) {
    return g.window.$i18n;
  }
  return g.$i18n;
}

function interpolate(template: string, args: unknown[]): string {
  if (!args.length) {
    return template;
  }
  let i = 0;
  return template.replace(/%s|%d|%%/g, match => {
    if (match === '%%') {
      return '%';
    }
    if (i >= args.length) {
      return match;
    }
    const v = args[i++];
    return v == null ? '' : String(v);
  });
}

function isTranslateOptions(value: unknown): value is TranslateOptions {
  return (
    !!value &&
    typeof value === 'object' &&
    !Array.isArray(value) &&
    ('scope' in (value as object) || 'path' in (value as object) || 'location' in (value as object) || 'kind' in (value as object))
  );
}

function currentLocale(i18n: VueI18nLike): string {
  const locale = i18n.locale;
  if (typeof locale === 'string') {
    return locale;
  }
  return String(locale?.value || '').trim();
}

function catalogValue(i18n: VueI18nLike, module: string, scope: string, src: string): string | undefined {
  if (!scope || typeof i18n.getLocaleMessage !== 'function') {
    return undefined;
  }
  const locale = currentLocale(i18n);
  if (!locale) {
    return undefined;
  }
  const root = i18n.getLocaleMessage(locale);
  if (!root || typeof root !== 'object') {
    return undefined;
  }
  const byModule = (root as Record<string, unknown>)[module];
  if (!byModule || typeof byModule !== 'object') {
    return undefined;
  }
  const byScope = (byModule as Record<string, unknown>)[scope];
  if (!byScope || typeof byScope !== 'object') {
    return undefined;
  }
  const value = (byScope as Record<string, unknown>)[src];
  return typeof value === 'string' && value ? value : undefined;
}

/**
 * Bind frontend translation helpers to a terminology owner module.
 */
export function createTranslate(module: string, defaults?: TranslateOptions): {
  _t: (src: string, ...args: unknown[]) => string;
  _tr: (src: string, ...args: unknown[]) => ComputedRef<string>;
} {
  const mod = String(module || '').trim() || 'web';
  const defaultScope = resolveI18nScope(defaults);

  const _t = (src: string, ...args: unknown[]): string => {
    let opts: TranslateOptions | undefined;
    let interp: unknown[] = args;
    if (args.length > 0 && isTranslateOptions(args[0])) {
      opts = args[0] as TranslateOptions;
      interp = args.slice(1);
    }

    const scope = resolveI18nScope(opts) || defaultScope;
    const i18n = getI18n();
    if (!i18n) {
      return interpolate(src, interp);
    }

    try {
      const translated = catalogValue(i18n, mod, scope, src);
      return interpolate(translated || src, interp);
    } catch {
      return interpolate(src, interp);
    }
  };

  const _tr = (src: string, ...args: unknown[]): ComputedRef<string> => computed(() => _t(src, ...args));

  return { _t, _tr };
}
