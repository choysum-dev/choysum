<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest marketing top bar aligned to shadcn SiteHeader density. -->
  <header
    class="choy-guest-header sticky top-0 z-50 w-full shrink-0 border-b border-border/40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60"
    data-testid="choy-shell-header"
  >
    <div class="container mx-auto flex h-14 max-w-screen-2xl items-center gap-2 px-4 md:gap-4 md:px-8">
      <a
        href="/"
        class="choy-shell__brand mr-2 inline-flex min-w-0 items-center gap-2 text-sm font-bold tracking-tight text-foreground no-underline hover:opacity-90 md:mr-4"
        data-testid="choy-shell-brand"
        @click.prevent="goHome"
      >
        <img :src="logoUrl" alt="" class="size-5 shrink-0" width="20" height="20" />
        <span class="truncate">Choysum</span>
      </a>
      <nav class="flex min-w-0 items-center gap-4 text-sm md:gap-6" aria-label="Guest">
        <a
          href="/"
          class="text-foreground/80 hover:text-foreground font-medium no-underline transition-colors"
          data-testid="choy-guest-nav-home"
          @click.prevent="goHome"
        >
          {{ homeLabel }}
        </a>
      </nav>
      <div class="ms-auto flex items-center gap-2 md:flex-1 md:justify-end">
        <nav class="flex items-center gap-0.5 [&_button]:size-8">
          <ChoyShellThemeToggle />
          <ChoyShellLocaleMenu />
        </nav>
        <slot name="header-actions" />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import logoUrl from '../../assets/logo-32.png'
import ChoyShellLocaleMenu from './ChoyShellLocaleMenu.vue'
import ChoyShellThemeToggle from './ChoyShellThemeToggle.vue'

defineOptions({ name: 'ChoyGuestHeader' })

defineProps<{
  goHome: () => void
}>()

let tLayout: (key: string) => string = (key) => key
try {
  const i18n = useI18n({ useScope: 'global' })
  tLayout = (key) => String(i18n.t(key))
} catch {
  tLayout = (key) => key
}

const homeLabel = computed(() => tLayout('layout.header.home'))
</script>
