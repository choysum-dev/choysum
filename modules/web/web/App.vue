<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Reka portals (DropdownMenu / Select / Popover / …) default dir=ltr unless a
    ConfigProvider supplies direction. Keep that in sync with html[dir] so icon+label
    rows flip under RTL without per-menu hacks.
  -->
  <ConfigProvider :dir="rekaDir">
    <div class="choy-app h-full w-full overflow-hidden bg-background text-foreground">
      <router-view />
      <ChoyToaster />
      <ChoyConfirmHost />
      <ChoyDialogHost />
      <ChoyLoadingSlowHint />
    </div>
  </ConfigProvider>
</template>

<script setup lang="ts">
/**
 * Root host for the web SPA. Theme class / density live on documentElement
 * (see applyChoyThemePreference from setupApp).
 */
import { computed, onMounted, onUnmounted } from 'vue';
import { ConfigProvider } from 'reka-ui';
import ChoyToaster from './components/layout/ChoyToaster.vue';
import ChoyConfirmHost from './components/layout/ChoyConfirmHost.vue';
import ChoyDialogHost from './components/layout/ChoyDialogHost.vue';
import ChoyLoadingSlowHint from './components/layout/ChoyLoadingSlowHint.vue';
import { bindChromeLoadingBar, unbindChromeLoadingBar } from './composables/chromeLoadingFeedback';
import { useI18nStore } from './stores/i18nStore';

const i18nStore = useI18nStore();
const rekaDir = computed<'ltr' | 'rtl'>(() =>
  i18nStore.activeLanguage.textDirection === 'rtl' ? 'rtl' : 'ltr',
);

onMounted(() => {
  bindChromeLoadingBar();
});
onUnmounted(() => {
  unbindChromeLoadingBar();
});
</script>
