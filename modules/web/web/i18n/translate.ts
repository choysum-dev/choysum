// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend terminology `_t` / `_lt` bound to vue-i18n (§7.2).
 * `_t` looks up module/scope/msgid in the active locale catalog.
 * `_lt` pins a serializable TermReference (no lookup).
 */

import { ref } from 'vue';
import type { PostTranslationHandler, VueMessageType } from 'vue-i18n';

import { resolveI18nScope } from '../../../core/service/i18n/scope';
import {
  createTermReference,
  type CreateTranslateOptions as CoreCreateTranslateOptions,
  type CreateTranslateResult,
  type LazyTranslateFn,
  type TermReference,
  type TranslateFn,
  type TranslateOptions as CoreTranslateOptions,
} from '../../../core/service/i18n/translate';

export type TranslateOptions = CoreTranslateOptions;
export type CreateTranslateOptions = CoreCreateTranslateOptions;
export type TextSource = string | TermReference;

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
export const trackComposerMessageRevision: PostTranslationHandler<VueMessageType> = (
  translated
) => {
  void composerMessageRevision.value;
  return translated;
};

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
 * Translate a static term reference at the frontend display boundary.
 *
 * vue-i18n treats the second string argument as the default message, so a
 * separate `te` probe is unnecessary and would incorrectly bypass locale
 * fallback. Calling `t` also keeps the active locale reactive for Composer
 * consumers; the revision dependency covers runtime catalog merges.
 *
 * `composer` is intentionally `unknown` so callers can pass a real vue-i18n
 * Composer without triggering excessively deep generic instantiation.
 */
export function translateTerm(
  composer: unknown,
  reference?: TermReference,
  fallback = ''
): string {
  void composerMessageRevision.value;
  const defaultText = reference?.src || fallback;
  const bridge = composer as ComposerLike | null | undefined;
  if (!reference || !bridge || typeof bridge.t !== 'function') {
    return defaultText;
  }
  try {
    const translated = bridge.t(reference.key, reference.src || fallback);
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

function translateReference(
  reference: TermReference,
  interpolation: unknown[]
): string {
  return interpolate(translateTerm(getGlobalComposer(), reference, reference.src), interpolation);
}

/**
 * Bind frontend translation helpers `_t` (text) and `_lt` (TermReference)
 * to a terminology owner module. Shape matches core `createTranslate`.
 */
export function createTranslate(
  module: string,
  defaults?: CreateTranslateOptions
): CreateTranslateResult {
  const mod = String(module || '').trim() || 'web';
  const defaultScope = resolveI18nScope(defaults);
  const defaultKind = defaults?.kind;
  const defaultReferences = new Map<string, TermReference>();

  const resolveReference = (src: string, opts?: TranslateOptions): TermReference => {
    if (!opts && defaultScope) {
      const cached = defaultReferences.get(src);
      if (cached) {
        return cached;
      }
    }
    const scope = resolveI18nScope(opts) || defaultScope;
    const reference = createTermReference(mod, src, {
      ...opts,
      scope,
      kind: opts?.kind ?? defaultKind,
    });
    if (!opts && defaultScope) {
      defaultReferences.set(src, reference);
    }
    return reference;
  };

  const _t = (src: string, ...args: unknown[]): string => {
    const { opts, interpolation } = parseTranslateArgs(args);
    return translateReference(resolveReference(src, opts), interpolation);
  };

  const _lt = (src: string, opts?: TranslateOptions, ...rest: unknown[]): TermReference => {
    if (rest.length > 0) {
      throw new Error('_lt does not accept interpolation arguments');
    }
    return resolveReference(src, opts);
  };

  return {
    _t: _t as TranslateFn,
    _lt: _lt as LazyTranslateFn,
  };
}
