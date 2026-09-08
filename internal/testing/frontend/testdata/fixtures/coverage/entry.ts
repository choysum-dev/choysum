// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { createApp } from 'vue';
import App from './SpikeCounter.vue';

const el = document.createElement('div');
document.body.appendChild(el);
createApp(App).mount(el);
