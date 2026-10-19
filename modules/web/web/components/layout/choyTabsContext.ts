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
  /** Host v-model; tab panels render their slot only when this matches. */
  activeValue: Ref<string | undefined>;
  register: (tab: ChoyTabRegistration) => boolean;
  unregister: (value: string) => void;
  update: (value: string, patch: Partial<ChoyTabRegistration>) => boolean;
};

export const ChoyTabsContextKey: InjectionKey<ChoyTabsContext> = Symbol.for('choysum.choyTabs');

function isUsableTabValue(value: string | undefined): boolean {
  return !!String(value ?? '').trim();
}

/**
 * Mutable tab registry shared by ChoyTabs and ChoyTab children.
 * Kept here (not in the SFC) so register/update/unregister are unit-testable.
 */
export function createChoyTabsContext(
  tabs: Ref<ChoyTabRegistration[]> = ref([]),
  activeValue: Ref<string | undefined> = ref(undefined),
): ChoyTabsContext {
  return {
    tabs,
    activeValue,
    register(tab) {
      if (!isUsableTabValue(tab.value)) {
        return false;
      }
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
      if (!isUsableTabValue(nextValue)) {
        return false;
      }
      const valueTaken =
        nextValue !== value && tabs.value.some((item) => item.value === nextValue);
      // A colliding rename keeps the owned value but still applies the rest of
      // the patch so callers do not silently lose label / disabled updates.
      const effectiveValue = valueTaken ? value : nextValue;
      const current = tabs.value.find((item) => item.value === value);
      const nextLabel = patch.label ?? current?.label ?? '';
      const nextDisabled = patch.disabled ?? current?.disabled ?? false;
      if (
        current &&
        current.value === effectiveValue &&
        current.label === nextLabel &&
        current.disabled === nextDisabled
      ) {
        return !valueTaken;
      }
      tabs.value = tabs.value.map((item) =>
        item.value === value
          ? {
              value: effectiveValue,
              label: nextLabel,
              disabled: nextDisabled,
            }
          : item,
      );
      return !valueTaken;
    },
  };
}
