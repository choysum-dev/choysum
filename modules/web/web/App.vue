<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-app min-h-screen w-full bg-background text-foreground">
    <router-view />
    <Toaster />
    <ChoyConfirmHost />
  </div>
</template>

<script setup lang="ts">
/**
 * Root host for the web SPA. Theme class / density live on documentElement
 * (see applyChoyThemePreference from setupApp).
 */
import Toaster from './components/vendor/ui/toast/Toaster.vue';
import ChoyConfirmHost from './components/layout/ChoyConfirmHost.vue';
</script>
