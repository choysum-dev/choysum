<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div :class="layoutClass">
    <div class="o-layout__container">
      <!-- Header section. -->
      <OHeader v-if="showHeader" :fixed="fixedHeader" />
      <OSidebar v-if="showSidebar" />

      <!-- Main content area. -->
      <el-container class="o-layout__main-container">
        <div class="o-layout__content-wrapper">
          <OContent v-bind="contentSpacing">
            <router-view v-slot="{ Component }">
              <transition :name="getTransitionName">
                <keep-alive v-if="$route.meta.keepAlive">
                  <component :is="Component" />
                </keep-alive>
                <component v-else :is="Component" />
              </transition>
            </router-view>
          </OContent>
          <!-- Footer. -->
          <OFooter v-if="showFooter" />
        </div>
      </el-container>
    </div>
  </div>
</template>

<script setup lang="ts">
// Copyright 2025 The Choysum Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { computed } from 'vue';
import { useLayoutStore } from '@/web/web/stores';
import OHeader from './OHeader.vue';
import OSidebar from './OSidebar.vue';
import OContent from './OContent.vue';
import OFooter from './OFooter.vue';
import { ElContainer } from 'element-plus';

const props = defineProps({
  /**
   * Whether to show the sidebar.
   */
  showSidebar: {
    type: Boolean,
    default: true,
  },

  /**
   * Whether to show the top navigation bar.
   */
  showHeader: {
    type: Boolean,
    default: true,
  },

  /**
   * Whether to keep the header fixed.
   */
  fixedHeader: {
    type: Boolean,
    default: true,
  },

  /**
   * Whether to show the footer.
   */
  showFooter: {
    type: Boolean,
    default: false,
  },

  /**
   * Layout spacing applied to both inner and outer page margins.
   * @values none, small, medium, large
   */
  spacing: {
    type: String,
    default: 'medium',
    validator: (value: string) => ['none', 'small', 'medium', 'large'].includes(value),
  },

  /**
   * Route transition effect.
   * @values fade, slide, zoom, none
   */
  transition: {
    type: String,
    default: 'fade',
    validator: (value: string) => ['fade', 'slide', 'zoom', 'none'].includes(value),
  },
});

// Layout state.
const layoutStore = useLayoutStore();

// Compute layout classes.
const layoutClass = computed(() => {
  let sidebarClass = '';
  if (props.showSidebar) {
    if (layoutStore.sidebarMode === 'collapsed' || layoutStore.sidebarMode === 'hover') {
      sidebarClass = 'o-layout--with-sidebar--collapsed';
    } else if (layoutStore.sidebarMode === 'expanded') {
      sidebarClass = 'o-layout--with-sidebar--expanded';
    }
  } else {
    sidebarClass = 'o-layout--without-sidebar';
  }

  return [
    'o-layout',
    sidebarClass,
    props.showHeader ? 'o-layout--with-header' : 'o-layout--without-header',
    props.showFooter ? 'o-layout--with-footer' : 'o-layout--without-footer',
    `o-layout--spacing-${props.spacing}`,
    props.showHeader && props.fixedHeader ? 'o-layout--fixed-header' : '',
  ]
    .filter(Boolean)
    .join(' ');
});

/**
 * Transition name.
 */
const getTransitionName = computed(() => {
  return props.transition === 'none' ? '' : props.transition;
});

// Map layout spacing to OContent padding props so spacing is controlled
// in a single place instead of via :deep() overrides.
const contentSpacing = computed(() => {
  if (props.spacing === 'none') return { padding: false };
  return { paddingSize: props.spacing as 'small' | 'medium' | 'large' };
});
</script>

<style lang="scss" scoped>
/* Layout component. */
.o-layout {
  width: 100%;

  /* Element Plus container overrides. */
  :deep(.el-container) {
    width: 100%;
    margin: 0;
    padding: 0;
  }

  /* Main content container with fixed layout behavior. */
  &__main-container {
    overflow: visible;
    min-width: 0; /* Prevent content overflow. */

    /* Start at the top when no header is shown. */
    .o-layout--without-header & {
      top: 0;
      height: 100vh;
    }

    /* Remove the left inset when no sidebar is shown. */
    .o-layout--without-sidebar & {
      padding-inline-start: 0 !important;
    }

    /* Apply the inset for the collapsed sidebar. */
    .o-layout--with-sidebar--collapsed & {
      padding-inline-start: var(--o-sidebar-collapsed-width) !important;
    }

    /* Apply the inset for the expanded sidebar. */
    .o-layout--with-sidebar--expanded & {
      padding-inline-start: var(--o-sidebar-width) !important;
    }

    /* Mobile adjustments. */
    @media only screen and (max-width: 991px) {
      top: var(--o-header-height-mobile, 50px);
    }

    /* Mobile adjustments. */
    @media only screen and (max-width: 768px) {
      padding-inline-start: 0 !important; /* Remove the inset. */
    }
  }

  /* Content wrapper. */
  &__content-wrapper {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: calc(100vh - var(--o-header-height, 60px));
    min-width: 0;
  }

  /* Layout variant without a top navigation bar. */
  &--without-header {
    :deep(.o-sidebar) {
      height: 100vh; /* Override the default OSidebar height. */
      top: 0;
    }

    .o-layout__main-container {
      height: 100vh;
      margin-block-start: 0;
    }
  }
}
</style>
