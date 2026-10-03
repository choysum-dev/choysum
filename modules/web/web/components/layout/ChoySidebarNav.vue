<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import { Bookmark, CircleHelp } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import type { MenuItem } from '@/core/web/menu'
import { translateTerm, createTranslate } from '../../i18n'
import { useMenu } from '../../composables/useMenu'
import { useMenuStore } from '../../stores/menuStore'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
} from '../vendor/ui/sidebar/index'
import ChoySidebarNavEntry from './ChoySidebarNavEntry.vue'

/**
 * Product sidebar nav: menu store → SidebarMenu* + Collapsible groups.
 */
const props = withDefaults(
  defineProps<{
    useDefaultIcon?: boolean
  }>(),
  { useDefaultIcon: true },
)

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

const items = computed(() => {
  if (!menuApi || !menuStore) return [] as MenuItem[]
  if (menuApi.activeApp.value?.children?.length) return menuApi.activeApp.value.children
  return menuStore.getMenus()
})

const emptyText = computed(() => _tMenu('No menus available'))

function labelOf(item: MenuItem): string {
  if (!composer) return String(item.titleText || item.title || '')
  return translateTerm(composer, item.titleText, item.title)
}

function iconOf(item: MenuItem) {
  return item.icon || (props.useDefaultIcon ? Bookmark : CircleHelp)
}

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
  <SidebarGroup data-testid="choy-shell-aside">
    <SidebarGroupContent>
      <p
        v-if="!items.length"
        class="text-sidebar-foreground/70 px-2 py-2 text-sm"
        data-testid="choy-sidebar-nav-empty"
      >
        {{ emptyText }}
      </p>
      <SidebarMenu v-else>
        <ChoySidebarNavEntry
          v-for="item in items.filter((i) => !i.hidden)"
          :key="item.id || item.path || item.title"
          :item="item"
          :depth="0"
          :label-of="labelOf"
          :icon-of="iconOf"
          :leaf-active="leafActive"
          :group-expanded="groupExpanded"
          :on-group-open-change="onGroupOpenChange"
          :on-leaf-click="onLeafClick"
        />
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
