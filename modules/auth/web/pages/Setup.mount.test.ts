// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { setActivePinia, type Pinia } from 'pinia';
import Setup from './Setup.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';

async function mountSetup(opts?: {
  path?: string;
  query?: Record<string, string>;
  failSetup?: string;
  failLogin?: string;
}) {
  const replaces: unknown[] = [];
  const global = buildPageMountGlobal({
    route: { path: opts?.path || '/setup', query: opts?.query || {} },
    router: {
      replace: (to: unknown) => {
        replaces.push(to);
        return Promise.resolve(to);
      },
    },
  });
  const pinia = global.plugins[0] as Pinia;
  setActivePinia(pinia);
  const store = useAuthStore() as {
    completeInitialSetup: (username: string, password: string) => Promise<unknown>;
    login: (username: string, password: string) => Promise<unknown>;
  };
  if (opts?.failSetup) {
    const message = opts.failSetup;
    store.completeInitialSetup = async () => {
      throw new ChoysumError({ domain: 'auth', code: 'VALIDATION_FAILED', message });
    };
  }
  if (opts?.failLogin) {
    const message = opts.failLogin;
    store.login = async () => {
      throw new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message });
    };
  }
  const wrapper = mount(Setup as any, { global });
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

async function fillField(wrapper: { find: (sel: string) => any }, selector: string, value: string) {
  const input = fieldInput(wrapper, selector);
  const el = input.element as HTMLInputElement;
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
  el.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
  el.dispatchEvent(new Event('blur', { bubbles: true, cancelable: true }));
  await flushPromises();
}

function submitForm(wrapper: { find: (sel: string) => any }) {
  const form = wrapper.find('form');
  (form.element as HTMLFormElement).dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

async function afterSubmit() {
  for (let i = 0; i < 8; i++) await flushPromises();
}

async function fillValidCredentials(wrapper: { find: (sel: string) => any }) {
  await fillField(wrapper, '.setup-username', 'admin');
  await fillField(wrapper, '.setup-password', 'secret1');
  await fillField(wrapper, '.setup-confirm', 'secret1');
}

test('Setup.vue mounts under choysumMount and shows admin credential fields', async () => {
  const { wrapper } = await mountSetup();
  expect(fieldInput(wrapper, '.setup-username').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-password').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-confirm').exists()).toBe(true);
  expect(wrapper.text().includes('Initial Setup') || wrapper.text().includes('初始设置')).toBe(true);
  wrapper.unmount();
});

test('Setup.vue: empty submit keeps the form and shows field errors', async () => {
  const { wrapper, replaces } = await mountSetup();
  submitForm(wrapper);
  await afterSubmit();
  expect(
    wrapper.text().includes('Enter username') ||
      wrapper.find('.text-destructive').exists() ||
      wrapper.find('.text-danger').exists(),
  ).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Setup.vue: valid submit completes setup, logs in, and redirects', async () => {
  const { wrapper, replaces } = await mountSetup({ query: { redirect: '/auth/users' } });
  await fillValidCredentials(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.find('.setup-error').exists()).toBe(false);
  expect(replaces).toEqual(['/auth/users']);
  wrapper.unmount();
});

test('Setup.vue: setup RPC failure stays on the page with an error', async () => {
  const { wrapper, replaces } = await mountSetup({ failSetup: 'already done' });
  await fillValidCredentials(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  const alert = wrapper.find('.setup-error');
  expect(alert.exists()).toBe(true);
  expect(wrapper.text().includes('already done')).toBe(true);
  expect(replaces).toEqual([]);
  const close = (alert.element as HTMLElement).querySelector('button');
  expect(!!close).toBe(true);
  close!.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
  await flushPromises();
  expect(wrapper.find('.setup-error').exists()).toBe(false);
  wrapper.unmount();
});

test('Setup.vue: login failure after setup navigates to login', async () => {
  const { wrapper, replaces } = await mountSetup({
    query: { redirect: '/auth/users' },
    failLogin: 'bad login',
  });
  await fillValidCredentials(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(replaces.length).toBeGreaterThan(0);
  expect(replaces[replaces.length - 1]).toEqual({
    name: 'login',
    query: { redirect: '/auth/users' },
  });
  wrapper.unmount();
});
