<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MenuItem } from '@/core/web/menu'
import { translateTerm } from '../../i18n'
import { useMenu } from '../../composables/useMenu'
import { useMenuStore } from '../../stores/menuStore'
import { firstGrapheme } from './appRailGlyph'
import { Tooltip, TooltipTrigger } from '../vendor/ui/tooltip'
import { TooltipContent, TooltipPortal } from 'reka-ui'
import {
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '../vendor/ui/sidebar/index'
import { useSidebar } from '../vendor/ui/sidebar/utils'

/**
 * Always-visible L1 app icon rail (sidebar-09 inner icon column).
 * Clicks navigate to the first navigable leaf of that app root.
 */
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

let sidebarIsMobile = computed(() => false)
try {
  const { isMobile } = useSidebar()
  sidebarIsMobile = computed(() => !!isMobile.value)
} catch {
  sidebarIsMobile = computed(() => false)
}

function visibleItems(list: MenuItem[] | undefined): MenuItem[] {
  return (list || []).filter((item) => !item.hidden)
}

function labelOf(item: MenuItem): string {
  if (!composer) return String(item.titleText || item.title || '')
  return translateTerm(composer, item.titleText, item.title)
}

const apps = computed(() => {
  if (!menuStore) return []
  return visibleItems(menuStore.getMenus())
})

function isActive(app: MenuItem): boolean {
  return menuApi?.activeApp.value?.id === app.id
}

function tileClass(app: MenuItem): string {
  return isActive(app)
    ? 'bg-sidebar-primary text-sidebar-primary-foreground'
    : 'text-sidebar-foreground/80 group-hover/rail:bg-sidebar-accent group-hover/rail:text-sidebar-accent-foreground'
}

function onSelect(app: MenuItem) {
  void menuApi?.navigateTo(app)
}

function appKey(app: MenuItem): string {
  return app.id || app.path || app.title || ''
}

const tooltipOpen = reactive<Record<string, boolean>>({})

function onTooltipOpen(app: MenuItem, open: boolean) {
  const key = appKey(app)
  if (!key) return
  tooltipOpen[key] = open
}
</script>

<template>
  <div
    class="bg-muted text-sidebar-foreground flex h-full shrink-0 flex-col overflow-hidden"
    data-testid="choy-shell-app-rail"
    :style="{ width: 'calc(var(--sidebar-width-icon) + 1px)' }"
  >
    <SidebarContent class="pt-2">
      <SidebarGroup class="px-2 py-0">
        <SidebarGroupContent>
          <SidebarMenu class="gap-1.5">
            <SidebarMenuItem v-for="app in apps" :key="appKey(app)">
              <Tooltip
                :open="!!tooltipOpen[appKey(app)]"
                @update:open="(open: boolean) => onTooltipOpen(app, open)"
              >
                <TooltipTrigger as-child>
                  <SidebarMenuButton
                    size="lg"
                    type="button"
                    class="group/rail h-10 justify-center p-1 hover:bg-transparent data-[active=true]:bg-transparent"
                    :is-active="isActive(app)"
                    data-testid="choy-shell-app-rail-item"
                    @click="onSelect(app)"
                  >
                    <span
                      class="flex size-8 shrink-0 items-center justify-center rounded-lg [&>svg]:size-4"
                      data-testid="choy-shell-app-rail-tile"
                      :class="tileClass(app)"
                    >
                      <component :is="app.icon" v-if="app.icon" />
                      <span
                        v-else
                        class="text-xs font-medium leading-none"
                        data-testid="choy-shell-app-rail-fallback"
                      >
                        {{ firstGrapheme(labelOf(app)) }}
                      </span>
                    </span>
                    <span class="sr-only">{{ labelOf(app) }}</span>
                  </SidebarMenuButton>
                </TooltipTrigger>
                <TooltipPortal v-if="tooltipOpen[appKey(app)] && !sidebarIsMobile">
                  <TooltipContent
                    data-testid="choy-shell-app-rail-tooltip"
                    side="right"
                    align="center"
                    :side-offset="8"
                    class="z-50 overflow-hidden rounded-md border border-border bg-background px-3 py-1.5 text-xs text-foreground shadow-md"
                  >
                    {{ labelOf(app) }}
                  </TooltipContent>
                </TooltipPortal>
              </Tooltip>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
  </div>
</template>
