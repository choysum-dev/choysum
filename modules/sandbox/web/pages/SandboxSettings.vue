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
  ChoyVarcharField,
  type ChoyViewSwitcherOption,
} from '@/web';
import { useSandboxSettingsPage } from './sandbox_settings_page';

defineOptions({ name: 'SandboxSettingsPage' });

const { _t } = createTranslate('sandbox', { scope: 'web/pages/SandboxSettings' });
const pageTitle = computed(() => _t('Settings: sandbox'));

const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'settings', label: _t('Settings View') },
  { value: 'list', label: _t('Parameters View') },
]);

const { formStore, loading, dirty, load, onSave, onDiscard } = useSandboxSettingsPage({
  translate: _t,
});

onMounted(() => {
  void load();
});
</script>
