// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { setActivePinia, type Pinia } from 'pinia';
import Logout from './Logout.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { useAuthStore } from '../stores/auth';

function installFakeInterval() {
  const realSet = globalThis.setInterval.bind(globalThis);
  const realClear = globalThis.clearInterval.bind(globalThis);
  const ticks: Array<() => void> = [];
  let seq = 1;
  (globalThis as unknown as { setInterval: typeof setInterval }).setInterval = ((fn: TimerHandler) => {
    ticks.push(fn as () => void);
    return seq++ as unknown as ReturnType<typeof setInterval>;
  }) as unknown as typeof setInterval;
  (globalThis as unknown as { clearInterval: typeof clearInterval }).clearInterval = ((_id: ReturnType<typeof setInterval>) => {
    ticks.length = 0;
  }) as unknown as typeof clearInterval;
  return {
    tick() {
      const fns = ticks.slice();
      for (let i = 0; i < fns.length; i++) fns[i]();
    },
    restore() {
      globalThis.setInterval = realSet;
      globalThis.clearInterval = realClear;
    },
  };
}

async function mountLogout(opts?: { failLogout?: string }) {
  const pushes: unknown[] = [];
  const global = buildPageMountGlobal({
    route: { path: '/logout', query: {} },
    router: {
      push: (to: unknown) => {
        pushes.push(to);
        return Promise.resolve(to);
      },
    },
  });
  if (opts?.failLogout) {
    const pinia = global.plugins[0] as Pinia;
    setActivePinia(pinia);
    const store = useAuthStore() as { failNextLogout?: (message: string) => void };
    store.failNextLogout?.(opts.failLogout);
  }
  const wrapper = mount(Logout as any, { global });
  for (let i = 0; i < 8; i++) await flushPromises();
  return { wrapper, pushes };
}

test('Logout.vue: success uses a login-style header and inline skip link', async () => {
  const { wrapper } = await mountLogout();
  expect(wrapper.text().includes('Signed Out Successfully')).toBe(true);
  expect(wrapper.text().includes('Redirecting to the login page in 3 seconds')).toBe(true);
  expect(wrapper.text().includes('Thank you for using our service')).toBe(false);
  const link = wrapper.find('[data-testid="logout-login-again"]');
  expect(link.exists()).toBe(true);
  expect(wrapper.text().includes('Go now')).toBe(true);
  expect(wrapper.text().includes('Go to login')).toBe(false);
  wrapper.unmount();
});

test('Logout.vue: Go now navigates to login', async () => {
  const { wrapper, pushes } = await mountLogout();
  const cta = wrapper.find('[data-testid="logout-login-again"]');
  expect(cta.exists()).toBe(true);
  await cta.trigger('click');
  await flushPromises();
  expect(pushes).toEqual(['/login']);
  wrapper.unmount();
});

test('Logout.vue: failed logout shows retry and Back to Login', async () => {
  const { wrapper, pushes } = await mountLogout({ failLogout: 'logout failed' });
  expect(wrapper.text().includes('Sign-out Failed')).toBe(true);
  expect(wrapper.text().includes('logout failed')).toBe(true);
  const retry = wrapper.find('[data-testid="logout-retry"]');
  expect(retry.exists()).toBe(true);
  expect(String((retry.element as HTMLElement).className).includes('w-full')).toBe(true);
  expect(wrapper.text().includes('Back to Login')).toBe(true);
  const back = wrapper.find('.logout-back-to-login');
  expect(back.exists()).toBe(true);
  await back.trigger('click');
  await flushPromises();
  expect(pushes).toEqual(['/login']);
  wrapper.unmount();
});

test('Logout.vue: Retry after failed logout reaches success', async () => {
  const { wrapper } = await mountLogout({ failLogout: 'logout failed' });
  expect(wrapper.text().includes('Sign-out Failed')).toBe(true);
  await wrapper.find('[data-testid="logout-retry"]').trigger('click');
  for (let i = 0; i < 8; i++) await flushPromises();
  expect(wrapper.text().includes('Signed Out Successfully')).toBe(true);
  expect(wrapper.find('[data-testid="logout-login-again"]').exists()).toBe(true);
  wrapper.unmount();
});

test('Logout.vue: countdown auto-redirects to login', async () => {
  const fake = installFakeInterval();
  try {
    const { wrapper, pushes } = await mountLogout();
    expect(wrapper.text().includes('Redirecting to the login page in 3 seconds')).toBe(true);
    fake.tick();
    await flushPromises();
    expect(wrapper.text().includes('Redirecting to the login page in 2 seconds')).toBe(true);
    fake.tick();
    await flushPromises();
    fake.tick();
    await flushPromises();
    expect(pushes).toEqual(['/login']);
    wrapper.unmount();
  } finally {
    fake.restore();
  }
});
