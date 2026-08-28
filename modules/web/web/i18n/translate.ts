// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Vue-i18n hooks for terminology catalog reactivity and TermReference display.
 * `createTranslate` lives in `@/core/service/i18n` and resolves via `$choysum.i18n.t`.
 */

import { ref } from 'vue';
import type { PostTranslationHandler, VueMessageType } from 'vue-i18n';

import type { TermReference } from '@/core/service/i18n';
import { langToUiKey } from '../stores/i18nStore/lang';

export type TranslateOptions = import('@/core/service/i18n').TranslateOptions;
export type CreateTranslateOptions = import('@/core/service/i18n').CreateTranslateOptions;
export type TextSource = string | TermReference;

export type ComposerLike = {
  t: (key: string, fallback: string, options?: { locale?: string }) => unknown;
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
  fallback = '',
  terminologyLang?: string
): string {
  void composerMessageRevision.value;
  const defaultText = reference?.src || fallback;
  const bridge = composer as ComposerLike | null | undefined;
  if (!reference || !bridge || typeof bridge.t !== 'function') {
    return defaultText;
  }
  try {
    const locale = terminologyLang ? langToUiKey(terminologyLang) : undefined;
    const translated = locale
      ? bridge.t(reference.key, reference.src || fallback, { locale })
      : bridge.t(reference.key, reference.src || fallback);
    return typeof translated === 'string' && translated !== '' ? translated : defaultText;
  } catch {
    return defaultText;
  }
}

export { createTranslate } from '@/core/service/i18n';
