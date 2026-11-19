<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MenuItem } from '@/core/web/menu'
import { translateTerm } from '../../i18n'
import { useMenu } from '../../composables/useMenu'
import { useMenuStore } from '../../stores/menuStore'
import { useLayoutStore } from '../../stores/layoutStore'
import { useI18nStore } from '../../stores/i18nStore'
import { firstGrapheme } from './appRailGlyph'
import {
  APP_RAIL_HOVER_CLOSE_MS,
  APP_RAIL_HOVER_OPEN_MS,
  APP_RAIL_LEAF_SUPPRESS_MS,
} from './appRailHover'
import ChoyAppRailFlyout from './ChoyAppRailFlyout.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '../vendor/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '../vendor/ui/dropdown-menu'
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
 * Expanded: click navigates to the first navigable leaf; tooltip shows the name.
 * Collapsed: click/hover opens a right multi-level flyout (no blind first-leaf jump).
 */
let composer: { t: (...args: any[]) => unknown } | null = null
try {
  composer = useI18n({ useScope: 'global' }) as any
} catch {
  composer = null
}

let menuApi: ReturnType<typeof useMenu> | null = null
let menuStore: ReturnType<typeof useMenuStore> | null = null
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
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
try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
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

/** Collapsed desktop/tablet rail: flyout menus. Mobile sheet keeps expanded semantics. */
const useFlyout = computed(
  () => !!layoutStore && layoutStore.sidebarMode === 'collapsed' && !sidebarIsMobile.value,
)

/** Popovers open toward the content canvas (opposite the sidebar edge). */
function resolveRailPopupSide(): 'left' | 'right' {
  try {
    return useI18nStore().activeLanguage?.textDirection === 'rtl' ? 'left' : 'right'
  } catch {
    return 'right'
  }
}
const railPopupSide = computed(resolveRailPopupSide)

function isActive(app: MenuItem): boolean {
  return menuApi?.activeApp.value?.id === app.id
}

function tileClass(app: MenuItem): string {
  return isActive(app)
    ? 'bg-sidebar-primary/15 text-sidebar-primary'
    : 'text-sidebar-foreground/80 group-hover/rail:bg-sidebar-accent group-hover/rail:text-sidebar-accent-foreground'
}

function appKey(app: MenuItem): string {
  return app.id || app.path || app.title || ''
}

/**
 * Pointer press focuses the button before click/paint, so kit focus-visible:ring-2
 * flashes even when CSS/blur clear it a frame later. preventDefault keeps focus
 * where it was; keyboard Tab/Enter still focuses and activates normally.
 */
function onRailMouseDown(e: MouseEvent) {
  if (e.button !== 0) return
  e.preventDefault()
}

function onSelectApp(app: MenuItem) {
  if (useFlyout.value) return
  void menuApi?.navigateTo(app)
  blurRailFocus()
}

/**
 * After a leaf click: portal teardown often click-throughs to the rail trigger
 * and synthesizes mouseenter. Suppress opens briefly, then unlock so a later
 * hover on the same app icon can open again (leave-based unlock fails because
 * onFlyoutLeaf already zeros intent).
 */
const suppressFlyoutOpen = ref(false)
let suppressFlyoutTimer: ReturnType<typeof setTimeout> | null = null

function clearSuppressFlyoutTimer() {
  if (suppressFlyoutTimer) {
    clearTimeout(suppressFlyoutTimer)
    suppressFlyoutTimer = null
  }
}

function armFlyoutSuppress() {
  suppressFlyoutOpen.value = true
  clearSuppressFlyoutTimer()
  suppressFlyoutTimer = setTimeout(() => {
    suppressFlyoutTimer = null
    suppressFlyoutOpen.value = false
    // Drop intent accrued while suppressed so we do not instant-open under a
    // still-hovering pointer; the next enter (or leave→enter) can open.
    for (const k of Object.keys(hoverIntent)) hoverIntent[k] = 0
    clearAllHoverTimers()
  }, APP_RAIL_LEAF_SUPPRESS_MS)
}

function blurRailFocus() {
  const drop = () => {
    const el = document.activeElement
    if (el instanceof HTMLElement && el.closest('[data-testid="choy-shell-app-rail"]')) {
      el.blur()
    }
  }
  drop()
  // Reka may restore focus to the flyout trigger after close — clear again.
  queueMicrotask(() => {
    requestAnimationFrame(drop)
  })
}

function onFlyoutLeaf(item: MenuItem) {
  void menuApi?.navigateTo(item)
  openFlyoutKey.value = null
  for (const k of Object.keys(hoverIntent)) hoverIntent[k] = 0
  clearAllHoverTimers()
  armFlyoutSuppress()
  blurRailFocus()
}

const tooltipOpen = reactive<Record<string, boolean>>({})

function onTooltipOpen(app: MenuItem, open: boolean) {
  if (useFlyout.value) return
  const key = appKey(app)
  if (!key) return
  tooltipOpen[key] = open
}

const openFlyoutKey = ref<string | null>(null)
/** Pointer is over this app's trigger and/or portaled content (not a DOM parent — content is portaled). */
const hoverIntent = reactive<Record<string, number>>({})
const hoverOpenTimers = new Map<string, ReturnType<typeof setTimeout>>()
const hoverCloseTimers = new Map<string, ReturnType<typeof setTimeout>>()
function clearHoverTimers(key: string) {
  const openT = hoverOpenTimers.get(key)
  if (openT) {
    clearTimeout(openT)
    hoverOpenTimers.delete(key)
  }
  const closeT = hoverCloseTimers.get(key)
  if (closeT) {
    clearTimeout(closeT)
    hoverCloseTimers.delete(key)
  }
}

function clearAllHoverTimers() {
  for (const key of new Set([...hoverOpenTimers.keys(), ...hoverCloseTimers.keys()])) {
    clearHoverTimers(key)
  }
}

onBeforeUnmount(() => {
  clearAllHoverTimers()
  clearSuppressFlyoutTimer()
})

function isFlyoutOpen(app: MenuItem): boolean {
  const key = appKey(app)
  return !!key && openFlyoutKey.value === key
}

function setFlyoutOpen(app: MenuItem, open: boolean) {
  const key = appKey(app)
  if (!key || !useFlyout.value) return
  if (open) {
    // Leaf-select suppress: ignore trigger re-click / Reka open while pointer never left.
    if (suppressFlyoutOpen.value) return
    clearHoverTimers(key)
    openFlyoutKey.value = key
    return
  }
  // Always honor close after a leaf select (do not keep open via hoverIntent).
  if (suppressFlyoutOpen.value) {
    clearHoverTimers(key)
    if (openFlyoutKey.value === key) openFlyoutKey.value = null
    return
  }
  // Ignore Reka dismiss while the pointer is still on the trigger/content (modal overlay flicker).
  if ((hoverIntent[key] || 0) > 0) return
  // Nested SubContent is portaled: leaving L1 briefly zeros intent and Reka emits
  // open=false before the pointer reaches L2. Keep open for the hover-close grace.
  if (hoverCloseTimers.has(key)) return
  clearHoverTimers(key)
  if (openFlyoutKey.value === key) openFlyoutKey.value = null
}

function bumpFlyoutHover(app: MenuItem, delta: number) {
  if (!useFlyout.value) return
  const key = appKey(app)
  if (!key) return
  const prev = hoverIntent[key] || 0
  const next = Math.max(0, prev + delta)
  hoverIntent[key] = next
  clearHoverTimers(key)
  if (next > 0) {
    // Any other app's pending hover is stale once the pointer moves here;
    // clear it so a late open timer cannot resurrect the wrong flyout.
    for (const other of Object.keys(hoverIntent)) {
      if (other !== key) {
        hoverIntent[other] = 0
        clearHoverTimers(other)
      }
    }
    if (openFlyoutKey.value && openFlyoutKey.value !== key) {
      openFlyoutKey.value = null
    }
    if (openFlyoutKey.value === key) return
    // After leaf select, wait for a full leave (intent→0) before hover can reopen.
    if (suppressFlyoutOpen.value) return
    hoverOpenTimers.set(
      key,
      setTimeout(() => {
        hoverOpenTimers.delete(key)
        if (suppressFlyoutOpen.value) return
        if ((hoverIntent[key] || 0) > 0) openFlyoutKey.value = key
      }, APP_RAIL_HOVER_OPEN_MS),
    )
  } else {
    // Real leave during suppress: unlock early so a quick return can hover-open.
    if (prev > 0 && suppressFlyoutOpen.value) {
      clearSuppressFlyoutTimer()
      suppressFlyoutOpen.value = false
    }
    hoverCloseTimers.set(
      key,
      setTimeout(() => {
        hoverCloseTimers.delete(key)
        if ((hoverIntent[key] || 0) === 0 && openFlyoutKey.value === key) {
          openFlyoutKey.value = null
        }
      }, APP_RAIL_HOVER_CLOSE_MS),
    )
  }
}

function onFlyoutHoverEnter(app: MenuItem) {
  bumpFlyoutHover(app, 1)
}

function onFlyoutHoverLeave(app: MenuItem) {
  bumpFlyoutHover(app, -1)
}

watch(useFlyout, (enabled) => {
  if (!enabled) {
    openFlyoutKey.value = null
    clearSuppressFlyoutTimer()
    suppressFlyoutOpen.value = false
    for (const k of Object.keys(hoverIntent)) hoverIntent[k] = 0
    clearAllHoverTimers()
  }
})
</script>

<template>
  <div
    class="choy-app-rail text-sidebar-foreground flex h-full shrink-0 flex-col overflow-hidden"
    data-testid="choy-shell-app-rail"
    :style="{ width: 'var(--sidebar-width-icon)' }"
  >
    <SidebarContent class="pt-1.5">
      <SidebarGroup class="px-2 py-0">
        <SidebarGroupContent>
          <SidebarMenu class="gap-1">
            <SidebarMenuItem v-for="app in apps" :key="appKey(app)">
              <!-- Collapsed: right multi-level flyout (hover + click open only). -->
              <DropdownMenu
                v-if="useFlyout"
                :modal="false"
                :open="isFlyoutOpen(app)"
                @update:open="(open: boolean) => setFlyoutOpen(app, open)"
              >
                <DropdownMenuTrigger as-child>
                  <SidebarMenuButton
                    size="lg"
                    type="button"
                    class="group/rail h-9 justify-center p-1 outline-none hover:bg-transparent focus:ring-0 focus-visible:ring-0 data-[active=true]:bg-transparent"
                    :is-active="isActive(app)"
                    data-testid="choy-shell-app-rail-item"
                    @mousedown="onRailMouseDown"
                    @mouseenter="onFlyoutHoverEnter(app)"
                    @mouseleave="onFlyoutHoverLeave(app)"
                  >
                    <span
                      class="flex size-7 shrink-0 items-center justify-center rounded-md [&>svg]:size-4"
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
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  :side="railPopupSide"
                  align="start"
                  :side-offset="4"
                  class="choy-app-rail-flyout min-w-[10rem] outline-none"
                  data-testid="choy-shell-app-rail-flyout"
                  @mouseenter="onFlyoutHoverEnter(app)"
                  @mouseleave="onFlyoutHoverLeave(app)"
                  @close-auto-focus="(e: Event) => e.preventDefault()"
                >
                  <ChoyAppRailFlyout
                    :root="app"
                    :label-of="labelOf"
                    :leaf-select="onFlyoutLeaf"
                    :hover-enter="() => onFlyoutHoverEnter(app)"
                    :hover-leave="() => onFlyoutHoverLeave(app)"
                  />
                </DropdownMenuContent>
              </DropdownMenu>

              <!-- Expanded: tooltip name + click navigates to first leaf. -->
              <Tooltip
                v-else
                :open="!!tooltipOpen[appKey(app)]"
                @update:open="(open: boolean) => onTooltipOpen(app, open)"
              >
                <TooltipTrigger as-child>
                  <SidebarMenuButton
                    size="lg"
                    type="button"
                    class="group/rail h-9 justify-center p-1 outline-none hover:bg-transparent focus:ring-0 focus-visible:ring-0 active:bg-transparent data-[active=true]:bg-transparent"
                    :is-active="isActive(app)"
                    data-testid="choy-shell-app-rail-item"
                    @mousedown="onRailMouseDown"
                    @click="onSelectApp(app)"
                  >
                    <span
                      class="flex size-7 shrink-0 items-center justify-center rounded-md [&>svg]:size-4"
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
                <TooltipContent
                  v-if="tooltipOpen[appKey(app)] && !sidebarIsMobile"
                  data-testid="choy-shell-app-rail-tooltip"
                  :side="railPopupSide"
                  align="center"
                  :side-offset="8"
                >
                  {{ labelOf(app) }}
                </TooltipContent>
              </Tooltip>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
  </div>
</template>

<style scoped>
/*
 * Expanded click must not paint a focus ring flash. Kit uses focus-visible:ring-2;
 * pointer mousedown preventDefault avoids focus, and this kills any residual ring.
 */
.choy-app-rail :deep([data-testid='choy-shell-app-rail-item']),
.choy-app-rail :deep([data-testid='choy-shell-app-rail-item']:focus),
.choy-app-rail :deep([data-testid='choy-shell-app-rail-item']:focus-visible),
.choy-app-rail :deep([data-testid='choy-shell-app-rail-item']:active) {
  outline: none;
  -webkit-tap-highlight-color: transparent;
  --tw-ring-offset-shadow: 0 0 #0000;
  --tw-ring-shadow: 0 0 #0000;
  box-shadow: none;
}
</style>
