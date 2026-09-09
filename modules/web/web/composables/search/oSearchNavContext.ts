// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey } from 'vue';

/**
 * Optional nav sources for OSearch default-favorite naming and scopeKey.
 * When provided, OSearch skips breadcrumb / menu / route setup hooks.
 */
export type OSearchNavContext = {
  breadcrumbStore?: { breadcrumbStack: Array<{ title?: string; titleText?: any }> } | null;
  menuStore?: { activeMenu: { title?: string; titleText?: any } | null } | null;
  route?: { path?: string; meta?: Record<string, unknown> } | null;
};

export const OSearchNavContextKey: InjectionKey<OSearchNavContext> = Symbol('OSearchNavContext');
