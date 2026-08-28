// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Creates the Choysum web application instance.
 * Uses core/web/application to create the app and register Pinia and router plugins.
 */
import { createApp } from '@/core/web/application';
import { setupApp } from './app_setup';

import App from './App.vue';

import 'normalize.css/normalize.css';
import 'element-plus/dist/index.css';
import 'element-plus/theme-chalk/display.css';
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css';
import './styles/index.scss';

const app = createApp(App).setup(setupApp);

export default app;
