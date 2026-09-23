// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

export type ChoyTabRegistration = {
  value: string;
  label: string;
  disabled: boolean;
};

export type ChoyTabsContext = {
  tabs: Ref<ChoyTabRegistration[]>;
  register: (tab: ChoyTabRegistration) => void;
  unregister: (value: string) => void;
  update: (value: string, patch: Partial<ChoyTabRegistration>) => void;
};

export const ChoyTabsContextKey: InjectionKey<ChoyTabsContext> = Symbol.for('choysum.choyTabs');
