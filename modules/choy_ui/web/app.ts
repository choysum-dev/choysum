// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import app from '@/web/web';
import type { ChoysumWebApp } from '@/core/web/application';
import { setupRouter } from './route';

/**
 * Registers the Choy UI gallery router with the shared web application.
 */
export function setupApp(webApp: ChoysumWebApp): void {
  setupRouter(webApp);
}

/**
 * Choy UI web application instance (gallery / dogfood only during isolation).
 */
const choyUiApp: ChoysumWebApp = app.setup(setupApp);
export default choyUiApp;
