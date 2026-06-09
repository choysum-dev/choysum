// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import app from '@/web/web';
import type { ChoysumWebApp } from '@/core/web/application';

/**
 * Initializes the partner commercial web module. UI extension registration is provided by side-effect imports.
 */
export function setupApp(_: ChoysumWebApp): void {}

/**
 * Partner commercial web application instance.
 */
const partnerCommercialApp: ChoysumWebApp = app.setup(setupApp);
export default partnerCommercialApp;
