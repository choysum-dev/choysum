// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { App } from 'vue';
import { vAction } from './action';

export function registerGlobalDirectives(app: App): void {
  app.directive('action', vAction);
}
