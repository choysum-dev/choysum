// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getContextLang } from '../runtime/context/scope';
import { resolveI18nScope, withI18nScope, type ResolveI18nScopeOptions } from './scope';
import { resolveRequestLang } from './request_lang';

type TranslateOptions = ResolveI18nScopeOptions & { kind?: string };

type ChoysumI18n = {
  t: (module: string, lang: string, scope: string, src: string, kind?: string) => string;
};

function getBridge(): ChoysumI18n | undefined {
  const root = globalThis as { $choysum?: { i18n?: ChoysumI18n } };
  return root.$choysum?.i18n;
}

function isTranslateOptions(value: unknown): value is TranslateOptions {
  return !!value && typeof value === 'object' && !Array.isArray(value) && ('scope' in (value as object) || 'kind' in (value as object) || 'path' in (value as object) || 'location' in (value as object));
}

function interpolate(template: string, args: unknown[]): string {
  if (!args.length) {
    return template;
  }
  let i = 0;
  return template.replace(/%s|%d|%%/g, (match) => {
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

function lookup(module: string, src: string, opts?: TranslateOptions): string {
  const fromServiceCtx = getContextLang()?.trim();
  const lang = fromServiceCtx || resolveRequestLang(undefined, { final: 'en_US' });
  const scope = resolveI18nScope(opts);
  const kind = opts?.kind || 'literal';
  const bridge = getBridge();
  if (!bridge || typeof bridge.t !== 'function') {
    return '';
  }
  try {
    return bridge.t(module, lang, scope, src, kind) || '';
  } catch {
    return '';
  }
}

export type TranslateFn = {
  (src: string, ...args: unknown[]): string;
};

export type LazyTranslate = {
  toString(): string;
  valueOf(): string;
  [Symbol.toPrimitive](): string;
};

/**
 * Bind `_t` / `_lt` to a module name (term owner module).
 */
export function createTranslate(module: string): {
  _t: TranslateFn;
  _lt: (src: string, opts?: TranslateOptions) => LazyTranslate;
} {
  const mod = module;

  const _t: TranslateFn = (src: string, ...args: unknown[]): string => {
    let opts: TranslateOptions | undefined;
    let interp: unknown[] = args;
    if (args.length > 0 && isTranslateOptions(args[0])) {
      opts = args[0] as TranslateOptions;
      interp = args.slice(1);
    }
    const hit = lookup(mod, src, opts);
    const base = hit || src;
    return interpolate(base, interp);
  };

  const _lt = (src: string, opts?: TranslateOptions): LazyTranslate => {
    const resolve = () => {
      const hit = lookup(mod, src, opts);
      return hit || src;
    };
    return {
      toString: resolve,
      valueOf: resolve,
      [Symbol.toPrimitive]: resolve,
    };
  };

  return { _t, _lt };
}

export { withI18nScope, resolveI18nScope, formatScope } from './scope';
export { resolveRequestLang } from './request_lang';
