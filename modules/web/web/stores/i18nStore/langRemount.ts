// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Post-language-change strategy: always location.reload in product paths.
 */

export type AfterLangChangeOptions = {
  /** Injected location.reload for tests. */
  reload?: () => void;
};

/**
 * Run after a successful setLang that should remount the SPA.
 * Product path is location.reload only.
 */
export async function afterLangChange(options?: AfterLangChangeOptions): Promise<void> {
  const reload =
    options?.reload ||
    (() => {
      const loc = (globalThis as { location?: Location }).location;
      if (loc?.reload) {
        loc.reload();
        return;
      }
      console.warn('afterLangChange: location.reload unavailable; language switch not applied');
    });
  reload();
}
