<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-header :class="headerClass" role="banner">
    <div class="o-header__inner">
      <!-- Navigation section. -->
      <div class="o-header__nav">
        <!-- Menu toggle button area. -->
        <div class="o-header__menu-toggle-area">
          <el-button class="o-header__menu-toggle" text @click="handleMenuToggle" :aria-label="isSidebarCollapsed ? '展开菜单' : '折叠菜单'">
            <el-icon :size="24">
              <component :is="menuToggleIcon" />
            </el-icon>
          </el-button>
        </div>

        <!-- Logo section. -->
        <div class="o-header__logo">
          <router-link to="/" class="o-header__logo-link" :aria-label="`${appName} - 首页`">
            <img v-if="logoUrl" :src="logoUrl" :alt="appName" class="o-header__logo-img" />
          </router-link>
        </div>

        <!-- App selector using a dedicated popper class. -->
        <el-dropdown v-if="!isSidebarHidden" trigger="click" @command="handleAppChange" popper-class="o-header__app-selector" placement="bottom-start">
          <div class="o-header__current-app">
            <template v-if="menu.activeApp.value">
              <el-icon v-if="menu.activeApp.value.icon" class="o-header__app-icon">
                <component :is="menu.activeApp.value.icon" />
              </el-icon>
              <span class="o-header__app-name">{{ menu.activeApp.value.title }}</span>
            </template>
            <span v-else class="o-header__app-name">选择应用</span>
            <el-icon class="o-header__dropdown-arrow" size="1.5em">
              <component :is="isMobile ? '' : ArrowDropDownOutlined" />
            </el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="app in menu.getMenus()" :key="app.id" :command="app.id" :class="{ 'is-active': isActiveApp(app.id) }">
                <div class="o-header__app-option">
                  <el-icon v-if="app.icon" class="o-header__app-option-icon">
                    <component :is="app.icon" />
                  </el-icon>
                  <span>{{ app.title }}</span>
                </div>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <!-- Center section with the app selector and search box. -->
      <div class="o-header__center">
        <el-input
          v-model="searchValue"
          placeholder="搜索..."
          class="o-header__search"
          :prefix-icon="Search"
          @keyup.enter="handleSearch(searchValue)"
          clearable
        />
      </div>

      <!-- Action area split into secondary and primary sections. -->
      <div class="o-header__actions">
        <!-- Secondary actions. -->
        <div class="o-header__actions-secondary">
          <!-- Language switch dropdown. -->
          <el-dropdown
            trigger="click"
            @command="handleLanguageChange"
            class="o-header__action-item"
            placement="bottom-end"
            :max-height="400"
            popper-class="o-header__language-dropdown"
          >
            <el-button text class="o-header__action-btn" aria-label="切换语言">
              <el-icon :size="20">
                <component :is="TranslateOutlined" />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="locale in localeOptions"
                  :key="locale.code"
                  :command="locale.code"
                  :class="{ 'is-active': currentLocaleCode === locale.code }"
                >
                  <div class="locale-item">
                    <span :dir="locale.textDirection">{{ locale.name }}</span>
                    <span v-if="locale.textDirection === 'rtl'" class="rtl-indicator">(RTL)</span>
                  </div>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <!-- Support dropdown using a dedicated popper class. -->
          <el-dropdown trigger="click" class="o-header__action-item" placement="bottom-end" popper-class="o-header__support-dropdown">
            <el-button text class="o-header__action-btn" aria-label="获取帮助">
              <el-icon :size="20">
                <QuestionFilled />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>
                  <span>帮助文档</span>
                </el-dropdown-item>
                <el-dropdown-item>
                  <span>视频教程</span>
                </el-dropdown-item>
                <el-dropdown-item>
                  <span>联系支持</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <!-- Primary actions. -->
        <div class="o-header__actions-primary"></div>
      </div>
    </div>

    <!-- App menu drawer used when the sidebar is hidden. -->
    <el-drawer
      v-model="appDrawerVisible"
      title="应用导航"
      :direction="drawerDirection"
      :size="280"
      :with-header="true"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
      class="o-app-drawer"
    >
      <el-scrollbar>
        <component
          :is="
            renderAppDrawerMenu({
              onItemClick: handleDrawerMenuItemClick,
              className: 'o-app-menu',
            })
          "
        />
      </el-scrollbar>
    </el-drawer>
  </el-header>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useCssVar, useTextDirection } from '@vueuse/core';
import { useLayoutStore } from '@/web/web/stores';
import { useI18nStore, SUPPORTED_LOCALES } from '@/web/web/stores/i18nStore';
import { useMenu } from '@/web/web/composables';
import type { MenuItem } from '@/core/web/menu';
import { ElButton, ElDrawer, ElIcon, ElHeader, ElDropdown, ElDropdownMenu, ElDropdownItem, ElInput, ElScrollbar } from 'element-plus';
import { Search, QuestionFilled } from '@element-plus/icons-vue';
import { MenuOutlined, AppsOutlined, TranslateOutlined, ArrowDropDownOutlined } from '@vicons/material';
import defaultLogo from '@/web/web/assets/logo.svg';

// Fixed height constants.
const HEADER_HEIGHT = 48;
const HEADER_HEIGHT_MOBILE = 40;

// Set CSS variables when the component mounts.
onMounted(() => {
  useCssVar('--o-header-height').value = `${HEADER_HEIGHT}px`;
  useCssVar('--o-header-height-mobile').value = `${HEADER_HEIGHT_MOBILE}px`;
});

const props = defineProps({
  appName: {
    type: String,
    default: 'Choysum',
  },
  logoUrl: {
    type: String,
    default: defaultLogo,
  },
  fixed: {
    type: Boolean,
    default: false,
  },
});

// State stores.
const layoutStore = useLayoutStore();
const i18nStore = useI18nStore();

// Menu system.
const menu = useMenu();

// Drawer state.
const appDrawerVisible = ref(false);
const searchValue = ref('');

// Menu renderer.
const { renderAppDrawerMenu } = useMenu();

// Derived state.
const isSidebarHidden = computed(() => layoutStore.sidebarMode === 'hidden');
const isSidebarCollapsed = computed(() => layoutStore.sidebarMode === 'collapsed' || layoutStore.sidebarMode === 'hover');
const isMobile = computed(() => layoutStore.isMobile);

// Text direction.
const direction = useTextDirection();

// Compute the drawer direction from the text direction.
const drawerDirection = computed(() => (direction.value === 'rtl' ? 'rtl' : 'ltr'));

// Supported locale list.
const localeOptions = computed(() =>
  i18nStore.supportedLocales.map(code => {
    const locale = SUPPORTED_LOCALES[code];
    return {
      code,
      name: locale.name,
      textDirection: locale.textDirection,
    };
  })
);

// Currently selected locale code.
const currentLocaleCode = computed(() => i18nStore.currentLocale.code);

// Header class computation.
const headerClass = computed(() => (props.fixed ? 'o-header o-header--fixed' : 'o-header'));

// Menu toggle icon.
const menuToggleIcon = computed(() => (isSidebarHidden.value ? AppsOutlined : MenuOutlined));

// Helper for active-app checks.
const isActiveApp = computed(() => (appId?: string) => menu.activeApp.value?.id === appId);

function handleMenuToggle() {
  if (isSidebarHidden.value) {
    appDrawerVisible.value = true;
  } else {
    layoutStore.toggleSidebar();
  }
}

function handleAppChange(appId: string) {
  menu.navigateTo(appId);
}

async function handleLanguageChange(locale: string) {
  await i18nStore.setLocale(locale);
}

function handleSearch(value: string) {
  //
}

function handleDrawerMenuItemClick(item: MenuItem) {
  menu.navigateTo(item);
  appDrawerVisible.value = false;
}
</script>

<style lang="scss" scoped>
@use '@/web/web/styles/tokens.scss' as *;

.o-header {
  height: var(--o-header-height);
  width: 100%;
  background-color: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-lighter);
  padding: 0;
  z-index: $z-index-fixed;

  &.el-header {
    height: var(--o-header-height) !important;
    padding: 0;

    @media only screen and (max-width: 991px) {
      height: var(--o-header-height-mobile) !important;
    }
  }

  &__inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
    width: 100%;
  }

  &__nav {
    display: flex;
    align-items: center;
    height: 100%;
    width: auto;
    min-width: var(--o-sidebar-width);
    padding-inline: 0px;
    box-sizing: border-box;
    transition: width var(--el-transition-duration) var(--el-transition-function-ease-in-out);
    flex: 0 0 auto;
    gap: 4px;
  }

  &__menu-toggle-area {
    display: flex;
    align-items: center;
    justify-content: center;
    width: var(--o-sidebar-collapsed-width);
    height: 100%;
    padding: 0 4px;
  }

  &__menu-toggle {
    height: 36px;
    width: 36px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    border-radius: var(--el-border-radius-base);

    &:hover {
      color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
    }
  }

  &__logo {
    display: flex;
    align-items: center;
    height: 100%;
    margin-inline-end: 0;
    transition: opacity var(--el-transition-duration) var(--el-transition-function-ease-in-out);

    &-link {
      display: flex;
      align-items: center;
      height: 100%;
    }

    &-img {
      height: 32px;
      width: auto;
    }

    &-text {
      font-size: var(--el-font-size-base);
      font-weight: var(--el-font-weight-bold);
      color: var(--el-text-color-primary);
      white-space: nowrap;
    }
  }

  &__center {
    display: flex;
    align-items: center;
    height: 100%;
    flex: 1 1 auto;
    padding-inline: var(--el-padding-medium, 16px);
    justify-content: flex-start;
    overflow: hidden;
    gap: 16px;
  }

  &__app-selector {
    flex-shrink: 0;
  }

  &__current-app {
    display: flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    padding: 0 10px;
    height: 36px;
    border-radius: var(--el-border-radius-base);
    transition: background-color var(--el-transition-duration-fast);

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }

  &__app-icon {
    font-size: 18px;
    color: var(--el-color-primary);
  }

  &__app-name {
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-primary);
    white-space: nowrap;
  }

  &__dropdown-arrow {
    color: var(--el-text-color-secondary);
  }

  &__app-option {
    display: flex;
    align-items: center;
    gap: 8px;

    /* Keep the icon color inherited so inactive items do not always look primary. */
    &-icon {
      color: inherit;
    }
  }

  &__search {
    max-width: 400px;
    width: 100%;

    &:deep(.el-input__wrapper) {
      background-color: var(--el-fill-color-light);
      border-radius: 20px;
    }
  }

  &__actions {
    display: flex;
    align-items: center;
    height: 100%;
    padding-inline: var(--el-padding-medium, 16px);
    gap: 8px;
  }

  &__actions-secondary {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__actions-primary {
    display: flex;
    align-items: center;
    margin-inline-start: 4px;
  }

  &__action-item {
    display: flex;
    align-items: center;
  }

  &__action-btn {
    height: 36px;
    width: 36px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    border-radius: var(--el-border-radius-base);
    transition: all var(--el-transition-duration-fast);

    &:hover {
      color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
    }
  }

  .locale-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .rtl-indicator {
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
    margin-inline-start: 8px;
    flex-shrink: 0;
  }

  &--fixed {
    position: sticky;
    top: 0;
    left: 0;
    inset-inline-start: 0;
    inset-inline-end: 0;
  }

  @media only screen and (max-width: 991px) {
    height: var(--o-header-height-mobile);

    &__nav,
    &__center,
    &__actions {
      padding-inline: var(--el-padding-small, 8px);
    }

    &__nav {
      width: auto;
      min-width: 0;
      border-inline-end: none;
    }

    &__logo-text,
    &__app-name {
      display: none;
    }
  }

  @media only screen and (max-width: 767px) {
    &__center {
      max-width: 160px;
    }
  }

  @media only screen and (max-width: 480px) {
    &__search {
      display: none;
    }
  }

  /* App selector dropdown panel targeted through popper-class. */
  :global(.o-header__app-selector .el-dropdown-menu__item) {
    color: var(--el-text-color-regular);
  }
  :global(.o-header__app-selector .el-dropdown-menu__item:hover) {
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }
  :global(.o-header__app-selector .el-dropdown-menu__item.is-active) {
    color: var(--el-color-primary);
    font-weight: 500;
    background-color: var(--el-color-primary-light-9);
  }
  /* Let icons inherit text color to avoid unrelated global overrides. */
  :global(.o-header__app-selector .o-header__app-option-icon) {
    color: inherit;
  }

  /* Language dropdown panel. */
  :global(.o-header__language-dropdown .el-dropdown-menu__item.is-active) {
    color: var(--el-color-primary);
    font-weight: 500;
    background-color: var(--el-color-primary-light-9);
  }

  /* Support dropdown panel. */
  :global(.o-header__support-dropdown) {
    /* Custom support menu styles. */
  }
}
</style>
