<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue'
import { Bookmark, ChevronRight, CircleHelp } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import type { MenuItem } from '@/core/web/menu'
import { translateTerm, createTranslate } from '../../i18n'
import { useMenu } from '../../composables/useMenu'
import { useMenuStore } from '../../stores/menuStore'
import Collapsible from '../vendor/ui/collapsible/Collapsible.vue'
import CollapsibleContent from '../vendor/ui/collapsible/CollapsibleContent.vue'
import CollapsibleTrigger from '../vendor/ui/collapsible/CollapsibleTrigger.vue'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from '../vendor/ui/sidebar/index'

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
        <template v-for="item in items.filter((i) => !i.hidden)" :key="item.id || item.path || item.title">
          <SidebarMenuItem v-if="item.children?.length">
            <Collapsible
              :open="groupExpanded(item)"
              class="group/collapsible"
              @update:open="(v) => onGroupOpenChange(item, !!v)"
            >
              <CollapsibleTrigger as-child>
                <SidebarMenuButton
                  type="button"
                  :tooltip="labelOf(item)"
                  :disabled="item.disabled || undefined"
                  :aria-expanded="groupExpanded(item) ? 'true' : 'false'"
                  data-testid="choy-sidebar-nav-group"
                >
                  <component :is="iconOf(item)" />
                  <span>{{ labelOf(item) }}</span>
                  <ChevronRight
                    class="ms-auto transition-transform group-data-[state=open]/collapsible:rotate-90"
                  />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem
                    v-for="child in item.children.filter((c) => !c.hidden)"
                    :key="child.id || child.path || child.title"
                  >
                    <template v-if="child.children?.length">
                      <Collapsible
                        :open="groupExpanded(child)"
                        class="group/collapsible"
                        @update:open="(v) => onGroupOpenChange(child, !!v)"
                      >
                        <CollapsibleTrigger as-child>
                          <SidebarMenuSubButton
                            as="button"
                            type="button"
                            :aria-expanded="groupExpanded(child) ? 'true' : 'false'"
                            data-testid="choy-sidebar-nav-group"
                          >
                            <component :is="iconOf(child)" />
                            <span>{{ labelOf(child) }}</span>
                          </SidebarMenuSubButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            <SidebarMenuSubItem
                              v-for="grand in child.children.filter((c) => !c.hidden)"
                              :key="grand.id || grand.path || grand.title"
                            >
                              <SidebarMenuSubButton
                                as="button"
                                type="button"
                                :is-active="leafActive(grand)"
                                :disabled="grand.disabled || undefined"
                                data-testid="choy-sidebar-nav-leaf"
                                @click="onLeafClick(grand)"
                              >
                                <component :is="iconOf(grand)" />
                                <span>{{ labelOf(grand) }}</span>
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </Collapsible>
                    </template>
                    <SidebarMenuSubButton
                      v-else
                      as="button"
                      type="button"
                      :is-active="leafActive(child)"
                      :disabled="child.disabled || undefined"
                      data-testid="choy-sidebar-nav-leaf"
                      @click="onLeafClick(child)"
                    >
                      <component :is="iconOf(child)" />
                      <span>{{ labelOf(child) }}</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </Collapsible>
          </SidebarMenuItem>
          <SidebarMenuItem v-else>
            <SidebarMenuButton
              type="button"
              :is-active="leafActive(item)"
              :tooltip="labelOf(item)"
              :disabled="item.disabled || undefined"
              data-testid="choy-sidebar-nav-leaf"
              @click="onLeafClick(item)"
            >
              <component :is="iconOf(item)" />
              <span>{{ labelOf(item) }}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </template>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
