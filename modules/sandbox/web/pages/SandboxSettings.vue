<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="pageTitle">
    <ChoySettingsView
      :local-store="formStore"
      :loading="loading"
      :dirty="dirty"
      :active-view="activeView"
      :view-switcher-options="viewSwitcherOptions"
      @save="onSave"
      @discard="onDiscard"
      @view-select="onViewSelect"
    >
      <div data-anchor="sandbox.settings.sections" class="flex max-w-md flex-col gap-8">
        <ChoySection :title="_t('Demo')" data-anchor="sandbox.settings.demo">
          <ChoyGrid :cols="12" class="gap-y-4">
            <ChoyCol :span="12">
              <ChoyBooleanField
                v-if="formStore"
                :store="formStore"
                prop="DemoFlag"
                :label="_t('Demo flag')"
                :help="_t('Turn sandbox demo mode on or off.')"
                widget="switch"
              />
            </ChoyCol>
            <ChoyCol :span="12">
              <ChoyVarcharField
                v-if="formStore"
                :store="formStore"
                prop="DemoLabel"
                :label="_t('Demo label')"
                :help="_t('Short label shown with sandbox demo mode.')"
              />
            </ChoyCol>
          </ChoyGrid>
        </ChoySection>
      </div>
    </ChoySettingsView>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createLocalFormStore, type LocalFormStore } from '@/web/web/stores/localFormStore';
import { useChoyViewRouteSwitch } from '@/web/web/composables/useChoyViewRouteSwitch';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyGrid,
  ChoyPage,
  ChoySection,
  ChoySettingsView,
  ChoyVarcharField,
  type ChoyViewSwitcherOption,
} from '@/web';

defineOptions({ name: 'SandboxSettingsPage' });

/** Sandbox dogfood keys (table-local; not auth keys). */
const SANDBOX_DEMO_FLAG_KEY = 'demo_flag';
const SANDBOX_DEMO_LABEL_KEY = 'demo_label';

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxSettings' });
const pageTitle = computed(() => _t('Settings: sandbox'));

const formStore = shallowRef<LocalFormStore | null>(null);
const baselineFlag = ref('0');
const baselineLabel = ref('');
const loading = ref(false);

const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'settings', label: _t('Settings View') },
  { value: 'list', label: _t('Parameters View') },
]);

const dirty = computed(() => {
  if (!formStore.value) return false;
  const flag = formStore.value.getField('DemoFlag') === true ? '1' : '0';
  const label = String(formStore.value.getField('DemoLabel') ?? '');
  return flag !== baselineFlag.value || label !== baselineLabel.value;
});

type AppSettingKvStore = {
  Get: (key: string, defaultValue?: string | null) => Promise<string | null>;
  Set: (key: string, value: string | null) => Promise<unknown>;
};

function appSettingStore(): AppSettingKvStore {
  return createStoreByModel('sandbox.AppSetting', {
    storeId: 'AppSetting_sandbox_settings',
    scopeManager: useScopeManager().menuScopeManager,
  }) as unknown as AppSettingKvStore;
}

function rebuildForm(flag: string, label: string) {
  formStore.value = createLocalFormStore({
    storeId: `local-form:sandbox-settings:${Date.now()}`,
    fields: [
      { name: 'DemoFlag', label: _t('Demo flag'), type: 'boolean' },
      { name: 'DemoLabel', label: _t('Demo label'), type: 'varchar' },
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
    const store = appSettingStore();
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
    const store = appSettingStore();
    const flag = formStore.value.getField('DemoFlag') === true ? '1' : '0';
    const label = String(formStore.value.getField('DemoLabel') ?? '');
    // Only persist keys that actually changed (one edit must not Set siblings).
    if (flag !== baselineFlag.value) {
      await store.Set(SANDBOX_DEMO_FLAG_KEY, flag);
      baselineFlag.value = flag;
    }
    if (label !== baselineLabel.value) {
      await store.Set(SANDBOX_DEMO_LABEL_KEY, label);
      baselineLabel.value = label;
    }
    ChoyMessage.success(_t('Saved successfully'));
  } catch (err) {
    ChoyMessage.error(_t('Save failed'));
    throw err;
  } finally {
    loading.value = false;
  }
}

function onDiscard() {
  rebuildForm(baselineFlag.value, baselineLabel.value);
}

onMounted(() => {
  void load();
});
</script>
