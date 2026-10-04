<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Auth / fullscreen canvas: no Sidebar chrome. -->
  <div
    v-if="isAuthPage || !sidebarAllowed"
    class="flex min-h-svh w-full flex-col text-foreground"
    :class="isAuthPage ? 'choy-shell--auth bg-muted' : 'bg-background'"
    data-testid="choy-shell"
    data-shell-mode="canvas"
  >
    <header
      v-if="effectiveShowHeader"
      class="shrink-0 border-b border-border"
      data-testid="choy-shell-header"
    >
      <div
        class="flex items-center gap-2 px-3 text-sm"
        :style="{ height: 'var(--choy-layout-header-height)' }"
      >
        <a
          href="/"
          class="choy-shell__brand inline-flex min-w-0 items-center gap-2 font-semibold tracking-tight text-foreground no-underline hover:opacity-90"
          data-testid="choy-shell-brand"
          @click.prevent="onBrandClick"
        >
          <img :src="logoUrl" alt="" class="size-6 shrink-0" width="24" height="24" />
          <span class="truncate">Choysum</span>
        </a>
        <div data-anchor="choy.shell.header-actions" class="ms-auto flex items-center gap-1">
          <slot name="header-actions" />
        </div>
      </div>
    </header>
      <div
        class="choy-shell__main-inner min-h-0 flex-1 overflow-auto"
        :class="isAuthPage ? 'bg-muted' : 'bg-muted/30'"
        data-testid="choy-shell-canvas"
      >
        <slot>
          <router-view v-slot="{ Component, route: viewRoute }">
            <KeepAlive>
              <component
                :is="Component"
                v-if="Component && viewRoute.meta?.keepAlive"
                :key="viewRoute.path"
              />
            </KeepAlive>
            <component
              :is="Component"
              v-if="Component && !viewRoute.meta?.keepAlive"
              :key="viewRoute.fullPath"
            />
          </router-view>
        </slot>
      </div>
    <footer
      v-if="showFooter && $slots.footer"
      class="shrink-0 border-t border-border px-4 py-2 text-xs text-foreground/60"
    >
      <slot name="footer" />
    </footer>
  </div>

  <!-- Product shell: SidebarProvider + Sidebar + SidebarInset. -->
  <SidebarProvider
    v-else
    class="choy-shell min-h-svh"
    data-testid="choy-shell"
    data-shell-mode="sidebar"
  >
    <ChoySidebarBridge />
    <Sidebar collapsible="icon" side="left">
      <SidebarHeader class="gap-2 border-b border-sidebar-border px-2 py-2">
        <a
          href="/"
          class="choy-shell__brand flex min-w-0 items-center gap-2 rounded-md px-2 py-1.5 font-semibold tracking-tight text-sidebar-foreground no-underline hover:bg-sidebar-accent"
          data-testid="choy-shell-brand-rail"
          @click.prevent="onBrandClick"
        >
          <img :src="logoUrl" alt="" class="size-6 shrink-0" width="24" height="24" />
          <span class="truncate group-data-[collapsible=icon]:hidden">Choysum</span>
        </a>
      </SidebarHeader>
      <SidebarContent>
        <slot name="aside">
          <ChoySidebarNav />
        </slot>
      </SidebarContent>
      <SidebarFooter class="border-t border-sidebar-border">
        <div
          class="choy-shell__attrib px-2 py-2 text-xs text-sidebar-foreground/70"
          data-testid="choy-shell-attrib"
        >
          <p class="truncate leading-snug group-data-[collapsible=icon]:hidden">
            {{ copyrightLine }}
          </p>
          <p class="truncate leading-snug group-data-[collapsible=icon]:hidden">
            {{ poweredLine }}
          </p>
          <p
            class="truncate leading-snug group-data-[collapsible=icon]:text-center"
            :title="versionLine"
          >
            <span class="group-data-[collapsible=icon]:hidden">{{ versionLine }}</span>
            <span class="hidden font-medium tabular-nums group-data-[collapsible=icon]:inline">
              {{ versionShort }}
            </span>
          </p>
        </div>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>

    <SidebarInset>
      <header
        v-if="effectiveShowHeader"
        class="flex shrink-0 items-center gap-2 border-b border-border px-3 text-sm"
        :style="{ height: 'var(--choy-layout-header-height)' }"
        data-testid="choy-shell-header"
      >
        <SidebarTrigger
          data-testid="choy-shell-menu-trigger"
          :aria-label="menuTriggerLabel"
        />
        <ChoyShellBreadcrumb />
        <div class="ms-auto flex items-center gap-1">
          <ChoyCommandPalette />
          <div data-anchor="choy.shell.header-actions" class="flex items-center gap-1">
            <slot name="header-actions" />
          </div>
        </div>
      </header>
      <div
        class="choy-shell__main-inner min-h-0 flex-1 overflow-auto bg-muted/30"
        data-testid="choy-shell-canvas"
      >
        <slot>
          <router-view v-slot="{ Component, route: viewRoute }">
            <KeepAlive>
              <component
                :is="Component"
                v-if="Component && viewRoute.meta?.keepAlive"
                :key="viewRoute.path"
              />
            </KeepAlive>
            <component
              :is="Component"
              v-if="Component && !viewRoute.meta?.keepAlive"
              :key="viewRoute.fullPath"
            />
          </router-view>
        </slot>
      </div>
      <footer
        v-if="showFooter && $slots.footer"
        class="shrink-0 border-t border-border px-4 py-2 text-xs text-foreground/60"
      >
        <slot name="footer" />
      </footer>
    </SidebarInset>
  </SidebarProvider>
</template>

<script setup lang="ts">
import { KeepAlive, computed, type ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import logoUrl from '../../assets/logo-32.png'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import ChoyCommandPalette from './ChoyCommandPalette.vue'
import ChoyShellBreadcrumb from './ChoyShellBreadcrumb.vue'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath'
import { shellMenuTriggerLabel, shortAppVersion } from './choyWebShellChrome'

/**
 * Dense Admin product shell: Sidebar* chrome + Canvas with KeepAlive-aware router-view.
 * Auth pages suppress the rail and top bar so Auth is a fullscreen canvas.
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
    showFooter: false,
  },
)

let isAuthPage: ComputedRef<boolean> = computed(() => false)
let onBrandClick = () => {}
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key

try {
  const route = useRoute()
  if (route) {
    isAuthPage = computed(() => !!route.meta?.isAuthPage)
  }
} catch {
  // Unit mounts may omit vue-router.
}

try {
  const router = useRouter()
  if (router) {
    onBrandClick = () => {
      void router.push(resolveRuntimeDefaultLandPath())
    }
  }
} catch {
  onBrandClick = () => {}
}

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

const effectiveShowHeader = computed(() => props.showHeader && !isAuthPage.value)
const sidebarAllowed = computed(() => props.showSidebar && !isAuthPage.value)

const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: layoutStore?.sidebarMode === 'collapsed',
    t: (key) => tLayout(key),
  }),
)

const appVersion = computed(
  () => String((import.meta as ImportMeta).env?.CHOYSUM_APP_VERSION || '').trim() || 'dev',
)
const versionShort = computed(() => shortAppVersion(appVersion.value))
const year = computed(() => new Date().getFullYear())
const copyrightLine = computed(() =>
  tLayout('layout.footer.copyright', { year: year.value }),
)
const poweredLine = computed(() => tLayout('layout.footer.powered'))
const versionLine = computed(() =>
  tLayout('layout.footer.version', { version: appVersion.value }),
)
</script>
