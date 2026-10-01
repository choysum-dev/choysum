// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Runtime wrapper around {@link resolveDefaultLandPath} for router redirects
 * and brand click. Reads the live menu tree when Pinia/menu injection is ready;
 * falls back to Module Board when stores are unavailable (early boot / unit mounts).
 */
import { useMenuStore } from '../stores/menuStore';
import { resolveDefaultLandPath, MODULE_BOARD_PATH } from './resolveDefaultLandPath';

export function resolveRuntimeDefaultLandPath(): string {
  try {
    const menus = useMenuStore().getMenus?.() ?? [];
    return resolveDefaultLandPath({ menus });
  } catch {
    return MODULE_BOARD_PATH;
  }
}
