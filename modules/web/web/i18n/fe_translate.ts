// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend terminology `_t` bound to vue-i18n (§7.2).
 * Looks up `namespace = ${module}.${scope}` then falls back to msgid.
 */

import { resolveI18nScope, type ResolveI18nScopeOptions } from '../../../core/service/i18n/scope';

type FeTranslateOptions = ResolveI18nScopeOptions;

type VueI18nLike = {
  t: (key: string, ...args: any[]) => string;
  te?: (key: string, ...args: any[]) => boolean;
  locale?: { value?: string } | string;
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

function isTranslateOptions(value: unknown): value is FeTranslateOptions {
  return (
    !!value &&
    typeof value === 'object' &&
    !Array.isArray(value) &&
    ('scope' in (value as object) || 'path' in (value as object) || 'location' in (value as object) || 'kind' in (value as object))
  );
}

/**
 * Bind FE `_t` to a module name (term owner). Uses Gateway-merged vue-i18n messages.
 */
export function createFeTranslate(module: string): {
  _t: (src: string, ...args: unknown[]) => string;
} {
  const mod = String(module || '').trim() || 'web';

  const _t = (src: string, ...args: unknown[]): string => {
    let opts: FeTranslateOptions | undefined;
    let interp: unknown[] = args;
    if (args.length > 0 && isTranslateOptions(args[0])) {
      opts = args[0] as FeTranslateOptions;
      interp = args.slice(1);
    }

    const scope = resolveI18nScope(opts);
    const i18n = getI18n();
    if (!i18n || typeof i18n.t !== 'function') {
      return interpolate(src, interp);
    }

    const namespace = scope ? `${mod}.${scope}` : mod;
    try {
      if (typeof i18n.te === 'function') {
        const exists = i18n.te(src, namespace);
        if (!exists) {
          return interpolate(src, interp);
        }
      }
      const translated =
        interp.length > 0 ? i18n.t(src, interp as any, { namespace }) : i18n.t(src, { namespace });
      const text = translated == null ? '' : String(translated);
      if (!text || text === src || text === `${namespace}.${src}`) {
        return interpolate(src, interp);
      }
      return text;
    } catch {
      return interpolate(src, interp);
    }
  };

  return { _t };
}
