// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import app from '@/web/web';
import type { ChoysumWebApp } from '@/core/web/application';
import { registerChoyGalleryRoute } from '@/web/web';

/**
 * Registers the Choy UI gallery router with the shared web application.
 * Kit source lives under modules/web; this shell only boots the routes.
 */
export function setupApp(webApp: ChoysumWebApp): void {
  registerChoyGalleryRoute(webApp);
}

/**
 * Choy UI web application instance (gallery / dogfood registration host).
 */
const choyUiApp: ChoysumWebApp = app.setup(setupApp);
export default choyUiApp;
