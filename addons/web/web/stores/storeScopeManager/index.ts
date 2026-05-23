// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GlobalScopeManager } from './global';
import { MenuScopeManager } from './menu';
import { ComponentScopeManager } from './component';

/**
 * Returns the shared scope managers used by web stores and components.
 */
export function useScopeManager() {
  const globalScopeManager = GlobalScopeManager.instance();
  const menuScopeManager = MenuScopeManager.instance();

  /**
   * Creates a fresh component scope manager per caller.
   */
  const componentScopeManager = () => new ComponentScopeManager();

  return {
    globalScopeManager,
    menuScopeManager,
    componentScopeManager,
  };
}
