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
 * Renders nothing; must be a child of SidebarProvider.
 */
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
}

const { open, setOpen, openMobile, setOpenMobile, isMobile } = useSidebar()

let syncing = false

function applyStoreToSidebar() {
  if (!layoutStore) return
  const mode = layoutStore.sidebarMode
  syncing = true
  try {
    if (isMobile.value) {
      setOpenMobile(mode === 'expanded')
    } else {
      setOpen(mode === 'expanded')
    }
  } finally {
    syncing = false
  }
}

function applySidebarToStore() {
  if (!layoutStore || syncing) return
  if (isMobile.value) {
    const next = openMobile.value ? 'expanded' : 'hidden'
    if (layoutStore.sidebarMode !== next) {
      layoutStore.setSidebarMode(next, { isUserAction: true })
    }
    return
  }
  const next = open.value ? 'expanded' : 'collapsed'
  if (layoutStore.sidebarMode !== next) {
    layoutStore.setSidebarMode(next, { isUserAction: true })
  }
}

if (layoutStore) {
  watch(
    () => [layoutStore!.sidebarMode, isMobile.value] as const,
    () => applyStoreToSidebar(),
    { immediate: true },
  )
  watch([open, openMobile, isMobile], () => applySidebarToStore())
}
</script>

<template>
  <span hidden data-testid="choy-sidebar-bridge" aria-hidden="true" />
</template>
