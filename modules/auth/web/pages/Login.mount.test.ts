// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Login from './Login.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

async function mountLogin(opts?: {
  path?: string;
  query?: Record<string, string>;
}) {
  const replaces: unknown[] = [];
  const wrapper = mount(Login as any, {
    global: buildPageMountGlobal({
      route: { path: opts?.path || '/login', query: opts?.query || {} },
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

test('Login.vue mounts under choysumMount and runs script setup', async () => {
  const { wrapper } = await mountLogin();
  expect(fieldInput(wrapper, '.login-username').exists()).toBe(true);
  expect(fieldInput(wrapper, '.login-password').exists()).toBe(true);
  // Required rules must not paint until blur or submit.
  expect(wrapper.text().includes('Enter username')).toBe(false);
  expect(wrapper.text().includes('Enter password')).toBe(false);
  wrapper.unmount();
});

test('Login.vue: empty submit keeps the form and shows field errors', async () => {
  const { wrapper, replaces } = await mountLogin();
  const form = wrapper.find('form');
  expect(form.exists()).toBe(true);
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('Enter username') || wrapper.find('.text-destructive').exists() || wrapper.find('.text-danger').exists()).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Login.vue: filled native submit does not show required field errors', async () => {
  const { wrapper, replaces } = await mountLogin({ query: { redirect: '/auth/tokens' } });
  expect(fieldInput(wrapper, '.login-username').exists()).toBe(true);
  expect(fieldInput(wrapper, '.login-password').exists()).toBe(true);
  await fillField(wrapper, '.login-username', 'admin');
  await fillField(wrapper, '.login-password', 'secret');
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('Enter username')).toBe(false);
  // Page error swaps into the card header only when present; success keeps description.
  expect(wrapper.find('.login-error').exists()).toBe(false);
  expect(wrapper.text().includes('Login failed')).toBe(false);
  const fieldErrors = (wrapper.element as HTMLElement).querySelectorAll('.choy-field-base__error');
  expect(fieldErrors.length).toBeGreaterThan(0);
  for (let i = 0; i < fieldErrors.length; i++) {
    expect(fieldErrors[i]!.getAttribute('role')).toBe(null);
    expect(fieldErrors[i]!.className.includes('invisible')).toBe(true);
  }
  expect(replaces).toEqual(['/auth/tokens']);
  wrapper.unmount();
});
