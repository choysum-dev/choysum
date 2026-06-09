// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseScopeManager } from './base';

export class GlobalScopeManager extends BaseScopeManager {
  private static _inst: GlobalScopeManager | null = null;
  static instance() {
    return (this._inst ??= new GlobalScopeManager());
  }
  private constructor() {
    super('global');
  }
  protected getCurrentScopeId(): string | undefined {
    return 'global';
  }
}
