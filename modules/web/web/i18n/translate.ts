// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend terminology `_t` and reactive `_tr` bound to vue-i18n (§7.2).
 * Looks up module/scope/msgid in the active locale catalog.
 */

import { computed, ref, type ComputedRef } from 'vue';

import { resolveI18nScope } from '../../../core/service/i18n/scope';
import {
  createTextDescriptor,
  type TextDescriptor,
  type TranslateOptions as CoreTranslateOptions,
} from '../../../core/service/i18n/translate';

export type TranslateOptions = CoreTranslateOptions;
export type TextSource = string | TextDescriptor;

export type ComposerLike = {
  t: (key: string, fallback: string) => unknown;
};

const composerMessageRevision = ref(0);

export function notifyComposerMessagesChanged(): void {
  composerMessageRevision.value += 1;
}

/**
 * vue-i18n post-translation hook that makes native template `$t()` calls
 * depend on runtime catalog revisions without changing translated content.
 *
 * `createI18n` accepts one postTranslation handler. If another handler is
 * added later, it must be composed with this hook rather than replacing it.
 */
export function trackComposerMessageRevision(translated: string): string {
  void composerMessageRevision.value;
  return translated;
}

export function getGlobalComposer(): ComposerLike | undefined {
  // Register catalog replacement/merge as a dependency for computed consumers.
  void composerMessageRevision.value;
  const g = globalThis as { window?: { $i18n?: ComposerLike }; $i18n?: ComposerLike };
  if (g.window?.$i18n) {
    return g.window.$i18n;
  }
  return g.$i18n;
}

/**
 * Translate a static text descriptor at the frontend display boundary.
 *
 * vue-i18n treats the second string argument as the default message, so a
 * separate `te` probe is unnecessary and would incorrectly bypass locale
 * fallback. Calling `t` also keeps the active locale reactive for Composer
 * consumers; the revision dependency covers runtime catalog merges.
 */
export function translateTerm(
  composer: ComposerLike | undefined,
  descriptor?: TextDescriptor,
  fallback = ''
): string {
  void composerMessageRevision.value;
  const defaultText = descriptor?.src || fallback;
  if (!descriptor || !composer || typeof composer.t !== 'function') {
    return defaultText;
  }
  try {
    const translated = composer.t(descriptor.key, descriptor.src || fallback);
    return typeof translated === 'string' && translated !== '' ? translated : defaultText;
  } catch {
    return defaultText;
  }
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

function parseTranslateArgs(args: unknown[]): {
  opts: TranslateOptions | undefined;
  interpolation: unknown[];
} {
  if (args.length > 0 && isTranslateOptions(args[0])) {
    return {
      opts: args[0],
      interpolation: args.slice(1),
    };
  }
  return { opts: undefined, interpolation: args };
}

function translateDescriptor(
  descriptor: TextDescriptor,
  interpolation: unknown[]
): string {
  return interpolate(translateTerm(getGlobalComposer(), descriptor, descriptor.src), interpolation);
}

/**
 * Bind frontend translation helpers to a terminology owner module.
 */
export function createTranslate(module: string, defaults?: TranslateOptions): {
  _t: (src: string, ...args: unknown[]) => string;
  _tr: (src: string, ...args: unknown[]) => ComputedRef<string>;
  _td: (src: string, opts?: TranslateOptions) => TextDescriptor;
} {
  const mod = String(module || '').trim() || 'web';
  const defaultScope = resolveI18nScope(defaults);

  const resolveDescriptor = (src: string, opts?: TranslateOptions): TextDescriptor => {
    const scope = resolveI18nScope(opts) || defaultScope;
    return createTextDescriptor(mod, src, { ...opts, scope });
  };

  const _t = (src: string, ...args: unknown[]): string => {
    const { opts, interpolation } = parseTranslateArgs(args);
    return translateDescriptor(resolveDescriptor(src, opts), interpolation);
  };

  const _tr = (src: string, ...args: unknown[]): ComputedRef<string> => {
    const { opts, interpolation } = parseTranslateArgs(args);
    const descriptor = resolveDescriptor(src, opts);
    return computed(() => translateDescriptor(descriptor, interpolation));
  };
  const _td = (src: string, opts?: TranslateOptions): TextDescriptor =>
    resolveDescriptor(src, opts);

  return { _t, _tr, _td };
}
