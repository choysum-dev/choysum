<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest: viewport-locked column, no nav rail. -->
  <div
    class="choy-shell choy-shell--guest flex h-full min-h-0 w-full flex-col overflow-hidden bg-background text-foreground"
    data-testid="choy-shell"
    data-shell-mode="guest"
  >
    <ChoyGuestHeader v-if="effectiveShowHeader" :go-home="onBrandClick" :home-href="homeHref">
      <template #header-actions>
        <div
          data-anchor="choy.shell.header-actions"
          class="flex items-center gap-2"
        >
          <slot name="header-actions" />
        </div>
      </template>
    </ChoyGuestHeader>
    <!--
      Sticky footer column: short pages pin Powered-by to the viewport bottom;
      tall pages place it after content inside the same scrollport.
    -->
    <div
      class="choy-shell__scroll min-h-0 flex-1 overflow-y-auto bg-background"
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
          class="choy-shell__footer shrink-0 bg-background"
          data-testid="choy-shell-footer"
        >
          <div
            class="choy-inset-pad-x mx-auto flex w-full max-w-screen-2xl items-center justify-center"
            :style="{ minHeight: 'var(--choy-layout-footer-height)' }"
          >
            <ChoyAppFooter />
            <slot name="footer" />
          </div>
        </footer>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useRouter } from 'vue-router'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyGuestHeader from './ChoyGuestHeader.vue'
import { createGuestHomeNavigate } from './guestHomeNavigate'

/** Guest "Home" is the site root, not the authenticated default land path. */
const GUEST_HOME_HREF = '/'

/**
 * Guest shell: marketing-style header + main canvas + shared footer, without the app nav rail.
 */
const props = withDefaults(
  defineProps<{
    showHeader?: boolean
    showFooter?: boolean
  }>(),
  {
    showHeader: true,
    showFooter: true,
  },
)

const homeHref = GUEST_HOME_HREF
const onBrandClick = createGuestHomeNavigate(
  () => GUEST_HOME_HREF,
  () => useRouter(),
)

const effectiveShowHeader = computed(() => props.showHeader)
</script>
