// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

/**
 * Vue-managed host for the narrow-bar search row under ChoyViewActionBar.
 * ChoySearchFit teleports its field here instead of inserting DOM imperatively.
 */
export const ChoySearchFitRowKey: InjectionKey<Ref<HTMLElement | null>> = Symbol.for(
  'choysum.choySearchFitRow',
);
