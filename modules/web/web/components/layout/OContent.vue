<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-main :class="contentClass" :style="rtlCssVariables" role="main" data-print="content">
    <slot></slot>
  </el-main>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useCssVar } from '@vueuse/core';
import { useLayoutStore, useI18nStore } from '@/web/web/stores';
import { ElMain } from 'element-plus';

// Content padding constants.
const CONTENT_PADDING = 16;
const CONTENT_PADDING_MOBILE = 8;

// Sync JS constants to global CSS variables.
onMounted(() => {
  useCssVar('--o-content-padding').value = `${CONTENT_PADDING}px`;
  useCssVar('--o-content-padding-mobile').value = `${CONTENT_PADDING_MOBILE}px`;
});

// Define content padding and transition types.
type ContentPaddingSize = 'none' | 'small' | 'medium' | 'large';
type TransitionType = 'fade' | 'slide';

const props = defineProps({
  /**
   * Whether to add padding.
   */
  padding: {
    type: Boolean,
    default: true,
  },

  /**
   * Padding size.
   * @values none, small, medium, large
   */
  paddingSize: {
    type: String as () => ContentPaddingSize,
    default: 'medium',
    validator: (value: string) => ['none', 'small', 'medium', 'large'].includes(value),
  },

  /**
   * Whether to show a border.
   */
  bordered: {
    type: Boolean,
    default: false,
  },

  /**
   * Whether to show a background.
   */
  background: {
    type: Boolean,
    default: false,
  },

  /**
   * Whether to enable page transitions.
   */
  transition: {
    type: Boolean,
    default: true,
  },

  /**
   * Transition style.
   * @values fade, slide
   */
  transitionType: {
    type: String as () => TransitionType,
    default: 'fade',
    validator: (value: string) => ['fade', 'slide'].includes(value),
  },
});

// Layout store.
const layoutStore = useLayoutStore();

// Use the i18n store to detect text direction.
const i18nStore = useI18nStore();
const isRtlMode = computed(() => i18nStore.currentLocale.textDirection === 'rtl');

// Compute RTL CSS variables so slide transitions move in the right direction.
const rtlCssVariables = computed(() => ({
  '--o-slide-enter-translate': isRtlMode.value ? '-20px' : '20px',
  '--o-slide-leave-translate': isRtlMode.value ? '20px' : '-20px',
}));

const contentWidthClass = computed(() => {
  switch (layoutStore.sidebarMode) {
    case 'expanded':
      return 'o-content--sidebar-expanded';
    case 'collapsed':
      return 'o-content--sidebar-collapsed';
    case 'hidden':
      return 'o-content--sidebar-hidden';
    default:
      return '';
  }
});

// Compute content area classes using the Element Plus naming style.
const contentClass = computed(() => {
  return [
    'o-content',
    contentWidthClass,
    props.background ? 'o-content--with-background' : '',
    props.bordered ? 'o-content--bordered' : '',
    props.padding ? `o-content--padding-${props.paddingSize}` : 'o-content--padding-none',
  ]
    .filter(Boolean)
    .join(' ');
});

// Keep the transition name available for router-view integration.
const transitionName = computed(() => (props.transition ? props.transitionType : ''));

// Compute the slide direction with RTL awareness.
const slideDirection = computed(() => (isRtlMode.value ? 'rtl' : 'ltr'));
</script>

<style lang="scss" scoped>
.o-content {
  width: 100%;
  transition: padding var(--el-transition-duration) var(--el-transition-function);
  display: flex;
  flex-direction: column;
  height: auto;
  flex: 1;
  overflow: visible;

  &.el-main {
    padding: 0;
    overflow: visible;
  }

  &--with-background {
    background-color: var(--el-bg-color);
  }

  &--bordered {
    border: 1px solid var(--el-border-color-light);
  }

  &--padding-none {
    padding: 0;
  }

  &--padding-small {
    padding: 8px;
  }

  &--padding-medium {
    padding: var(--o-content-padding);

    @media only screen and (max-width: 991px) {
      padding: var(--o-content-padding-mobile);
    }
  }

  &--padding-large {
    padding: 24px;

    @media only screen and (max-width: 991px) {
      padding: 16px;
    }
  }
}
</style>
