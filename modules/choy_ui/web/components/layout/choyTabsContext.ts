// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type InjectionKey, type Ref } from 'vue';

export type ChoyTabRegistration = {
  value: string;
  label: string;
  disabled: boolean;
};

export type ChoyTabsContext = {
  tabs: Ref<ChoyTabRegistration[]>;
  register: (tab: ChoyTabRegistration) => boolean;
  unregister: (value: string) => void;
  update: (value: string, patch: Partial<ChoyTabRegistration>) => boolean;
};

export const ChoyTabsContextKey: InjectionKey<ChoyTabsContext> = Symbol.for('choysum.choyTabs');

/**
 * Mutable tab registry shared by ChoyTabs and ChoyTab children.
 * Kept here (not in the SFC) so register/update/unregister are unit-testable.
 */
export function createChoyTabsContext(
  tabs: Ref<ChoyTabRegistration[]> = ref([]),
): ChoyTabsContext {
  return {
    tabs,
    register(tab) {
      if (tabs.value.some((item) => item.value === tab.value)) {
        return false;
      }
      tabs.value = [...tabs.value, tab];
      return true;
    },
    unregister(value) {
      tabs.value = tabs.value.filter((item) => item.value !== value);
    },
    update(value, patch) {
      if (!tabs.value.some((item) => item.value === value)) {
        return false;
      }
      const nextValue = patch.value ?? value;
      if (nextValue !== value && tabs.value.some((item) => item.value === nextValue)) {
        return false;
      }
      tabs.value = tabs.value.map((item) =>
        item.value === value ? { ...item, ...patch, value: nextValue } : item,
      );
      return true;
    },
  };
}
