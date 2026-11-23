<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <header
    class="choy-app-header sticky top-0 z-50 flex shrink-0 items-center gap-2 bg-transparent text-sm"
    :style="headerStyle"
    data-testid="choy-shell-header"
  >
    <!--
      Trigger + sep + breadcrumb share one items-center row.
      Header T/L/R pad is chrome-padding (shell CSS); bottom is 0. No -ms on
      the trigger: size-4 glyph sits chrome-padding + 8px from the inset corner
      on both top and left (equal air measured on the icon).
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
        class="shrink-0 leading-none"
        data-testid="choy-shell-menu-trigger"
        data-sidebar="trigger"
        data-slot="sidebar-trigger"
        :aria-label="menuTriggerLabel"
        @click="onMenuTriggerClick"
      >
        <PanelLeft class="size-4 rtl:rotate-180" aria-hidden="true" />
      </ChoyButton>
      <Separator
        orientation="vertical"
        class="!h-4 w-px shrink-0"
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
import { runShellMenuToggle } from './runShellMenuToggle'
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

let toggleSidebar: () => void = () => {}
try {
  toggleSidebar = useSidebar().toggleSidebar
} catch {
  toggleSidebar = () => {}
}

function onMenuTriggerClick() {
  runShellMenuToggle(layoutStore, toggleSidebar)
}

/** Height from chrome token; block/inline pad comes from shell product CSS. */
const headerStyle = {
  height: 'var(--choy-layout-chrome-height)',
  boxSizing: 'border-box' as const,
}

</script>
