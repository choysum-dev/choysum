<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { watch } from 'vue'
import { useSidebar } from '../vendor/ui/sidebar/utils'
import { useLayoutStore } from '../../stores/layoutStore'

/**
 * Keeps layoutStore.sidebarMode in sync with SidebarProvider open / openMobile.
 * Store→sidebar runs on mode or viewport class changes.
 * Sidebar→store only follows open / openMobile toggles (not isMobile alone),
 * so resizing does not rewrite a persisted preference.
 */
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
}

const { open, setOpen, openMobile, setOpenMobile, isMobile } = useSidebar()

function applyStoreToSidebar() {
  if (!layoutStore) return
  const mode = layoutStore.sidebarMode
  if (isMobile.value) {
    const wantOpen = mode === 'expanded'
    if (openMobile.value !== wantOpen) setOpenMobile(wantOpen)
  } else {
    const wantOpen = mode === 'expanded'
    if (open.value !== wantOpen) setOpen(wantOpen)
  }
}

/** Write store only when expanded-ness actually changed for the active rail. */
function applySidebarToStore(isOpen: boolean, closedMode: 'hidden' | 'collapsed') {
  if (!layoutStore) return
  const storeOpen = layoutStore.sidebarMode === 'expanded'
  if (storeOpen === isOpen) return
  layoutStore.setSidebarMode(isOpen ? 'expanded' : closedMode, { isUserAction: true })
}

if (layoutStore) {
  watch(
    () => [layoutStore!.sidebarMode, isMobile.value] as const,
    () => applyStoreToSidebar(),
    { immediate: true },
  )
  watch(open, (v) => {
    if (!isMobile.value) applySidebarToStore(!!v, 'collapsed')
  })
  watch(openMobile, (v) => {
    if (isMobile.value) applySidebarToStore(!!v, 'hidden')
  })
}
</script>

<template>
  <span hidden data-testid="choy-sidebar-bridge" aria-hidden="true" />
</template>
