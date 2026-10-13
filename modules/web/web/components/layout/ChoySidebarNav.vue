<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MenuItem } from '@/core/web/menu'
import { translateTerm, createTranslate } from '../../i18n'
import { useMenu } from '../../composables/useMenu'
import { useMenuStore } from '../../stores/menuStore'
import {
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
} from '../vendor/ui/sidebar/index'
import { buildAppNavGroups, type AppNavGroup } from './buildAppNavGroups'
import ChoySidebarNavEntry from './ChoySidebarNavEntry.vue'

/**
 * Product nested sidebar: L2+ of the active app (or the first visible root).
 * Icons are optional; when absent, items render as text-only labels.
 * Branch nodes are Collapsible only (no navigate); expand uses sibling
 * accordion with active-menu ancestry kept open via useMenu.isExpanded.
 */
const { _t: _tMenu } = createTranslate('web', { scope: 'web/components/layout/ChoySidebarNav' })

let composer: { t: (...args: any[]) => unknown } | null = null
try {
  composer = useI18n({ useScope: 'global' }) as any
} catch {
  composer = null
}

let menuApi: ReturnType<typeof useMenu> | null = null
let menuStore: ReturnType<typeof useMenuStore> | null = null
try {
  menuStore = useMenuStore()
} catch {
  menuStore = null
}
try {
  menuApi = useMenu()
} catch {
  menuApi = null
}

function visibleItems(list: MenuItem[] | undefined): MenuItem[] {
  return (list || []).filter((item) => !item.hidden)
}

function labelOf(item: MenuItem): string {
  if (!composer) return String(item.titleText || item.title || '')
  return translateTerm(composer, item.titleText, item.title)
}

const scopedRoot = computed((): MenuItem | null => {
  if (!menuApi || !menuStore) return null
  const roots = visibleItems(menuStore.getMenus())
  const active = menuApi.activeApp.value
  if (active && !active.hidden) return active
  return roots[0] ?? null
})

const navGroups = computed((): AppNavGroup[] => buildAppNavGroups(scopedRoot.value))

const isEmpty = computed(() => !navGroups.value.some((group) => group.items.length))

const emptyText = computed(() => _tMenu('No menus available'))

function leafActive(item: MenuItem): boolean {
  return menuApi?.activeMenu.value?.id === item.id
}

function groupExpanded(item: MenuItem): boolean {
  return !!menuApi?.isExpanded(item.id)
}

function onGroupOpenChange(item: MenuItem, open: boolean) {
  const key = item.id || ''
  if (!key || !menuApi) return
  if (open) menuApi.openSubMenu(key)
  else menuApi.closeSubMenu(key)
}

function onLeafClick(item: MenuItem) {
  void menuApi?.navigateTo(item)
}
</script>

<template>
  <div class="choy-sidebar-nav flex h-full min-h-0 flex-1 flex-col" data-testid="choy-shell-aside">
    <SidebarContent>
    <p
      v-if="isEmpty"
      class="text-sidebar-foreground/70 px-4 py-3 text-sm"
      data-testid="choy-sidebar-nav-empty"
    >
      {{ emptyText }}
    </p>
    <div v-else class="flex flex-col px-1 py-2">
      <SidebarGroup v-for="group in navGroups" :key="group.key">
        <SidebarGroupContent>
          <SidebarMenu>
            <ChoySidebarNavEntry
              v-for="item in group.items"
              :key="item.id || item.path || item.title"
              :item="item"
              :depth="0"
              :label-of="labelOf"
              :leaf-active="leafActive"
              :group-expanded="groupExpanded"
              :on-group-open-change="onGroupOpenChange"
              :on-leaf-click="onLeafClick"
            />
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </div>
    </SidebarContent>
  </div>
</template>

<style scoped>
.choy-sidebar-nav :deep([data-sidebar='group']) {
  padding: 0.25rem 0.5rem;
}
.choy-sidebar-nav :deep([data-sidebar='group-content']) {
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.choy-sidebar-nav :deep([data-sidebar='menu']) {
  gap: 0.125rem;
  width: 100%;
}
.choy-sidebar-nav :deep([data-sidebar='menu-button']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']) {
  height: 2rem;
  padding-inline: 0.5rem;
  font-size: inherit;
  font-weight: 400;
  line-height: 1.25rem;
  border-radius: 0.5rem;
  color: var(--color-sidebar-foreground);
}
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']) {
  width: 100%;
  padding-inline-start: 1.25rem;
}
.choy-sidebar-nav :deep([data-sidebar='menu-button']:hover),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']:hover) {
  background-color: var(--color-sidebar-accent);
  color: var(--color-sidebar-accent-foreground);
}
.choy-sidebar-nav :deep([data-sidebar='menu-button'][data-active='true']),
.choy-sidebar-nav :deep([data-sidebar='menu-button'][data-active='']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button'][data-active='true']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button'][data-active='']) {
  background-color: var(--color-sidebar-accent);
  color: var(--color-sidebar-accent-foreground);
  font-weight: 500;
}
.choy-sidebar-nav :deep([data-testid='choy-sidebar-nav-group']) {
  color: var(--color-sidebar-foreground);
  font-weight: 400;
}
.choy-sidebar-nav :deep([data-testid='choy-sidebar-nav-group']:hover) {
  background-color: var(--color-sidebar-accent);
  color: var(--color-sidebar-accent-foreground);
}
</style>
