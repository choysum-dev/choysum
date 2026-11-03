// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Setup from './Setup.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

async function mountSetup() {
  const replaces: unknown[] = [];
  const wrapper = mount(Setup as any, {
    global: buildPageMountGlobal({
      route: { path: '/setup', query: {} },
      router: {
        replace: (to: unknown) => {
          replaces.push(to);
          return Promise.resolve(to);
        },
      },
    }),
  });
  await flushPromises();
  return { wrapper, replaces };
}

function fieldInput(wrapper: { find: (sel: string) => any }, selector: string) {
  const root = wrapper.find(selector);
  if (!root.exists()) return root;
  const el = root.element as HTMLElement;
  if (String(el.tagName || '').toLowerCase() === 'input') return root;
  const inner = typeof el.querySelector === 'function' ? el.querySelector('input') : null;
  if (inner) {
    return {
      exists: () => true,
      element: inner,
      trigger: async (type: string) => {
        inner.dispatchEvent(new Event(type, { bubbles: true }));
      },
    };
  }
  return root;
}

test('Setup.vue mounts under choysumMount and shows admin credential fields', async () => {
  const { wrapper } = await mountSetup();
  expect(fieldInput(wrapper, '.setup-username').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-password').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-confirm').exists()).toBe(true);
  expect(wrapper.text().includes('Initial Setup') || wrapper.text().includes('初始设置')).toBe(true);
});
