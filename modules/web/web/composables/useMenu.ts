// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMenuStore } from '../stores/menuStore'
import type { MenuItem } from '@/core/web/menu'

const ANCESTRY_HOP_CAP = 64

/**
 * Menu navigation helpers and group expand state for Sidebar* nav.
 * Branch nodes toggle Collapsible only (no navigate). Manual expands use
 * sibling accordion; active-menu ancestry always stays open via isExpanded.
 */
export function useMenu() {
  const router = useRouter()
  const menuStore = useMenuStore()
  menuStore.bindRouteFromInjection()
  // Prefer store proxy reads over storeToRefs: setup-store computeds are not always
  // present on the object returned by storeToRefs during early shell mount.
  const activeMenu = computed(() => menuStore.activeMenu)
  const activeApp = computed(() => menuStore.activeApp)
  // Manual expand/collapse for groups without an active descendant.
  const openedSubMenus = ref(new Set<string>())

  /**
   * Navigates to a menu item or menu id.
   * Resolves folders to the first navigable leaf (L1 / breadcrumb). L2 folders
   * do not call this — they only toggle Collapsible.
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
    pruneOpenedToAncestry(targetMenuItem)

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
   * Collects folder ids on the path from `menu` up to the app root.
   * Empty-string ids are kept so pathless folders still ancestry-expand.
   */
  function ancestryFolderIds(menu: MenuItem | null | undefined): Set<string> {
    const ids = new Set<string>()
    let current: MenuItem | null | undefined = menu
    let hops = 0
    while (current && hops < ANCESTRY_HOP_CAP) {
      hops += 1
      if (current.__parent) {
        ids.add(current.__parent.id ?? '')
      }
      current = current.__parent || null
    }
    return ids
  }

  /**
   * True when `menuId` is under `ancestorId` in the menu tree.
   */
  function isUnderAncestor(menuId: string, ancestorId: string): boolean {
    if (!menuId || !ancestorId || menuId === ancestorId) return false
    let current: MenuItem | null | undefined = menuStore.getMenu(menuId)
    let hops = 0
    while (current && hops < ANCESTRY_HOP_CAP) {
      hops += 1
      if (current.__parent?.id === ancestorId) return true
      current = current.__parent || null
    }
    return false
  }

  /**
   * Drops manual expands that are not ancestors of the active leaf.
   */
  function pruneOpenedToAncestry(leaf: MenuItem) {
    const keep = ancestryFolderIds(leaf)
    const next = new Set<string>()
    for (const id of openedSubMenus.value) {
      if (keep.has(id)) next.add(id)
    }
    openedSubMenus.value = next
  }

  /**
   * Reports whether a submenu should be expanded (manual toggle or active ancestry).
   */
  function isExpanded(menuId: string): boolean {
    if (openedSubMenus.value.has(menuId)) return true

    const currentActiveMenu = activeMenu.value
    if (!currentActiveMenu) return false

    return ancestryFolderIds(currentActiveMenu).has(menuId)
  }

  /**
   * Opens a folder and closes manually opened siblings (and their subtrees).
   * Active-menu ancestry remains visible via isExpanded even if removed here.
   */
  function openSubMenu(key: string) {
    if (!key) return
    const item = menuStore.getMenu(key)
    const parent = item?.__parent
    const siblingIds = new Set(
      (parent?.children || [])
        .map((c) => c.id || '')
        .filter((id) => !!id && id !== key),
    )

    const next = new Set<string>()
    for (const id of openedSubMenus.value) {
      if (siblingIds.has(id)) continue
      let underSibling = false
      for (const siblingId of siblingIds) {
        if (isUnderAncestor(id, siblingId)) {
          underSibling = true
          break
        }
      }
      if (underSibling) continue
      next.add(id)
    }
    next.add(key)
    openedSubMenus.value = next
  }

  function closeSubMenu(key: string) {
    if (!key || !openedSubMenus.value.has(key)) return
    const next = new Set(openedSubMenus.value)
    next.delete(key)
    // Drop nested manual expands under this folder.
    for (const id of openedSubMenus.value) {
      if (isUnderAncestor(id, key)) next.delete(id)
    }
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
