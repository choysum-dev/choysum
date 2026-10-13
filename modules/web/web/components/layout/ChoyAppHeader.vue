<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <header
    class="choy-app-header choy-inset-pad-x sticky top-0 z-50 flex shrink-0 items-center gap-2 bg-background/95 text-sm backdrop-blur"
    :style="headerStyle"
    data-testid="choy-shell-header"
  >
    <ChoyButton
      v-if="showSidebarChrome"
      variant="ghost"
      size="icon"
      type="button"
      class="size-7 shrink-0"
      :style="{ width: '1.75rem', height: '1.75rem' }"
      data-testid="choy-shell-menu-trigger"
      data-sidebar="trigger"
      data-slot="sidebar-trigger"
      :aria-label="menuTriggerLabel"
      @click="onMenuToggle"
    >
      <PanelLeft class="size-4" aria-hidden="true" />
    </ChoyButton>
    <Separator
      v-if="showSidebarChrome"
      orientation="vertical"
      class="mx-2 !h-4 self-center"
      data-testid="choy-shell-header-sep-trigger"
    />
    <ChoyShellBreadcrumb v-if="showSidebarChrome" />
    <div class="ms-auto flex items-center gap-2">
      <ChoyCommandPalette v-if="showSidebarChrome" />
      <Separator
        v-if="$slots['header-actions']"
        orientation="vertical"
        class="mx-1 !h-4"
        data-testid="choy-shell-header-sep"
      />
      <slot name="header-actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { PanelLeft } from 'lucide-vue-next'
import ChoyButton from './ChoyButton.vue'
import ChoyCommandPalette from './ChoyCommandPalette.vue'
import ChoyShellBreadcrumb from './ChoyShellBreadcrumb.vue'
import { Separator } from '../vendor/ui/separator'
import { useSidebar } from '../vendor/ui/sidebar/utils'
import { useLayoutStore } from '../../stores/layoutStore'

defineOptions({ name: 'ChoyAppHeader' })

defineProps<{
  showSidebarChrome: boolean
  menuTriggerLabel: string
}>()

let layoutStore: ReturnType<typeof useLayoutStore> | null = null
try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
}

const { toggleSidebar } = useSidebar()

/** Height only — shared shell top chrome is translucent + blur with no hairline border. */
const headerStyle = {
  height: 'var(--choy-layout-header-height)',
  paddingTop: '0',
  paddingBottom: '0',
  boxSizing: 'border-box' as const,
}

/**
 * Prefer layoutStore so ChoySidebarBridge remains the single writer to kit open state.
 * Falls back to kit toggle when the layout store is unavailable (tests).
 */
function onMenuToggle() {
  if (layoutStore) {
    layoutStore.toggleSidebar()
    return
  }
  toggleSidebar()
}
</script>
