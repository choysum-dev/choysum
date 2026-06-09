// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import app from '@/web/web';
import type { ChoysumWebApp } from '@/core/web/application';
import { setupRouter } from './route';
import { setupAppMenu } from './menu';

export function setupApp(app: ChoysumWebApp): void {
  setupRouter(app);
  setupAppMenu(app);
}

const metaApp: ChoysumWebApp = app.setup(setupApp);
export default metaApp;
