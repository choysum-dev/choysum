<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest marketing-style top bar: brand + Home nav, no app sidebar chrome. -->
  <header
    class="choy-guest-header sticky top-0 z-40 w-full shrink-0 border-b bg-background/95 text-sm backdrop-blur"
    :style="{ height: 'var(--choy-layout-header-height)' }"
    data-testid="choy-shell-header"
  >
    <div
      class="mx-auto flex h-full w-full max-w-screen-2xl items-center gap-4 px-4"
    >
      <a
        href="/"
        class="choy-shell__brand inline-flex min-w-0 items-center gap-2 font-medium tracking-tight text-foreground no-underline hover:opacity-90"
        data-testid="choy-shell-brand"
        @click.prevent="goHome"
      >
        <img :src="logoUrl" alt="" class="size-5 shrink-0 saturate-50" width="20" height="20" />
        <span class="truncate">Choysum</span>
      </a>
      <nav class="flex min-w-0 items-center gap-4" aria-label="Guest">
        <a
          href="/"
          class="text-muted-foreground hover:text-foreground font-medium no-underline transition-colors"
          data-testid="choy-guest-nav-home"
          @click.prevent="goHome"
        >
          {{ homeLabel }}
        </a>
      </nav>
      <div class="ms-auto flex items-center gap-2">
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
