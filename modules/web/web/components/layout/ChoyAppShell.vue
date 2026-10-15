<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- App shell: full-height sidebar + Inset header/canvas + shared footer. -->
  <SidebarProvider
    class="choy-shell flex h-full min-h-0 w-full flex-col overflow-hidden"
    data-testid="choy-shell"
    data-shell-mode="app"
  >
    <div class="choy-shell__body relative flex min-h-0 flex-1 overflow-hidden">
      <template v-if="showSidebar">
        <ChoySidebarBridge />
        <Sidebar collapsible="icon" side="left">
          <!-- Height/padding locked in product CSS so kit p-2 cannot shrink the band. -->
          <SidebarHeader class="choy-shell__sidebar-header">
            <div
              data-anchor="choy.shell.sidebar-header"
              class="flex w-full min-w-0 items-center"
              data-testid="choy-shell-sidebar-header"
            >
              <slot name="sidebar-header" />
            </div>
          </SidebarHeader>
          <div
            class="flex min-h-0 w-full flex-1 flex-row overflow-hidden"
            data-testid="choy-shell-nav-split"
          >
            <ChoyAppRail />
            <div
              class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
              data-testid="choy-shell-nav-pane"
            >
              <div
                class="flex min-h-0 flex-1 flex-col overflow-auto"
                data-testid="choy-shell-nav-pane-scroll"
              >
                <slot name="aside">
                  <ChoySidebarNav />
                </slot>
              </div>
            </div>
          </div>
          <SidebarFooter class="choy-shell__sidebar-footer">
            <div
              data-anchor="choy.shell.sidebar-footer"
              class="flex w-full min-w-0 items-center"
              data-testid="choy-shell-sidebar-footer"
            >
              <slot name="sidebar-footer" />
            </div>
          </SidebarFooter>
          <SidebarRail />
        </Sidebar>
      </template>
      <SidebarInset class="min-h-0 min-w-0 overflow-hidden">
        <ChoyAppHeader
          v-if="effectiveShowHeader"
          :show-sidebar-chrome="showSidebar"
          :menu-trigger-label="menuTriggerLabel"
        >
          <template #header-actions>
            <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
              <slot name="header-actions" />
            </div>
          </template>
        </ChoyAppHeader>
        <!--
          Workspace layout (ERP): main column does not scroll; footer is pinned
          below the canvas. List/Kanban/Form fill the canvas and scroll inside.
        -->
        <div
          class="choy-shell__workspace flex min-h-0 flex-1 flex-col overflow-hidden"
          data-testid="choy-shell-scroll"
          data-shell-layout="workspace"
        >
          <div
            class="choy-shell__main-inner flex min-h-0 flex-1 flex-col overflow-hidden"
            data-testid="choy-shell-canvas"
          >
            <slot>
              <router-view v-slot="{ Component, route: viewRoute }">
                <KeepAlive>
                  <component
                    :is="Component"
                    v-if="Component && viewRoute.meta?.keepAlive"
                    class="min-h-0 flex-1"
                    :key="viewRoute.path"
                  />
                </KeepAlive>
                <component
                  :is="Component"
                  v-if="Component && !viewRoute.meta?.keepAlive"
                  class="min-h-0 flex-1"
                  :key="viewRoute.fullPath"
                />
              </router-view>
            </slot>
          </div>
          <footer
            v-if="showFooter"
            class="choy-shell__footer choy-inset-pad-x flex shrink-0 items-center justify-center bg-background"
            :style="{ minHeight: 'var(--choy-layout-footer-height)' }"
            data-testid="choy-shell-footer"
          >
            <ChoyAppFooter />
            <slot name="footer" />
          </footer>
        </div>
      </SidebarInset>
    </div>
  </SidebarProvider>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyAppHeader from './ChoyAppHeader.vue'
import ChoyAppRail from './ChoyAppRail.vue'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import {
  Sidebar,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/**
 * App shell: full-height sidebar (company header / L1+L2 / user footer)
 * plus Inset chrome (trigger, breadcrumb, search, notifications) and canvas.
 */
const props = withDefaults(
  defineProps<{
    showHeader?: boolean
    showSidebar?: boolean
    showFooter?: boolean
  }>(),
  {
    showHeader: true,
    showSidebar: true,
    showFooter: true,
  },
)

let layoutStore: ReturnType<typeof useLayoutStore> | null = null
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key

try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
}

try {
  const i18n = useI18n({ useScope: 'global' })
  tLayout = (key, values) => String(i18n.t(key, values as any))
} catch {
  tLayout = (key) => key
}

const effectiveShowHeader = computed(() => props.showHeader)

const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: layoutStore?.sidebarMode === 'collapsed',
    t: (key) => tLayout(key),
  }),
)
</script>

<style>
/* Fill the app host; override kit SidebarProvider min-h-svh so inset chrome can size the column. */
.choy-shell[data-shell-mode='app'] {
  --sidebar-width: calc(var(--sidebar-width-icon) + 13.5rem) !important;
  /* Chrome + L1 + header/footer; L2 is a lighter nested pane. Both neutral. */
  --choy-shell-chrome: oklch(0.965 0 0);
  --choy-shell-pane: oklch(0.985 0 0);
  /* Identity row hover/open on chrome — keep in sync with --choy-color-sidebar-accent. */
  --choy-shell-chrome-hover: var(--choy-color-sidebar-accent);
  min-height: 0 !important;
  height: 100%;
  max-height: 100%;
}

html.dark .choy-shell[data-shell-mode='app'],
.dark .choy-shell[data-shell-mode='app'] {
  --choy-shell-chrome: oklch(0.18 0 0);
  --choy-shell-pane: oklch(0.205 0 0);
  --choy-shell-chrome-hover: var(--choy-color-sidebar-accent);
}

/*
 * Kit desktop sidebar uses position:fixed + h-svh and would paint over a shell-level
 * footer. Pin the panel to .choy-shell__body instead; powered-by lives in the inset.
 *
 * Collapse motion matches sidebar-07 (width / left / right over 200ms linear).
 * Product override — tailwind-go emits transition-[width] without `width` in
 * transition-property, so the kit utility classes alone do not animate.
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > div {
  transition-property: width;
  transition-duration: 200ms;
  transition-timing-function: linear;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed {
  position: absolute !important;
  top: 0 !important;
  bottom: 0 !important;
  height: auto !important;
  box-sizing: border-box !important;
  /* Real border steals 1px from the icon rail; outer/inset shadows get covered. */
  border-right: 0 !important;
  box-shadow: none !important;
  transition-property: left, right, width;
  transition-duration: 200ms;
  transition-timing-function: linear;
}

/* Full-height hairline above chrome children; does not consume layout width. */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 40;
  width: 1px;
  pointer-events: none;
  background-color: var(--color-sidebar-border, var(--choy-color-sidebar-border));
}

/*
 * Collapsed = L1 only.
 * Hide the L2 pane and force the kit gap + fixed panel to icon width.
 * Tailwind group-data width utilities do not always win against the expanded
 * --sidebar-width override, so product CSS owns the collapsed width.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='choy-shell-nav-pane'] {
  display: none !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-collapsible='icon'] > div {
  width: var(--sidebar-width-icon) !important;
}

/*
 * Collapsed: hide labels / meta / chevrons only. Mark size + left inset stay
 * identical to expanded so width animation does not jump the glyphs.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-label'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-meta'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-label'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-meta'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-chevrons'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-chevrons'] {
  display: none !important;
}

/*
 * Chrome (L1 + spanning header/footer) = 0.965; nested L2 pane = 0.985.
 * Collapsed: whole panel is chrome (L2 hidden).
 */
.choy-shell[data-shell-mode='app'] [data-sidebar='sidebar'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-app-rail'] {
  background-color: var(--choy-shell-chrome) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-nav-pane'] {
  background-color: var(--choy-shell-pane) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-scroll'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-canvas'] {
  background-color: var(--color-background, var(--choy-color-background)) !important;
}

/*
 * sidebar-07-like header/footer: size=lg (h-12) identity rows.
 * Inline pad stays 0.5rem: in a 3rem icon rail that leaves a 2rem slot
 * (size-8), so collapse only clips text and mark glyphs do not shift.
 * Block pad matches inline (0.5rem): enough edge air without a tall chrome band.
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-footer'] {
  box-sizing: border-box;
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: flex-start !important;
  gap: 0 !important;
  height: auto !important;
  min-height: 0 !important;
  max-height: none !important;
  padding-block: 0.5rem !important;
  padding-inline: 0.5rem !important;
  border: 0 !important;
  background-color: var(--choy-shell-chrome) !important;
  background-image: none !important;
  box-shadow: none !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'],
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'] {
  justify-content: flex-start !important;
  align-items: center !important;
  width: 100% !important;
  height: 3rem !important;
  min-height: 3rem !important;
  min-width: 0 !important;
  /* No inline pad: mark x stays at header inset in both expand and collapse. */
  padding-block: 0.5rem !important;
  padding-inline: 0 !important;
  margin-inline: 0 !important;
  border-radius: var(--radius-md, 0.375rem);
  gap: 0.5rem !important;
  overflow: hidden !important;
}

/* Chrome rows need an explicit hover fill; kit accent alone was ≈ chrome in light. */
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'][data-state='open'],
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'][data-state='open'] {
  background-color: var(--choy-shell-chrome-hover) !important;
}

/* Defeat kit icon-mode size-control shrink — keep the same h-12 hit row. */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-trigger'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-trigger'] {
  width: 100% !important;
  height: 3rem !important;
  min-height: 3rem !important;
  min-width: 0 !important;
  padding-block: 0.5rem !important;
  padding-inline: 0 !important;
  margin-inline: 0 !important;
  justify-content: flex-start !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='company-switch-logo-wrap'],
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-logo'] {
  width: 2rem !important;
  height: 2rem !important;
  min-width: 2rem !important;
  min-height: 2rem !important;
}

.choy-shell[data-shell-mode='app']
  [data-testid='auth-user-menu-trigger']
  [data-slot='avatar'],
.choy-shell[data-shell-mode='app']
  [data-testid='auth-user-menu-heading']
  [data-slot='avatar'] {
  width: 2rem !important;
  height: 2rem !important;
  min-width: 2rem !important;
  min-height: 2rem !important;
  border-radius: 9999px !important;
}
</style>
