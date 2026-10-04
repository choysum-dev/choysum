<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <header
    class="choy-shell-header sticky top-0 z-40 flex shrink-0 items-center gap-2 bg-background/95 px-4 text-sm backdrop-blur"
    :style="{ height: 'var(--choy-layout-header-height)' }"
    data-testid="choy-shell-header"
  >
    <SidebarTrigger
      v-if="showSidebarChrome"
      data-testid="choy-shell-menu-trigger"
      :aria-label="menuTriggerLabel"
    />
    <a
      href="/"
      class="choy-shell__brand inline-flex min-w-0 items-center gap-2 font-medium tracking-tight text-foreground no-underline hover:opacity-90"
      data-testid="choy-shell-brand"
      @click.prevent="goHome"
    >
      <img :src="logoUrl" alt="" class="size-5 shrink-0 saturate-50" width="20" height="20" />
      <span class="truncate">Choysum</span>
    </a>
    <ChoyShellBreadcrumb v-if="showSidebarChrome" />
    <div class="ms-auto flex items-center gap-2">
      <ChoyCommandPalette v-if="showSidebarChrome" />
      <ChoyShellThemeToggle />
      <ChoyShellLocaleMenu />
      <div
        v-if="$slots['header-actions']"
        class="mx-1 h-4 w-px shrink-0 bg-border"
        data-testid="choy-shell-header-sep"
        aria-hidden="true"
      />
      <slot name="header-actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import logoUrl from '../../assets/logo-32.png'
import ChoyCommandPalette from './ChoyCommandPalette.vue'
import ChoyShellBreadcrumb from './ChoyShellBreadcrumb.vue'
import ChoyShellLocaleMenu from './ChoyShellLocaleMenu.vue'
import ChoyShellThemeToggle from './ChoyShellThemeToggle.vue'
import { SidebarTrigger } from '../vendor/ui/sidebar/index'

defineOptions({ name: 'ChoyShellHeader' })

defineProps<{
  showSidebarChrome: boolean
  menuTriggerLabel: string
  goHome: () => void
}>()
</script>
