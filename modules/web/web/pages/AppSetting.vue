<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="appSettingStore" :store="appSettingStore">
    <AppSettingFormView
      :key="application"
      :application="application"
      :store="appSettingStore"
      :initial-values="initialValues"
      :create-action="createPath"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import AppSettingFormView from '../views/AppSettingFormView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useAppSettingDetailPage } from './app_setting_detail_page';

defineOptions({ name: 'AppSettingPage' });

const route = useRoute();
const { application, appSettingStore } = useAppSettingDetailPage({
  route,
  scopeManager: useScopeManager().menuScopeManager,
});

const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/app-settings/${app}/new` : '';
});

// Management Form uses standard CRUD (align FieldDefault); program path uses Get/Set.
const initialValues: Record<string, unknown> = {
  Key: null,
  Value: null,
};
</script>
