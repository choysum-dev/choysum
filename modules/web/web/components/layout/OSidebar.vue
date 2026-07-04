<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-aside :class="sidebarClass" ref="sidebarRef" role="navigation" aria-label="主导航">
    <div class="o-sidebar__inner" @mouseleave="handleMenuMouseLeave">
      <!-- Top section. -->
      <div class="o-sidebar__header"></div>

      <!-- Menu section. -->
      <div class="o-sidebar__menu" @mouseenter="handleMenuMouseEnter">
        <el-scrollbar>
          <!-- Render the menu through the composable. -->
          <component
            :is="
              renderSidebarMenu({
                onItemClick: handleMenuSelect,
                onSubMenuOpen: handleSubMenuOpen,
                onSubMenuClose: handleSubMenuClose,
                useDefaultIcon: true,
                defaultIcon: BookmarkBorderOutlined,
                uniqueOpened: false,
              })
            "
          />
        </el-scrollbar>
      </div>

      <!-- Bottom section. -->
      <div class="o-sidebar__footer"></div>
    </div>
  </el-aside>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue';
import { useTimeoutFn, type UseTimeoutFnReturn } from '@vueuse/core';
import { useLayoutStore } from '@/web/web/stores';
import { useMenu } from '@/web/web/composables';
import type { MenuItem } from '@/core/web/menu';
import { ElAside, ElScrollbar } from 'element-plus';
import { BookmarkBorderOutlined } from '@vicons/material';

// State store.
const layoutStore = useLayoutStore();

// Menu system.
const menu = useMenu();

// Menu renderer.
const { renderSidebarMenu } = useMenu();

// Sidebar DOM reference.
const sidebarRef = ref<InstanceType<typeof ElAside> | null>(null);

// Attach the global mousemove listener on mount.
onMounted(() => {
  document.addEventListener('mousemove', handleGlobalMouseMove);
});

// Clean up event listeners.
onBeforeUnmount(() => {
  document.removeEventListener('mousemove', handleGlobalMouseMove);
  if (mouseMoveDebounceTimer) {
    clearTimeout(mouseMoveDebounceTimer);
  }
  if (leaveTimeout) {
    leaveTimeout.stop();
    leaveTimeout = null;
  }
  if (enterTimeout) {
    enterTimeout.stop();
    enterTimeout = null;
  }
});

const props = defineProps({
  showScrollbar: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits<{
  (e: 'select', menuItem: MenuItem): void;
  (e: 'open', index: string): void;
  (e: 'close', index: string): void;
}>();

const sidebarClass = computed(() =>
  ['o-sidebar', `o-sidebar--${layoutStore.sidebarMode}`, props.showScrollbar ? '' : 'o-sidebar--no-scrollbar'].filter(Boolean).join(' ')
);

function handleMenuSelect(menuItem: MenuItem) {
  menu.navigateTo(menuItem);
  emit('select', menuItem);
  if (layoutStore.isMobile) {
    layoutStore.closeSidebar();
  }
}

function handleSubMenuOpen(index: string) {
  emit('open', index);
}

function handleSubMenuClose(index: string) {
  emit('close', index);
}

// Timer references.
let leaveTimeout: UseTimeoutFnReturn<any> | null = null;
let enterTimeout: UseTimeoutFnReturn<any> | null = null;

// Check whether the mouse is currently inside the sidebar region.
function isMouseInSidebar(x: number, y: number): boolean {
  if (!sidebarRef.value) return false;

  // Get the actual DOM element from the ElAside component.
  const sidebarElement = sidebarRef.value.$el as HTMLElement;
  if (!sidebarElement || typeof sidebarElement.getBoundingClientRect !== 'function') {
    return false;
  }

  const rect = sidebarElement.getBoundingClientRect();
  return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom;
}

// Debounce state.
let mouseMoveDebounceTimer: number | null = null;

// Debounced global mousemove handler.
function handleGlobalMouseMove(event: MouseEvent) {
  // Only check while the sidebar is in hover mode.
  if (layoutStore.sidebarMode !== 'hover') return;

  // Clear the previous debounce timer.
  if (mouseMoveDebounceTimer) {
    clearTimeout(mouseMoveDebounceTimer);
  }

  // Start a new debounce timer.
  mouseMoveDebounceTimer = window.setTimeout(() => {
    const isInSidebar = isMouseInSidebar(event.clientX, event.clientY);

    if (!isInSidebar) {
      // Collapse again once the pointer leaves the sidebar.
      if (leaveTimeout) {
        leaveTimeout.stop();
      }
      leaveTimeout = useTimeoutFn(() => {
        if (layoutStore.sidebarMode === 'hover') {
          layoutStore.setSidebarMode('collapsed', { isUserAction: true });
        }
      }, 100);
    }
  }, 50); // 50ms debounce delay.
}

function handleMenuMouseEnter() {
  if (layoutStore.sidebarMode === 'collapsed') {
    // Clear any pending leave timer.
    if (leaveTimeout) {
      leaveTimeout.stop();
      leaveTimeout = null;
    }

    // Start the enter timer.
    enterTimeout = useTimeoutFn(() => {
      if (layoutStore.sidebarMode === 'collapsed') {
        layoutStore.setSidebarMode('hover', { isUserAction: true });
      }
    }, 50);
  }
}

function handleMenuMouseLeave() {
  // Clear the enter timer.
  if (enterTimeout) {
    enterTimeout.stop();
    enterTimeout = null;
  }

  if (layoutStore.sidebarMode === 'hover') {
    leaveTimeout = useTimeoutFn(() => {
      if (layoutStore.sidebarMode === 'hover') {
        layoutStore.setSidebarMode('collapsed', { isUserAction: true });
      }
    }, 100);
  }
}
</script>

<style lang="scss" scoped>
@use '../../styles/tokens.scss' as *;

.o-sidebar {
  --el-menu-base-level-padding: 12px;
  --el-menu-level-padding: 8px;
  --el-menu-item-height: 48px;
  --el-menu-icon-width: 20px;

  position: fixed;
  inset-inline-start: 0;
  z-index: $z-index-fixed;
  background-color: var(--el-menu-bg-color, var(--el-bg-color));
  transition: width var(--el-transition-duration) var(--el-transition-function-ease-in-out);
  border-inline-end: 1px solid var(--el-border-color-light);
  padding: 0 5px;
  overflow: hidden;

  .o-layout--with-header & {
    height: calc(100vh - var(--o-header-height, 48px));
    top: var(--o-header-height, 48px);

    @media only screen and (max-width: 991px) {
      height: calc(100vh - var(--o-header-height-mobile, 40px));
      top: var(--o-header-height-mobile, 40px);
    }
  }

  &.el-aside {
    width: var(--o-sidebar-width) !important;

    &.o-sidebar--collapsed {
      width: var(--o-sidebar-collapsed-width) !important;
    }

    &.o-sidebar--hidden {
      width: 0 !important;
    }
  }

  &__inner {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    overflow: hidden;
  }

  &__header {
    flex-shrink: 0;
  }

  &__menu {
    flex: 1;
    overflow: hidden;
    padding: 16px 0;

    :deep(.el-scrollbar) {
      height: 100%;
    }

    :deep(.el-menu) {
      border-inline-end: none;
      background-color: transparent;
    }

    :deep(.el-menu-item) {
      &:hover {
        background-color: var(--el-menu-hover-bg-color, var(--el-color-primary-light-9));
      }
      &.is-active,
      &[aria-current='page'] {
        background-color: var(--el-menu-hover-bg-color, var(--el-color-primary-light-9));
      }
    }

    :deep(.el-sub-menu__title) {
      &:hover {
        background-color: var(--el-menu-hover-bg-color, var(--el-color-primary-light-9));
      }
    }

    :deep(.el-menu-item),
    :deep(.el-sub-menu__title) {
      .el-icon {
        margin-inline-end: var(--el-menu-icon-width, 16px);
        margin-inline-start: 0;
      }
    }
  }

  &__footer {
    flex-shrink: 0;
    padding: 0;
  }

  &--collapsed {
    .o-sidebar__header,
    .o-sidebar__footer {
      padding: 0px;
      text-align: center;
    }
    .o-sidebar__user {
      justify-content: center;
      &-avatar {
        margin-inline-end: 0;
      }
      &-info {
        display: none;
      }
    }
    :deep(.el-menu) {
      .el-menu-item,
      .el-sub-menu__title {
        padding: 0 !important;
        align-items: center;
        justify-content: center;
        span {
          display: none;
        }
      }
      .el-icon {
        margin-inline: 0 !important;
      }
      .el-sub-menu .el-sub-menu__icon-arrow {
        display: none;
      }
      .el-menu-item {
        align-items: center;
        padding: 0;
        &.is-active {
          border-radius: 50%;
        }
      }
    }
  }

  &--hover {
    box-shadow: 2px 0 8px -4px rgba(0, 0, 0, 0.1);
    border-inline-end: 1px solid var(--el-border-color-lighter);
    transition:
      width var(--el-transition-duration) var(--el-transition-function-ease-in-out),
      box-shadow var(--el-transition-duration) var(--el-transition-function-ease-in-out);
  }

  &--hidden {
    width: 0 !important;
    padding: 0;
    border: none;
    overflow: hidden;
  }

  &:not(&--collapsed) {
    :deep(.el-menu) {
      > .el-menu-item,
      > .el-sub-menu > .el-sub-menu__title {
        padding-inline-start: var(--el-menu-base-level-padding, 20px) !important;
        padding-inline-end: 45px !important;
      }
      .el-sub-menu > .el-sub-menu__title {
        padding-inline-end: 15px !important;
        position: relative;
      }
      .el-menu-item {
        padding-inline-end: var(--el-menu-base-level-padding, 20px) !important;
      }
      .el-menu {
        > .el-menu-item,
        > .el-sub-menu > .el-sub-menu__title {
          padding-inline-start: calc(var(--el-menu-base-level-padding, 20px) + var(--el-menu-level-padding, 20px) * 1) !important;
        }
      }
      .el-menu .el-menu {
        > .el-menu-item,
        > .el-sub-menu > .el-sub-menu__title {
          padding-inline-start: calc(var(--el-menu-base-level-padding, 20px) + var(--el-menu-level-padding, 20px) * 2) !important;
        }
      }
      .el-menu .el-menu .el-menu {
        > .el-menu-item,
        > .el-sub-menu > .el-sub-menu__title {
          padding-inline-start: calc(var(--el-menu-base-level-padding, 20px) + var(--el-menu-level-padding, 20px) * 3) !important;
        }
      }
      .el-menu .el-menu .el-menu .el-menu {
        > .el-menu-item,
        > .el-sub-menu > .el-sub-menu__title {
          padding-inline-start: calc(var(--el-menu-base-level-padding, 20px) + var(--el-menu-level-padding, 20px) * 4) !important;
        }
      }
      .el-menu .el-menu .el-menu .el-menu .el-menu {
        .el-menu-item,
        .el-sub-menu > .el-sub-menu__title {
          padding-inline-start: calc(var(--el-menu-base-level-padding, 20px) + var(--el-menu-level-padding, 20px) * 5) !important;
        }
      }
    }
    :deep(.el-sub-menu__icon-arrow) {
      position: absolute;
      inset-inline-end: 0px;
      inset-inline-start: auto;
    }
    :deep(.el-menu--collapse .el-sub-menu__icon-arrow) {
      inset-inline-end: auto;
    }
    :deep(.el-menu--popup) {
      text-align: start;
      .el-menu-item {
        padding-inline: 20px !important;
        .el-icon {
          margin-inline-end: var(--el-menu-icon-width, 16px) !important;
          margin-inline-start: 0 !important;
        }
      }
      .el-sub-menu {
        .el-sub-menu__title {
          position: relative;
        }
        .el-sub-menu__icon-arrow {
          position: absolute;
          inset-inline-end: 0px !important;
          inset-inline-start: auto !important;
        }
      }
    }
  }
}
</style>
