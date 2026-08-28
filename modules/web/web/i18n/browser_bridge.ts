// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTermReference } from '@/core/service/i18n';
import { getGlobalComposer, translateTerm } from './translate';

type ChoysumRoot = {
  $choysum?: {
    i18n?: {
      t: (module: string, lang: string, scope: string, src: string, kind?: string) => string;
    };
  };
};

/**
 * Install browser `$choysum.i18n.t` so `@/core/service/i18n` lookup resolves
 * against the active vue-i18n catalog (same contract as the QuickJS Go bridge).
 */
export function installBrowserI18nBridge(): void {
  const root = globalThis as ChoysumRoot;
  root.$choysum ??= {};
  root.$choysum.i18n = {
    t(module, lang, scope, src, kind) {
      const reference = createTermReference(module, src, { scope, kind });
      return translateTerm(getGlobalComposer(), reference, src, lang);
    },
  };
}

/**
 * Expose the vue-i18n composer on `window.$i18n` and install `$choysum.i18n.t`.
 */
export function exposeBrowserI18nOnWindow(composer: unknown): void {
  if (typeof window === 'undefined') {
    return;
  }
  (window as { $i18n?: unknown }).$i18n = composer;
  installBrowserI18nBridge();
}
