<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-app h-full w-full overflow-hidden bg-background text-foreground">
    <router-view />
    <Toaster rich-colors close-button />
    <ChoyConfirmHost />
    <ChoyDialogHost />
    <ChoyLoadingSlowHint />
  </div>
</template>

<script setup lang="ts">
/**
 * Root host for the web SPA. Theme class / density live on documentElement
 * (see applyChoyThemePreference from setupApp).
 */
import { onMounted, onUnmounted } from 'vue';
import Toaster from './components/vendor/ui/sonner/Sonner.vue';
import ChoyConfirmHost from './components/layout/ChoyConfirmHost.vue';
import ChoyDialogHost from './components/layout/ChoyDialogHost.vue';
import ChoyLoadingSlowHint from './components/layout/ChoyLoadingSlowHint.vue';
import { bindChromeLoadingBar, unbindChromeLoadingBar } from './composables/chromeLoadingFeedback';

onMounted(() => {
  bindChromeLoadingBar();
});
onUnmounted(() => {
  unbindChromeLoadingBar();
});
</script>
