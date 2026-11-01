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
      <div data-anchor="auth.settings.sections" class="flex max-w-md flex-col gap-8">
        <ChoySection :title="_t('Browser')" data-anchor="auth.settings.browser">
          <ChoyGrid :cols="12">
            <ChoyCol :span="12">
              <ChoyBooleanField
                v-if="formStore"
                :store="formStore"
                prop="PersistBrowserTimezone"
                :label="_t('Browser timezone')"
                :help="_t('On sign-in, fill an empty user timezone from the browser.')"
                widget="switch"
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
import { PERSIST_BROWSER_TIMEZONE_KEY } from '@/auth/service/models/user/persist_browser_timezone_key';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyGrid,
  ChoyPage,
  ChoySection,
  ChoySettingsView,
  type ChoyViewSwitcherOption,
} from '@/web';

defineOptions({ name: 'AuthSettingsPage' });

const { _t } = createTranslate('auth', { scope: 'web/pages/AuthSettings' });
const pageTitle = computed(() => _t('Settings: auth'));

const formStore = shallowRef<LocalFormStore | null>(null);
const baseline = ref('1');
const loading = ref(false);

const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'settings', label: _t('Settings View') },
  { value: 'list', label: _t('Parameters View') },
]);

const dirty = computed(() => {
  if (!formStore.value) return false;
  const current = formStore.value.getField('PersistBrowserTimezone') === true ? '1' : '0';
  return current !== baseline.value;
});

type AppSettingKvStore = {
  Get: (key: string, defaultValue?: string | null) => Promise<string | null>;
  Set: (key: string, value: string | null) => Promise<unknown>;
};

function appSettingStore(): AppSettingKvStore {
  return createStoreByModel('auth.AppSetting', {
    storeId: 'AppSetting_auth_settings',
    scopeManager: useScopeManager().menuScopeManager,
  }) as unknown as AppSettingKvStore;
}

function rebuildForm(flag: string) {
  const on = flag === '1';
  formStore.value = createLocalFormStore({
    storeId: `local-form:auth-settings:${Date.now()}`,
    fields: [
      {
        name: 'PersistBrowserTimezone',
        label: _t('Browser timezone'),
        type: 'boolean',
      },
    ],
    initialValues: { PersistBrowserTimezone: on },
  });
  baseline.value = on ? '1' : '0';
}

async function load() {
  loading.value = true;
  try {
    const flag = (await appSettingStore().Get(PERSIST_BROWSER_TIMEZONE_KEY, '1')) ?? '1';
    rebuildForm(flag === '0' ? '0' : '1');
  } finally {
    loading.value = false;
  }
}

async function onSave() {
  if (!formStore.value || !dirty.value) return;
  loading.value = true;
  try {
    const on = formStore.value.getField('PersistBrowserTimezone') === true;
    const value = on ? '1' : '0';
    await appSettingStore().Set(PERSIST_BROWSER_TIMEZONE_KEY, value);
    baseline.value = value;
    ChoyMessage.success(_t('Saved successfully'));
  } catch (err) {
    ChoyMessage.error(_t('Save failed'));
    throw err;
  } finally {
    loading.value = false;
  }
}

function onDiscard() {
  rebuildForm(baseline.value);
}

onMounted(() => {
  void load();
});
</script>
