// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseScopeManager } from './base';
import { getCurrentScope, onScopeDispose } from 'vue';

export class ComponentScopeManager extends BaseScopeManager {
  private readonly sid: string;
  constructor(scopeId?: string) {
    super('component');
    this.sid = scopeId || `cmp_${Math.random().toString(36).slice(2)}`;
    if (getCurrentScope()) {
      onScopeDispose(() => this.destroyScope(this.sid));
    }
  }
  protected getCurrentScopeId(): string | undefined {
    return this.sid;
  }
}
