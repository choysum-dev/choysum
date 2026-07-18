<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-footer v-if="showFooter" :class="footerClass" role="contentinfo" height="auto">
    <div class="o-footer__content">
      <!-- Left section. -->
      <div class="o-footer__left">
        <div class="o-footer__copyright">{{ formattedCopyright }}</div>
      </div>
      <!-- Center section. -->
      <div class="o-footer__center">
        <!-- Extension point for version info or custom content. -->
      </div>
      <!-- Right section. -->
      <div class="o-footer__right">
        <a href="#" class="o-footer__link" :aria-label="_t('View privacy policy')">{{ _t('Privacy policy') }}</a>
        <a href="#" class="o-footer__link" :aria-label="_t('View terms of use')">{{ _t('Terms of use') }}</a>
        <a href="#" class="o-footer__link" :aria-label="_t('Contact us')">{{ _t('Contact us') }}</a>
      </div>
    </div>
  </el-footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useLayoutStore } from '@/web/web/stores';
import { ElFooter } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/layout/OFooter' });

const props = defineProps({
  copyright: {
    type: String,
    default: '© {year} Choysum. All rights reserved.',
  },
  year: {
    type: [String, Number],
    default: () => new Date().getFullYear(),
  },
  divider: {
    type: Boolean,
    default: false,
  },
  hideOnMobile: {
    type: Boolean,
    default: false,
  },
  show: {
    type: Boolean,
    default: true,
  },
});

const layoutStore = useLayoutStore();

const footerClass = computed(() => {
  return ['o-footer', props.divider ? 'o-footer--divider' : '', props.hideOnMobile && layoutStore.isMobile ? 'o-footer--hidden' : ''].filter(Boolean).join(' ');
});

const formattedCopyright = computed(() => {
  return props.copyright.replace(/\{year\}/g, props.year.toString());
});

const showFooter = computed(() => props.show && !(props.hideOnMobile && layoutStore.isMobile));
</script>

<style lang="scss" scoped>
.o-footer {
  width: 100%;
  padding: 0;
  background-color: transparent;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-small);
  margin-top: auto;

  &.el-footer {
    background-color: transparent;
    border-top: none;
    height: auto;
    padding: 0;
  }

  &--divider {
    border-top: 1px solid var(--el-border-color-light);
  }

  &--hidden {
    display: none;
  }

  &__content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--el-gap-base, 16px);
    margin: 0 auto;
    padding: var(--el-padding-base, 16px);
  }

  &__left {
    flex: 1;
    min-width: 200px;
    text-align: start;
  }

  &__center {
    flex: 0 1 auto;
    text-align: center;
  }

  &__right {
    flex: 1;
    min-width: 200px;
    display: flex;
    gap: var(--el-gap-base, 16px);
    justify-content: flex-end;
    text-align: end;
  }

  &__copyright {
    opacity: 0.9;
  }

  &__link {
    color: var(--el-text-color-secondary);
    text-decoration: none;
    transition: color var(--el-transition-duration) var(--el-transition-function-ease-in-out);

    &:hover {
      color: var(--el-color-primary);
    }

    &:focus-visible {
      outline: var(--o-aria-focus-outline-width) solid var(--o-aria-focus-outline-color);
      outline-offset: var(--o-aria-focus-outline-offset);
    }

    &:not(:last-child) {
      margin-inline-end: var(--el-gap-base, 16px);
    }
  }

  @media only screen and (max-width: 991px) {
    &__content {
      flex-direction: column;
      gap: 8px;
      padding: var(--el-padding-small, 8px);
    }
    &__left,
    &__center,
    &__right {
      width: 100%;
      text-align: center;
    }
    &__right {
      justify-content: center;
    }
  }
}
</style>
