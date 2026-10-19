// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey } from 'vue';

/**
 * Optional nav sources for Search default-favorite naming and scopeKey.
 * When provided, Search skips breadcrumb / menu / route setup hooks.
 */
export type SearchNavContext = {
  breadcrumbStore?: { breadcrumbStack: Array<{ title?: string; titleText?: any }> } | null;
  menuStore?: { activeMenu: { title?: string; titleText?: any } | null } | null;
  route?: { path?: string; meta?: Record<string, unknown> } | null;
};

export const SearchNavContextKey: InjectionKey<SearchNavContext> = Symbol('SearchNavContext');
