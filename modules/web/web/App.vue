<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-app min-h-screen w-full bg-background text-foreground">
    <router-view />
  </div>
</template>

<script setup lang="ts">
/**
 * Root host for the web SPA. Theme class / density live on documentElement
 * (see applyChoyThemePreference from setupApp); no Element Plus provider.
 */
</script>
