// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, computed, ref, type VNode } from 'vue';
import { storeToRefs } from 'pinia';
import { useRouter } from 'vue-router';
import { Bookmark, CircleHelp } from 'lucide-vue-next';
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
  // Manual expand/collapse for groups without an active descendant.
  const openedSubMenus = ref(new Set<string>());

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
   * Reports whether a submenu should be expanded (manual toggle or active ancestry).
   */
  function isExpanded(menuId: string): boolean {
    if (openedSubMenus.value.has(menuId)) return true;

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

  function openSubMenu(key: string) {
    if (!key || openedSubMenus.value.has(key)) return;
    const next = new Set(openedSubMenus.value);
    next.add(key);
    openedSubMenus.value = next;
  }

  function closeSubMenu(key: string) {
    if (!key || !openedSubMenus.value.has(key)) return;
    const next = new Set(openedSubMenus.value);
    next.delete(key);
    openedSubMenus.value = next;
  }

  /**
   * Renders a menu icon node, optionally falling back to a default icon.
   */
  const renderIcon = (icon: any, useDefault = false, defaultIcon: any = CircleHelp): VNode | null => {
    const resolved = icon || (useDefault ? defaultIcon : null);
    if (!resolved) return null;
    return h('span', { class: 'choy-menu__icon inline-flex shrink-0' }, [h(resolved)]);
  };

  /**
   * Renders a menu tree into semantic list items.
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
  ): VNode[] => {
    const {
      onItemClick = item => navigateTo(item),
      onItemSelect,
      onSubMenuOpen,
      onSubMenuClose,
      defaultIcon = Bookmark,
      useDefaultIcon = false,
    } = options;

    return menuItems
      .filter(item => !item.hidden)
      .map(item => {
        const hasChildren = item.children && item.children.length > 0;
        const isActive = activeMenu.value?.id === item.id;
        const itemId = item.id || item.path || '';

        if (hasChildren) {
          const expanded = isExpanded(item.id);
          return h(
            'li',
            {
              key: itemId,
              class: ['choy-menu__sub', { 'is-expanded': expanded }],
              role: 'none',
            },
            [
              h(
                'button',
                {
                  type: 'button',
                  class: 'choy-menu__sub-title',
                  'aria-expanded': expanded ? 'true' : 'false',
                  disabled: item.disabled || undefined,
                  onClick: () => {
                    const key = item.id || '';
                    if (expanded) {
                      closeSubMenu(key);
                      onSubMenuClose?.(key);
                    } else {
                      openSubMenu(key);
                      onSubMenuOpen?.(key);
                    }
                  },
                },
                [
                  renderIcon(item.icon, useDefaultIcon, defaultIcon),
                  h('span', {}, translateTerm(composer, item.titleText, item.title)),
                ]
              ),
              expanded
                ? h(
                    'ul',
                    { class: 'choy-menu__sub-list', role: 'group' },
                    renderMenuItems(item.children || [], options)
                  )
                : null,
            ]
          );
        }

        return h(
          'li',
          { key: itemId, role: 'none' },
          h(
            'button',
            {
              type: 'button',
              class: ['choy-menu__item', { 'is-active': isActive }],
              disabled: item.disabled || undefined,
              onClick: () => {
                onItemClick(item);
                onItemSelect?.(item.id || '', item);
              },
            },
            [
              renderIcon(item.icon, useDefaultIcon, defaultIcon),
              h('span', {}, translateTerm(composer, item.titleText, item.title)),
            ]
          )
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
      className = '',
      onItemClick,
      onItemSelect,
      onSubMenuOpen,
      onSubMenuClose,
      defaultIcon,
      useDefaultIcon = false,
      emptyText = _tMenu('No menus available'),
    } = options;

    const displayItems = computed(() => {
      if (options.items) return options.items;
      if (activeApp.value && appMenuItems.value.length) {
        return appMenuItems.value;
      }
      return menuStore.getMenus();
    });

    if (!displayItems.value.length) {
      return h('p', { class: 'choy-menu__empty' }, emptyText);
    }

    return h(
      'ul',
      {
        class: className ? `choy-menu ${className}` : 'choy-menu',
        role: 'menu',
      },
      renderMenuItems(displayItems.value, {
        onItemClick,
        onItemSelect,
        onSubMenuOpen,
        onSubMenuClose,
        defaultIcon,
        useDefaultIcon,
      })
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
    const { onItemClick, className = 'choy-app-menu' } = options;
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
      /** Icon-rail mode: hide label text visually; keep accessible name via title. */
      collapsed?: boolean;
    } = {}
  ) => {
    const {
      onItemClick,
      onSubMenuOpen,
      onSubMenuClose,
      useDefaultIcon = true,
      defaultIcon = Bookmark,
      collapsed = false,
    } = options;

    return renderMenu({
      defaultActive: activeMenu.value?.id,
      uniqueOpened: false,
      className: collapsed ? 'choy-menu--collapsed' : '',
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
