// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMenuStore } from '../stores/menuStore'
import type { MenuItem } from '@/core/web/menu'

/**
 * Menu navigation helpers and group expand state for Sidebar* nav.
 * DOM rendering lives in ChoySidebarNav (SidebarMenu* + Collapsible).
 */
export function useMenu() {
  const router = useRouter()
  const menuStore = useMenuStore()
  // Prefer store proxy reads over storeToRefs: setup-store computeds are not always
  // present on the object returned by storeToRefs during early shell mount.
  const activeMenu = computed(() => menuStore.activeMenu)
  const activeApp = computed(() => menuStore.activeApp)
  // Manual expand/collapse for groups without an active descendant.
  const openedSubMenus = ref(new Set<string>())

  /**
   * Navigates to a menu item or menu id.
   */
  async function navigateTo(menuIdOrItem: string | MenuItem): Promise<boolean> {
    const menuItem = typeof menuIdOrItem === 'string' ? menuStore.getMenu(menuIdOrItem) : menuIdOrItem
    if (!menuItem) {
      console.warn('Menu item does not exist')
      return false
    }

    const targetMenuItem = menuStore.findFirstNavigableMenu(menuItem)
    if (!targetMenuItem || !targetMenuItem.path) {
      console.warn('Menu item has no navigable path')
      return false
    }

    menuStore.setActiveMenu(targetMenuItem)

    try {
      if (targetMenuItem.externalLink) {
        const target = getExternalTarget(targetMenuItem.openMode)
        window.open(targetMenuItem.path, target)
      } else {
        await router.push(targetMenuItem.path)
      }
      return true
    } catch (error) {
      console.error('Menu navigation failed:', error)
      return false
    }
  }

  /**
   * Resolves the browser target for an external menu link.
   */
  function getExternalTarget(openMode?: string): string {
    switch (openMode) {
      case 'window':
        return '_blank'
      case 'parent':
        return '_parent'
      case 'top':
        return '_top'
      default:
        return '_self'
    }
  }

  /**
   * Reports whether a submenu should be expanded (manual toggle or active ancestry).
   */
  function isExpanded(menuId: string): boolean {
    if (openedSubMenus.value.has(menuId)) return true

    const currentActiveMenu = activeMenu.value
    if (!currentActiveMenu) return false

    let current: MenuItem | null = currentActiveMenu
    while (current) {
      if (current.__parent?.id === menuId) {
        return true
      }
      current = current.__parent || null
    }
    return false
  }

  function openSubMenu(key: string) {
    if (!key || openedSubMenus.value.has(key)) return
    const next = new Set(openedSubMenus.value)
    next.add(key)
    openedSubMenus.value = next
  }

  function closeSubMenu(key: string) {
    if (!key || !openedSubMenus.value.has(key)) return
    const next = new Set(openedSubMenus.value)
    next.delete(key)
    openedSubMenus.value = next
  }

  return {
    activeMenu,
    activeApp,

    navigateTo,
    isExpanded,
    openSubMenu,
    closeSubMenu,
    setActiveMenu: menuStore.setActiveMenu,

    hasMenu: menuStore.hasMenu,
    getMenu: menuStore.getMenu,
    getMenuByPath: menuStore.getMenuByPath,
    getMenuChildren: menuStore.getMenuChildren,
    getMenuParent: menuStore.getMenuParent,
    getMenus: menuStore.getMenus,
  }
}
