<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest header only; App keeps theme/locale in the account menu. -->
  <nav class="flex items-center gap-0.5" data-testid="choy-shell-header-tools">
    <ChoyShellThemeToggle />
    <Separator
      orientation="vertical"
      class="mx-1 hidden !h-4 sm:block"
      data-testid="choy-shell-header-sep-tools"
    />
    <ChoyShellLocaleMenu />
  </nav>
</template>

<script setup lang="ts">
import ChoyShellLocaleMenu from './ChoyShellLocaleMenu.vue'
import ChoyShellThemeToggle from './ChoyShellThemeToggle.vue'
import { Separator } from '../vendor/ui/separator'

defineOptions({ name: 'ChoyShellHeaderTools' })
</script>
