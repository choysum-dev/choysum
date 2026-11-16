// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Whether setLang can skip Gateway work: store already initialized on that POSIX lang.
 */
export function shouldNoopSetLang(opts: {
  lang: string;
  currentLang: string | null | undefined;
  isInitialized: boolean;
}): boolean {
  return opts.isInitialized && opts.currentLang === opts.lang;
}
