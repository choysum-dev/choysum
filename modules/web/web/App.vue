<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-config-provider :locale="i18nStore.currentLocale.elementLocale" :size="componentSize">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { useI18nStore } from './stores';
import { ref } from 'vue';
import { ElConfigProvider } from 'element-plus';
import type { ComponentSize } from 'element-plus';

const i18nStore = useI18nStore();
const componentSize = ref<ComponentSize>('default');
</script>

<style lang="scss" scoped>
@use './styles/index.scss' as *;

.el-config-provider {
  min-height: 100vh;
  width: 100%;
  background-color: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
  font-family: var(--el-font-family);
  font-size: var(--el-font-size-base);
}
</style>
