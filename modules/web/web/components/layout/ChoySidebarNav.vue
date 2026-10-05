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
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
} from '../vendor/ui/sidebar/index'
import ChoySidebarNavEntry from './ChoySidebarNavEntry.vue'

type NavGroup = {
  key: string
  label: string
  items: MenuItem[]
}

/**
 * Product sidebar nav: full menu tree as one SidebarGroup per app root + Collapsible.
 * Icons are optional; when absent, items render as text-only labels.
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
  menuApi = useMenu()
} catch {
  menuStore = null
  menuApi = null
}

function visibleItems(list: MenuItem[] | undefined): MenuItem[] {
  return (list || []).filter((item) => !item.hidden)
}

function labelOf(item: MenuItem): string {
  if (!composer) return String(item.titleText || item.title || '')
  return translateTerm(composer, item.titleText, item.title)
}

const navGroups = computed((): NavGroup[] => {
  if (!menuApi || !menuStore) return []
  // Full registered menu tree (every app root), not only activeApp.
  const groups: NavGroup[] = []
  const ungrouped: MenuItem[] = []
  for (const root of visibleItems(menuStore.getMenus())) {
    const children = visibleItems(root.children)
    if (children.length) {
      groups.push({
        key: root.id || root.path || root.title,
        label: labelOf(root),
        items: children,
      })
    } else {
      ungrouped.push(root)
    }
  }
  if (ungrouped.length) {
    groups.push({ key: 'ungrouped', label: '', items: ungrouped })
  }
  return groups
})

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
  <div class="choy-sidebar-nav pt-3" data-testid="choy-shell-aside">
    <p
      v-if="isEmpty"
      class="text-sidebar-foreground/70 px-2 py-2 text-sm"
      data-testid="choy-sidebar-nav-empty"
    >
      {{ emptyText }}
    </p>
    <div v-else class="flex flex-col gap-4">
      <SidebarGroup v-for="group in navGroups" :key="group.key">
        <SidebarGroupLabel
          v-if="group.label"
          data-testid="choy-sidebar-nav-group-label"
        >
          {{ group.label }}
        </SidebarGroupLabel>
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
  </div>
</template>

<style scoped>
.choy-sidebar-nav :deep([data-sidebar='group']) {
  padding-block: 0;
}
.choy-sidebar-nav :deep([data-sidebar='group-content']) {
  font-size: 0.875rem; /* text-sm */
  line-height: 1.25rem;
}
.choy-sidebar-nav :deep([data-sidebar='group-label']) {
  margin-bottom: 0.25rem;
  height: auto;
  min-height: 1.75rem;
  font-size: 0.75rem; /* text-xs */
  font-weight: 500;
  line-height: 1rem;
  /* Softer than leaves: opaque mix of muted into the sidebar surface (not transparent). */
  color: color-mix(
    in oklab,
    var(--color-muted-foreground) 55%,
    var(--color-sidebar, var(--color-background))
  ) !important;
}
.choy-sidebar-nav :deep([data-sidebar='menu']),
.choy-sidebar-nav :deep([data-sidebar='menu-badge']) {
  gap: 0.25rem; /* gap-1 */
  margin-inline-start: 0;
  transform: none;
  border-inline-start-width: 0;
  padding-inline: 0;
  width: 100%;
}
.choy-sidebar-nav :deep([data-sidebar='menu-button']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']) {
  height: 30px;
  padding: 0.5rem; /* p-2 */
  font-size: inherit; /* inherit group-content text-sm */
  font-weight: 400;
  line-height: 1.25rem;
  color: var(--color-foreground);
}
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']) {
  width: 100%;
  padding-inline-start: 1rem;
}
.choy-sidebar-nav :deep([data-sidebar='menu-button']:hover),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button']:hover) {
  background-color: color-mix(in oklab, var(--color-accent) 50%, transparent);
  color: var(--color-foreground);
}
.choy-sidebar-nav :deep([data-sidebar='menu-button'][data-active='true']),
.choy-sidebar-nav :deep([data-sidebar='menu-button'][data-active='']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button'][data-active='true']),
.choy-sidebar-nav :deep([data-sidebar='menu-sub-button'][data-active='']) {
  background-color: var(--color-accent);
  color: var(--color-foreground);
  font-weight: 500;
}
.choy-sidebar-nav :deep([data-testid='choy-sidebar-nav-group']) {
  color: var(--color-foreground);
  font-weight: 400;
}
.choy-sidebar-nav :deep([data-testid='choy-sidebar-nav-group']:hover) {
  background-color: color-mix(in oklab, var(--color-accent) 30%, transparent);
  color: var(--color-foreground);
}
</style>

