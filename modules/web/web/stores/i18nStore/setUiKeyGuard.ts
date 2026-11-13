// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Whether setUiKey can skip Gateway/dayjs work: store already initialized on that uiKey.
 */
export function shouldNoopSetUiKey(opts: {
  locale: string;
  currentLocaleCode: string | null | undefined;
  isInitialized: boolean;
}): boolean {
  return opts.isInitialized && opts.currentLocaleCode === opts.locale;
}
