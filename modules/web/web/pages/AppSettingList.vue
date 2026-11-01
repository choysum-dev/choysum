<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage v-if="appSettingStore" :title="pageTitle" :store="appSettingStore">
    <AppSettingListView
      :key="application"
      :application="application"
      :store="appSettingStore"
    />
  </ChoyPage>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import AppSettingListView from '../views/AppSettingListView.vue';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyPage } from '@/web';
import { useAppSettingListPage } from './app_setting_list_page';

defineOptions({ name: 'AppSettingListPage' });

const { application, pageTitle, appSettingStore } = useAppSettingListPage({
  route: useRoute(),
  scopeManager: useScopeManager().menuScopeManager,
});
</script>
