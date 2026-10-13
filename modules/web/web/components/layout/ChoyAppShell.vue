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
          <SidebarHeader
            class="choy-shell__sidebar-header"
            :style="{ height: 'var(--choy-layout-header-height)' }"
          >
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
          Same sticky-footer contract as GuestShell: short canvas pins Powered-by
          to the inset bottom; tall canvas scrolls with footer after content.
        -->
        <div
          class="choy-shell__scroll min-h-0 flex-1 overflow-y-auto"
          data-testid="choy-shell-scroll"
        >
          <div class="choy-shell__scroll-inner flex min-h-full flex-col">
            <div
              class="choy-shell__main-inner flex flex-1 flex-col"
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
              class="choy-shell__footer flex shrink-0 items-center justify-center bg-background px-4 sm:px-6"
              :style="{ minHeight: 'var(--choy-layout-footer-height)' }"
              data-testid="choy-shell-footer"
            >
              <ChoyAppFooter />
              <slot name="footer" />
            </footer>
          </div>
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
  min-height: 0 !important;
  height: 100%;
  max-height: 100%;
}

html.dark .choy-shell[data-shell-mode='app'],
.dark .choy-shell[data-shell-mode='app'] {
  --choy-shell-chrome: oklch(0.18 0 0);
  --choy-shell-pane: oklch(0.205 0 0);
}

/*
 * Kit desktop sidebar uses position:fixed + h-svh and would paint over a shell-level
 * footer. Pin the panel to .choy-shell__body instead; powered-by lives in the inset.
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed {
  position: absolute !important;
  top: 0 !important;
  bottom: 0 !important;
  height: auto !important;
  /* Match sidebar-16: outer edge against the inset canvas. */
  border-right: 1px solid var(--color-sidebar-border, var(--choy-color-sidebar-border)) !important;
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
 * Collapsed: hide labels only. Keep the same justify-start + padding as expanded
 * so the company / user glyphs do not jump horizontally when the L2 pane closes.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-label'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-label'],
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

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-scroll'] {
  background-color: var(--color-background, var(--choy-color-background)) !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-footer'] {
  box-sizing: border-box;
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: flex-start !important;
  gap: 0 !important;
  height: var(--choy-layout-header-height) !important;
  min-height: var(--choy-layout-header-height) !important;
  max-height: var(--choy-layout-header-height) !important;
  /* Match rail icon column inset so the mark stays put when collapsing. */
  padding: 0 0.5rem !important;
  border: 0 !important;
  background-color: var(--choy-shell-chrome) !important;
  background-image: none !important;
  box-shadow: none !important;
}

/*
 * size-6 marks + rounded-md; fixed inline padding keeps expand/collapse x-stable.
 * Vertical center still shares the header band midline with Inset PanelLeft.
 */
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'],
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'] {
  justify-content: flex-start !important;
  width: 100%;
  height: 1.75rem !important;
  min-height: 1.75rem !important;
  padding-block: 0 !important;
  padding-inline: 0.25rem !important;
  border-radius: var(--radius-md, 0.375rem);
  gap: 0.5rem !important;
}

/* Match company mark (size-6); Avatar defaults to size-control without twMerge. */
.choy-shell[data-shell-mode='app']
  [data-testid='auth-user-menu-trigger']
  [data-slot='avatar'] {
  width: 1.5rem !important;
  height: 1.5rem !important;
  min-width: 1.5rem !important;
  min-height: 1.5rem !important;
  border-radius: var(--radius-md, 0.375rem) !important;
}
</style>
