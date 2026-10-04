// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Creates the Choysum web application instance.
 * Uses core/web/application to create the app and register Pinia and router plugins.
 */
import { createApp } from '@/core/web/application';
import { setupApp } from './appSetup';

import App from './App.vue';

import 'vue-virtual-scroller/dist/vue-virtual-scroller.css';
import 'vue-sonner/style.css';
import './styles/tokens.css';
import './styles/choy-tailwind.generated.css';
import './styles/index.css';

const app = createApp(App).setup(setupApp);

export default app;
