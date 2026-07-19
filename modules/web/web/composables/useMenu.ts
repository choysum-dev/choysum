// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useRouter } from 'vue-router';
import { ElMenu, ElSubMenu, ElMenuItem, ElIcon, ElEmpty } from 'element-plus';
import { QuestionFilled } from '@element-plus/icons-vue';
import { BookmarkBorderOutlined } from '@vicons/material';
import { useI18n } from 'vue-i18n';
import { useMenuStore } from '../stores/menuStore';
import type { MenuItem } from '@/core/web/menu';
import { translateTerm, createTranslate } from '../i18n';

const { _t: _tMenu } = createTranslate('web', { scope: 'web/composables/useMenu' });

/**
 * Provides menu navigation helpers and render utilities backed by the menu store.
 */
export function useMenu() {
  const router = useRouter();
  const composer = useI18n({ useScope: 'global' });

  const menuStore = useMenuStore();
  const { activeMenu, activeApp } = storeToRefs(menuStore);

  /**
   * Navigates to a menu item or menu id.
   */
  async function navigateTo(menuIdOrItem: string | MenuItem): Promise<boolean> {
    const menuItem = typeof menuIdOrItem === 'string' ? menuStore.getMenu(menuIdOrItem) : menuIdOrItem;
    if (!menuItem) {
      console.warn('Menu item does not exist');
      return false;
    }

    const targetMenuItem = menuStore.findFirstNavigableMenu(menuItem);
    if (!targetMenuItem || !targetMenuItem.path) {
      console.warn('Menu item has no navigable path');
      return false;
    }

    menuStore.setActiveMenu(targetMenuItem);

    try {
      if (targetMenuItem.externalLink) {
        const target = getExternalTarget(targetMenuItem.openMode);
        window.open(targetMenuItem.path, target);
      } else {
        await router.push(targetMenuItem.path);
      }
      return true;
    } catch (error) {
      console.error('Menu navigation failed:', error);
      return false;
    }
  }

  /**
   * Resolves the browser target for an external menu link.
   */
  function getExternalTarget(openMode?: string): string {
    switch (openMode) {
      case 'window':
        return '_blank';
      case 'parent':
        return '_parent';
      case 'top':
        return '_top';
      default:
        return '_self';
    }
  }

  /**
   * Reports whether a submenu should be expanded for the active menu.
   */
  function isExpanded(menuId: string): boolean {
    const currentActiveMenu = activeMenu.value;
    if (!currentActiveMenu) return false;

    let current: MenuItem | null = currentActiveMenu;
    while (current) {
      if (current.__parent?.id === menuId) {
        return true;
      }
      current = current.__parent || null;
    }
    return false;
  }

  /**
   * Renders a menu icon node, optionally falling back to a default icon.
   */
  const renderIcon = (icon: any, useDefault = false, defaultIcon: any = QuestionFilled) => {
    if (!icon && useDefault) {
      return h(ElIcon, {}, () => h(defaultIcon));
    } else if (!icon) {
      return null;
    }
    return h(ElIcon, {}, () => h(icon));
  };

  /**
   * Renders a menu tree into Element Plus menu items.
   */
  const renderMenuItems = (
    menuItems: MenuItem[],
    options: {
      onItemClick?: (item: MenuItem) => void;
      onItemSelect?: (key: string, item: MenuItem) => void;
      onSubMenuOpen?: (key: string) => void;
      onSubMenuClose?: (key: string) => void;
      defaultIcon?: any;
      useDefaultIcon?: boolean;
    } = {}
  ) => {
    const {
      onItemClick = item => navigateTo(item),
      onItemSelect,
      onSubMenuOpen,
      onSubMenuClose,
      defaultIcon = BookmarkBorderOutlined,
      useDefaultIcon = false,
    } = options;

    return menuItems
      .filter(item => !item.hidden)
      .map(item => {
        const hasChildren = item.children && item.children.length > 0;
        const isActive = activeMenu.value?.id === item.id;
        const props = {
          index: item.id || '',
          key: item.id || item.path,
          disabled: item.disabled,
        };

        if (hasChildren) {
          return h(
            ElSubMenu,
            {
              ...props,
              class: { 'is-expanded': isExpanded(item.id) },
              onOpen: () => onSubMenuOpen?.(item.id || ''),
              onClose: () => onSubMenuClose?.(item.id || ''),
            },
            {
              title: () => [
                renderIcon(item.icon, useDefaultIcon, defaultIcon),
                h('span', {}, translateTerm(composer, item.titleText, item.title)),
              ],
              default: () => renderMenuItems(item.children || [], options),
            }
          );
        }

        return h(
          ElMenuItem,
          {
            ...props,
            class: { 'is-active': isActive },
            onClick: () => onItemClick(item),
            onSelect: () => onItemSelect?.(item.id || '', item),
          },
          {
            default: () => [
              renderIcon(item.icon, useDefaultIcon, defaultIcon),
              h('span', {}, translateTerm(composer, item.titleText, item.title)),
            ],
          }
        );
      });
  };

  /**
   * Returns the menu items for the currently active application.
   */
  const appMenuItems = computed(() => {
    return activeApp.value?.children || [];
  });

  /**
   * Renders a complete menu widget from store or provided items.
   */
  const renderMenu = (
    options: {
      items?: MenuItem[];
      defaultActive?: string;
      uniqueOpened?: boolean;
      className?: string;
      onItemClick?: (item: MenuItem) => void;
      onItemSelect?: (key: string, item: MenuItem) => void;
      onSubMenuOpen?: (key: string) => void;
      onSubMenuClose?: (key: string) => void;
      defaultIcon?: any;
      useDefaultIcon?: boolean;
      emptyText?: string;
      menuProps?: Record<string, any>;
    } = {}
  ) => {
    const {
      defaultActive = activeMenu.value?.id,
      uniqueOpened = true,
      className = '',
      onItemClick,
      onItemSelect,
      onSubMenuOpen,
      onSubMenuClose,
      defaultIcon,
      useDefaultIcon = false,
      emptyText = _tMenu('No menus available'),
      menuProps = {},
    } = options;

    const displayItems = computed(() => {
      if (options.items) return options.items;
      if (activeApp.value && appMenuItems.value.length) {
        return appMenuItems.value;
      }
      return menuStore.getMenus();
    });

    if (!displayItems.value.length) {
      return h(ElEmpty, { description: emptyText });
    }

    return h(
      ElMenu,
      {
        defaultActive,
        collapse: false,
        collapseTransition: false,
        uniqueOpened,
        class: className ? `o-menu ${className}` : 'o-menu',
        ...menuProps,
      },
      {
        default: () =>
          renderMenuItems(displayItems.value, {
            onItemClick,
            onItemSelect,
            onSubMenuOpen,
            onSubMenuClose,
            defaultIcon,
            useDefaultIcon,
          }),
      }
    );
  };

  /**
   * Renders the application drawer menu.
   */
  const renderAppDrawerMenu = (
    options: {
      onItemClick?: (item: MenuItem) => void;
      className?: string;
    } = {}
  ) => {
    const { onItemClick, className = 'o-app-menu' } = options;
    return renderMenu({
      items: menuStore.getMenus(),
      defaultActive: activeMenu.value?.id,
      className,
      uniqueOpened: true,
      onItemClick: onItemClick || (item => navigateTo(item)),
    });
  };

  /**
   * Renders the sidebar menu for the current application.
   */
  const renderSidebarMenu = (
    options: {
      onItemClick?: (item: MenuItem) => void;
      onSubMenuOpen?: (key: string) => void;
      onSubMenuClose?: (key: string) => void;
      useDefaultIcon?: boolean;
      uniqueOpened?: boolean;
      defaultIcon?: any;
    } = {}
  ) => {
    const { uniqueOpened = false, onItemClick, onSubMenuOpen, onSubMenuClose, useDefaultIcon = true, defaultIcon = BookmarkBorderOutlined } = options;

    return renderMenu({
      defaultActive: activeMenu.value?.id,
      uniqueOpened,
      onItemClick: onItemClick || (item => navigateTo(item)),
      onSubMenuOpen,
      onSubMenuClose,
      useDefaultIcon,
      defaultIcon,
    });
  };

  return {
    activeMenu,
    activeApp,

    navigateTo,
    isExpanded,
    setActiveMenu: menuStore.setActiveMenu,

    renderIcon,
    renderMenuItems,
    renderMenu,
    renderAppDrawerMenu,
    renderSidebarMenu,

    hasMenu: menuStore.hasMenu,
    getMenu: menuStore.getMenu,
    getMenuByPath: menuStore.getMenuByPath,
    getMenuChildren: menuStore.getMenuChildren,
    getMenuParent: menuStore.getMenuParent,
    getMenus: menuStore.getMenus,
  };
}
