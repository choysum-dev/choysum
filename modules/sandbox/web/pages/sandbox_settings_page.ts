// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, shallowRef, type ComputedRef, type Ref, type ShallowRef } from 'vue';
import { createLocalFormStore, type LocalFormStore } from '@/web/web/stores/localFormStore';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';

/** Sandbox dogfood keys (table-local; not auth keys). */
export const SANDBOX_DEMO_FLAG_KEY = 'demo_flag';
export const SANDBOX_DEMO_LABEL_KEY = 'demo_label';

export type SandboxAppSettingKv = {
  Get: (key: string, defaultValue?: string | null) => Promise<string | null>;
  Set: (key: string, value: string | null) => Promise<unknown>;
};

export type SandboxSettingsPageDeps = {
  appSettingStore: () => SandboxAppSettingKv;
  translate: (msgid: string, ...args: unknown[]) => string;
};

export type SandboxSettingsPageApi = {
  formStore: ShallowRef<LocalFormStore | null>;
  loading: Ref<boolean>;
  dirty: ComputedRef<boolean>;
  load: () => Promise<void>;
  onSave: () => Promise<void>;
  onDiscard: () => void;
};

/** Curated sandbox Settings: demo flag/label via AppSetting Get/Set. */
export function useSandboxSettingsPage(deps: SandboxSettingsPageDeps): SandboxSettingsPageApi {
  const formStore = shallowRef<LocalFormStore | null>(null);
  const baselineFlag = ref('0');
  const baselineLabel = ref('');
  const loading = ref(false);

  const dirty = computed(() => {
    if (!formStore.value) return false;
    const flag = formStore.value.getField('DemoFlag') === true ? '1' : '0';
    const label = String(formStore.value.getField('DemoLabel') ?? '');
    return flag !== baselineFlag.value || label !== baselineLabel.value;
  });

  function rebuildForm(flag: string, label: string) {
    formStore.value = createLocalFormStore({
      storeId: `local-form:sandbox-settings:${Date.now()}`,
      fields: [
        { name: 'DemoFlag', label: deps.translate('Demo flag'), type: 'boolean' },
        { name: 'DemoLabel', label: deps.translate('Demo label'), type: 'varchar' },
      ],
      initialValues: {
        DemoFlag: flag === '1',
        DemoLabel: label,
      },
    });
    baselineFlag.value = flag === '1' ? '1' : '0';
    baselineLabel.value = label;
  }

  async function load() {
    loading.value = true;
    try {
      const store = deps.appSettingStore();
      const flag = (await store.Get(SANDBOX_DEMO_FLAG_KEY, '0')) ?? '0';
      const label = (await store.Get(SANDBOX_DEMO_LABEL_KEY, '')) ?? '';
      rebuildForm(flag === '1' ? '1' : '0', label);
    } finally {
      loading.value = false;
    }
  }

  async function onSave() {
    if (!formStore.value || !dirty.value) return;
    loading.value = true;
    try {
      const store = deps.appSettingStore();
      const flag = formStore.value.getField('DemoFlag') === true ? '1' : '0';
      const label = String(formStore.value.getField('DemoLabel') ?? '');
      // Only persist keys that actually changed (one edit must not Set siblings).
      // Advance each baseline after its own Set so a later failure still retries
      // only the remaining dirty key(s).
      if (flag !== baselineFlag.value) {
        await store.Set(SANDBOX_DEMO_FLAG_KEY, flag);
        baselineFlag.value = flag;
      }
      if (label !== baselineLabel.value) {
        await store.Set(SANDBOX_DEMO_LABEL_KEY, label);
        baselineLabel.value = label;
      }
      ChoyMessage.success(deps.translate('Saved successfully'));
    } catch {
      ChoyMessage.error(deps.translate('Save failed'));
    } finally {
      loading.value = false;
    }
  }

  function onDiscard() {
    rebuildForm(baselineFlag.value, baselineLabel.value);
  }

  return { formStore, loading, dirty, load, onSave, onDiscard };
}
