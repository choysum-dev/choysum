// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';
import { canMenu, type PermissionState } from '@/auth/web/permission';

/**
 * Company-scope context used for menu permission evaluation.
 */
export type PermissionCtx = {
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
};

type MenuPermissionMeta = {
  permissionMode?: 'hide' | 'disable';
  __permBaseHidden?: boolean;
  __permBaseDisabled?: boolean;
};

/**
 * Normalize menu metadata so permission state can be cached in place.
 */
function getMeta(item: MenuItem): MenuPermissionMeta {
  const meta = (item.meta ?? {}) as MenuPermissionMeta;
  item.meta = meta as any;
  return meta;
}

/**
 * Apply permission projection to a menu item and its descendants.
 */
function applyToItem(item: MenuItem, state: PermissionState | null | undefined, ctx: PermissionCtx): void {
  const meta = getMeta(item);

  if (meta.__permBaseHidden === undefined) meta.__permBaseHidden = !!item.hidden;
  if (meta.__permBaseDisabled === undefined) meta.__permBaseDisabled = !!item.disabled;

  const baseHidden = !!meta.__permBaseHidden;
  const baseDisabled = !!meta.__permBaseDisabled;

  // Process children first so parent visibility can react to their final state.
  if (Array.isArray(item.children) && item.children.length > 0) {
    for (const child of item.children) {
      applyToItem(child, state, ctx);
    }
  }

  const allowed = canMenu(item.id, state, ctx);
  const mode = meta.permissionMode ?? 'hide';

  if (mode === 'disable') {
    item.hidden = baseHidden;
    item.disabled = baseDisabled || !allowed;
  } else {
    item.hidden = baseHidden || !allowed;
    item.disabled = baseDisabled;
  }

  // Hide non-navigable parents when every child is hidden.
  if (!item.hidden && !item.path && Array.isArray(item.children) && item.children.length > 0) {
    const allHidden = item.children.every(c => !!c.hidden);
    if (allHidden) item.hidden = true;
  }
}

/**
 * Recompute permission-derived hidden and disabled states for root menus.
 */
export function applyPermissionToMenus(rootMenus: MenuItem[], state: PermissionState | null | undefined, ctx: PermissionCtx): void {
  for (const item of rootMenus) {
    applyToItem(item, state, ctx);
  }
}
