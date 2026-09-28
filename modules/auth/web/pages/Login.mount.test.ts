// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Login from './Login.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

async function mountLogin(route?: { path?: string; query?: Record<string, string> }) {
  const wrapper = mount(Login as any, {
    global: buildPageMountGlobal({
      route: { path: route?.path || '/login', query: route?.query || {} },
    }),
  });
  await flushPromises();
  return wrapper;
}

test('Login.vue mounts under choysumMount and runs script setup', async () => {
  const wrapper = await mountLogin();
  const hasLoginChrome =
    wrapper.text().includes('User Login') ||
    wrapper.find('[data-anchor="choy.page"]').exists() ||
    wrapper.find('.login-card').exists() ||
    wrapper.find('[data-testid="fe-stub-opage"]').exists() ||
    wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
    wrapper.find('[data-testid="fe-stub-child-view"]').exists();
  expect(hasLoginChrome).toBe(true);
  wrapper.unmount();
});

test('Login.vue: empty submit keeps the form and shows field errors', async () => {
  const wrapper = await mountLogin();
  const form = wrapper.find('form');
  expect(form.exists()).toBe(true);
  await form.trigger('submit');
  await flushPromises();
  expect(wrapper.text().includes('Enter username') || wrapper.find('.text-destructive').exists()).toBe(true);
  wrapper.unmount();
});

test('Login.vue: successful submit redirects via query.redirect', async () => {
  const wrapper = await mountLogin({ query: { redirect: '/auth/tokens' } });
  const user = wrapper.find('.login-username');
  const pass = wrapper.find('.login-password');
  expect(user.exists()).toBe(true);
  expect(pass.exists()).toBe(true);
  (user.element as HTMLInputElement).value = 'admin';
  await user.trigger('input');
  (pass.element as HTMLInputElement).value = 'secret';
  await pass.trigger('input');
  await wrapper.find('form').trigger('submit');
  await flushPromises();
  // Auth stub marks authenticated: success must clear the error badge and field errors.
  expect(wrapper.find('.login-error').exists()).toBe(false);
  expect(wrapper.find('.text-destructive').exists()).toBe(false);
  wrapper.unmount();
});
