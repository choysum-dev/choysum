<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyLayout
    :show-header="effectiveShowHeader"
    :show-aside="railVisible"
    :show-footer="showFooter && !!$slots.footer"
    :aside-overlay="railIsDrawer"
    :aside-collapsed="railCollapsed"
    :aside-aria-label="tLayout('layout.sidebar.menu')"
    :class="isAuthPage ? 'min-h-screen choy-shell--auth' : 'min-h-screen'"
    @aside-dismiss="closeMobileRail"
  >
    <template v-if="effectiveShowHeader" #header>
      <div
        class="flex items-center gap-2 px-3 text-sm"
        :style="{ height: 'var(--choy-layout-header-height)' }"
      >
        <ChoyButton
          v-if="showMenuTrigger"
          variant="ghost"
          size="icon"
          type="button"
          :aria-label="menuTriggerLabel"
          :aria-expanded="railIsDrawer ? true : railVisible && !railCollapsed"
          data-testid="choy-shell-menu-trigger"
          @click="onMenuTriggerClick"
        >
          <Menu class="size-4" aria-hidden="true" />
        </ChoyButton>
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
    </template>
    <template v-if="railVisible" #aside>
      <div class="flex h-full flex-col">
        <div v-if="railIsDrawer" class="flex shrink-0 justify-end p-1">
          <ChoyButton
            variant="ghost"
            size="icon"
            type="button"
            :aria-label="tLayout('layout.sidebar.collapse')"
            data-testid="choy-shell-drawer-close"
            @click="closeMobileRail"
          >
            <X class="size-4" aria-hidden="true" />
          </ChoyButton>
        </div>
        <nav
          class="choy-shell__aside flex min-h-0 flex-1 flex-col p-2 text-sm text-foreground/80"
          :class="{ 'choy-shell__aside--collapsed': railCollapsed }"
          aria-label="Main"
          data-testid="choy-shell-aside"
        >
          <slot name="aside">
            <component :is="sidebarMenu" />
          </slot>
        </nav>
      </div>
    </template>
    <template v-if="railVisible" #aside-foot>
      <div
        class="choy-shell__attrib px-2 py-2 text-xs text-muted-foreground"
        data-testid="choy-shell-attrib"
      >
        <template v-if="railCollapsed">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger as-child>
                <button
                  type="button"
                  class="flex w-full items-center justify-center rounded-md px-1 py-1 hover:bg-muted"
                  :aria-label="attribTooltip"
                >
                  <span class="font-medium tabular-nums">{{ versionShort }}</span>
                </button>
              </TooltipTrigger>
              <TooltipContent side="right" class="max-w-xs text-xs">
                {{ attribTooltip }}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </template>
        <template v-else>
          <p class="m-0 truncate leading-snug">{{ copyrightLine }}</p>
          <p class="m-0 truncate leading-snug">{{ poweredLine }}</p>
          <p class="m-0 truncate leading-snug">{{ versionLine }}</p>
        </template>
      </div>
    </template>
    <slot>
      <div
        class="choy-shell__main-inner min-h-full"
        :class="isAuthPage ? 'bg-muted/50' : 'bg-muted/30'"
        data-testid="choy-shell-canvas"
      >
        <router-view v-slot="{ Component, route: viewRoute }">
          <!-- KeepAlive stays mounted so cached views survive non-keepAlive navigations. -->
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
      </div>
    </slot>
    <template v-if="showFooter && $slots.footer" #footer>
      <div class="px-4 py-2 text-xs text-foreground/60">
        <slot name="footer" />
      </div>
    </template>
  </ChoyLayout>
</template>

<script setup lang="ts">
import { KeepAlive, computed, h, onMounted, onUnmounted, watch, type ComputedRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { Menu, X } from 'lucide-vue-next';
import logoUrl from '../../assets/logo-32.png';
import ChoyLayout from './ChoyLayout.vue';
import ChoyButton from './ChoyButton.vue';
import Tooltip from '../vendor/ui/tooltip/Tooltip.vue';
import TooltipContent from '../vendor/ui/tooltip/TooltipContent.vue';
import TooltipProvider from '../vendor/ui/tooltip/TooltipProvider.vue';
import TooltipTrigger from '../vendor/ui/tooltip/TooltipTrigger.vue';
import { useMenu } from '../../composables/useMenu';
import { useLayoutStore } from '../../stores/layoutStore';
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath';
import {
  shellMenuTriggerLabel,
  shortAppVersion,
  shouldCloseDrawerOnEscape,
  setDrawerBodyOverflow,
} from './choyWebShellChrome';

/**
 * Dense Admin product shell: Top bar + Nav rail (drawer / collapsed / expanded)
 * + Canvas with KeepAlive-aware router-view. Rail foot holds open-source attribution.
 * Auth pages suppress the rail. Header actions anchor stays for auth xpath inject.
 */
const props = withDefaults(
  defineProps<{
    showHeader?: boolean;
    showSidebar?: boolean;
    showFooter?: boolean;
  }>(),
  {
    showHeader: true,
    showSidebar: true,
    showFooter: false,
  },
);

let isAuthPage: ComputedRef<boolean> = computed(() => false);
let onBrandClick = () => {};
let layoutStore: ReturnType<typeof useLayoutStore> | null = null;
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key;

try {
  const route = useRoute();
  if (route) {
    isAuthPage = computed(() => !!route.meta?.isAuthPage);
  }
} catch {
  // Unit mounts may omit vue-router.
}

try {
  const router = useRouter();
  if (router) {
    onBrandClick = () => {
      void router.push(resolveRuntimeDefaultLandPath());
    };
  }
} catch {
  onBrandClick = () => {};
}

try {
  layoutStore = useLayoutStore();
} catch {
  layoutStore = null;
}

try {
  const i18n = useI18n({ useScope: 'global' });
  tLayout = (key, values) => String(i18n.t(key, values as any));
} catch {
  tLayout = (key) => key;
}

const effectiveShowHeader = computed(() => props.showHeader);
const sidebarAllowed = computed(() => props.showSidebar && !isAuthPage.value);

const railMode = computed(() => layoutStore?.sidebarMode ?? 'expanded');
const railVisible = computed(() => {
  if (!sidebarAllowed.value) return false;
  if (railMode.value === 'hidden') return false;
  return true;
});
const railIsDrawer = computed(
  () => !!layoutStore?.isMobile && railMode.value === 'expanded' && sidebarAllowed.value,
);
const railCollapsed = computed(
  () => railMode.value === 'collapsed' && !railIsDrawer.value,
);

let sidebarMenu: ComputedRef<() => ReturnType<typeof h>> = computed(
  () => () => h('p', { class: 'choy-menu__empty' }, ''),
);

try {
  const { renderSidebarMenu } = useMenu();
  sidebarMenu = computed(() => () =>
    renderSidebarMenu({
      useDefaultIcon: true,
      collapsed: railCollapsed.value,
    }),
  );
} catch {
  // Unit mounts may omit pinia / menu store.
}

const showMenuTrigger = computed(() => sidebarAllowed.value && !!layoutStore);
const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: railCollapsed.value,
    t: (key) => tLayout(key),
  }),
);

function onMenuTriggerClick() {
  layoutStore?.toggleSidebar();
}

function closeMobileRail() {
  layoutStore?.closeSidebar();
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (shouldCloseDrawerOnEscape(event, railIsDrawer.value)) {
    closeMobileRail();
  }
}

watch(
  railIsDrawer,
  (open, wasOpen) => {
    setDrawerBodyOverflow(!!open);
    if (!open && wasOpen) {
      document.querySelector<HTMLElement>('[data-testid="choy-shell-menu-trigger"]')?.focus();
    }
  },
  { immediate: true, flush: 'post' },
);

onMounted(() => {
  document.addEventListener('keydown', onDocumentKeydown);
});
onUnmounted(() => {
  setDrawerBodyOverflow(false);
  document.removeEventListener('keydown', onDocumentKeydown);
});

const appVersion = computed(
  () => String((import.meta as ImportMeta).env?.CHOYSUM_APP_VERSION || '').trim() || 'dev',
);
const versionShort = computed(() => shortAppVersion(appVersion.value));
const year = computed(() => new Date().getFullYear());
const copyrightLine = computed(() =>
  tLayout('layout.footer.copyright', { year: year.value }),
);
const poweredLine = computed(() => tLayout('layout.footer.powered'));
const versionLine = computed(() =>
  tLayout('layout.footer.version', { version: appVersion.value }),
);
const attribTooltip = computed(
  () => `${copyrightLine.value} · ${poweredLine.value} · ${versionLine.value}`,
);
</script>
