<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <header
    class="flex shrink-0 items-center gap-2 border-b border-border px-3 text-sm"
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
      class="choy-shell__brand inline-flex min-w-0 items-center gap-2 font-semibold tracking-tight text-foreground no-underline hover:opacity-90"
      data-testid="choy-shell-brand"
      @click.prevent="goHome"
    >
      <img :src="logoUrl" alt="" class="size-6 shrink-0" width="24" height="24" />
      <span v-if="showSidebarChrome" class="truncate">Choysum</span>
    </a>
    <ChoyShellBreadcrumb v-if="showSidebarChrome" />
    <div class="ms-auto flex items-center gap-1">
      <ChoyCommandPalette v-if="showSidebarChrome" />
      <ChoyShellThemeToggle />
      <ChoyShellLocaleMenu />
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
