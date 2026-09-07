// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { mount } from '@choysum/test-utils';
import HostCounter from './HostCounter.vue';

const w = mount(HostCounter);
const btn = w.find('.bump');
const marker = w.find('.marker');

(globalThis as any).__hostResult = {
  hasBtn: btn.exists(),
  markerText: marker.text(),
  btnTextBefore: btn.text(),
};

btn.trigger('click').then(() => {
  (globalThis as any).__hostResult.btnTextAfter = w.find('.bump').text();
  (globalThis as any).__hostResult.ready = true;
});
