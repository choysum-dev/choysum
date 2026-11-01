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
import { computed, onMounted } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import { useChoyViewRouteSwitch } from '@/web/web/composables/useChoyViewRouteSwitch';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyGrid,
  ChoyPage,
  ChoySection,
  ChoySettingsView,
  type ChoyViewSwitcherOption,
} from '@/web';
import { useAuthSettingsPage } from './auth_settings_page';

defineOptions({ name: 'AuthSettingsPage' });

const { _t } = createTranslate('auth', { scope: 'web/pages/AuthSettings' });
const pageTitle = computed(() => _t('Settings: auth'));

const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'settings', label: _t('Settings View') },
  { value: 'list', label: _t('Parameters View') },
]);

const { formStore, loading, dirty, load, onSave, onDiscard } = useAuthSettingsPage({
  translate: _t,
});

onMounted(() => {
  void load();
});
</script>
