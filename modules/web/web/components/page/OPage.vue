<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div :class="pageClass" role="region" data-print="page" :aria-busy="loading" :aria-labelledby="title && !$slots.header ? pageTitleId : undefined">
    <!-- Page header section. -->
    <div v-if="$slots.header || title || showBreadcrumb || $slots.breadcrumb" class="o-page__header">
      <slot name="header">
        <div v-if="showBreadcrumb || $slots.breadcrumb" class="o-page__breadcrumb">
          <slot name="breadcrumb">
            <OBreadcrumb />
          </slot>
        </div>
        <h1 v-if="title" class="o-page__title" :id="pageTitleId">{{ title }}</h1>
      </slot>
    </div>

    <!-- Page toolbar section. -->
    <div v-if="$slots.toolbar" class="o-page__toolbar" role="toolbar">
      <slot name="toolbar"></slot>
    </div>

    <!-- Page body section. -->
    <div class="o-page__body" :class="{ 'o-page__body--with-footer': $slots.footer }">
      <slot></slot>
    </div>

    <!-- Page footer section. -->
    <div v-if="$slots.footer" class="o-page__footer">
      <slot name="footer"></slot>
    </div>

    <!-- Loading overlay. -->
    <div v-if="loading" class="o-page__loading-mask" aria-hidden="true">
      <el-icon class="o-page__loading-icon is-loading" aria-label="页面加载中">
        <Loading />
      </el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue';
import { Loading } from '@element-plus/icons-vue';
import OBreadcrumb from '@/web/web/components/view/OBreadcrumb.vue';

type PageWidth = '' | 'narrow' | 'medium' | 'wide' | 'full';

const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  showBreadcrumb: {
    type: Boolean,
    default: true,
  },
  padding: {
    type: Boolean,
    default: true,
  },
  width: {
    type: String as () => PageWidth,
    default: '',
    validator: (value: string) => ['', 'narrow', 'medium', 'wide', 'full'].includes(value),
  },
  elevated: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
});

const pageTitleId = useId();

const pageClass = computed(() => {
  return [
    'o-page',
    props.padding === false ? 'o-page--without-padding' : 'o-page--with-padding',
    props.width ? `o-page--${props.width}` : '',
    props.elevated ? 'o-page--elevated' : '',
    props.loading ? 'o-page--loading' : '',
  ]
    .filter(Boolean)
    .join(' ');
});
</script>

<style lang="scss" scoped>
@use '../../styles/tokens.scss' as *;

.o-page {
  width: 100%;
  background-color: var(--el-bg-color);
  color: var(--el-text-color-primary);
  display: flex;
  flex-direction: column;
  position: relative;
  transition: all var(--el-transition-duration) var(--el-transition-function-ease-in-out);
  overflow: visible;

  &--with-padding {
    padding: var(--el-padding-medium, 16px);

    @media only screen and (max-width: 991px) {
      padding: var(--el-padding-small, 8px);
    }
  }

  &--without-padding {
    padding: 0;
  }

  &--narrow {
    max-width: 768px;
    margin-inline: auto;
  }

  &--medium {
    max-width: 1024px;
    margin-inline: auto;
  }

  &--wide {
    max-width: 1280px;
    margin-inline: auto;
  }

  &--full {
    max-width: 100%;
  }

  &--elevated {
    box-shadow: var(--el-box-shadow-light);
  }

  &--loading {
    pointer-events: none;
    opacity: 0.7;
  }

  &__header {
    margin-block-end: var(--el-margin-medium, 8px);
  }

  &__title {
    margin: 0;
    font-size: var(--el-font-size-extra-large, 20px);
    font-weight: var(--el-font-weight-bold, 500);
    color: var(--el-text-color-primary);
    line-height: 1.4;
  }

  &__toolbar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--el-gap-small, 8px);
    padding-block: var(--el-padding-small, 8px);
    margin-block-end: var(--el-margin-medium, 16px);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  &__body {
    flex: 1;
    width: 100%;
    min-width: 0;
    display: flex;
    flex-direction: column;

    &--with-footer {
      margin-block-end: var(--el-margin-medium, 16px);
    }
  }

  &__footer {
    padding-block-start: var(--el-padding-medium, 16px);
    border-top: 1px solid var(--el-border-color-light);
    margin-block-start: auto;
  }

  &__loading-mask {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--el-overlay-color-lighter, rgba(255, 255, 255, 0.5));
    z-index: $z-index-modal;
  }

  &__loading-icon {
    font-size: 24px;
    color: var(--el-color-primary);
    animation: rotating var(--el-transition-duration-fast) linear infinite;
  }

  @media only screen and (max-width: 991px) {
    &__header {
      margin-block-end: var(--el-margin-small, 8px);
    }

    &__title {
      font-size: var(--el-font-size-large, 18px);
    }

    &__toolbar {
      margin-block-end: var(--el-margin-small, 8px);
    }

    &__body--with-footer {
      margin-block-end: var(--el-margin-small, 8px);
    }

    &__footer {
      padding-block-start: var(--el-padding-small, 8px);
    }
  }
}
</style>
