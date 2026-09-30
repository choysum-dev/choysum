<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyLayout
    :show-header="effectiveShowHeader"
    :show-aside="effectiveShowSidebar"
    :show-footer="showFooter && !!$slots.footer"
    :class="isAuthPage ? 'min-h-screen choy-shell--auth' : 'min-h-screen'"
  >
    <template v-if="effectiveShowHeader" #header>
      <div
        class="flex items-center gap-3 px-4 text-sm"
        :style="{ height: 'var(--choy-layout-header-height)' }"
      >
        <a
          href="/web/home"
          class="choy-shell__brand inline-flex items-center gap-2 font-semibold tracking-tight text-foreground no-underline hover:opacity-90"
          @click.prevent="onBrandClick"
        >
          <span
            class="inline-flex size-6 items-center justify-center rounded-md bg-primary text-[11px] font-bold text-background"
            aria-hidden="true"
          >C</span>
          <span>Choysum</span>
        </a>
        <div data-anchor="choy.shell.header-actions" class="ms-auto flex items-center gap-1">
          <slot name="header-actions" />
        </div>
      </div>
    </template>
    <template v-if="effectiveShowSidebar" #aside>
      <nav
        class="choy-shell__aside flex h-full flex-col overflow-auto bg-muted/40 p-2 text-sm text-foreground/80"
        aria-label="Main"
      >
        <slot name="aside">
          <component :is="sidebarMenu" />
        </slot>
      </nav>
    </template>
    <slot>
      <div
        class="choy-shell__main-inner min-h-full"
        :class="isAuthPage ? 'bg-muted/50' : 'bg-muted/30'"
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
import { KeepAlive, computed, h, type ComputedRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import ChoyLayout from './ChoyLayout.vue';
import { useMenu } from '../../composables/useMenu';

/**
 * Web product shell around ChoyLayout + router-view: brand bar,
 * built-in sidebar menu, header-actions anchor, KeepAlive-aware host.
 * Auth pages (meta.isAuthPage) suppress the sidebar.
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
let sidebarMenu: ComputedRef<() => ReturnType<typeof h>> = computed(
  () => () => h('p', { class: 'choy-menu__empty' }, ''),
);

try {
  const route = useRoute();
  // useRoute() returns undefined (does not throw) when no router is installed.
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
      void router.push('/home');
    };
  }
} catch {
  onBrandClick = () => {};
}

try {
  const { renderSidebarMenu } = useMenu();
  sidebarMenu = computed(() => () => renderSidebarMenu({ useDefaultIcon: true }));
} catch {
  // Unit mounts may omit pinia / menu store.
}

const effectiveShowHeader = computed(() => props.showHeader);
const effectiveShowSidebar = computed(() => props.showSidebar && !isAuthPage.value);
</script>
