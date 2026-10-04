// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Logout from './Logout.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

async function mountLogout() {
  const pushes: unknown[] = [];
  const wrapper = mount(Logout as any, {
    global: buildPageMountGlobal({
      route: { path: '/logout', query: {} },
      router: {
        push: (to: unknown) => {
          pushes.push(to);
          return Promise.resolve(to);
        },
      },
    }),
  });
  for (let i = 0; i < 8; i++) await flushPromises();
  return { wrapper, pushes };
}

test('Logout.vue: success uses a login-style header and full-width CTA', async () => {
  const { wrapper } = await mountLogout();
  expect(wrapper.text().includes('Signed Out Successfully')).toBe(true);
  expect(wrapper.text().includes('Redirecting to the login page in 3 seconds')).toBe(true);
  expect(wrapper.text().includes('Thank you for using our service')).toBe(false);
  const cta = wrapper.find('[data-testid="logout-login-again"]');
  expect(cta.exists()).toBe(true);
  expect(String((cta.element as HTMLElement).className).includes('w-full')).toBe(true);
  wrapper.unmount();
});

test('Logout.vue: Log In Again navigates to login', async () => {
  const { wrapper, pushes } = await mountLogout();
  const cta = wrapper.find('[data-testid="logout-login-again"]');
  expect(cta.exists()).toBe(true);
  await cta.trigger('click');
  await flushPromises();
  expect(pushes).toEqual(['/login']);
  wrapper.unmount();
});
