<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-app min-h-screen w-full bg-background text-foreground">
    <!-- Resolves via Element Plus global registration from setupApp (dual-stack). -->
    <el-config-provider
      data-test="ep-config"
      :locale="currentLocale?.elementLocale"
      :size="componentSize"
    >
      <router-view />
    </el-config-provider>
  </div>
</template>

<script setup lang="ts">
/**
 * Root host for the web SPA. Theme class / density live on documentElement
 * (see applyChoyThemePreference from setupApp). Element Plus ConfigProvider
 * keeps O* locale/size reactive during dual-stack cutover.
 */
import { ref } from 'vue';
import { storeToRefs } from 'pinia';
import type { ComponentSize } from 'element-plus';
import { useI18nStore } from './stores';

const i18nStore = useI18nStore();
const { currentLocale } = storeToRefs(i18nStore);
const componentSize = ref<ComponentSize>('default');
</script>
