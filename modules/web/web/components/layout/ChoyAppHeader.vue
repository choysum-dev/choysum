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
    <!--
      Trigger + sep + breadcrumb share one items-center row.
      size=icon is size-control (32px) with a size-4 (16px) glyph, so the
      hit-box has 8px inset on each side. -ms-2 pulls the button back so the
      glyph’s inline-start edge lands on the same 16px gutter as ChoyPage
      (shadcn sidebar blocks use the same optical -m* on SidebarTrigger).
    -->
    <div
      v-if="showSidebarChrome"
      class="flex min-w-0 flex-1 items-center gap-2"
      data-testid="choy-shell-header-start"
    >
      <ChoyButton
        variant="ghost"
        size="icon"
        type="button"
        class="-ms-2 shrink-0 leading-none"
        data-testid="choy-shell-menu-trigger"
        data-sidebar="trigger"
        data-slot="sidebar-trigger"
        :aria-label="menuTriggerLabel"
        @click="onMenuToggle"
      >
        <PanelLeft class="size-4" aria-hidden="true" />
      </ChoyButton>
      <Separator
        orientation="vertical"
        class="mx-1 !h-4 w-px shrink-0"
        data-testid="choy-shell-header-sep-trigger"
      />
      <ChoyShellBreadcrumb />
    </div>
    <div class="ms-auto flex items-center gap-2">
      <ChoyCommandPalette v-if="showSidebarChrome" />
      <Separator
        v-if="$slots['header-actions']"
        orientation="vertical"
        class="mx-1 !h-4 w-px shrink-0"
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
