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
    t(module, _lang, scope, src, kind) {
      const reference = createTermReference(module, src, { scope, kind });
      return translateTerm(getGlobalComposer(), reference, src);
    },
  };
}
